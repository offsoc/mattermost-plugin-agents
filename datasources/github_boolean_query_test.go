// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildGitHubSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple query passthrough",
			query:    "authentication",
			expected: "authentication",
		},
		{
			name:     "boolean AND query extracts keywords",
			query:    "mobile AND notification",
			expected: "mobile notification",
		},
		{
			name:     "boolean OR query extracts keywords",
			query:    "deployment OR installation",
			expected: "deployment installation",
		},
		{
			name:     "complex nested query",
			query:    "(mobile OR web) AND (bug OR issue)",
			expected: "mobile web bug issue",
		},
		{
			name:     "quoted phrases",
			query:    "\"user management\" OR admin",
			expected: "user management admin",
		},
		{
			name:     "large boolean query includes all unique keywords",
			query:    "auth OR beta OR cache OR data OR email OR func OR guard OR hash OR init OR json OR key OR load OR main OR node OR open",
			expected: "auth beta cache data email func guard hash init json key load main node open",
		},
		{
			name:     "filters out short keywords",
			query:    "authentication AND to AND of",
			expected: "authentication",
		},
		{
			name:     "complex query from real test case",
			query:    "(api OR example OR usage OR how to use OR how to call) AND (MessageWillBePosted hook implementation)",
			expected: "api example usage how use call messagewillbeposted hook implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol := &GitHubProtocol{}
			result := protocol.buildGitHubSearchQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSimpleTerms(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "query with boolean operators and parentheses",
			query:    "(api OR example) AND (hook implementation)",
			expected: "api example hook implementation",
		},
		{
			name:     "query with quotes",
			query:    `"plugin hook" AND "message posted"`,
			expected: "plugin hook message posted",
		},
		{
			name:     "mixed case with operators",
			query:    "Error OR Debug AND Issue",
			expected: "error debug issue",
		},
		{
			name:     "filters short terms",
			query:    "a OR in AND the OR plugin",
			expected: "the plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSimpleTerms(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateAndJoinKeywords(t *testing.T) {
	tests := []struct {
		name     string
		keywords []string
		expected string
	}{
		{
			name:     "removes duplicates case-insensitive",
			keywords: []string{"Plugin", "hook", "PLUGIN", "Hook", "implementation"},
			expected: "plugin hook implementation",
		},
		{
			name:     "removes quoted strings",
			keywords: []string{`"plugin"`, `"hook"`, "implementation"},
			expected: "plugin hook implementation",
		},
		{
			name:     "filters short keywords",
			keywords: []string{"a", "in", "plugin", "to", "hook"},
			expected: "plugin hook",
		},
		{
			name:     "empty list",
			keywords: []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateAndJoinKeywords(tt.keywords)
			assert.Equal(t, tt.expected, result)
		})
	}
}
