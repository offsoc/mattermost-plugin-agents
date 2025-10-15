// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/mmtools/pm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBooleanQuery(t *testing.T) {
	tests := []struct {
		name             string
		intentKeywords   []string
		featureKeywords  []string
		expectedContains []string
		expectedPattern  string
	}{
		{
			name:            "Single word keywords",
			intentKeywords:  []string{"market", "competitive"},
			featureKeywords: []string{"channels", "playbooks"},
			expectedPattern: "(market OR competitive) AND (channels OR playbooks)",
		},
		{
			name:            "Multi-word intent keywords",
			intentKeywords:  []string{"user needs", "market share", "competitive advantage"},
			featureKeywords: []string{"channels", "ai"},
			expectedContains: []string{
				"\"user needs\"",
				"\"market share\"",
				"\"competitive advantage\"",
				"channels",
				"ai",
			},
			expectedPattern: `("user needs" OR "market share" OR "competitive advantage") AND (channels OR ai)`,
		},
		{
			name:            "Multi-word feature keywords",
			intentKeywords:  []string{"market", "trends"},
			featureKeywords: []string{"ai channels", "workflow automation", "boards"},
			expectedContains: []string{
				"market",
				"trends",
				"\"ai channels\"",
				"\"workflow automation\"",
				"boards",
			},
			expectedPattern: `(market OR trends) AND ("ai channels" OR "workflow automation" OR boards)`,
		},
		{
			name:            "Mixed single and multi-word keywords",
			intentKeywords:  []string{"pain points", "limitations", "not supported"},
			featureKeywords: []string{"mobile", "push notifications", "sso"},
			expectedContains: []string{
				"\"pain points\"",
				"limitations",
				"\"not supported\"",
				"mobile",
				"\"push notifications\"",
				"sso",
			},
			expectedPattern: `("pain points" OR limitations OR "not supported") AND (mobile OR "push notifications" OR sso)`,
		},
		{
			name:            "Empty intent keywords",
			intentKeywords:  []string{},
			featureKeywords: []string{"channels"},
			expectedPattern: "",
		},
		{
			name:            "Empty feature keywords",
			intentKeywords:  []string{"market"},
			featureKeywords: []string{},
			expectedPattern: "market",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildBooleanQuery(tt.intentKeywords, tt.featureKeywords)

			if tt.expectedPattern == "" {
				assert.Empty(t, result, "Expected empty query for empty inputs")
				return
			}

			assert.Equal(t, tt.expectedPattern, result, "Query should match expected pattern")

			// Verify all expected strings are present
			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected, "Query should contain: %s", expected)
			}

			// Verify multi-word phrases (spaces or hyphens) are quoted
			for _, keyword := range tt.intentKeywords {
				if strings.Contains(keyword, " ") || strings.Contains(keyword, "-") {
					assert.Contains(t, result, "\""+keyword+"\"", "Multi-word intent keyword should be quoted: %s", keyword)
				}
			}

			for _, keyword := range tt.featureKeywords {
				if strings.Contains(keyword, " ") || strings.Contains(keyword, "-") {
					assert.Contains(t, result, "\""+keyword+"\"", "Multi-word feature keyword should be quoted: %s", keyword)
				}
			}
		})
	}
}

func TestBuildBooleanQuery_RealToolMetadata(t *testing.T) {
	// Test with actual metadata from PMToolMetadata to ensure real queries are properly formatted
	tests := []struct {
		name              string
		toolName          string
		expectedQuoted    []string
		expectedNotQuoted []string
	}{
		{
			name:     "CompileMarketResearch metadata",
			toolName: "CompileMarketResearch",
			expectedQuoted: []string{
				"user needs",
				"third-party",
			},
			expectedNotQuoted: []string{
				"market",
				"competitive",
				"analysis",
			},
		},
		{
			name:     "AnalyzeFeatureGaps metadata",
			toolName: "AnalyzeFeatureGaps",
			expectedQuoted: []string{
				"not supported",
				"pain points",
				"customer objections",
				"deal blockers",
				"adoption barriers",
				"feature specs",
				"customer needs",
				"acceptance criteria",
				"user stories",
			},
			expectedNotQuoted: []string{
				"gaps",
				"limitations",
				"missing",
			},
		},
		{
			name:     "AnalyzeStrategicAlignment metadata",
			toolName: "AnalyzeStrategicAlignment",
			expectedQuoted: []string{
				"north star",
				"strategic plan",
				"business case",
			},
			expectedNotQuoted: []string{
				"vision",
				"mission",
				"strategy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, exists := pm.GetToolMetadata(tt.toolName)
			require.True(t, exists, "Tool metadata should exist")

			// Use a simple feature keyword for testing
			featureKeywords := []string{"channels", "ai integration"}

			query := BuildBooleanQuery(metadata.IntentKeywords, featureKeywords)
			require.NotEmpty(t, query, "Query should not be empty")

			// Verify multi-word phrases are quoted
			for _, phrase := range tt.expectedQuoted {
				assert.Contains(t, query, "\""+phrase+"\"", "Multi-word phrase should be quoted: %s", phrase)
			}

			// Verify single words are not quoted
			for _, word := range tt.expectedNotQuoted {
				// Check that the word appears but not as "word" (with quotes)
				assert.Contains(t, query, word, "Single word should appear in query: %s", word)
				// Make sure it's not quoted (this is a bit tricky as it might be part of a larger quoted phrase)
				// We check that it appears without surrounding quotes in the OR clause
				assert.True(t,
					strings.Contains(query, " "+word+" ") ||
						strings.Contains(query, "("+word+" ") ||
						strings.Contains(query, " "+word+")"),
					"Single word should not be quoted: %s", word)
			}

			// Verify the feature keywords are also properly quoted
			assert.Contains(t, query, "channels", "Feature keyword should be present")
			assert.Contains(t, query, "\"ai integration\"", "Multi-word feature should be quoted")
		})
	}
}
