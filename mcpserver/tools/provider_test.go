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
	Username string `jsonschema_description:"The username for the test"`
	Count    int    `jsonschema_description:"Number of items to process"`
	Enabled  bool   `jsonschema_description:"Whether the feature is enabled"`
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
		Schema:      llm.NewJSONSchemaFromStruct(TestSchemaArgs{}),
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
	var schema map[string]interface{}
	err := json.Unmarshal(libTool.RawInputSchema, &schema)
	require.NoError(t, err, "RawInputSchema should be valid JSON")

	// Verify the schema structure contains expected properties
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "Schema should have properties field")

	// Check that our test struct fields are in the schema (using Go field names)
	assert.Contains(t, properties, "Username", "Schema should contain Username field")
	assert.Contains(t, properties, "Count", "Schema should contain Count field")
	assert.Contains(t, properties, "Enabled", "Schema should contain Enabled field")

	// Verify field descriptions are preserved
	usernameField, ok := properties["Username"].(map[string]interface{})
	require.True(t, ok, "Username field should be an object")
	assert.Equal(t, "The username for the test", usernameField["description"], "Field descriptions should be preserved")
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
