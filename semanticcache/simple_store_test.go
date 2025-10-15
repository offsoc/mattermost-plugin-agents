// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semanticcache

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDB creates a test database and returns a connection to it.
func getRootDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://mmuser:mostest@localhost:5432/postgres?sslmode=disable"
}

func testDB(t *testing.T) *sql.DB {
	rootDSN := getRootDSN()
	rootDB, err := sqlx.Connect("postgres", rootDSN)
	require.NoError(t, err, "Failed to connect to PostgreSQL. Is PostgreSQL running?")
	defer rootDB.Close()

	// Check if pgvector extension is available
	var hasVector bool
	err = rootDB.Get(&hasVector, "SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = 'vector')")
	require.NoError(t, err, "Failed to check for vector extension")
	require.True(t, hasVector, "pgvector extension not available in PostgreSQL. Please install it to run these tests.")

	// Create a unique database name with a timestamp
	dbName := fmt.Sprintf("semanticcache_test_%d", model.GetMillis())

	// Create the test database
	_, err = rootDB.Exec("CREATE DATABASE " + dbName)
	require.NoError(t, err, "Failed to create test database")
	t.Logf("Created test database: %s", dbName)

	// Connect to the new database
	testDSN := fmt.Sprintf("postgres://mmuser:mostest@localhost:5432/%s?sslmode=disable", dbName)
	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		// Try to clean up the database even if connection fails
		_, _ = rootDB.Exec("DROP DATABASE " + dbName)
		require.NoError(t, err, "Failed to connect to test database")
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		db.Close()
		_, _ = rootDB.Exec("DROP DATABASE " + dbName)
		require.NoError(t, err, "Failed to ping test database")
	}

	// Store the database name for cleanup
	t.Setenv("SEMANTICCACHE_TEST_DB", dbName)

	return db
}

// dropTestDB drops the temporary test database
func dropTestDB(t *testing.T) {
	dbName := os.Getenv("SEMANTICCACHE_TEST_DB")
	if dbName == "" {
		return
	}

	rootDSN := getRootDSN()
	rootDB, err := sqlx.Connect("postgres", rootDSN)
	require.NoError(t, err, "Failed to connect to PostgreSQL to drop test database")
	defer rootDB.Close()

	// Drop the test database
	if !t.Failed() {
		_, err = rootDB.Exec("DROP DATABASE " + dbName)
		require.NoError(t, err, "Failed to drop test database")
	}
}

// cleanupDB cleans up test database state and drops the database
func cleanupDB(t *testing.T, db *sql.DB) {
	if db == nil {
		return
	}

	err := db.Close()
	require.NoError(t, err, "Failed to close database connection")

	dropTestDB(t)
}

// mockEmbedder creates a deterministic embedder for testing
func mockEmbedder(text string) ([]float32, error) {
	// Create deterministic embeddings based on text content
	// Use configured dimensions to match database schema
	dimensions := getEmbeddingDimensions()
	embedding := make([]float32, dimensions)
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}

	// Fill the embedding with deterministic values based on the hash
	for i := range embedding {
		embedding[i] = float32((hash*i+i)%1000) / 1000.0
	}

	return embedding, nil
}

// failingEmbedder always returns an error
func failingEmbedder(text string) ([]float32, error) {
	return nil, fmt.Errorf("embedder error")
}

func TestNewSimpleCache(t *testing.T) {
	t.Run("creates enabled cache with default values when env vars not set", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := NewSimpleCache(db)
		require.NotNil(t, cache)
		assert.Equal(t, db, cache.db)
		assert.Equal(t, float32(0.85), cache.threshold)
		assert.True(t, cache.enabled) // Default is true unless VECTOR_CACHE_ENABLED == "0"
	})

	t.Run("creates enabled cache when env vars are set", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		t.Setenv("VECTOR_CACHE_ENABLED", "1")
		t.Setenv("VECTOR_CACHE_THRESHOLD", "0.90")

		cache := NewSimpleCache(db)
		require.NotNil(t, cache)
		assert.Equal(t, float32(0.90), cache.threshold)
		assert.True(t, cache.enabled)

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'semantic_cache'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("creates disabled cache when VECTOR_CACHE_ENABLED=0", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		t.Setenv("VECTOR_CACHE_ENABLED", "0")

		cache := NewSimpleCache(db)
		require.NotNil(t, cache)
		assert.Equal(t, db, cache.db)
		assert.Equal(t, float32(0.85), cache.threshold)
		assert.False(t, cache.enabled) // Explicitly disabled
	})

	t.Run("creates disabled cache when db is nil", func(t *testing.T) {
		cache := NewSimpleCache(nil)
		require.NotNil(t, cache)
		assert.Nil(t, cache.db)
		assert.False(t, cache.enabled)               // Should be disabled when db is nil
		assert.Equal(t, float32(0), cache.threshold) // Zero value
	})

	t.Run("disables cache when schema setup fails", func(t *testing.T) {
		// Use a closed database to force schema setup failure
		db := testDB(t)
		db.Close()

		t.Setenv("VECTOR_CACHE_ENABLED", "1")

		cache := NewSimpleCache(db)
		require.NotNil(t, cache)
		assert.False(t, cache.enabled) // Should be disabled due to setup failure
	})
}

