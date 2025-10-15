// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package semanticcache

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests that verify semanticcache works with actual pgvector functionality
// These tests use real vector similarity calculations and test the end-to-end flow

func testIntegrationDB(t *testing.T) *sql.DB {
	rootDB, err := sqlx.Connect("postgres", rootDSN)
	require.NoError(t, err, "Failed to connect to PostgreSQL. Is PostgreSQL running?")
	defer rootDB.Close()

	var hasVector bool
	err = rootDB.Get(&hasVector, "SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = 'vector')")
	require.NoError(t, err, "Failed to check for vector extension")
	require.True(t, hasVector, "pgvector extension not available in PostgreSQL. Please install it to run these tests.")

	dbName := fmt.Sprintf("semanticcache_integration_test_%d", model.GetMillis())

	_, err = rootDB.Exec("CREATE DATABASE " + dbName)
	require.NoError(t, err, "Failed to create test database")
	t.Logf("Created integration test database: %s", dbName)

	// Ensure cleanup happens regardless of test outcome
	t.Cleanup(func() {
		rootCleanupDB, cleanupErr := sqlx.Connect("postgres", rootDSN)
		if cleanupErr == nil {
			defer rootCleanupDB.Close()
			_, _ = rootCleanupDB.Exec("DROP DATABASE " + dbName)
			t.Logf("Cleaned up integration test database: %s", dbName)
		}
	})

	testDSN := fmt.Sprintf("postgres://mmuser:mostest@localhost:5432/%s?sslmode=disable", dbName)
	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		require.NoError(t, err, "Failed to connect to test database")
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		require.NoError(t, err, "Failed to ping test database")
	}

	t.Setenv("SEMANTICCACHE_INTEGRATION_TEST_DB", dbName)

	return db
}

func cleanupIntegrationDB(t *testing.T, db *sql.DB) {
	if db == nil {
		return
	}

	err := db.Close()
	require.NoError(t, err, "Failed to close database connection")
}

// realEmbedder creates embeddings with actual vector similarity properties
func realEmbedder(text string) ([]float32, error) {
	// Create embeddings with meaningful similarity relationships
	// Use configured dimensions to match database schema
	dimensions := getEmbeddingDimensions()
	embedding := make([]float32, dimensions)

	hash := 0
	for _, char := range text {
		hash = (hash * 31) + int(char)
	}

	for i := range embedding {
		embedding[i] = float32((hash+i*7)%100) * 0.001
	}

	regionSize := dimensions / 8
	switch {
	case contains(text, "search"):
		// "search" related queries get a strong positive pattern in first region
		for i := 0; i < regionSize; i++ {
			embedding[i] = 0.8 + float32(i%5)*0.02
		}
	case contains(text, "query"):
		// "query" related text gets a pattern in different region
		for i := regionSize; i < 2*regionSize; i++ {
			embedding[i] = 0.8 + float32(i%5)*0.02
		}
	case contains(text, "database"):
		// "database" related text gets a third distinct pattern
		for i := 2 * regionSize; i < 3*regionSize; i++ {
			embedding[i] = 0.8 + float32(i%5)*0.02
		}
	case contains(text, "hello"):
		// "hello" gets a very specific pattern for greeting tests
		for i := 3 * regionSize; i < 4*regionSize; i++ {
			embedding[i] = 0.9 + float32(i%3)*0.01
		}
	default:
		// Completely different pattern for unrelated text
		for i := 4 * regionSize; i < 5*regionSize && i < dimensions; i++ {
			embedding[i] = 0.3 + float32(i%7)*0.01
		}
	}

	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(1.0 / (float64(norm) + 0.001)) // Add small epsilon to avoid division by zero

	for i := range embedding {
		embedding[i] *= norm
	}

	return embedding, nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntegrationSemanticSimilarity(t *testing.T) {
	t.Run("finds semantically similar queries with real vectors", func(t *testing.T) {
		db := testIntegrationDB(t)
		defer cleanupIntegrationDB(t, db)

		t.Setenv("VECTOR_CACHE_ENABLED", "1")
		t.Setenv("VECTOR_CACHE_THRESHOLD", "0.80")

		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.80,
			embedder:  realEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		cache.Store("search_tool", "search for users", "Found 10 users")
		cache.Store("search_tool", "search in database", "Database search completed")
		cache.Store("query_tool", "query the API", "API query successful")

		response, found := cache.Lookup("search_tool", "search users")
		assert.True(t, found)
		assert.Contains(t, []string{"Found 10 users", "Database search completed"}, response)

		_, found = cache.Lookup("search_tool", "query data")
		assert.False(t, found, "Should not find query_tool responses when searching search_tool")

		cache.Store("db_tool", "database connection", "Connected to DB")
		response, found = cache.Lookup("db_tool", "database connect")
		assert.True(t, found)
		assert.Equal(t, "Connected to DB", response)
	})

	t.Run("respects similarity threshold", func(t *testing.T) {
		db := testIntegrationDB(t)
		defer cleanupIntegrationDB(t, db)

		// Use a high threshold to test filtering
		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.95, // Very high threshold
			embedder:  realEmbedder,
		}

		// Setup schema
		err := cache.ensureSchema()
		require.NoError(t, err)

		// Store a response with "hello" semantic category
		cache.Store("test_tool", "hello world", "Hello response")

		// Query with completely different semantic category - should not find due to high threshold
		response, found := cache.Lookup("test_tool", "search files")
		assert.False(t, found, "Should not find response with high similarity threshold")
		assert.Empty(t, response)

		// Query with exact same text - should find
		response, found = cache.Lookup("test_tool", "hello world")
		assert.True(t, found)
		assert.Equal(t, "Hello response", response)
	})
}

