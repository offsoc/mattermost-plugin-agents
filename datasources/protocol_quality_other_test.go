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

// TestProtocolQuality_Confluence measures Confluence protocol search quality
func TestProtocolQuality_Confluence(t *testing.T) {
	confluenceToken := os.Getenv("MM_AI_CONFLUENCE_DOCS_TOKEN")
	if confluenceToken == "" {
		t.Skip("Skipping Confluence test - no MM_AI_CONFLUENCE_DOCS_TOKEN")
	}

	config := CreateDefaultConfig()

	// Enable Confluence and set auth
	for i := range config.Sources {
		if config.Sources[i].Name == SourceConfluenceDocs {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth = AuthConfig{
				Type: AuthTypeAPIKey,
				Key:  confluenceToken,
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
			name:           "Playbooks workflow",
			query:          "playbooks workflow automation",
			expectedTopics: []string{"playbook", "workflow", "automation", "run"},
			limit:          10,
		},
		{
			name:           "Product requirements",
			query:          "product requirements feature specs",
			expectedTopics: []string{"product", "requirement", "feature", "spec"},
			limit:          10,
		},
		{
			name:           "Mobile strategy",
			query:          "mobile app strategy roadmap",
			expectedTopics: []string{"mobile", "app", "strategy", "roadmap"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceConfluenceDocs, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("Confluence fetch failed: %v", err)
			}

			t.Logf("\n=== Confluence Protocol Quality Test: %s ===", tc.name)
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
				t.Logf("⚠️  WARNING: Relevance rate below 50%% - Confluence search quality needs improvement")
			} else {
				t.Logf("✅ Confluence search quality is acceptable (>50%%)")
			}
		})
	}
}

// TestProtocolQuality_MattermostDocs measures HTTP protocol quality for docs.mattermost.com
func TestProtocolQuality_MattermostDocs(t *testing.T) {
	config := CreateDefaultConfig()

	// Enable Mattermost Docs
	for i := range config.Sources {
		if config.Sources[i].Name == SourceMattermostDocs {
			config.Sources[i].Enabled = true
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
			name:           "LDAP authentication",
			query:          "LDAP authentication configuration",
			expectedTopics: []string{"ldap", "authentication", "auth", "active directory", "ad"},
			limit:          10,
		},
		{
			name:           "High availability",
			query:          "high availability clustering deployment",
			expectedTopics: []string{"high availability", "cluster", "ha", "deployment"},
			limit:          10,
		},
		{
			name:           "Plugin development",
			query:          "plugin development API guide",
			expectedTopics: []string{"plugin", "develop", "api", "guide"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceMattermostDocs, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("Mattermost Docs fetch failed: %v", err)
			}

			t.Logf("\n=== Mattermost Docs Protocol Quality Test: %s ===", tc.name)
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
				t.Logf("⚠️  WARNING: Relevance rate below 50%% - Mattermost Docs search quality needs improvement")
			} else {
				t.Logf("✅ Mattermost Docs search quality is acceptable (>50%%)")
			}
		})
	}
}

// TestProtocolQuality_Forum measures Discourse/Forum protocol quality
func TestProtocolQuality_Forum(t *testing.T) {
	forumToken := os.Getenv("MM_AI_COMMUNITY_FORUM_TOKEN")
	if forumToken == "" {
		t.Skip("Skipping Forum test - no MM_AI_COMMUNITY_FORUM_TOKEN")
	}

	config := CreateDefaultConfig()

	// Enable Community Forum and set auth
	for i := range config.Sources {
		if config.Sources[i].Name == SourceCommunityForum {
			config.Sources[i].Enabled = true
			config.Sources[i].Auth = AuthConfig{
				Type: AuthTypeAPIKey,
				Key:  forumToken,
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
			name:           "Plugin installation",
			query:          "plugin installation marketplace",
			expectedTopics: []string{"plugin", "install", "marketplace"},
			limit:          10,
		},
		{
			name:           "Mobile push notifications",
			query:          "mobile push notifications not working",
			expectedTopics: []string{"mobile", "push", "notification"},
			limit:          10,
		},
		{
			name:           "Webhook integration",
			query:          "webhook integration custom bot",
			expectedTopics: []string{"webhook", "integration", "bot"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeout)
			defer cancel()

			docs, err := client.FetchFromSource(ctx, SourceCommunityForum, tc.query, tc.limit)
			if err != nil {
				t.Fatalf("Forum fetch failed: %v", err)
			}

			t.Logf("\n=== Community Forum Protocol Quality Test: %s ===", tc.name)
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
				t.Logf("⚠️  WARNING: Relevance rate below 50%% - Forum search quality needs improvement")
			} else {
				t.Logf("✅ Forum search quality is acceptable (>50%%)")
			}
		})
	}
}

// TestProtocolQuality_FileBasedSources tests Hub, Productboard, UserVoice (file protocol)
func TestProtocolQuality_FileBasedSources(t *testing.T) {
	config := CreateDefaultConfig()

	testCases := []struct {
		sourceName     string
		query          string
		expectedTopics []string
		limit          int
	}{
		{
			sourceName:     SourceMattermostHub,
			query:          "enterprise sales customer feedback",
			expectedTopics: []string{"enterprise", "sales", "customer", "feedback"},
			limit:          10,
		},
		{
			sourceName:     SourceProductBoardFeatures,
			query:          "feature requests voting roadmap",
			expectedTopics: []string{"feature", "request", "vote", "roadmap"},
			limit:          10,
		},
		{
			sourceName:     SourceFeatureRequests,
			query:          "user feedback feature suggestions",
			expectedTopics: []string{"user", "feedback", "feature", "suggest"},
			limit:          10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.sourceName, func(t *testing.T) {
			// Enable source
			sourceEnabled := false
			for i := range config.Sources {
				if config.Sources[i].Name == tc.sourceName {
					config.Sources[i].Enabled = true
					sourceEnabled = true
					break
				}
			}

			if !sourceEnabled {
				t.Skipf("Source %s not found in config", tc.sourceName)
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
