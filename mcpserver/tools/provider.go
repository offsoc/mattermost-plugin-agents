// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/auth"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
)

// MCPToolContext provides MCP-specific functionality with the authenticated client
type MCPToolContext struct {
	Client     *model.Client4
	AccessMode types.AccessMode
}

// MCPToolResolver defines the signature for MCP tool resolvers
type MCPToolResolver func(*MCPToolContext, llm.ToolArgumentGetter) (string, error)

// MCPTool represents a tool specifically for MCP use with our custom context
type MCPTool struct {
	Name        string
	Description string
	Schema      interface{}
	Resolver    MCPToolResolver
}

type ToolProvider interface {
	ProvideTools(*server.MCPServer)
}

// MattermostToolProvider provides Mattermost tools following the mmtools pattern
type MattermostToolProvider struct {
	authProvider        auth.AuthenticationProvider
	logger              mlog.LoggerIFace
	mmServerURL         string // External server URL for OAuth redirects
	mmInternalServerURL string // Internal server URL for API communication
	devMode             bool
	accessMode          types.AccessMode
}

// NewMattermostToolProvider creates a new tool provider
func NewMattermostToolProvider(authProvider auth.AuthenticationProvider, logger mlog.LoggerIFace, mmServerURL, mmInternalServerURL string, devMode bool, accessMode types.AccessMode) *MattermostToolProvider {
	// Use internal URL for API communication if provided, otherwise fallback to external URL
	internalURL := mmInternalServerURL
	if internalURL == "" {
		internalURL = mmServerURL
	}

	return &MattermostToolProvider{
		authProvider:        authProvider,
		logger:              logger,
		mmServerURL:         mmServerURL,
		mmInternalServerURL: internalURL,
		devMode:             devMode,
		accessMode:          accessMode,
	}
}

// ProvideTools provides all tools to the MCP server by registering them
func (p *MattermostToolProvider) ProvideTools(mcpServer *server.MCPServer) {
	mcpTools := []MCPTool{}

	// Add regular tools
	mcpTools = append(mcpTools, p.getPostTools()...)
	mcpTools = append(mcpTools, p.getChannelTools()...)
	mcpTools = append(mcpTools, p.getTeamTools()...)
	mcpTools = append(mcpTools, p.getSearchTools()...)

	// Add dev tools if dev mode is enabled
	if p.devMode {
		mcpTools = append(mcpTools, p.getDevUserTools()...)
		mcpTools = append(mcpTools, p.getDevPostTools()...)
		mcpTools = append(mcpTools, p.getDevTeamTools()...)
		mcpTools = append(mcpTools, p.getDevChannelTools()...)
	}

	// Convert and register each tool
	for _, mcpTool := range mcpTools {
		libMCPTool := p.convertMCPToolToLibMCPTool(mcpTool)
		mcpServer.AddTool(libMCPTool, p.createMCPToolHandler(mcpTool.Resolver))
	}
}

// convertMCPToolToLibMCPTool converts our MCPTool to a library mcp.Tool
func (p *MattermostToolProvider) convertMCPToolToLibMCPTool(mcpTool MCPTool) mcp.Tool {
	// Try to convert the JSON schema to MCP format
	if schema, ok := mcpTool.Schema.(*jsonschema.Schema); ok && schema != nil {
		// Marshal the jsonschema.Schema to JSON for use as raw schema
		schemaBytes, err := json.Marshal(schema)
		if err == nil {
			// Use the raw JSON schema - this provides proper parameter validation and documentation
			return mcp.NewToolWithRawSchema(mcpTool.Name, mcpTool.Description, schemaBytes)
		}
		// Log the error but continue with fallback
		p.logger.Warn("Failed to marshal JSON schema for tool", mlog.String("tool", mcpTool.Name), mlog.Err(err))
	}

	// Fallback to basic tool creation without schema
	// This still works but provides less rich client experience
	return mcp.NewTool(mcpTool.Name, mcp.WithDescription(mcpTool.Description))
}

// createMCPToolHandler creates an MCP tool handler that wraps an MCP tool resolver
func (p *MattermostToolProvider) createMCPToolHandler(resolver MCPToolResolver) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Create MCP tool context from MCP context
		mcpContext, err := p.createMCPToolContext(ctx)
		if err != nil {
			p.logger.Debug("Failed to create LLM context", mlog.Err(err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: " + err.Error(),
					},
				},
				IsError: true,
			}, nil
		}

		// Create an argument getter that extracts arguments from the MCP request
		argsGetter := func(target interface{}) error {
			// Convert MCP arguments to the target struct
			argumentsBytes, marshalErr := json.Marshal(request.Params.Arguments)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal arguments: %w", marshalErr)
			}

			// Validate access restrictions before unmarshaling
			if validationErr := validateAccessRestrictions(argumentsBytes, target, string(mcpContext.AccessMode)); validationErr != nil {
				return fmt.Errorf("access validation failed: %w", validationErr)
			}

			return json.Unmarshal(argumentsBytes, target)
		}

		// Call the MCP tool resolver
		result, err := resolver(mcpContext, argsGetter)
		if err != nil {
			p.logger.Debug("LLM tool resolver failed", mlog.Err(err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: " + err.Error(),
					},
				},
				IsError: true,
			}, nil
		}

		// Return successful result
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: result,
				},
			},
			IsError: false,
		}, nil
	}
}

