// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestProtocolQuality_JIRA measures the actual quality of JIRA protocol search results
func TestProtocolQuality_JIRA(t *testing.T) {
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	if jiraToken == "" {
		t.Skip("Skipping JIRA quality test - no MM_AI_JIRA_TOKEN environment variable")
	}

	config := CreateDefaultConfig()

	// Enable jira_docs and set auth token
	for i := range config.Sources {
		if config.Sources[i].Name == SourceJiraDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth = AuthConfig{
				Type: AuthTypeAPIKey,
				Key:  jiraToken,
			}
			break
		}
	}

	client := NewClient(config, nil)

	testCases := []struct {
		name           string
		query          string
		expectedTopics []string // Keywords that should appear in relevant results
		limit          int
	}{
		{
			name:           "SAML authentication",
			query:          "SAML authentication SSO",
			expectedTopics: []string{"saml", "sso", "authentication", "auth", "login", "ldap"},
			limit:          10,
		},
		{
			name:           "Mobile performance",
			query:          "mobile app performance slow",
			expectedTopics: []string{"mobile", "performance", "slow", "app", "android", "ios"},
			limit:          10,
		},
		{
			name:           "Channels feature",
			query:          "channels messaging notifications",
			expectedTopics: []string{"channel", "message", "notification", "post", "thread"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceJiraDocs, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("JIRA fetch failed: %v", err)
			}

			t.Logf("\n=== JIRA Protocol Quality Test: %s ===", tc.name)
			t.Logf("Query: %s", tc.query)
			t.Logf("Results returned: %d", len(docs))

			if len(docs) == 0 {
				t.Logf("⚠️  WARNING: No results returned")
				return
			}

			// Measure relevance
			relevantCount := 0
			for i, doc := range docs {
				isRelevant := false
				content := strings.ToLower(doc.Title + " " + doc.Content)

				// Check if any expected topic appears
				for _, topic := range tc.expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						isRelevant = true
						break
					}
				}

				status := "❌ IRRELEVANT"
				if isRelevant {
					relevantCount++
					status = "✅ RELEVANT"
				}

				t.Logf("[%d] %s - %s", i+1, status, doc.Title)
				t.Logf("     URL: %s", doc.URL)
			}

			relevanceRate := float64(relevantCount) / float64(len(docs)) * 100
			t.Logf("\n📊 Relevance: %d/%d (%.1f%%)", relevantCount, len(docs), relevanceRate)

			if relevanceRate < 50 {
				t.Logf("⚠️  WARNING: Relevance rate below 50%% - JIRA search quality is PROBLEM B")
			} else {
				t.Logf("✅ JIRA search quality is acceptable (>50%%)")
			}
		})
	}
}

// TestProtocolQuality_GitHub measures the actual quality of GitHub protocol search results
func TestProtocolQuality_GitHub(t *testing.T) {
	githubToken := os.Getenv("MM_AI_GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("Skipping GitHub quality test - no MM_AI_GITHUB_TOKEN environment variable")
	}

	config := CreateDefaultConfig()
	config.GitHubToken = githubToken // Still set this for the protocol initialization

	// Enable github_repos and set auth token
	for i := range config.Sources {
		if config.Sources[i].Name == SourceGitHubRepos {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth = AuthConfig{
				Type: AuthTypeToken,
				Key:  githubToken,
			}
			break
		}
	}

	client := NewClient(config, nil)

	testCases := []struct {
		name           string
		query          string
		expectedTopics []string
		limit          int
	}{
		{
			name:           "Authentication bugs",
			query:          "authentication SAML login",
			expectedTopics: []string{"auth", "saml", "login", "sso", "ldap"},
			limit:          10,
		},
		{
			name:           "Mobile performance",
			query:          "mobile performance slow",
			expectedTopics: []string{"mobile", "performance", "slow", "app", "android", "ios"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceGitHubRepos, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("GitHub fetch failed: %v", err)
			}

			t.Logf("\n=== GitHub Protocol Quality Test: %s ===", tc.name)
			t.Logf("Query: %s", tc.query)
			t.Logf("Results returned: %d", len(docs))

			if len(docs) == 0 {
				t.Logf("⚠️  WARNING: No results returned")
				return
			}

			relevantCount := 0
			for i, doc := range docs {
				isRelevant := false
				content := strings.ToLower(doc.Title + " " + doc.Content)

				for _, topic := range tc.expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						isRelevant = true
						break
					}
				}

				status := "❌ IRRELEVANT"
				if isRelevant {
					relevantCount++
					status = "✅ RELEVANT"
				}

				t.Logf("[%d] %s - %s", i+1, status, doc.Title)
				t.Logf("     URL: %s", doc.URL)
			}

			relevanceRate := float64(relevantCount) / float64(len(docs)) * 100
			t.Logf("\n📊 Relevance: %d/%d (%.1f%%)", relevantCount, len(docs), relevanceRate)

			if relevanceRate < 50 {
				t.Logf("⚠️  WARNING: Relevance rate below 50%% - GitHub search quality is PROBLEM B")
			} else {
				t.Logf("✅ GitHub search quality is acceptable (>50%%)")
			}
		})
	}
}