func TestIntegrationCacheLifecycle(t *testing.T) {
	t.Run("complete cache lifecycle with real data", func(t *testing.T) {
		db := testIntegrationDB(t)
		defer cleanupIntegrationDB(t, db)

		t.Setenv("VECTOR_CACHE_ENABLED", "1")
		t.Setenv("VECTOR_CACHE_THRESHOLD", "0.75")

		cache := NewSimpleCache(db)
		require.True(t, cache.enabled)

		// Replace the embedder with our test one for predictable results
		cache.embedder = realEmbedder

		// Phase 1: Store multiple related responses
		testCases := map[string]string{
			"search for files":    "Found 25 files",
			"search documents":    "Found 12 documents",
			"database query":      "Query returned 5 rows",
			"database connection": "Connected successfully",
			"hello user":          "Hello! How can I help?",
			"hello there":         "Hi there!",
		}

		for query, response := range testCases {
			cache.Store("test_tool", query, response)
		}

		// Verify all were stored
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = 'test_tool'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 6, count)

		// Phase 2: Test semantic lookup for similar queries
		similarityTests := []struct {
			query           string
			expectFound     bool
			expectedOptions []string // Multiple valid responses due to similarity
		}{
			{"search files", true, []string{"Found 25 files", "Found 12 documents"}},
			{"database connect", true, []string{"Connected successfully", "Query returned 5 rows"}},
			{"hello world", true, []string{"Hello! How can I help?", "Hi there!"}},
			{"completely random unrelated words", false, nil},
		}

		for _, test := range similarityTests {
			response, found := cache.Lookup("test_tool", test.query)
			if test.expectFound {
				assert.True(t, found, "Expected to find response for: %s", test.query)
				assert.Contains(t, test.expectedOptions, response, "Response should be one of expected options for: %s", test.query)
			} else {
				assert.False(t, found, "Expected not to find response for: %s", test.query)
				assert.Empty(t, response)
			}
		}

		// Phase 3: Test tool isolation
		cache.Store("different_tool", "search files", "Different tool response")

		response, found := cache.Lookup("different_tool", "search files")
		assert.True(t, found)
		assert.Equal(t, "Different tool response", response)

		// Original tool should still return its own responses
		response, found = cache.Lookup("test_tool", "search files")
		assert.True(t, found)
		assert.Contains(t, []string{"Found 25 files", "Found 12 documents"}, response)
	})
}

