// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

// Generic semantic cache test infrastructure that can be used by all role-specific tests

var rootDSN = "postgres://mmuser:mostest@localhost:5432/postgres?sslmode=disable"

func testSemanticCacheDB(t *testing.T) *sql.DB {
	rootDB, err := sqlx.Connect("postgres", rootDSN)
	require.NoError(t, err, "Failed to connect to PostgreSQL. Is PostgreSQL running?")
	defer rootDB.Close()

	// Check if pgvector extension is available
	var hasVector bool
	err = rootDB.Get(&hasVector, "SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = 'vector')")
	require.NoError(t, err, "Failed to check for vector extension")
	require.True(t, hasVector, "pgvector extension not available in PostgreSQL. Please install it to run these tests.")

	// Create a unique database name with a timestamp
	dbName := fmt.Sprintf("mmtools_semantic_cache_test_%d", model.GetMillis())

	// Create the test database
	_, err = rootDB.Exec("CREATE DATABASE " + dbName)
	require.NoError(t, err, "Failed to create test database")
	t.Logf("Created semantic cache test database: %s", dbName)

	// Ensure cleanup happens regardless of test outcome
	t.Cleanup(func() {
		rootCleanupDB, cleanupErr := sql.Open("postgres", rootDSN)
		if cleanupErr == nil {
			defer rootCleanupDB.Close()
			_, _ = rootCleanupDB.Exec("DROP DATABASE " + dbName)
			t.Logf("Cleaned up semantic cache test database: %s", dbName)
		}
	})

	// Connect to the new database
	testDSN := fmt.Sprintf("postgres://mmuser:mostest@localhost:5432/%s?sslmode=disable", dbName)
	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		require.NoError(t, err, "Failed to connect to test database")
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		db.Close()
		require.NoError(t, err, "Failed to ping test database")
	}

	// Store the database name for cleanup
	t.Setenv("SEMANTIC_CACHE_INTEGRATION_TEST_DB", dbName)

	return db
}

func cleanupSemanticCacheDB(t *testing.T, db *sql.DB) {
	if db == nil {
		return
	}

	err := db.Close()
	require.NoError(t, err, "Failed to close database connection")

	// Database cleanup is handled automatically by t.Cleanup()
}