// TestProtocolQuality_MultiSourceComparison compares single-source vs multi-source search quality
func TestProtocolQuality_MultiSourceComparison(t *testing.T) {
	jiraToken := os.Getenv("MM_AI_JIRA_TOKEN")
	githubToken := os.Getenv("MM_AI_GITHUB_TOKEN")

	if jiraToken == "" || githubToken == "" {
		t.Skip("Skipping comparison test - need both MM_AI_JIRA_TOKEN and MM_AI_GITHUB_TOKEN")
	}

	config := CreateDefaultConfig()
	config.GitHubToken = githubToken // Still set for protocol initialization

	// Enable both sources and set auth
	for i := range config.Sources {
		if config.Sources[i].Name == SourceJiraDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth = AuthConfig{
				Type: AuthTypeAPIKey,
				Key:  jiraToken,
			}
		}
		if config.Sources[i].Name == SourceGitHubRepos {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth = AuthConfig{
				Type: AuthTypeToken,
				Key:  githubToken,
			}
		}
	}

	client := NewClient(config, nil)

	// Test multiple MM-specific queries
	testCases := []struct {
		name           string
		query          string
		expectedTopics []string
	}{
		{
			name:           "SAML authentication",
			query:          "SAML authentication SSO",
			expectedTopics: []string{"saml", "sso", "auth", "login"},
		},
		{
			name:           "Channels performance",
			query:          "channels performance slow loading",
			expectedTopics: []string{"channel", "performance", "slow", "load"},
		},
		{
			name:           "Playbooks feature",
			query:          "playbooks workflow automation",
			expectedTopics: []string{"playbook", "workflow", "automation"},
		},
		{
			name:           "Calls plugin",
			query:          "calls plugin webrtc audio video",
			expectedTopics: []string{"call", "plugin", "webrtc", "audio", "video"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testMultiSourceQuery(t, client, tc.query, tc.expectedTopics)
		})
	}
}

func testMultiSourceQuery(t *testing.T, client *Client, testQuery string, expectedTopics []string) {
	t.Run("Single JIRA", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
		defer cancel()

		results, err := client.FetchFromMultipleSources(ctx, []string{SourceJiraDocs}, testQuery, 10)
		if err != nil {
			t.Fatalf("Single-source fetch failed: %v", err)
		}

		totalDocs := 0
		relevantDocs := 0

		for sourceName, docs := range results {
			totalDocs += len(docs)
			for i, doc := range docs {
				isRelevant := false
				content := strings.ToLower(doc.Title + " " + doc.Content)
				for _, topic := range expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						isRelevant = true
						break
					}
				}
				if isRelevant {
					relevantDocs++
				}
				status := "❌"
				if isRelevant {
					status = "✅"
				}
				t.Logf("[%s] [%d] %s %s", sourceName, i+1, status, doc.Title)
			}
		}

		if totalDocs > 0 {
			rate := float64(relevantDocs) / float64(totalDocs) * 100
			t.Logf("📊 JIRA only: %d/%d (%.1f%%)", relevantDocs, totalDocs, rate)
		}
	})

	t.Run("Multi JIRA+GitHub", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		results, err := client.FetchFromMultipleSources(ctx, []string{SourceJiraDocs, SourceGitHubRepos}, testQuery, 10)
		if err != nil {
			t.Fatalf("Multi-source fetch failed: %v", err)
		}

		totalDocs := 0
		relevantDocs := 0
		sourceRelevance := make(map[string]int)
		sourceTotals := make(map[string]int)

		for sourceName, docs := range results {
			sourceTotals[sourceName] = len(docs)
			totalDocs += len(docs)

			for i, doc := range docs {
				isRelevant := false
				content := strings.ToLower(doc.Title + " " + doc.Content)
				for _, topic := range expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						isRelevant = true
						break
					}
				}

				if isRelevant {
					relevantDocs++
					sourceRelevance[sourceName]++
				}

				status := "❌"
				if isRelevant {
					status = "✅"
				}
				t.Logf("[%s] [%d] %s %s", sourceName, i+1, status, doc.Title)
			}
		}

		if totalDocs > 0 {
			rate := float64(relevantDocs) / float64(totalDocs) * 100
			t.Logf("\n📊 Multi-Source: %d/%d (%.1f%%)", relevantDocs, totalDocs, rate)
			for source, total := range sourceTotals {
				if total > 0 {
					relevant := sourceRelevance[source]
					sourceRate := float64(relevant) / float64(total) * 100
					t.Logf("  %s: %d/%d (%.1f%%)", source, relevant, total, sourceRate)
				}
			}
		}
	})
}

