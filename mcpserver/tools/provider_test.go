// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// TestSchemaArgs is a test struct for schema conversion testing
type TestSchemaArgs struct {
	Username string `json:"username" jsonschema_description:"The username for the test"`
	Count    int    `json:"count" jsonschema_description:"Number of items to process"`
	Enabled  bool   `json:"enabled" jsonschema_description:"Whether the feature is enabled"`
}

// TestTransportArgs is a test struct for transport validation testing
type TestTransportArgs struct {
	Message       string   `json:"message" jsonschema_description:"The message content"`
	Attachments   []string `json:"attachments,omitempty" transport:"stdio" jsonschema_description:"Optional list of file attachments"`
	HTTPOnlyField string   `json:"http_only_field,omitempty" transport:"http" jsonschema_description:"Field only available in HTTP mode"`
}

func TestConvertMCPToolToLibMCPTool_WithSchema(t *testing.T) {
	// Create a mock provider
	provider := &MattermostToolProvider{
		logger: mlog.CreateTestLogger(t),
	}

	// Create a test tool with schema
	testTool := MCPTool{
		Name:        "test_tool",
		Description: "A test tool for schema conversion",
		Schema:      llm.NewJSONSchemaFromStruct[TestSchemaArgs](),
		Resolver:    nil, // Not needed for this test
	}

	// Convert to MCP library tool
	libTool := provider.convertMCPToolToLibMCPTool(testTool)

	// Verify basic properties
	assert.Equal(t, "test_tool", libTool.Name)
	assert.Equal(t, "A test tool for schema conversion", libTool.Description)

	// Verify that RawInputSchema is populated (indicating schema conversion worked)
	assert.NotEmpty(t, libTool.RawInputSchema, "RawInputSchema should be populated when schema conversion succeeds")

	// Parse the raw schema to verify it's valid JSON and contains expected fields
	var parsedSchema map[string]interface{}
	err := json.Unmarshal(libTool.RawInputSchema, &parsedSchema)
	require.NoError(t, err, "RawInputSchema should be valid JSON")

	// Verify the schema structure contains expected properties
	properties, ok := parsedSchema["properties"].(map[string]interface{})
	require.True(t, ok, "Schema should have properties field")

	// Check that our test struct fields are in the schema (using JSON field names)
	assert.Contains(t, properties, "username", "Schema should contain username field")
	assert.Contains(t, properties, "count", "Schema should contain count field")
	assert.Contains(t, properties, "enabled", "Schema should contain enabled field")

	// Verify field types are correct
	usernameField, ok := properties["username"].(map[string]interface{})
	require.True(t, ok, "username field should be an object")
	assert.Equal(t, "string", usernameField["type"], "Username field should be string type")

	countField, ok := properties["count"].(map[string]interface{})
	require.True(t, ok, "count field should be an object")
	assert.Equal(t, "integer", countField["type"], "Count field should be integer type")

	enabledField, ok := properties["enabled"].(map[string]interface{})
	require.True(t, ok, "enabled field should be an object")
	assert.Equal(t, "boolean", enabledField["type"], "Enabled field should be boolean type")
}

func TestConvertMCPToolToLibMCPTool_WithoutSchema(t *testing.T) {
	// Create a mock provider
	provider := &MattermostToolProvider{
		logger: mlog.CreateTestLogger(t),
	}

	// Create a test tool without schema
	testTool := MCPTool{
		Name:        "test_tool_no_schema",
		Description: "A test tool without schema",
		Schema:      nil,
		Resolver:    nil, // Not needed for this test
	}

	// Convert to MCP library tool
	libTool := provider.convertMCPToolToLibMCPTool(testTool)

	// Verify basic properties
	assert.Equal(t, "test_tool_no_schema", libTool.Name)
	assert.Equal(t, "A test tool without schema", libTool.Description)

	// Verify that RawInputSchema is empty (fallback to basic tool creation)
	assert.Empty(t, libTool.RawInputSchema, "RawInputSchema should be empty when no schema is provided")
}

func TestConvertMCPToolToLibMCPTool_WithInvalidSchema(t *testing.T) {
	// Create a mock provider
	provider := &MattermostToolProvider{
		logger: mlog.CreateTestLogger(t),
	}

	// Create a test tool with invalid schema (not a *jsonschema.Schema)
	testTool := MCPTool{
		Name:        "test_tool_invalid_schema",
		Description: "A test tool with invalid schema",
		Schema:      "invalid_schema_type", // This should cause fallback
		Resolver:    nil,                   // Not needed for this test
	}

	// Convert to MCP library tool
	libTool := provider.convertMCPToolToLibMCPTool(testTool)

	// Verify basic properties
	assert.Equal(t, "test_tool_invalid_schema", libTool.Name)
	assert.Equal(t, "A test tool with invalid schema", libTool.Description)

	// Verify that RawInputSchema is empty (fallback due to invalid schema type)
	assert.Empty(t, libTool.RawInputSchema, "RawInputSchema should be empty when schema is invalid type")
}

