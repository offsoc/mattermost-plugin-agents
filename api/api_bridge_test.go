// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJSONSchema(t *testing.T) {
	tests := []struct {
		name        string
		schemaJSON  string
		expectError bool
	}{
		{
			name: "valid simple schema",
			schemaJSON: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}`,
			expectError: false,
		},
		{
			name: "valid complex schema",
			schemaJSON: `{
				"type": "object",
				"properties": {
					"summary": {"type": "string"},
					"items": {
						"type": "array",
						"items": {"type": "string"}
					}
				},
				"required": ["summary"]
			}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			schemaJSON:  `{invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := parseJSONSchema([]byte(tt.schemaJSON))
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, schema)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "JSON in code block",
			content:  "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "JSON in plain code block",
			content:  "```\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "JSON with text before",
			content:  "Here's the JSON:\n```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "Plain JSON object",
			content:  `{"key": "value", "nested": {"foo": "bar"}}`,
			expected: `{"key": "value", "nested": {"foo": "bar"}}`,
		},
		{
			name:     "JSON with extra text after",
			content:  `{"key": "value"} some extra text`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON array",
			content:  `[{"key": "value"}, {"key2": "value2"}]`,
			expected: `[{"key": "value"}, {"key2": "value2"}]`,
		},
		{
			name:     "No JSON",
			content:  "This is just plain text without JSON",
			expected: "",
		},
		{
			name:     "Multiline JSON in code block",
			content:  "```json\n{\n  \"key\": \"value\",\n  \"key2\": \"value2\"\n}\n```",
			expected: "{\n  \"key\": \"value\",\n  \"key2\": \"value2\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromMarkdown(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "hello world this is a test",
			maxLen:   10,
			expected: "hello worl...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
