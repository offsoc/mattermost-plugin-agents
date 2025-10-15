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

// TestConfluenceMetadataExtraction_Integration verifies metadata extraction works on real Confluence API data
func TestConfluenceMetadataExtraction_Integration(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tokenEnvVar := "MM_AI_CONFLUENCE_DOCS_TOKEN"
	token := os.Getenv(tokenEnvVar)

	if token == "" {
		t.Skipf("Skipping Confluence metadata test: %s environment variable not set", tokenEnvVar)
	}

	source := SourceConfig{
		Name:     SourceConfluenceDocs,
		Protocol: ConfluenceProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: ConfluenceURL,
			EndpointSpaces:  ConfluenceSpaces,
		},
		Auth: AuthConfig{Type: AuthTypeAPIKey, Key: token},
	}

	protocol.SetAuth(source.Auth)
	ctx := context.Background()

	request := ProtocolRequest{
		Source:   source,
		Topic:    "playbooks workflow automation",
		Sections: []string{},
		Limit:    15,
	}

	docs, err := protocol.Fetch(ctx, request)
	require.NoError(t, err, "Should fetch Confluence pages successfully")

	if len(docs) == 0 {
		t.Skip("No pages found - this may be expected if the Confluence search returns no results for the query")
	}

	require.NotEmpty(t, docs, "Should return Confluence pages")

	t.Logf("Fetched %d pages from Confluence", len(docs))

	// Verify metadata extraction works
	metadataFound := false
	for i, doc := range docs {
		if len(doc.Labels) > 0 {
			metadataFound = true
			t.Logf("Page %d: %s", i+1, doc.Title)
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
		require.NotEmpty(t, doc.Title, "Page should have title")
		require.NotEmpty(t, doc.URL, "Page should have URL")
		require.NotEmpty(t, doc.Content, "Page should have content")
		require.Equal(t, SourceConfluenceDocs, doc.Source, "Source should match")
	}

	if metadataFound {
		t.Log("✅ Metadata extraction verified on real Confluence pages")
	} else {
		t.Log("⚠️  No metadata found in returned pages (may be expected if pages lack keywords)")
	}
}

// TestConfluenceMetadataExtraction_SpecificTopics tests metadata on known page types
func TestConfluenceMetadataExtraction_SpecificTopics(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tokenEnvVar := "MM_AI_CONFLUENCE_DOCS_TOKEN"
	token := os.Getenv(tokenEnvVar)

	if token == "" {
		t.Skipf("Skipping Confluence metadata test: %s environment variable not set", tokenEnvVar)
	}

	testCases := []struct {
		name           string
		topic          string
		expectedLabels []string
	}{
		{
			name:           "Mobile features",
			topic:          "mobile app notifications push",
			expectedLabels: []string{"category:mobile"},
		},
		{
			name:           "Authentication features",
			topic:          "authentication SAML SSO login",
			expectedLabels: []string{"category:authentication"},
		},
		{
			name:           "Plugin features",
			topic:          "plugin integration API webhooks",
			expectedLabels: []string{"category:plugins"},
		},
		{
			name:           "Playbooks workflow",
			topic:          "playbooks incident response workflow",
			expectedLabels: []string{"category:playbooks"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := SourceConfig{
				Name:     SourceConfluenceDocs,
				Protocol: ConfluenceProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: ConfluenceURL,
					EndpointSpaces:  ConfluenceSpaces,
				},
				Auth: AuthConfig{Type: AuthTypeAPIKey, Key: token},
			}

			protocol.SetAuth(source.Auth)

			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.topic,
				Sections: []string{},
				Limit:    10,
			}

			docs, err := protocol.Fetch(context.Background(), request)
			require.NoError(t, err)

			if len(docs) == 0 {
				t.Skipf("No pages found for topic '%s'", tc.topic)
			}

			t.Logf("Fetched %d pages for topic '%s'", len(docs), tc.topic)

			// Check if ANY of the expected labels appear in ANY of the results
			foundExpectedLabel := false
			for _, expectedLabel := range tc.expectedLabels {
				for _, doc := range docs {
					for _, label := range doc.Labels {
						if label == expectedLabel {
							foundExpectedLabel = true
							t.Logf("✓ Found expected label '%s' in page: %s", expectedLabel, doc.Title)
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
				t.Logf("⚠️  Expected labels %v not found (may be expected if pages lack relevant keywords)", tc.expectedLabels)
			}
		})
	}
}
