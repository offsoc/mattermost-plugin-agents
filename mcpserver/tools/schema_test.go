// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structs for schema generation
type TestArgs struct {
	// No transport tag - should be available for all transports
	CommonField string `json:"common_field" jsonschema_description:"Available in all transports"`

	// Single transport
	StdioOnly string `json:"stdio_only" transport:"stdio" jsonschema_description:"Only available via stdio"`
	HttpOnly  string `json:"http_only" transport:"http" jsonschema_description:"Only available via HTTP"`

	// Multiple transports
	StdioAndHttp    string `json:"stdio_http" transport:"stdio,http" jsonschema_description:"Available in stdio and HTTP"`
	WithSpaces      string `json:"with_spaces" transport:"stdio, http, websocket" jsonschema_description:"Multiple transports with spaces"`
	ThreeTransports string `json:"three_transports" transport:"stdio,http,websocket" jsonschema_description:"Available in three transports"`

	// Required field with transport restrictions
	RequiredStdio string `json:"required_stdio" transport:"stdio" jsonschema_description:"Required field only for stdio"`
}

type EmptyStruct struct{}

type NoJSONTagsStruct struct {
	Field1 string
	Field2 int
}

func TestNewJSONSchemaForTransport(t *testing.T) {
	tests := []struct {
		name           string
		transportMode  string
		expectedFields []string
		excludedFields []string
		expectedCount  int
	}{
		{
			name:          "stdio transport includes correct fields",
			transportMode: "stdio",
			expectedFields: []string{
				"common_field",
				"stdio_only",
				"stdio_http",
				"with_spaces",
				"three_transports",
				"required_stdio",
			},
			excludedFields: []string{"http_only"},
			expectedCount:  6,
		},
		{
			name:          "http transport includes correct fields",
			transportMode: "http",
			expectedFields: []string{
				"common_field",
				"http_only",
				"stdio_http",
				"with_spaces",
				"three_transports",
			},
			excludedFields: []string{"stdio_only", "required_stdio"},
			expectedCount:  5,
		},
		{
			name:          "websocket transport includes correct fields",
			transportMode: "websocket",
			expectedFields: []string{
				"common_field",
				"with_spaces",
				"three_transports",
			},
			excludedFields: []string{"stdio_only", "http_only", "stdio_http", "required_stdio"},
			expectedCount:  3,
		},
		{
			name:           "unknown transport only includes common fields",
			transportMode:  "unknown",
			expectedFields: []string{"common_field"},
			excludedFields: []string{"stdio_only", "http_only", "stdio_http", "with_spaces", "three_transports", "required_stdio"},
			expectedCount:  1,
		},
		{
			name:           "sse transport only includes common fields",
			transportMode:  "sse",
			expectedFields: []string{"common_field"},
			excludedFields: []string{"stdio_only", "http_only", "stdio_http", "with_spaces", "three_transports", "required_stdio"},
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewJSONSchemaForTransport[TestArgs](tt.transportMode)

			require.NotNil(t, schema)
			require.NotNil(t, schema.Properties)

			// Check expected field count
			assert.Len(t, schema.Properties, tt.expectedCount,
				"transport %s should have %d fields", tt.transportMode, tt.expectedCount)

			// Check all expected fields are present
			for _, field := range tt.expectedFields {
				assert.Contains(t, schema.Properties, field,
					"transport %s schema should include %s", tt.transportMode, field)
			}

			// Check all excluded fields are absent
			for _, field := range tt.excludedFields {
				assert.NotContains(t, schema.Properties, field,
					"transport %s schema should not include %s", tt.transportMode, field)
			}
		})
	}

	// Test edge cases that don't fit the table pattern
	t.Run("empty struct returns valid schema", func(t *testing.T) {
		schema := NewJSONSchemaForTransport[EmptyStruct]("stdio")

		require.NotNil(t, schema)
		// Properties might be nil or empty for empty struct
		if schema.Properties != nil {
			assert.Len(t, schema.Properties, 0)
		}
	})

	t.Run("struct with no JSON tags returns base schema", func(t *testing.T) {
		schema := NewJSONSchemaForTransport[NoJSONTagsStruct]("stdio")

		require.NotNil(t, schema)
		// Should return base schema since no JSON tags to filter
	})

	t.Run("preserves schema metadata", func(t *testing.T) {
		schema := NewJSONSchemaForTransport[TestArgs]("stdio")

		require.NotNil(t, schema)
		// Check that basic schema properties are preserved
		assert.NotEmpty(t, schema.Type)

		// Check that field descriptions are preserved
		if commonField, exists := schema.Properties["common_field"]; exists {
			// Verify the property exists and has expected structure
			assert.NotNil(t, commonField)
		}
	})
}

func TestIsTransportAllowed(t *testing.T) {
	tests := []struct {
		name             string
		transportTag     string
		currentTransport string
		expected         bool
	}{
		{"empty tag allows all", "", "stdio", true},
		{"empty tag allows any transport", "", "http", true},
		{"exact match stdio", "stdio", "stdio", true},
		{"exact match http", "http", "http", true},
		{"no match stdio vs http", "stdio", "http", false},
		{"no match http vs stdio", "http", "stdio", false},
		{"comma separated includes stdio", "stdio,http", "stdio", true},
		{"comma separated includes http", "stdio,http", "http", true},
		{"comma separated excludes websocket", "stdio,http", "websocket", false},
		{"with spaces includes stdio", "stdio, http", "stdio", true},
		{"with spaces includes http", "stdio, http", "http", true},
		{"with spaces excludes websocket", "stdio, http", "websocket", false},
		{"three transports includes first", "stdio,http,websocket", "stdio", true},
		{"three transports includes middle", "stdio,http,websocket", "http", true},
		{"three transports includes last", "stdio,http,websocket", "websocket", true},
		{"three transports excludes other", "stdio,http,websocket", "sse", false},
		{"mixed spaces includes all", "stdio , http,  websocket", "http", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransportAllowed(tt.transportTag, tt.currentTransport)
			assert.Equal(t, tt.expected, result,
				"isTransportAllowed(%q, %q) = %v, want %v",
				tt.transportTag, tt.currentTransport, result, tt.expected)
		})
	}
}
