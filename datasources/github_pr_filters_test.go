// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubPRFilters_Integration tests the new filtering capabilities
func TestGitHubPRFilters_Integration(t *testing.T) {
	token := os.Getenv("MM_AI_GITHUB_TOKEN")
	if token == "" {
		t.Skip("MM_AI_GITHUB_TOKEN not set, skipping integration test")
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	owner := "mattermost"
	repo := "mattermost"

	t.Run("date range filtering - last week", func(t *testing.T) {
		lastWeek := time.Now().AddDate(0, 0, -7)

		filters := &PRFilterOptions{
			Since: &lastWeek,
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 50, "github_repos", filters)

		t.Logf("Found %d PRs updated in the last week", len(docs))

		// Verify all returned PRs have LastModified within range
		for i, doc := range docs {
			if doc.LastModified != "" {
				lastMod, err := time.Parse(time.RFC3339, doc.LastModified)
				if err == nil {
					assert.True(t, lastMod.After(lastWeek) || lastMod.Equal(lastWeek),
						"PR %d should be updated after %s, got %s", i, lastWeek.Format("2006-01-02"), lastMod.Format("2006-01-02"))
				}
			}
		}
	})

	t.Run("date range filtering - last month", func(t *testing.T) {
		lastMonth := time.Now().AddDate(0, -1, 0)

		filters := &PRFilterOptions{
			Since: &lastMonth,
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 50, "github_repos", filters)

		t.Logf("Found %d PRs updated in the last month", len(docs))

		require.NotEmpty(t, docs, "Should find PRs from last month")
	})

	t.Run("date range with until - specific window", func(t *testing.T) {
		// Get PRs from 30 days ago to 7 days ago
		since := time.Now().AddDate(0, 0, -30)
		until := time.Now().AddDate(0, 0, -7)

		filters := &PRFilterOptions{
			Since: &since,
			Until: &until,
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 50, "github_repos", filters)

		t.Logf("Found %d PRs between %s and %s",
			len(docs),
			since.Format("2006-01-02"),
			until.Format("2006-01-02"))
	})

	t.Run("state filtering - open only", func(t *testing.T) {
		filters := &PRFilterOptions{
			State: "open",
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 20, "github_repos", filters)

		t.Logf("Found %d open PRs", len(docs))

		// Verify all returned PRs are open
		for _, doc := range docs {
			assert.Contains(t, doc.Content, "State: open", "PR should be in open state")
		}
	})

	t.Run("state filtering - closed only", func(t *testing.T) {
		filters := &PRFilterOptions{
			State: "closed",
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 20, "github_repos", filters)

		t.Logf("Found %d closed PRs", len(docs))

		// Verify all returned PRs are closed
		for _, doc := range docs {
			assert.Contains(t, doc.Content, "State: closed", "PR should be in closed state")
		}
	})

	t.Run("author filtering - exact match", func(t *testing.T) {
		// First, get any PR to find a real author
		allDocs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 10, "github_repos", nil)
		require.NotEmpty(t, allDocs, "Should have PRs to test with")

		testAuthor := allDocs[0].Author
		require.NotEmpty(t, testAuthor, "First PR should have an author")

		t.Logf("Testing author filtering with: %s", testAuthor)

		filters := &PRFilterOptions{
			Author: testAuthor,
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 20, "github_repos", filters)

		t.Logf("Found %d PRs by author %s", len(docs), testAuthor)

		// Verify all returned PRs are by this author
		for _, doc := range docs {
			assert.Equal(t, testAuthor, doc.Author, "PR should be by specified author")
			assert.Contains(t, doc.Content, "Author: "+testAuthor, "Content should contain author")
		}
	})

	t.Run("label filtering - single label", func(t *testing.T) {
		// Find PRs with common labels first
		allDocs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 50, "github_repos", nil)

		var testLabel string
		for _, doc := range allDocs {
			if len(doc.Labels) > 0 {
				// Extract a real label from metadata labels
				for _, label := range doc.Labels {
					// Skip internal metadata labels
					if !hasPrefix(label, []string{"segment:", "category:", "priority:", "recent_", "gh:"}) {
						continue
					}
					if hasPrefix(label, []string{"gh:"}) {
						// Extract GitHub label (format: "gh:labelname")
						testLabel = label[3:]
						break
					}
				}
				if testLabel != "" {
					break
				}
			}
		}

		if testLabel == "" {
			t.Skip("No suitable label found for testing")
		}

		t.Logf("Testing label filtering with: %s", testLabel)

		filters := &PRFilterOptions{
			Labels: []string{testLabel},
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 50, "github_repos", filters)

		t.Logf("Found %d PRs with label '%s'", len(docs), testLabel)

		if len(docs) > 0 {
			// Verify PRs have the label
			for _, doc := range docs {
				hasLabel := false
				for _, label := range doc.Labels {
					if label == "gh:"+testLabel {
						hasLabel = true
						break
					}
				}
				assert.True(t, hasLabel, "PR should have the specified label")
			}
		}
	})

	t.Run("combined filters - state and author", func(t *testing.T) {
		// Get a real author first
		allDocs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 10, "github_repos", nil)
		require.NotEmpty(t, allDocs, "Should have PRs")

		testAuthor := allDocs[0].Author

		filters := &PRFilterOptions{
			State:  "open",
			Author: testAuthor,
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "", 20, "github_repos", filters)

		t.Logf("Found %d open PRs by %s", len(docs), testAuthor)

		for _, doc := range docs {
			assert.Equal(t, testAuthor, doc.Author, "Should be by specified author")
			assert.Contains(t, doc.Content, "State: open", "Should be open")
		}
	})

	t.Run("combined filters - date and topic", func(t *testing.T) {
		lastWeek := time.Now().AddDate(0, 0, -7)

		filters := &PRFilterOptions{
			Since: &lastWeek,
		}

		docs := protocol.fetchRecentPRsWithFilters(ctx, owner, repo, "performance", 50, "github_repos", filters)

		t.Logf("Found %d performance-related PRs from last week", len(docs))

		// Each PR should be recent and performance-related
		for _, doc := range docs {
			if doc.LastModified != "" {
				lastMod, _ := time.Parse(time.RFC3339, doc.LastModified)
				if !lastMod.IsZero() {
					assert.True(t, lastMod.After(lastWeek), "Should be from last week")
				}
			}
		}
	})
}

func hasPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
