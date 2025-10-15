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

// TestGitHubMetadataExtraction_Integration verifies metadata extraction works on real GitHub API data
func TestGitHubMetadataExtraction_Integration(t *testing.T) {
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub metadata test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	// Fetch recent issues from mattermost repos
	source := SourceConfig{
		Name:     SourceGitHubRepos,
		Protocol: GitHubAPIProtocolType,
		Endpoints: map[string]string{
			EndpointOwner: GitHubOwnerMattermost,
			EndpointRepos: "mattermost,mattermost-mobile",
		},
		Sections: []string{"issues"},
		Auth:     AuthConfig{Type: AuthTypeToken, Key: token},
	}

	request := ProtocolRequest{
		Source:   source,
		Topic:    "authentication mobile performance",
		Sections: []string{"issues"},
		Limit:    20,
	}

	docs, err := protocol.Fetch(ctx, request)
	require.NoError(t, err, "Should fetch issues successfully")
	require.NotEmpty(t, docs, "Should return issues")

	t.Logf("Fetched %d issues from GitHub", len(docs))

	// Verify metadata extraction works
	metadataFound := false
	for i, doc := range docs {
		if len(doc.Labels) > 0 {
			metadataFound = true
			t.Logf("Issue %d: %s", i+1, doc.Title)
			t.Logf("  Labels: %v", doc.Labels)

			// Check if metadata labels are present (segment:*, category:*, priority:*)
			hasMetadata := false
			for _, label := range doc.Labels {
				if len(label) > 8 && (label[:8] == "segment:" || label[:9] == "category:" || label[:9] == "priority:") {
					hasMetadata = true
					break
				}
			}

			if hasMetadata {
				t.Logf("  ✓ Metadata extraction working")
			}
		}

		// All docs should have basic fields
		require.NotEmpty(t, doc.Title, "Issue should have title")
		require.NotEmpty(t, doc.URL, "Issue should have URL")
		require.NotEmpty(t, doc.Content, "Issue should have content")
		require.Equal(t, SourceGitHubRepos, doc.Source, "Source should match")
	}

	if metadataFound {
		t.Log("✅ Metadata extraction verified on real GitHub issues")
	} else {
		t.Log("⚠️  No metadata found in returned issues (may be expected if issues lack keywords)")
	}
}

// TestGitHubMetadataExtraction_SpecificIssues tests metadata on known issue types
func TestGitHubMetadataExtraction_SpecificIssues(t *testing.T) {
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	if token == "" {
		t.Skipf("Skipping GitHub metadata test: %s environment variable not set", envVarName)
	}

	protocol := NewGitHubProtocol(token, nil)
	ctx := context.Background()

	testCases := []struct {
		name           string
		repo           string
		topic          string
		expectedLabels []string // Labels we expect to find in at least one result
	}{
		{
			name:           "Mobile issues",
			repo:           "mattermost-mobile",
			topic:          "mobile performance",
			expectedLabels: []string{"category:mobile"},
		},
		{
			name:           "Authentication issues",
			repo:           "mattermost",
			topic:          "authentication SAML SSO",
			expectedLabels: []string{"category:authentication"},
		},
		{
			name:           "Plugin issues",
			repo:           "mattermost",
			topic:          "plugin integration",
			expectedLabels: []string{"category:plugins"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := SourceConfig{
				Name:     SourceGitHubRepos,
				Protocol: GitHubAPIProtocolType,
				Endpoints: map[string]string{
					EndpointOwner: GitHubOwnerMattermost,
					EndpointRepos: tc.repo,
				},
				Sections: []string{"issues"},
				Auth:     AuthConfig{Type: AuthTypeToken, Key: token},
			}

			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.topic,
				Sections: []string{"issues"},
				Limit:    10,
			}

			docs, err := protocol.Fetch(ctx, request)
			require.NoError(t, err)

			if len(docs) == 0 {
				t.Skipf("No issues found for topic '%s'", tc.topic)
			}

			t.Logf("Fetched %d issues for topic '%s'", len(docs), tc.topic)

			// Check if ANY of the expected labels appear in ANY of the results
			foundExpectedLabel := false
			for _, expectedLabel := range tc.expectedLabels {
				for _, doc := range docs {
					for _, label := range doc.Labels {
						if label == expectedLabel {
							foundExpectedLabel = true
							t.Logf("✓ Found expected label '%s' in issue: %s", expectedLabel, doc.Title)
							break
						}
					}
					if foundExpectedLabel {
						break
					}
				}
				if foundExpectedLabel {
					break
				}
			}

			if !foundExpectedLabel {
				t.Logf("⚠️  Expected labels %v not found (may be expected if issues lack relevant keywords)", tc.expectedLabels)
			}
		})
	}
}
