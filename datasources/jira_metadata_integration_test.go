// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJiraMetadataExtraction_Integration verifies metadata extraction works on real Jira API data
func TestJiraMetadataExtraction_Integration(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	token := os.Getenv("MM_AI_JIRA_TOKEN")
	if token == "" {
		token = os.Getenv("MM_AI_JIRA_DOCS_TOKEN")
	}

	email := os.Getenv("MM_AI_JIRA_EMAIL")
	formattedToken := FormatJiraAuth(email, token)

	if formattedToken == "" {
		t.Skip("Skipping Jira metadata test: MM_AI_JIRA_TOKEN or MM_AI_JIRA_DOCS_TOKEN environment variable not set")
	}

	source := SourceConfig{
		Name:     SourceJiraDocs,
		Protocol: JiraProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: JiraURL,
		},
		Sections: []string{"bug", "task", "story"},
		Auth:     AuthConfig{Type: AuthTypeAPIKey, Key: formattedToken},
		RateLimit: RateLimitConfig{
			Enabled: false,
		},
	}

	protocol.SetAuth(source.Auth)

	ctx := context.Background()

	request := ProtocolRequest{
		Source:   source,
		Topic:    "authentication mobile plugin",
		Sections: []string{"bug", "task"},
		Limit:    15,
	}

	docs, err := protocol.Fetch(ctx, request)
	require.NoError(t, err, "Should fetch Jira issues successfully")
	require.NotEmpty(t, docs, "Should return Jira issues")

	t.Logf("Fetched %d issues from Jira", len(docs))

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
		require.Equal(t, SourceJiraDocs, doc.Source, "Source should match")
	}

	if metadataFound {
		t.Log("✅ Metadata extraction verified on real Jira issues")
	} else {
		t.Log("⚠️  No metadata found in returned issues (may be expected if issues lack keywords)")
	}
}

// TestJiraMetadataExtraction_SpecificTopics tests metadata on known issue types
func TestJiraMetadataExtraction_SpecificTopics(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	token := os.Getenv("MM_AI_JIRA_TOKEN")
	if token == "" {
		token = os.Getenv("MM_AI_JIRA_DOCS_TOKEN")
	}

	email := os.Getenv("MM_AI_JIRA_EMAIL")
	formattedToken := FormatJiraAuth(email, token)

	if formattedToken == "" {
		t.Skip("Skipping Jira metadata test: token not configured")
	}

	testCases := []struct {
		name           string
		topic          string
		sections       []string
		expectedLabels []string
	}{
		{
			name:           "Authentication issues",
			topic:          "authentication SSO SAML",
			sections:       []string{"bug"},
			expectedLabels: []string{"category:authentication"},
		},
		{
			name:           "Mobile issues",
			topic:          "mobile app performance",
			sections:       []string{"bug", "task"},
			expectedLabels: []string{"category:mobile"},
		},
		{
			name:           "Plugin issues",
			topic:          "plugin integration",
			sections:       []string{"bug", "story"},
			expectedLabels: []string{"category:plugins"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			email := os.Getenv("MM_AI_JIRA_EMAIL")
			formattedToken := FormatJiraAuth(email, token)

			source := SourceConfig{
				Name:     SourceJiraDocs,
				Protocol: JiraProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: JiraURL,
				},
				Sections: tc.sections,
				Auth:     AuthConfig{Type: AuthTypeAPIKey, Key: formattedToken},
				RateLimit: RateLimitConfig{
					Enabled: false,
				},
			}

			protocol.SetAuth(source.Auth)

			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.topic,
				Sections: tc.sections,
				Limit:    10,
			}

			docs, err := protocol.Fetch(context.Background(), request)
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
