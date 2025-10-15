// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"strings"
	"testing"
)

// TestHTTPProtocol_PMBotQueries verifies HTTP protocol works for common PM bot queries
func TestHTTPProtocol_PMBotQueries(t *testing.T) {
	config := CreateDefaultConfig()
	config.EnableSource(SourceMattermostDocs)

	client := NewClient(config, nil)

	testCases := []struct {
		name           string
		query          string
		expectedTopics []string
		minRelevance   int
	}{
		{
			name:           "LDAP authentication",
			query:          "LDAP authentication configuration",
			expectedTopics: []string{"ldap", "authentication", "auth"},
			minRelevance:   1,
		},
		{
			name:           "AI channels feature",
			query:          "AI channels capabilities features",
			expectedTopics: []string{"ai", "channel", "copilot"},
			minRelevance:   1,
		},
		{
			name:           "Mobile strategy",
			query:          "mobile strategy roadmap",
			expectedTopics: []string{"mobile", "strategy"},
			minRelevance:   1,
		},
		{
			name:           "Generic deployment",
			query:          "deployment best practices",
			expectedTopics: []string{"deploy", "install", "setup", "config"},
			minRelevance:   1,
		},
		{
			name:           "Performance optimization",
			query:          "server performance optimization",
			expectedTopics: []string{"performance", "optimization", "scale", "speed"},
			minRelevance:   1,
		},
		{
			name:           "Plugin development",
			query:          "how to develop plugins",
			expectedTopics: []string{"plugin", "develop", "api", "integration"},
			minRelevance:   1,
		},
		{
			name:           "SSO setup",
			query:          "single sign-on configuration",
			expectedTopics: []string{"sso", "saml", "oauth", "authentication"},
			minRelevance:   1,
		},
		{
			name:           "High availability",
			query:          "high availability clustering",
			expectedTopics: []string{"high availability", "cluster", "ha", "redundancy"},
			minRelevance:   1,
		},
		{
			name:           "Notifications",
			query:          "push notification setup",
			expectedTopics: []string{"notification", "push", "alert"},
			minRelevance:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), ExtendedTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceMattermostDocs, tc.query, 3)
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
