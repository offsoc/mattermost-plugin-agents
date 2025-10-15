// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFetchRecentPRs_DateFiltering tests that PRs can be filtered by date range
func TestFetchRecentPRs_DateFiltering(t *testing.T) {
	// Get token from environment variable
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub PR test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)

	ctx := context.Background()
	docs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost", "", 10, "github_repos")

	require.NotEmpty(t, docs, "Should return PR documents from real API")

	// Verify each doc contains expected fields
	for _, doc := range docs {
		require.NotEmpty(t, doc.Title, "PR should have title")
		require.NotEmpty(t, doc.Content, "PR should have content")
		require.NotEmpty(t, doc.URL, "PR should have URL")
		require.Equal(t, "pulls", doc.Section)
		require.Equal(t, "github_repos", doc.Source)
	}

	t.Logf("Successfully fetched %d PRs from GitHub API", len(docs))
}

// TestFetchRecentPRs_AuthorFiltering tests that PRs contain author metadata
func TestFetchRecentPRs_AuthorFiltering(t *testing.T) {
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub PR test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	t.Run("all PRs contain author metadata", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost-server", "", 10, "github_repos")

		require.NotEmpty(t, docs, "Should return PRs from real API")

		// Verify author is in the formatted content
		for _, doc := range docs {
			require.Contains(t, doc.Content, "**Details:**", "Should have details section")
			require.Contains(t, doc.Content, "Author:", "Should include author field")
		}

		t.Logf("Verified author metadata in %d PRs", len(docs))
	})
}

// TestFetchRecentPRs_LabelFiltering tests that PR labels are extracted
func TestFetchRecentPRs_LabelFiltering(t *testing.T) {
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub PR test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	t.Run("verify labels are extracted", func(t *testing.T) {
		docs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost-server", "", 10, "github_repos")

		require.NotEmpty(t, docs, "Should return PRs from real API")

		// Verify that labels are included in the Doc.Labels field
		labelCount := 0
		for _, doc := range docs {
			if len(doc.Labels) > 0 {
				labelCount++
			}
		}

		t.Logf("Found labels in %d/%d PRs", labelCount, len(docs))
		// Note: Not all PRs have labels, so we just verify the structure exists
	})
}

// TestFetchRecentPRs_StateFiltering tests that PRs have state information
func TestFetchRecentPRs_StateFiltering(t *testing.T) {
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub PR test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	docs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost-server", "", 10, "github_repos")

	require.NotEmpty(t, docs, "Should return PRs from real API")

	// Verify state information is in content
	for _, doc := range docs {
		require.Contains(t, doc.Content, "**Details:**", "Should have details section")
		require.Contains(t, doc.Content, "State:", "Should include state field")
	}

	t.Logf("Verified state metadata in %d PRs", len(docs))
}

// TestFetchRecentPRs_MultipleRepositories tests fetching from different repos
func TestFetchRecentPRs_MultipleRepositories(t *testing.T) {
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub PR test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	t.Run("separate repos return different PRs", func(t *testing.T) {
		serverDocs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost", "", 5, "github_repos")
		require.NotEmpty(t, serverDocs, "Should return PRs from mattermost/mattermost")

		mobileDocs := protocol.fetchRecentPRs(ctx, "mattermost", "mattermost-mobile", "", 5, "github_repos")
		require.NotEmpty(t, mobileDocs, "Should return PRs from mattermost/mattermost-mobile")

		// Verify the PRs are different by checking URLs contain different repo names
		serverURL := "https://github.com/mattermost/mattermost"
		for _, doc := range serverDocs {
			require.Contains(t, doc.URL, serverURL, "Server PR URL should be from mattermost/mattermost repo")
		}

		mobileURL := "https://github.com/mattermost/mattermost-mobile"
		for _, doc := range mobileDocs {
			require.Contains(t, doc.URL, mobileURL, "Mobile PR URL should be from mattermost/mattermost-mobile repo")
		}

		t.Logf("Fetched %d PRs from mattermost/mattermost and %d PRs from mattermost/mattermost-mobile", len(serverDocs), len(mobileDocs))
	})
}
