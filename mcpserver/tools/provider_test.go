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

// TestAccessArgs is a test struct for access validation testing
type TestAccessArgs struct {
	Message         string   `json:"message" jsonschema_description:"The message content"`
	Attachments     []string `json:"attachments,omitempty" access:"local" jsonschema_description:"Optional list of file attachments"`
	RemoteOnlyField string   `json:"remote_only_field,omitempty" access:"remote" jsonschema_description:"Field only available in remote mode"`
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

func TestValidateAccessRestrictions_ValidFields(t *testing.T) {
	testCases := []struct {
		name          string
		jsonData      string
		accessMode    string
		expectError   bool
		errorContains string
	}{
		{
			name:        "local access mode with local-only field should succeed",
			jsonData:    `{"message": "hello", "attachments": ["file1.txt"]}`,
			accessMode:  "local",
			expectError: false,
		},
		{
			name:        "remote access mode with remote-only field should succeed",
			jsonData:    `{"message": "hello", "remote_only_field": "value"}`,
			accessMode:  "remote",
			expectError: false,
		},
		{
			name:        "remote access mode without restricted fields should succeed",
			jsonData:    `{"message": "hello"}`,
			accessMode:  "remote",
			expectError: false,
		},
		{
			name:          "remote access mode with local-only field should fail",
			jsonData:      `{"message": "hello", "attachments": ["file1.txt"]}`,
			accessMode:    "remote",
			expectError:   true,
			errorContains: "field 'attachments' is not available in remote access mode",
		},
		{
			name:          "local access mode with remote-only field should fail",
			jsonData:      `{"message": "hello", "remote_only_field": "value"}`,
			accessMode:    "local",
			expectError:   true,
			errorContains: "field 'remote_only_field' is not available in local access mode",
		},
		{
			name:          "remote access mode with multiple restricted fields should fail on first",
			jsonData:      `{"message": "hello", "attachments": ["file1.txt"], "remote_only_field": "value"}`,
			accessMode:    "remote",
			expectError:   true,
			errorContains: "field 'attachments' is not available in remote access mode",
		},
	}

	var target TestAccessArgs

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAccessRestrictions([]byte(tc.jsonData), &target, tc.accessMode)

			if tc.expectError {
				require.Error(t, err, "Expected validation to fail")
				assert.Contains(t, err.Error(), tc.errorContains, "Error message should contain expected text")
			} else {
				require.NoError(t, err, "Expected validation to succeed")
			}
		})
	}
}

func TestValidateAccessRestrictions_NonStructTarget(t *testing.T) {
	// Test with a non-struct target (should succeed without validation)
	var target string
	jsonData := `"hello world"`

	err := validateAccessRestrictions([]byte(jsonData), &target, "remote")
	require.NoError(t, err, "Non-struct targets should not be validated")
}

func TestValidateAccessRestrictions_SliceTarget(t *testing.T) {
	// Test with a slice target (should succeed without validation)
	var target []string
	jsonData := `["item1", "item2"]`

	err := validateAccessRestrictions([]byte(jsonData), &target, "remote")
	require.NoError(t, err, "Non-struct targets should not be validated")
}

func TestValidateAccessRestrictions_InvalidJSON(t *testing.T) {
	var target TestAccessArgs
	invalidJSON := `{"message": "hello", "attachments"`

	err := validateAccessRestrictions([]byte(invalidJSON), &target, "local")
	require.NoError(t, err, "Invalid JSON that can't be parsed as object should be allowed (not parsed as struct)")
}

func TestValidateAccessRestrictions_AttackScenario(t *testing.T) {
	// This test simulates a realistic attack scenario:
	// Someone creates a remote HTTP request to a post creation tool and tries to
	// include attachments, which should only be available in local access mode

	// Simulate a malicious HTTP request trying to send attachments via remote access
	maliciousRemoteRequest := `{
		"channel_id": "channel123",
		"message": "This is a test post",
		"attachments": ["/etc/passwd"]
	}`

	// Use the actual CreatePostArgs-like structure from our codebase
	type CreatePostArgsSimulated struct {
		ChannelID   string   `json:"channel_id"`
		Message     string   `json:"message"`
		Attachments []string `json:"attachments,omitempty" access:"local"`
	}

	var target CreatePostArgsSimulated

	// Validate that remote access mode rejects local-only attachment fields
	err := validateAccessRestrictions([]byte(maliciousRemoteRequest), &target, "remote")
	require.Error(t, err, "Remote access mode should reject local-only attachments field")
	assert.Contains(t, err.Error(), "field 'attachments' is not available in remote access mode")

	// Validate that local access mode allows the same request
	err = validateAccessRestrictions([]byte(maliciousRemoteRequest), &target, "local")
	require.NoError(t, err, "Local access mode should allow attachments field")

	// Validate that a clean remote request without restricted fields works
	cleanRemoteRequest := `{
		"channel_id": "channel123", 
		"message": "This is a clean test post"
	}`

	err = validateAccessRestrictions([]byte(cleanRemoteRequest), &target, "remote")
	require.NoError(t, err, "Remote access mode should allow requests without restricted fields")
}
