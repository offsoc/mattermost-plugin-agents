// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBooleanQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple keyword",
			query:   "mobile",
			wantErr: false,
		},
		{
			name:    "AND expression",
			query:   "mobile AND bug",
			wantErr: false,
		},
		{
			name:    "OR expression",
			query:   "mobile OR web",
			wantErr: false,
		},
		{
			name:    "NOT expression",
			query:   "NOT obsolete",
			wantErr: false,
		},
		{
			name:    "parentheses",
			query:   "(mobile OR web) AND bug",
			wantErr: false,
		},
		{
			name:    "complex nested",
			query:   "(mobile OR web) AND (bug OR issue) AND NOT obsolete",
			wantErr: false,
		},
		{
			name:    "quoted phrase",
			query:   `"mobile app" AND bug`,
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseBooleanQuery(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
			}
		})
	}
}

func TestEvaluateBoolean(t *testing.T) {
	tests := []struct {
		name  string
		query string
		text  string
		want  bool
	}{
		{
			name:  "simple match",
			query: "mobile",
			text:  "This is about mobile app development",
			want:  true,
		},
		{
			name:  "simple no match",
			query: "mobile",
			text:  "This is about web development",
			want:  false,
		},
		{
			name:  "AND both match",
			query: "mobile AND bug",
			text:  "Mobile app has a bug",
			want:  true,
		},
		{
			name:  "AND one missing",
			query: "mobile AND bug",
			text:  "Mobile app works fine",
			want:  false,
		},
		{
			name:  "OR one matches",
			query: "mobile OR web",
			text:  "Web application issue",
			want:  true,
		},
		{
			name:  "OR both match",
			query: "mobile OR web",
			text:  "Mobile and web applications",
			want:  true,
		},
		{
			name:  "OR none match",
			query: "mobile OR web",
			text:  "Desktop application",
			want:  false,
		},
		{
			name:  "NOT matches (should fail)",
			query: "NOT obsolete",
			text:  "This is obsolete",
			want:  false,
		},
		{
			name:  "NOT doesn't match (should succeed)",
			query: "NOT obsolete",
			text:  "This is current",
			want:  true,
		},
		{
			name:  "complex expression - all conditions met",
			query: "(mobile OR web) AND (bug OR issue)",
			text:  "Mobile app has a critical bug",
			want:  true,
		},
		{
			name:  "complex expression - first part fails",
			query: "(mobile OR web) AND (bug OR issue)",
			text:  "Desktop has a bug",
			want:  false,
		},
		{
			name:  "complex expression - second part fails",
			query: "(mobile OR web) AND (bug OR issue)",
			text:  "Mobile app works perfectly",
			want:  false,
		},
		{
			name:  "complex with NOT",
			query: "(mobile OR web) AND bug AND NOT obsolete",
			text:  "Mobile app has a bug",
			want:  true,
		},
		{
			name:  "complex with NOT fails",
			query: "(mobile OR web) AND bug AND NOT obsolete",
			text:  "Mobile app has an obsolete bug",
			want:  false,
		},
		{
			name:  "quoted phrase match",
			query: `"mobile app"`,
			text:  "The mobile app is great",
			want:  true,
		},
		{
			name:  "quoted phrase no match (words separated)",
			query: `"mobile app"`,
			text:  "The mobile web app is great",
			want:  false,
		},
		{
			name:  "case insensitive",
			query: "MOBILE AND BUG",
			text:  "mobile app has a bug",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseBooleanQuery(tt.query)
			require.NoError(t, err, "Query parsing should succeed for test: %s", tt.name)

			result := EvaluateBoolean(node, tt.text)
			assert.Equal(t, tt.want, result, "Query: %s, Text: %s", tt.query, tt.text)
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "simple keyword",
			query:    "mobile",
			expected: []string{"mobile"},
		},
		{
			name:     "AND expression",
			query:    "mobile AND bug",
			expected: []string{"mobile", "bug"},
		},
		{
			name:     "OR expression",
			query:    "mobile OR web",
			expected: []string{"mobile", "web"},
		},
		{
			name:     "complex nested",
			query:    "(mobile OR web) AND (bug OR issue)",
			expected: []string{"mobile", "web", "bug", "issue"},
		},
		{
			name:     "with NOT",
			query:    "mobile AND NOT obsolete",
			expected: []string{"mobile", "obsolete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseBooleanQuery(tt.query)
			require.NoError(t, err)

			keywords := ExtractKeywords(node)
			assert.ElementsMatch(t, tt.expected, keywords)
		})
	}
}

func TestTokenizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "simple",
			query:    "mobile",
			expected: []string{"mobile"},
		},
		{
			name:     "with AND",
			query:    "mobile AND bug",
			expected: []string{"mobile", "AND", "bug"},
		},
		{
			name:     "with parentheses",
			query:    "(mobile OR web)",
			expected: []string{"(", "mobile", "OR", "web", ")"},
		},
		{
			name:     "quoted phrase",
			query:    `"mobile app"`,
			expected: []string{"mobile app"},
		},
		{
			name:     "mixed",
			query:    `(mobile OR "web app") AND bug`,
			expected: []string{"(", "mobile", "OR", "web app", ")", "AND", "bug"},
		},
		{
			name:     "extra whitespace",
			query:    "mobile  AND   bug",
			expected: []string{"mobile", "AND", "bug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizeQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
