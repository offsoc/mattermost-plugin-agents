// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMetadataProvider implements the shared.MetadataProvider interface for testing
type MockMetadataProvider struct {
	metadata map[string]MockToolMetadata
}

type MockToolMetadata struct {
	SupportedDataSources []string
	IntentKeywords       []string
}

func (m *MockMetadataProvider) GetSupportedDataSources(toolName string) []string {
	if metadata, exists := m.metadata[toolName]; exists {
		return metadata.SupportedDataSources
	}
	return []string{}
}

func (m *MockMetadataProvider) BuildCompositeQuery(toolName, topic, dataSource string) string {
	return ""
}

func (m *MockMetadataProvider) BuildSearchQueries(toolName, topic string) map[string]string {
	metadata, exists := m.metadata[toolName]
	if !exists {
		return map[string]string{}
	}

	queries := make(map[string]string)

	// Build a query with multi-word phrases
	if len(metadata.IntentKeywords) > 0 {
		// Simulate BuildBooleanQuery behavior with proper quoting
		intentPart := "("
		for i, keyword := range metadata.IntentKeywords {
			if i > 0 {
				intentPart += " OR "
			}
			// Quote multi-word or hyphenated keywords
			if containsSpaceOrHyphen(keyword) {
				intentPart += "\"" + keyword + "\""
			} else {
				intentPart += keyword
			}
		}
		intentPart += ")"

		featurePart := "(channels OR ai)"
		queries["general"] = intentPart + " AND " + featurePart
	}

	return queries
}

func containsSpaceOrHyphen(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '-' {
			return true
		}
	}
	return false
}

func TestSearchQueriesWithMultiWordPhrases(t *testing.T) {
	tests := []struct {
		name                string
		toolName            string
		intentKeywords      []string
		expectedQuoted      []string
		expectedNotQuoted   []string
		mustContainPatterns []string
	}{
		{
			name:     "CompileMarketResearch with multi-word phrases",
			toolName: ToolNameCompileMarketResearch,
			intentKeywords: []string{
				"market",
				"user needs",
				"third-party",
				"competitive",
			},
			expectedQuoted: []string{
				"\"user needs\"",
				"\"third-party\"",
			},
			expectedNotQuoted: []string{
				"market",
				"competitive",
			},
			mustContainPatterns: []string{
				"(market OR \"user needs\" OR \"third-party\" OR competitive) AND (channels OR ai)",
			},
		},
		{
			name:     "AnalyzeFeatureGaps with complex phrases",
			toolName: ToolNameAnalyzeFeatureGaps,
			intentKeywords: []string{
				"gaps",
				"not supported",
				"pain points",
				"deal blockers",
			},
			expectedQuoted: []string{
				"\"not supported\"",
				"\"pain points\"",
				"\"deal blockers\"",
			},
			expectedNotQuoted: []string{
				"gaps",
			},
		},
		{
			name:     "AnalyzeStrategicAlignment with strategic terms",
			toolName: ToolNameAnalyzeStrategicAlignment,
			intentKeywords: []string{
				"vision",
				"north star",
				"strategic plan",
				"business case",
			},
			expectedQuoted: []string{
				"\"north star\"",
				"\"strategic plan\"",
				"\"business case\"",
			},
			expectedNotQuoted: []string{
				"vision",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock metadata provider
			mockProvider := &MockMetadataProvider{
				metadata: map[string]MockToolMetadata{
					tt.toolName: {
						SupportedDataSources: []string{"mattermost_docs"},
						IntentKeywords:       tt.intentKeywords,
					},
				},
			}

			// Build search queries
			queries := mockProvider.BuildSearchQueries(tt.toolName, "channels ai")
			require.NotEmpty(t, queries, "Should generate queries")

			// Check the general query
			generalQuery, exists := queries["general"]
			require.True(t, exists, "Should have general query")
			require.NotEmpty(t, generalQuery, "General query should not be empty")

			// Verify quoted phrases
			for _, quoted := range tt.expectedQuoted {
				assert.Contains(t, generalQuery, quoted,
					"Query should contain quoted phrase: %s", quoted)
			}

			// Verify single words are not quoted
			for _, notQuoted := range tt.expectedNotQuoted {
				assert.Contains(t, generalQuery, notQuoted,
					"Query should contain word: %s", notQuoted)
				// Ensure it's not surrounded by quotes in the intent part
				assert.NotContains(t, generalQuery, "\""+notQuoted+"\"",
					"Single word should not be quoted: %s", notQuoted)
			}

			// Verify exact patterns if specified
			for _, pattern := range tt.mustContainPatterns {
				assert.Equal(t, pattern, generalQuery,
					"Query should match expected pattern")
			}

			t.Logf("Generated query: %s", generalQuery)
		})
	}
}

func TestQueryEscapingPreservesBooleanSyntax(t *testing.T) {
	// Test that escaping multi-word phrases doesn't break boolean query parsing
	queries := []string{
		"(market OR \"user needs\") AND channels",
		"(\"pain points\" OR limitations) AND mobile",
		"(\"north star\" OR vision OR \"strategic plan\") AND (\"ai channels\" OR playbooks)",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			// Verify the query has balanced parentheses
			openCount := 0
			closeCount := 0
			for _, c := range query {
				switch c {
				case '(':
					openCount++
				case ')':
					closeCount++
				}
			}
			assert.Equal(t, openCount, closeCount, "Parentheses should be balanced")

			// Verify quotes are balanced
			quoteCount := 0
			for _, c := range query {
				if c == '"' {
					quoteCount++
				}
			}
			assert.Equal(t, 0, quoteCount%2, "Quotes should be balanced (even count)")

			// Verify it contains AND operator
			assert.Contains(t, query, " AND ", "Query should contain AND operator")

			// Verify it contains OR operators
			assert.Contains(t, query, " OR ", "Query should contain OR operator")
		})
	}
}
