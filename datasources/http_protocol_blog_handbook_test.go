// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"strings"
	"testing"
)

// TestHTTPProtocol_BlogQueries verifies HTTP protocol works for Mattermost Blog queries
func TestHTTPProtocol_BlogQueries(t *testing.T) {
	config := CreateDefaultConfig()
	config.EnableSource(SourceMattermostBlog)

	client := NewClient(config, nil)

	testCases := []struct {
		name           string
		query          string
		expectedTopics []string
		minRelevance   int
	}{
		{
			name:           "Security blog posts",
			query:          "security best practices",
			expectedTopics: []string{"security", "secure", "vulnerability", "threat"},
			minRelevance:   1,
		},
		{
			name:           "AI/ML features",
			query:          "artificial intelligence machine learning",
			expectedTopics: []string{"ai", "machine learning", "ml", "copilot", "llm"},
			minRelevance:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), ExtendedTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceMattermostBlog, tc.query, 3)
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			if len(docs) == 0 {
				t.Fatalf("FAIL: Expected at least 1 document, got 0 - HTTP protocol must return documents")
			}

			relevantCount := 0
			for _, doc := range docs {
				content := strings.ToLower(doc.Title + " " + doc.Content)
				for _, topic := range tc.expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						relevantCount++
						break
					}
				}
			}

			if relevantCount < tc.minRelevance {
				t.Fatalf("FAIL: Expected at least %d relevant results, got %d - documents must match query topic", tc.minRelevance, relevantCount)
			}
		})
	}
}

// TestHTTPProtocol_HandbookQueries verifies HTTP protocol works for Mattermost Handbook queries
func TestHTTPProtocol_HandbookQueries(t *testing.T) {
	config := CreateDefaultConfig()

	config.EnableSource(SourceMattermostHandbook)

	client := NewClient(config, nil)

	testCases := []struct {
		name           string
		query          string
		expectedTopics []string
		minRelevance   int
	}{
		{
			name:           "Engineering practices",
			query:          "engineering development practices",
			expectedTopics: []string{"engineering", "development", "code", "review", "testing"},
			minRelevance:   1,
		},
		{
			name:           "Company values",
			query:          "company culture values",
			expectedTopics: []string{"culture", "value", "mission", "team", "company"},
			minRelevance:   1,
		},
		{
			name:           "Operations",
			query:          "operations processes",
			expectedTopics: []string{"operation", "process", "workflow", "procedure"},
			minRelevance:   1,
		},
		{
			name:           "Security policies",
			query:          "security policies procedures",
			expectedTopics: []string{"security", "policy", "compliance", "privacy"},
			minRelevance:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), ExtendedTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceMattermostHandbook, tc.query, 3)
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			if len(docs) == 0 {
				t.Fatalf("FAIL: Expected at least 1 document, got 0 - HTTP protocol must return documents")
			}

			relevantCount := 0
			for _, doc := range docs {
				content := strings.ToLower(doc.Title + " " + doc.Content)
				for _, topic := range tc.expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						relevantCount++
						break
					}
				}
			}

			if relevantCount < tc.minRelevance {
				t.Fatalf("FAIL: Expected at least %d relevant results, got %d - documents must match query topic", tc.minRelevance, relevantCount)
			}
		})
	}
}
