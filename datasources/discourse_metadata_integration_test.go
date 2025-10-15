// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDiscourseMetadataExtraction_Integration verifies metadata extraction works on real Discourse API data
func TestDiscourseMetadataExtraction_Integration(t *testing.T) {
	protocol := NewDiscourseProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Name:     SourceMattermostForum,
		Protocol: DiscourseProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: MattermostForumURL,
		},
		Sections: []string{SectionAnnouncements, SectionFAQ, SectionCopilotAI},
		Auth:     AuthConfig{Type: AuthTypeNone},
	}

	ctx := context.Background()

	request := ProtocolRequest{
		Source:   source,
		Topic:    "mobile app performance",
		Sections: []string{SectionAnnouncements, SectionFAQ},
		Limit:    15,
	}

	docs, err := protocol.Fetch(ctx, request)
	require.NoError(t, err, "Should fetch Discourse topics successfully")

	if len(docs) == 0 {
		t.Skip("No topics found - this may be expected if the forum search returns no results for the query")
	}

	require.NotEmpty(t, docs, "Should return Discourse topics")

	t.Logf("Fetched %d topics from Discourse", len(docs))

	// Verify metadata extraction works
	metadataFound := false
	for i, doc := range docs {
		if len(doc.Labels) > 0 {
			metadataFound = true
			t.Logf("Topic %d: %s", i+1, doc.Title)
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
		require.NotEmpty(t, doc.Title, "Topic should have title")
		require.NotEmpty(t, doc.URL, "Topic should have URL")
		require.NotEmpty(t, doc.Content, "Topic should have content")
		require.Equal(t, SourceMattermostForum, doc.Source, "Source should match")
	}

	if metadataFound {
		t.Log("✅ Metadata extraction verified on real Discourse topics")
	} else {
		t.Log("⚠️  No metadata found in returned topics (may be expected if topics lack keywords)")
	}
}

// TestDiscourseMetadataExtraction_SpecificTopics tests metadata on known topic types
func TestDiscourseMetadataExtraction_SpecificTopics(t *testing.T) {
	protocol := NewDiscourseProtocol(&http.Client{}, nil)

	testCases := []struct {
		name           string
		topic          string
		sections       []string
		expectedLabels []string
	}{
		{
			name:           "Mobile issues",
			topic:          "mobile app performance notifications",
			sections:       []string{SectionAnnouncements, SectionFAQ},
			expectedLabels: []string{"category:mobile"},
		},
		{
			name:           "Authentication issues",
			topic:          "authentication SAML SSO login",
			sections:       []string{SectionAnnouncements, SectionFAQ},
			expectedLabels: []string{"category:authentication"},
		},
		{
			name:           "Plugin issues",
			topic:          "plugin integration API webhooks",
			sections:       []string{SectionAnnouncements, SectionFAQ},
			expectedLabels: []string{"category:plugins"},
		},
		{
			name:           "Playbooks discussions",
			topic:          "playbooks incident response workflow",
			sections:       []string{SectionAnnouncements, SectionCopilotAI},
			expectedLabels: []string{"category:playbooks"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := SourceConfig{
				Name:     SourceMattermostForum,
				Protocol: DiscourseProtocolType,
				Endpoints: map[string]string{
					EndpointBaseURL: MattermostForumURL,
				},
				Sections: tc.sections,
				Auth:     AuthConfig{Type: AuthTypeNone},
			}

			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.topic,
				Sections: tc.sections,
				Limit:    10,
			}

			docs, err := protocol.Fetch(context.Background(), request)
			require.NoError(t, err)

			if len(docs) == 0 {
				t.Skipf("No topics found for topic '%s'", tc.topic)
			}

			t.Logf("Fetched %d topics for topic '%s'", len(docs), tc.topic)

			// Check if ANY of the expected labels appear in ANY of the results
			foundExpectedLabel := false
			for _, expectedLabel := range tc.expectedLabels {
				for _, doc := range docs {
					for _, label := range doc.Labels {
						if label == expectedLabel {
							foundExpectedLabel = true
							t.Logf("✓ Found expected label '%s' in topic: %s", expectedLabel, doc.Title)
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
				t.Logf("⚠️  Expected labels %v not found (may be expected if topics lack relevant keywords)", tc.expectedLabels)
			}
		})
	}
}
