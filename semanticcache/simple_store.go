// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semanticcache

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/pgvector/pgvector-go"
)

// SQL schema constants to prevent SQL injection via dynamic construction
const (
	tableName            = "semantic_cache"
	columnToolName       = "tool_name"
	columnQueryText      = "query_text"
	columnQueryEmbedding = "query_embedding"
	columnResponseData   = "response_data"
	columnCreatedAt      = "created_at"
	indexEmbedding       = "semantic_cache_embedding_idx"
	indexTool            = "idx_semantic_cache_tool"
	constraintUnique     = "semantic_cache_tool_query_unique"
	cacheRetentionDays   = 7
)

// NewSimpleCache creates a new semantic cache with pgvector
func NewSimpleCache(db *sql.DB) *SimpleCache {
	if db == nil {
		return &SimpleCache{
			enabled: false,
		}
	}

	enabled := os.Getenv("VECTOR_CACHE_ENABLED") != "0" // Default to enabled - can be disabled with VECTOR_CACHE_ENABLED=0
	threshold, _ := strconv.ParseFloat(os.Getenv("VECTOR_CACHE_THRESHOLD"), 32)
	if threshold == 0 {
		threshold = 0.85 // sensible default
	}

	cache := &SimpleCache{
		db:        db,
		threshold: float32(threshold),
		enabled:   enabled,
		embedder:  createEmbedder(),
	}

	if enabled {
		if err := cache.ensureSchema(); err != nil {
			fmt.Printf("[VECTOR-CACHE] Setup failed: %v, disabling\n", err)
			cache.enabled = false
		} else {
			fmt.Printf("[VECTOR-CACHE] Initialized with threshold=%.2f\n", threshold)
		}
	}

	return cache
}

// Lookup searches for cached responses similar to the query
func (c *SimpleCache) Lookup(toolName, query string) (string, bool) {
	if !c.enabled {
		return "", false
	}

	if c.db == nil {
		fmt.Printf("[VECTOR-CACHE] ERROR: Database is nil\n")
		return "", false
	}

	queryEmbed, err := c.embedder(query)
	if err != nil {
		fmt.Printf("[VECTOR-CACHE] WARN: Failed to generate embedding for lookup: %v\n", err)
		return "", false
	}

	var response string
	ctx := context.Background()
	err = c.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT response_data
		FROM semantic_cache
		WHERE tool_name = $1
		  AND created_at > NOW() - INTERVAL '%d days'
		  AND (1 - (query_embedding <=> $2)) >= $3
		ORDER BY (query_embedding <=> $2) ASC
		LIMIT 1`, cacheRetentionDays),
		toolName,
		pgvector.NewVector(queryEmbed),
		c.threshold).Scan(&response)

	if err == nil {
		return response, true
	}

	if err != sql.ErrNoRows {
		fmt.Printf("[VECTOR-CACHE] WARN: Query failed during lookup: %v\n", err)
	}

	return "", false
}

// Store caches a response for the given tool and query
func (c *SimpleCache) Store(toolName, query, response string) {
	if !c.enabled || response == "" {
		return
	}

	if c.db == nil {
		fmt.Printf("[VECTOR-CACHE] ERROR: Database is nil, cannot store cache entry\n")
		return
	}

	queryEmbed, err := c.embedder(query)
	if err != nil {
		fmt.Printf("[VECTOR-CACHE] WARN: Failed to generate embedding for storage: %v\n", err)
		return
	}

	ctx := context.Background()
	_, err = c.db.ExecContext(ctx, `
		INSERT INTO semantic_cache (tool_name, query_text, query_embedding, response_data)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tool_name, query_text)
		DO UPDATE SET
			query_embedding = EXCLUDED.query_embedding,
			response_data = EXCLUDED.response_data,
			created_at = NOW()`,
		toolName,
		query,
		pgvector.NewVector(queryEmbed),
		response)

	if err != nil {
		fmt.Printf("[VECTOR-CACHE] WARN: Failed to store cache entry: %v\n", err)
	}
}

// ensureSchema creates the necessary database schema
func (c *SimpleCache) ensureSchema() error {
	ctx := context.Background()

	if _, err := c.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	dimensions := getEmbeddingDimensions()

	var currentDimensions int
	err := c.db.QueryRowContext(ctx, `
		SELECT dimension
		FROM pg_attribute a
		JOIN pg_class c ON a.attrelid = c.oid
		JOIN pg_type t ON a.atttypid = t.oid
		WHERE c.relname = $1
		AND a.attname = $2
		AND t.typname = 'vector'
	`, tableName, columnQueryEmbedding).Scan(&currentDimensions)

	if err == nil && currentDimensions != dimensions {
		fmt.Printf("[VECTOR-CACHE] Table exists with wrong dimensions (%d vs %d), dropping and recreating\n", currentDimensions, dimensions)
		if _, dropErr := c.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)); dropErr != nil {
			return fmt.Errorf("failed to drop table: %w", dropErr)
		}
	}

	// Create table and indexes using constants to prevent SQL injection
	// Note: Table and column names cannot be parameterized in PostgreSQL,
	// so we use Go constants defined at package level to ensure safety
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		%s TEXT NOT NULL,
		%s TEXT NOT NULL,
		%s VECTOR(%d),
		%s TEXT NOT NULL,
		%s TIMESTAMPTZ DEFAULT NOW(),
		CONSTRAINT %s UNIQUE (%s, %s)
	);

	CREATE INDEX IF NOT EXISTS %s
		ON %s USING hnsw (%s vector_cosine_ops);
	CREATE INDEX IF NOT EXISTS %s
		ON %s(%s);
	`,
		tableName,
		columnToolName, columnQueryText, columnQueryEmbedding, dimensions,
		columnResponseData, columnCreatedAt,
		constraintUnique, columnToolName, columnQueryText,
		indexEmbedding, tableName, columnQueryEmbedding,
		indexTool, tableName, columnToolName,
	)

	_, err = c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}