// TestProtocolQuality_OtherSources tests Zendesk, Confluence, Discourse, and HTTP protocols
func TestProtocolQuality_OtherSources(t *testing.T) {
	config := CreateDefaultConfig()

	testCases := []struct {
		sourceName     string
		query          string
		expectedTopics []string
		limit          int
	}{
		{
			sourceName:     SourceZendeskTickets,
			query:          "SAML authentication login issues",
			expectedTopics: []string{"saml", "sso", "authentication", "auth", "login"},
			limit:          10,
		},
		{
			sourceName:     SourceConfluenceDocs,
			query:          "playbooks workflow automation",
			expectedTopics: []string{"playbook", "workflow", "automation", "run"},
			limit:          10,
		},
		{
			sourceName:     SourceMattermostForum,
			query:          "mobile app performance slow",
			expectedTopics: []string{"mobile", "performance", "slow", "app", "android", "ios"},
			limit:          10,
		},
		{
			sourceName:     SourceMattermostDocs,
			query:          "LDAP authentication configuration",
			expectedTopics: []string{"ldap", "authentication", "auth", "config", "active directory"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.sourceName, func(t *testing.T) {
			// Check if source is enabled in config
			sourceEnabled := false
			for _, source := range config.Sources {
				if source.Name == tc.sourceName && source.Enabled {
					sourceEnabled = true
					break
				}
			}

			if !sourceEnabled {
				t.Skipf("Source %s not enabled in config", tc.sourceName)
			}

			client := NewClient(config, nil)

			ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, tc.sourceName, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("%s fetch failed: %v", tc.sourceName, err)
			}

			t.Logf("\n=== %s Protocol Quality Test ===", tc.sourceName)
			t.Logf("Query: %s", tc.query)
			t.Logf("Results returned: %d", len(docs))

			if len(docs) == 0 {
				t.Logf("⚠️  WARNING: No results returned")
				return
			}

			relevantCount := 0
			for i, doc := range docs {
				isRelevant := false
				content := strings.ToLower(doc.Title + " " + doc.Content)

				for _, topic := range tc.expectedTopics {
					if strings.Contains(content, strings.ToLower(topic)) {
						isRelevant = true
						break
					}
				}

				status := "❌ IRRELEVANT"
				if isRelevant {
					relevantCount++
					status = "✅ RELEVANT"
				}

				t.Logf("[%d] %s - %s", i+1, status, doc.Title)
				t.Logf("     URL: %s", doc.URL)
			}

			relevanceRate := float64(relevantCount) / float64(len(docs)) * 100
			t.Logf("\n📊 Relevance: %d/%d (%.1f%%)", relevantCount, len(docs), relevanceRate)

			if relevanceRate < 50 {
				t.Logf("⚠️  WARNING: Relevance rate below 50%% - %s search quality needs improvement", tc.sourceName)
			} else {
				t.Logf("✅ %s search quality is acceptable (>50%%)", tc.sourceName)
			}
		})
	}
}