// createMCPToolContext creates an MCPToolContext from the Go context and authenticated client
func (p *MattermostToolProvider) createMCPToolContext(ctx context.Context) (*MCPToolContext, error) {
	client, err := p.authProvider.GetAuthenticatedMattermostClient(ctx)
	if err != nil {
		return nil, err
	}

	return &MCPToolContext{
		Client:     client,
		AccessMode: p.accessMode,
	}, nil
}

// NewJSONSchemaForAccessMode creates a JSONSchema from a Go struct, filtering fields based on access mode
//
// Access tag examples:
//   - access:"local" - only available for local access mode
//   - access:"remote" - only available for remote access mode
//   - access:"local,remote" - available for both local and remote access modes
//   - no access tag - available in all access modes
//
// The function uses comma-separated parsing, so you can specify multiple access modes.
func NewJSONSchemaForAccessMode[T any](accessMode string) *jsonschema.Schema {
	// Validate access mode - empty string indicates uninitialized AccessMode
	if accessMode == "" {
		panic("access mode cannot be empty - indicates uninitialized AccessMode")
	}

	// Get the base schema
	baseSchema, err := jsonschema.For[T]()
	if err != nil {
		panic(fmt.Sprintf("failed to create JSON schema from struct: %v", err))
	}

	// If no properties to filter, return the base schema
	if baseSchema.Properties == nil {
		return baseSchema
	}

	// Get the struct type to inspect field tags
	var zero T
	structType := reflect.TypeOf(zero)

	// If it's a pointer, get the underlying type
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	// If it's not a struct, return the base schema
	if structType.Kind() != reflect.Struct {
		return baseSchema
	}

	// Create a new schema with filtered properties
	filteredSchema := &jsonschema.Schema{
		Type:        baseSchema.Type,
		Title:       baseSchema.Title,
		Description: baseSchema.Description,
		Properties:  make(map[string]*jsonschema.Schema),
		Required:    []string{},
	}

	// Check each field and its access tag
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Get the JSON field name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name (ignore omitempty and other options)
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		if jsonFieldName == "" {
			continue
		}

		// Check access tag
		restrictionTag := field.Tag.Get("access")

		// Include field if:
		// - No restriction tag (available for all access modes)
		// - Current access mode is in the comma-separated list of allowed modes
		includeField := restrictionTag == "" || isAccessAllowed(restrictionTag, accessMode)

		if includeField {
			// Copy the property from base schema if it exists
			if baseProperty, exists := baseSchema.Properties[jsonFieldName]; exists {
				filteredSchema.Properties[jsonFieldName] = baseProperty
			}

			// Check if field was required in original schema
			for _, requiredField := range baseSchema.Required {
				if requiredField == jsonFieldName {
					filteredSchema.Required = append(filteredSchema.Required, jsonFieldName)
					break
				}
			}
		}
	}

	return filteredSchema
}

// isAccessAllowed checks if the current access mode is allowed based on the access tag
// Supports comma-separated access modes (e.g., "local", "remote", "local,remote")
func isAccessAllowed(restrictionTag, currentAccessMode string) bool {
	if restrictionTag == "" {
		return true // No restrictions
	}

	// Normalize and split by comma
	allowedValues := strings.Split(strings.ReplaceAll(restrictionTag, " ", ""), ",")

	// Check each allowed value
	for _, allowed := range allowedValues {
		// Direct access mode matching
		if allowed == currentAccessMode {
			return true
		}
	}

	return false
}

// validateAccessRestrictions validates that no access-restricted fields are present in the JSON data
// for the current access mode. This prevents clients from sending fields they shouldn't have access to.
func validateAccessRestrictions(jsonData []byte, target interface{}, currentAccessMode string) error {
	if currentAccessMode == "" {
		panic("access mode cannot be empty - indicates uninitialized AccessMode")
	}
	// Get the struct type to inspect field tags
	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// If it's not a struct, no validation needed
	if targetType.Kind() != reflect.Struct {
		return nil
	}

	// Check if the incoming JSON is actually an object/map
	// If it's not an object, we can't have field restrictions to validate
	var incomingData map[string]interface{}
	if err := json.Unmarshal(jsonData, &incomingData); err != nil {
		// If JSON can't be parsed as an object, it's likely a primitive value or array
		// In this case, there are no fields to validate transport restrictions for
		return nil
	}

	// Check each field in the struct
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)

		// Get the JSON field name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name (ignore omitempty and other options)
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		if jsonFieldName == "" {
			continue
		}

		// Check if this field is present in the incoming data
		if _, fieldPresent := incomingData[jsonFieldName]; !fieldPresent {
			continue // Field not provided, so no validation needed
		}

		// Check access tag
		restrictionTag := field.Tag.Get("access")

		// If field has access restrictions and current access mode is not allowed
		if restrictionTag != "" && !isAccessAllowed(restrictionTag, currentAccessMode) {
			return fmt.Errorf("field '%s' is not available in %s access mode (requires: %s)", jsonFieldName, currentAccessMode, restrictionTag)
		}
	}

	return nil
}
