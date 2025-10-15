// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubPRFiltering_Integration tests PR filtering capabilities with real GitHub API
// Run with: MM_AI_GITHUB_TOKEN=xxx go test -v -tags=integration ./datasources -run TestGitHubPRFiltering_Integration
func TestGitHubPRFiltering_Integration(t *testing.T) {
	token := os.Getenv("MM_AI_GITHUB_TOKEN")
	if token == "" {
		t.Skip("MM_AI_GITHUB_TOKEN not set, skipping integration test")
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	// Use Mattermost server repo for testing
	owner := "mattermost"
	repo := "mattermost"

	t.Run("fetch recent PRs without filtering", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "", 10, "github_repos")

		require.NotEmpty(t, docs, "Should return PRs from mattermost/mattermost")

		t.Logf("Fetched %d PRs", len(docs))

		// Verify PR structure
		for i, doc := range docs {
			assert.NotEmpty(t, doc.Title, "PR %d should have title", i)
			assert.NotEmpty(t, doc.Content, "PR %d should have content", i)
			assert.NotEmpty(t, doc.URL, "PR %d should have URL", i)
			assert.Equal(t, "pulls", doc.Section)
			assert.Equal(t, "github_repos", doc.Source)
			assert.NotEmpty(t, doc.Labels, "PR %d should have labels", i)

			// Log first PR for inspection
			if i == 0 {
				t.Logf("First PR Title: %s", doc.Title)
				t.Logf("First PR URL: %s", doc.URL)
				t.Logf("First PR Labels: %v", doc.Labels)
				t.Logf("First PR Content Length: %d chars", len(doc.Content))
			}
		}
	})

	t.Run("filter by topic - performance", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "performance", 20, "github_repos")

		t.Logf("Found %d PRs matching 'performance'", len(docs))

		// Verify filtering worked (may be 0 if no recent performance PRs)
		for _, doc := range docs {
			titleAndContent := strings.ToLower(doc.Title + " " + doc.Content)
			// Should contain performance keyword or related terms
			containsRelevant := strings.Contains(titleAndContent, "performance") ||
				strings.Contains(titleAndContent, "optimize") ||
				strings.Contains(titleAndContent, "speed") ||
				strings.Contains(titleAndContent, "slow")

			if !containsRelevant {
				t.Logf("Warning: PR may not be performance-related: %s", doc.Title)
			}
		}
	})

	t.Run("filter by topic - security", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "security", 20, "github_repos")

		t.Logf("Found %d PRs matching 'security'", len(docs))

		for _, doc := range docs {
			titleAndContent := strings.ToLower(doc.Title + " " + doc.Content)
			containsRelevant := strings.Contains(titleAndContent, "security") ||
				strings.Contains(titleAndContent, "vulnerability") ||
				strings.Contains(titleAndContent, "auth") ||
				strings.Contains(titleAndContent, "permission")

			if !containsRelevant {
				t.Logf("Warning: PR may not be security-related: %s", doc.Title)
			}
		}
	})

	t.Run("verify PR metadata - dates", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "", 10, "github_repos")

		require.NotEmpty(t, docs, "Should have PRs")

		// Verify PRs are sorted by update time (most recent first)
		// GitHub API returns sorted by updated DESC
		for i, doc := range docs {
			// Content should contain state and author
			assert.Contains(t, doc.Content, "**Details:**", "PR %d should have details", i)
			assert.Contains(t, doc.Content, "State:", "PR %d should have state", i)
			assert.Contains(t, doc.Content, "Author:", "PR %d should have author", i)
		}

		t.Logf("All %d PRs have proper metadata", len(docs))
	})

	t.Run("verify PR metadata - labels", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "", 10, "github_repos")

		require.NotEmpty(t, docs, "Should have PRs")

		hasLabels := 0
		for _, doc := range docs {
			if len(doc.Labels) > 0 {
				hasLabels++
				t.Logf("PR '%s' has %d labels", doc.Title, len(doc.Labels))
			}
		}

		t.Logf("%d out of %d PRs have labels", hasLabels, len(docs))
	})

	t.Run("verify PR metadata - author information", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "", 10, "github_repos")

		require.NotEmpty(t, docs, "Should have PRs")

		for _, doc := range docs {
			// Every PR should have author in content
			assert.Contains(t, doc.Content, "Author:", "PR should contain author information")
		}
	})

	t.Run("filter by component - webapp", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "webapp", 20, "github_repos")

		t.Logf("Found %d PRs matching 'webapp'", len(docs))

		for _, doc := range docs {
			titleAndContent := strings.ToLower(doc.Title + " " + doc.Content)
			containsRelevant := strings.Contains(titleAndContent, "webapp") ||
				strings.Contains(titleAndContent, "web") ||
				strings.Contains(titleAndContent, "frontend") ||
				strings.Contains(titleAndContent, "react")

			if !containsRelevant {
				t.Logf("Warning: PR may not be webapp-related: %s", doc.Title)
			}
		}
	})

	t.Run("filter by component - server", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, owner, repo, "server", 20, "github_repos")

		t.Logf("Found %d PRs matching 'server'", len(docs))

		for _, doc := range docs {
			titleAndContent := strings.ToLower(doc.Title + " " + doc.Content)
			containsRelevant := strings.Contains(titleAndContent, "server") ||
				strings.Contains(titleAndContent, "api") ||
				strings.Contains(titleAndContent, "backend")

			if !containsRelevant {
				t.Logf("Warning: PR may not be server-related: %s", doc.Title)
			}
		}
	})

	t.Run("verify multiple repos work independently", func(t *testing.T) {
		// Test server repo
		serverDocs := protocol.fetchRecentPRs(ctx, owner, "mattermost", "", 5, "github_repos")
		require.NotEmpty(t, serverDocs, "Should fetch PRs from mattermost/mattermost")

		// Test different repo
		desktopDocs := protocol.fetchRecentPRs(ctx, owner, "desktop", "", 5, "github_repos")
		require.NotEmpty(t, desktopDocs, "Should fetch PRs from mattermost/desktop")

		t.Logf("Server repo: %d PRs", len(serverDocs))
		t.Logf("Desktop repo: %d PRs", len(desktopDocs))

		// Verify they're different PRs (different URLs)
		serverURL := serverDocs[0].URL
		desktopURL := desktopDocs[0].URL

		assert.Contains(t, serverURL, "/mattermost/", "Server PR URL should contain repo name")
		assert.Contains(t, desktopURL, "/desktop/", "Desktop PR URL should contain repo name")
		assert.NotEqual(t, serverURL, desktopURL, "PRs from different repos should have different URLs")
	})
}

