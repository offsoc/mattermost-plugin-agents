// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestMattermostRealDataValidation tests search syntax validation against REAL Mattermost data sources
func TestMattermostRealDataValidation(t *testing.T) {
	// Test against actual Mattermost data sources with real queries
	testCases := []struct {
		name         string
		protocol     DataSourceProtocol
		sourceConfig SourceConfig
		complexQuery string
		simpleQuery  string
	}{
		{
			name:     "MattermostForum",
			protocol: NewDiscourseProtocol(&http.Client{}, nil),
			sourceConfig: SourceConfig{
				Name:     SourceMattermostForum,
				Protocol: DiscourseProtocolType,
				Enabled:  true,
				Endpoints: map[string]string{
					"base_url": "https://forum.mattermost.com",
				},
				Sections: []string{"announcements", "trouble-shoot", "feature-ideas"},
			},
			// This is the exact problematic query from production logs
			complexQuery: "(gaps OR limitations OR missing OR issues OR feedback OR requests OR needs OR problems OR suggestions OR improvements) AND (mobile OR ios OR android OR app OR smartphone OR tablet OR mobile app OR native app OR react native OR react-native OR push notifications OR push notification OR offline-first OR edge mobile)",
			simpleQuery:  "mobile",
		},
		{
			name:     "MattermostDocs",
			protocol: NewHTTPProtocol(&http.Client{}, nil),
			sourceConfig: SourceConfig{
				Name:     SourceMattermostDocs,
				Protocol: HTTPProtocolType,
				Enabled:  true,
				Endpoints: map[string]string{
					"base_url": "https://docs.mattermost.com",
				},
				Sections: []string{"install", "deploy", "configure"},
			},
			complexQuery: "(gaps OR limitations OR missing) AND (mobile OR ios OR android)",
			simpleQuery:  "mobile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
			defer cancel()

			t.Logf("Testing %s with real data source: %s", tc.name, tc.sourceConfig.Endpoints["base_url"])

			// Test 1: Complex query that typically fails
			t.Run("ComplexQuery", func(t *testing.T) {
				request := ProtocolRequest{
					Source:   tc.sourceConfig,
					Topic:    tc.complexQuery,
					Sections: tc.sourceConfig.Sections,
					Limit:    5,
				}

				result, err := tc.protocol.ValidateSearchSyntax(ctx, request)

				if err != nil {
					t.Logf("Expected: Complex query validation returned error: %v", err)
					return
				}

				if result == nil {
					t.Errorf("Got nil result for complex query")
					return
				}

				t.Logf("Complex query results:")
				t.Logf("  Original: %s", result.OriginalQuery)
				t.Logf("  Valid: %t", result.IsValidSyntax)
				t.Logf("  Test result count: %d", result.TestResultCount)
				t.Logf("  Errors: %v", result.SyntaxErrors)
				t.Logf("  Recommended: %s", result.RecommendedQuery)
				t.Logf("  Supports: %v", result.SupportsFeatures)

				// Key validation: if complex query returns 0 results, it should be flagged as invalid syntax
				if result.TestResultCount == 0 && result.IsValidSyntax {
					t.Errorf("Complex query returned 0 results but marked as valid - this suggests syntax issues are not being detected")
				}

				// If marked as invalid, should provide helpful recommendation
				if !result.IsValidSyntax {
					if result.RecommendedQuery == "" {
						t.Errorf("Invalid syntax but no recommended query provided")
					}
					if len(result.SyntaxErrors) == 0 {
						t.Errorf("Invalid syntax but no error messages provided")
					}
				}
			})

			// Test 2: Simple query that should work
			t.Run("SimpleQuery", func(t *testing.T) {
				request := ProtocolRequest{
					Source:   tc.sourceConfig,
					Topic:    tc.simpleQuery,
					Sections: tc.sourceConfig.Sections,
					Limit:    5,
				}

				result, err := tc.protocol.ValidateSearchSyntax(ctx, request)

				if err != nil {
					t.Logf("Simple query validation error: %v", err)
					return
				}

				if result == nil {
					t.Errorf("Got nil result for simple query")
					return
				}

				t.Logf("Simple query results:")
				t.Logf("  Original: %s", result.OriginalQuery)
				t.Logf("  Valid: %t", result.IsValidSyntax)
				t.Logf("  Test result count: %d", result.TestResultCount)
				t.Logf("  Errors: %v", result.SyntaxErrors)
				t.Logf("  Recommended: %s", result.RecommendedQuery)

				// Simple queries should typically work or at least provide results
				if result.TestResultCount > 0 {
					t.Logf("SUCCESS: Simple query '%s' returned %d results", tc.simpleQuery, result.TestResultCount)
				} else {
					t.Logf("INFO: Simple query '%s' returned 0 results - may indicate empty content or auth issues", tc.simpleQuery)
				}
			})

			// Test 3: Comparison test - verify our validation detects the difference
			t.Run("ValidationComparison", func(t *testing.T) {
				// Test complex query
				complexRequest := ProtocolRequest{
					Source:   tc.sourceConfig,
					Topic:    tc.complexQuery,
					Sections: tc.sourceConfig.Sections,
					Limit:    5,
				}
				complexResult, _ := tc.protocol.ValidateSearchSyntax(ctx, complexRequest)

				// Test simple query
				simpleRequest := ProtocolRequest{
					Source:   tc.sourceConfig,
					Topic:    tc.simpleQuery,
					Sections: tc.sourceConfig.Sections,
					Limit:    5,
				}
				simpleResult, _ := tc.protocol.ValidateSearchSyntax(ctx, simpleRequest)

				if complexResult != nil && simpleResult != nil {
					t.Logf("COMPARISON:")
					t.Logf("  Complex query ('%s'): %d results, valid=%t",
						strings.Split(tc.complexQuery, " ")[0]+"...", complexResult.TestResultCount, complexResult.IsValidSyntax)
					t.Logf("  Simple query ('%s'): %d results, valid=%t",
						tc.simpleQuery, simpleResult.TestResultCount, simpleResult.IsValidSyntax)

					// The validation should detect when complex queries fail due to syntax vs content
					if complexResult.TestResultCount == 0 && simpleResult.TestResultCount > 0 {
						if complexResult.IsValidSyntax {
							t.Errorf("VALIDATION ISSUE: Complex query returned 0 results but simple query returned %d results, yet complex query is marked as valid syntax. This suggests the validation is not detecting search syntax problems.", simpleResult.TestResultCount)
						} else {
							t.Logf("SUCCESS: Validation correctly detected that complex query has syntax issues (recommended: %s)", complexResult.RecommendedQuery)
						}
					}
				}
			})
		})
	}
}
