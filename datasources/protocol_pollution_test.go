// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"strings"
	"testing"
)

// TestProtocolQuality_WrongSourcePollution tests the actual PM bot problem:
// When querying for SAML from JIRA, does Zendesk/Hub pollute results?
func TestProtocolQuality_WrongSourcePollution(t *testing.T) {
	config := CreateDefaultConfig()

	// Simulate PM bot calling ALL sources for "SAML authentication" query
	sources := []string{
		SourceJiraDocs,
		SourceZendeskTickets,
		SourceMattermostHub,
	}

	query := "SAML authentication SSO"
	expectedTopics := []string{"saml", "sso", "authentication", "auth", "login"}

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), ExtendedTestTimeout)
	defer cancel()

	t.Logf("\n=== Testing Wrong Source Pollution ===")
	t.Logf("Query: %s", query)
	t.Logf("Simulating PM bot calling ALL sources (not just JIRA)\n")

	var allDocs []Doc
	sourceResults := make(map[string][]Doc)

	for _, sourceName := range sources {
		// Check if source is enabled
		sourceEnabled := false
		for _, source := range config.Sources {
			if source.Name == sourceName && source.Enabled {
				sourceEnabled = true
				break
			}
		}

		if !sourceEnabled {
			t.Logf("⚠️  Skipping %s (not enabled)", sourceName)
			continue
		}

		docs, err := client.FetchFromSource(ctx, sourceName, query, 5)
		if err != nil {
			t.Logf("⚠️  %s fetch failed: %v", sourceName, err)
			continue
		}

		sourceResults[sourceName] = docs
		allDocs = append(allDocs, docs...)
	}

	t.Logf("\n=== Results by Source ===")
	for sourceName, docs := range sourceResults {
		if len(docs) == 0 {
			t.Logf("\n%s: No results", sourceName)
			continue
		}

		relevantCount := 0
		t.Logf("\n%s: %d results", sourceName, len(docs))
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
				relevantCount++
			}

			status := "❌"
			if isRelevant {
				status = "✅"
			}

			t.Logf("  [%d] %s %s", i+1, status, doc.Title)
		}

		relevanceRate := float64(relevantCount) / float64(len(docs)) * 100
		t.Logf("  📊 %s relevance: %d/%d (%.1f%%)", sourceName, relevantCount, len(docs), relevanceRate)
	}

	// Overall analysis
	t.Logf("\n=== Overall Analysis ===")
	t.Logf("Total documents returned: %d", len(allDocs))

	relevantCount := 0
	for _, doc := range allDocs {
		content := strings.ToLower(doc.Title + " " + doc.Content)
		for _, topic := range expectedTopics {
			if strings.Contains(content, strings.ToLower(topic)) {
				relevantCount++
				break
			}
		}
	}

	if len(allDocs) > 0 {
		overallRate := float64(relevantCount) / float64(len(allDocs)) * 100
		t.Logf("Overall relevance: %d/%d (%.1f%%)", relevantCount, len(allDocs), overallRate)

		if overallRate < 50 {
			t.Logf("\n❌ PROBLEM CONFIRMED: Multi-source pollution reduces relevance below 50%%")
			t.Logf("This is NOT a protocol quality issue - protocols work fine individually")
			t.Logf("This IS a routing issue - PM service should not query wrong sources")
		} else {
			t.Logf("\n✅ No pollution detected - all sources return relevant results")
		}
	}
}