func TestLookup(t *testing.T) {
	t.Run("returns false when cache is disabled", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:      db,
			enabled: false,
		}

		response, found := cache.Lookup("test_tool", "test query")
		assert.False(t, found)
		assert.Empty(t, response)
	})

	t.Run("returns false when embedder fails", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:       db,
			enabled:  true,
			embedder: failingEmbedder,
		}

		response, found := cache.Lookup("test_tool", "test query")
		assert.False(t, found)
		assert.Empty(t, response)
	})

	t.Run("returns false when no similar entries found", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.85,
			embedder:  mockEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		response, found := cache.Lookup("test_tool", "test query")
		assert.False(t, found)
		assert.Empty(t, response)
	})

	t.Run("returns cached response when similar entry exists", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.5, // Lower threshold for easier testing
			embedder:  mockEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		cache.Store("test_tool", "hello world", "cached response")

		// Lookup with similar query
		response, found := cache.Lookup("test_tool", "hello world")
		assert.True(t, found)
		assert.Equal(t, "cached response", response)
	})

	t.Run("respects tool name filtering", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.5,
			embedder:  mockEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		cache.Store("tool_a", "test query", "response A")
		cache.Store("tool_b", "test query", "response B")

		// Lookup should return response for specific tool
		response, found := cache.Lookup("tool_a", "test query")
		assert.True(t, found)
		assert.Equal(t, "response A", response)

		response, found = cache.Lookup("tool_b", "test query")
		assert.True(t, found)
		assert.Equal(t, "response B", response)

		// Non-existent tool should return nothing
		response, found = cache.Lookup("tool_c", "test query")
		assert.False(t, found)
		assert.Empty(t, response)
	})

	t.Run("respects 7-day time filter", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.5,
			embedder:  mockEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		embedding, err := mockEmbedder("test query")
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO semantic_cache (tool_name, query_text, query_embedding, response_data, created_at)
			VALUES ($1, $2, $3, $4, $5)`,
			"test_tool",
			"test query",
			pgvector.NewVector(embedding), // Use proper pgvector format
			"old response",
			time.Now().AddDate(0, 0, -8)) // 8 days ago

		require.NoError(t, err)

		// Should not find the old entry
		response, found := cache.Lookup("test_tool", "test query")
		assert.False(t, found)
		assert.Empty(t, response)
	})
}

func TestStore(t *testing.T) {
	t.Run("does nothing when cache is disabled", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:      db,
			enabled: false,
		}

		// Should not panic or error
		cache.Store("test_tool", "test query", "test response")
	})

	t.Run("does nothing when response is empty", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:       db,
			enabled:  true,
			embedder: mockEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		cache.Store("test_tool", "test query", "")

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("does nothing when embedder fails", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:       db,
			enabled:  true,
			embedder: failingEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		cache.Store("test_tool", "test query", "test response")

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("successfully stores valid data", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{
			db:       db,
			enabled:  true,
			embedder: mockEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		cache.Store("test_tool", "test query", "test response")

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify stored data
		var toolName, queryText, responseData string
		err = db.QueryRow("SELECT tool_name, query_text, response_data FROM semantic_cache").Scan(
			&toolName, &queryText, &responseData)
		require.NoError(t, err)
		assert.Equal(t, "test_tool", toolName)
		assert.Equal(t, "test query", queryText)
		assert.Equal(t, "test response", responseData)
	})
}

func TestEnsureSchema(t *testing.T) {
	t.Run("creates schema successfully", func(t *testing.T) {
		db := testDB(t)
		defer cleanupDB(t, db)

		cache := &SimpleCache{db: db}

		err := cache.ensureSchema()
		require.NoError(t, err)

		// Verify table exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'semantic_cache'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify indexes exist
		var indexCount int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pg_indexes
			WHERE tablename = 'semantic_cache'
			AND indexname IN ('semantic_cache_embedding_idx', 'idx_semantic_cache_tool')
		`).Scan(&indexCount)
		require.NoError(t, err)
		assert.Equal(t, 2, indexCount)
	})

	t.Run("handles extension creation failure", func(t *testing.T) {
		// Use a closed database to force failure
		db := testDB(t)
		db.Close()

		cache := &SimpleCache{db: db}

		err := cache.ensureSchema()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create vector extension")
	})
}