func TestValidateTransportRestrictions_ValidFields(t *testing.T) {
	testCases := []struct {
		name          string
		jsonData      string
		transport     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "stdio transport with stdio-only field should succeed",
			jsonData:    `{"message": "hello", "attachments": ["file1.txt"]}`,
			transport:   "stdio",
			expectError: false,
		},
		{
			name:        "http transport with http-only field should succeed",
			jsonData:    `{"message": "hello", "http_only_field": "value"}`,
			transport:   "http",
			expectError: false,
		},
		{
			name:        "http transport without restricted fields should succeed",
			jsonData:    `{"message": "hello"}`,
			transport:   "http",
			expectError: false,
		},
		{
			name:          "http transport with stdio-only field should fail",
			jsonData:      `{"message": "hello", "attachments": ["file1.txt"]}`,
			transport:     "http",
			expectError:   true,
			errorContains: "field 'attachments' is not available in http transport mode",
		},
		{
			name:          "stdio transport with http-only field should fail",
			jsonData:      `{"message": "hello", "http_only_field": "value"}`,
			transport:     "stdio",
			expectError:   true,
			errorContains: "field 'http_only_field' is not available in stdio transport mode",
		},
		{
			name:          "http transport with multiple restricted fields should fail on first",
			jsonData:      `{"message": "hello", "attachments": ["file1.txt"], "http_only_field": "value"}`,
			transport:     "http",
			expectError:   true,
			errorContains: "field 'attachments' is not available in http transport mode",
		},
	}

	var target TestTransportArgs

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTransportRestrictions([]byte(tc.jsonData), &target, tc.transport)

			if tc.expectError {
				require.Error(t, err, "Expected validation to fail")
				assert.Contains(t, err.Error(), tc.errorContains, "Error message should contain expected text")
			} else {
				require.NoError(t, err, "Expected validation to succeed")
			}
		})
	}
}

func TestValidateTransportRestrictions_NonStructTarget(t *testing.T) {
	// Test with a non-struct target (should succeed without validation)
	var target string
	jsonData := `"hello world"`

	err := validateTransportRestrictions([]byte(jsonData), &target, "http")
	require.NoError(t, err, "Non-struct targets should not be validated")
}

func TestValidateTransportRestrictions_SliceTarget(t *testing.T) {
	// Test with a slice target (should succeed without validation)
	var target []string
	jsonData := `["item1", "item2"]`

	err := validateTransportRestrictions([]byte(jsonData), &target, "http")
	require.NoError(t, err, "Non-struct targets should not be validated")
}

func TestValidateTransportRestrictions_InvalidJSON(t *testing.T) {
	var target TestTransportArgs
	invalidJSON := `{"message": "hello", "attachments"`

	err := validateTransportRestrictions([]byte(invalidJSON), &target, "stdio")
	require.NoError(t, err, "Invalid JSON that can't be parsed as object should be allowed (not parsed as struct)")
}

func TestValidateTransportRestrictions_AttackScenario(t *testing.T) {
	// This test simulates the attack scenario described in the issue:
	// Someone creates a POST request to the create_post tool in HTTP mode
	// and tries to add an attachment field, which should only be available in stdio mode
	
	// Simulate a malicious HTTP request trying to send attachments
	maliciousHttpRequest := `{
		"channel_id": "channel123",
		"message": "This is a test post",
		"attachments": ["malicious_file.txt", "another_file.pdf"]
	}`

	// This should be similar to CreatePostArgs from posts.go
	type CreatePostArgsSimulated struct {
		ChannelID   string   `json:"channel_id"`
		Message     string   `json:"message"`
		Attachments []string `json:"attachments,omitempty" transport:"stdio"`
	}

	var target CreatePostArgsSimulated

	// Validate that HTTP transport rejects stdio-only fields
	err := validateTransportRestrictions([]byte(maliciousHttpRequest), &target, "http")
	require.Error(t, err, "HTTP transport should reject stdio-only attachments field")
	assert.Contains(t, err.Error(), "field 'attachments' is not available in http transport mode")
	
	// Validate that stdio transport allows the same request
	err = validateTransportRestrictions([]byte(maliciousHttpRequest), &target, "stdio")
	require.NoError(t, err, "stdio transport should allow attachments field")

	// Validate that a clean HTTP request without restricted fields works
	cleanHttpRequest := `{
		"channel_id": "channel123", 
		"message": "This is a clean test post"
	}`
	
	err = validateTransportRestrictions([]byte(cleanHttpRequest), &target, "http")
	require.NoError(t, err, "HTTP transport should allow requests without restricted fields")
}