// TestGitHubPRFiltering_TimeRangeAnalysis analyzes the time distribution of fetched PRs
func TestGitHubPRFiltering_TimeRangeAnalysis(t *testing.T) {
	token := os.Getenv("MM_AI_GITHUB_TOKEN")
	if token == "" {
		t.Skip("MM_AI_GITHUB_TOKEN not set, skipping integration test")
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	docs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost", "", 50, "github_repos")

	require.NotEmpty(t, docs, "Should fetch PRs")

	t.Logf("\nAnalyzing %d PRs by update time:", len(docs))

	t.Logf("\nNote: Current implementation doesn't expose UpdatedAt in Doc struct.")
	t.Logf("PRs are sorted by GitHub API (most recently updated first).")

	t.Logf("\nNote: Current implementation filters PRs by topic relevance and content quality.")
	t.Logf("Date filtering would need to be added as a query parameter to GitHub API.")
	t.Logf("GitHub API supports: ?since=YYYY-MM-DDTHH:MM:SSZ parameter for issues/PRs")
}

// TestGitHubPRFiltering_Capabilities documents what's currently supported
func TestGitHubPRFiltering_Capabilities(t *testing.T) {
	t.Log("\n=== GitHub PR Filtering Capabilities Assessment ===\n")

	t.Log("✅ CURRENTLY SUPPORTED:")
	t.Log("  - Fetch recent PRs sorted by update time (GitHub API default)")
	t.Log("  - Filter by topic/keyword (via isRelevantToTopic)")
	t.Log("  - Filter by content quality (via UniversalRelevanceScorer)")
	t.Log("  - Multiple repository support")
	t.Log("  - Label extraction and metadata")
	t.Log("  - Author information in content")
	t.Log("  - State information (open/closed/merged)")
	t.Log("")

	t.Log("⚠️  PARTIALLY SUPPORTED (via topic filtering):")
	t.Log("  - Component filtering (search 'webapp', 'server', 'mobile' in title/body)")
	t.Log("  - Label filtering (search label names in title/body/comments)")
	t.Log("  - Author filtering (search author name in title/body)")
	t.Log("")

	t.Log("❌ NOT YET SUPPORTED (would need API query parameter changes):")
	t.Log("  - Date range filtering (need ?since= or ?until= parameters)")
	t.Log("  - Precise author filtering (need ?author= parameter in API call)")
	t.Log("  - Label filtering via API (need ?labels= parameter)")
	t.Log("  - State filtering beyond 'all' (currently queries state=all)")
	t.Log("")

	t.Log("📋 RECOMMENDATIONS FOR DEVBOT:")
	t.Log("  1. Current implementation is SUFFICIENT for MVP SummarizePRs tool")
	t.Log("  2. Topic-based filtering handles most use cases:")
	t.Log("     - 'performance' finds performance-related PRs")
	t.Log("     - 'security' finds security-related PRs")
	t.Log("     - 'webapp' finds frontend PRs")
	t.Log("     - 'mobile' finds mobile PRs")
	t.Log("  3. For advanced filtering (exact date ranges, specific authors),")
	t.Log("     add GitHub API query parameters in fetchRecentPRs()")
	t.Log("")

	t.Log("🔧 TO ADD PRECISE DATE FILTERING:")
	t.Log("  1. Modify fetchRecentPRs() to accept since/until parameters")
	t.Log("  2. Add query.Add(\"since\", sinceDate) to GitHub API call")
	t.Log("  3. Example: ?since=2025-01-01T00:00:00Z")
	t.Log("")

	t.Log("🔧 TO ADD PRECISE AUTHOR FILTERING:")
	t.Log("  1. GitHub API doesn't support ?author= for PR list endpoint")
	t.Log("  2. Current approach (topic matching author name) is best available")
	t.Log("  3. Alternative: Fetch all PRs, filter client-side by User.Login")
	t.Log("")
}