func TestIntegrationVectorOperations(t *testing.T) {
	t.Run("direct pgvector operations work correctly", func(t *testing.T) {
		db := testIntegrationDB(t)
		defer cleanupIntegrationDB(t, db)

		// Enable pgvector extension
		_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
		require.NoError(t, err)

		// Create a simple test table with configured dimensions
		dimensions := getEmbeddingDimensions()
		_, err = db.Exec(fmt.Sprintf(`
			CREATE TABLE test_vectors (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				embedding VECTOR(%d)
			)
		`, dimensions))
		require.NoError(t, err)

		// Insert test vectors
		testVectors := map[string][]float32{
			"search_query": mustEmbedding("search for data", realEmbedder),
			"db_query":     mustEmbedding("database query", realEmbedder),
			"hello_query":  mustEmbedding("hello world", realEmbedder),
		}

		for name, vec := range testVectors {
			_, err = db.Exec("INSERT INTO test_vectors (name, embedding) VALUES ($1, $2)",
				name, pgvector.NewVector(vec))
			require.NoError(t, err)
		}

		// Test similarity search
		searchVec := mustEmbedding("search data", realEmbedder)

		var foundName string
		var similarity float64
		err = db.QueryRow(`
			SELECT name, (1 - (embedding <=> $1)) as similarity
			FROM test_vectors
			ORDER BY embedding <=> $1
			LIMIT 1
		`, pgvector.NewVector(searchVec)).Scan(&foundName, &similarity)

		require.NoError(t, err)
		assert.Equal(t, "search_query", foundName)
		assert.Greater(t, similarity, 0.5, "Similarity should be reasonably high for related terms")

		// Test that different semantic spaces have lower similarity
		helloVec := mustEmbedding("hello there", realEmbedder)
		err = db.QueryRow(`
			SELECT (1 - (embedding <=> $1)) as similarity
			FROM test_vectors
			WHERE name = 'search_query'
		`, pgvector.NewVector(helloVec)).Scan(&similarity)

		require.NoError(t, err)
		assert.Less(t, similarity, 0.9, "Different semantic spaces should have lower similarity")
	})
}

func TestIntegrationConcurrency(t *testing.T) {
	t.Run("handles concurrent operations safely", func(t *testing.T) {
		db := testIntegrationDB(t)
		defer cleanupIntegrationDB(t, db)

		cache := &SimpleCache{
			db:        db,
			enabled:   true,
			threshold: 0.75,
			embedder:  realEmbedder,
		}

		err := cache.ensureSchema()
		require.NoError(t, err)

		// Run concurrent stores and lookups
		done := make(chan bool, 10)

		// Start multiple goroutines storing data
		for i := 0; i < 5; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 3; j++ {
					query := fmt.Sprintf("search query %d-%d", id, j)
					response := fmt.Sprintf("response %d-%d", id, j)
					cache.Store("concurrent_tool", query, response)
				}
			}(i)
		}

		// Start multiple goroutines doing lookups
		for i := 0; i < 5; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 3; j++ {
					query := fmt.Sprintf("search query %d", id)
					cache.Lookup("concurrent_tool", query)
					// We don't assert results here since timing is unpredictable
					// The important thing is no panics or database errors occur
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Concurrent operations timed out")
			}
		}

		// Verify data was stored (some operations should have succeeded)
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = 'concurrent_tool'").Scan(&count)
		require.NoError(t, err)
		assert.Greater(t, count, 0, "Some concurrent stores should have succeeded")
	})
}

// mustEmbedding is a helper that panics on embedding errors (for test setup)
func mustEmbedding(text string, embedder func(string) ([]float32, error)) []float32 {
	embedding, err := embedder(text)
	if err != nil {
		panic(fmt.Sprintf("Failed to create embedding: %v", err))
	}
	return embedding
}
