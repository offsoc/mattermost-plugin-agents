// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/auth"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MCPToolContext provides MCP-specific functionality with the authenticated client
type MCPToolContext struct {
	Client *model.Client4
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

// MattermostToolRegistry provides Mattermost tools following the mmtools pattern
type MattermostToolRegistry struct {
	authProvider auth.AuthenticationProvider
	logger       mlog.LoggerIFace
	serverURL    string
	devMode      bool
}

// NewMattermostToolRegistry creates a new tool registry
func NewMattermostToolRegistry(authProvider auth.AuthenticationProvider, logger mlog.LoggerIFace, serverURL string, devMode bool) *MattermostToolRegistry {
	return &MattermostToolRegistry{
		authProvider: authProvider,
		logger:       logger,
		serverURL:    serverURL,
		devMode:      devMode,
	}
}

// getMCPTools returns all available MCP tools based on configuration
func (r *MattermostToolRegistry) getMCPTools() []MCPTool {
	tools := []MCPTool{}

	// Add regular tools
	tools = append(tools, r.getPostTools()...)
	tools = append(tools, r.getChannelTools()...)
	tools = append(tools, r.getTeamTools()...)
	tools = append(tools, r.getSearchTools()...)

	// Add dev tools if dev mode is enabled
	if r.devMode {
		tools = append(tools, r.getDevUserTools()...)
		tools = append(tools, r.getDevPostTools()...)
		tools = append(tools, r.getDevTeamTools()...)
		tools = append(tools, r.getDevChannelTools()...)
	}

	return tools
}

// RegisterWithMCPServer registers all tools with the provided MCP server
func (r *MattermostToolRegistry) RegisterWithMCPServer(mcpServer *server.MCPServer) {
	// Get all MCP tools from the provider
	mcpTools := r.getMCPTools()

	// Convert and register each tool
	for _, mcpTool := range mcpTools {
		libMCPTool := r.convertMCPToolToLibMCPTool(mcpTool)
		mcpServer.AddTool(libMCPTool, r.createMCPToolHandler(mcpTool.Resolver))
	}
}

// convertMCPToolToLibMCPTool converts our MCPTool to a library mcp.Tool
func (r *MattermostToolRegistry) convertMCPToolToLibMCPTool(mcpTool MCPTool) mcp.Tool {
	// Try to convert the JSON schema to MCP format
	if schema, ok := mcpTool.Schema.(*jsonschema.Schema); ok && schema != nil {
		// Marshal the jsonschema.Schema to JSON for use as raw schema
		schemaBytes, err := json.Marshal(schema)
		if err == nil {
			// Use the raw JSON schema - this provides proper parameter validation and documentation
			return mcp.NewToolWithRawSchema(mcpTool.Name, mcpTool.Description, schemaBytes)
		}
		// Log the error but continue with fallback
		r.logger.Warn("Failed to marshal JSON schema for tool", mlog.String("tool", mcpTool.Name), mlog.Err(err))
	}

	// Fallback to basic tool creation without schema
	// This still works but provides less rich client experience
	return mcp.NewTool(mcpTool.Name, mcp.WithDescription(mcpTool.Description))
}

// createMCPToolHandler creates an MCP tool handler that wraps an MCP tool resolver
func (r *MattermostToolRegistry) createMCPToolHandler(resolver MCPToolResolver) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Create MCP tool context from MCP context
		mcpContext, err := r.createMCPToolContext(ctx)
		if err != nil {
			r.logger.Debug("Failed to create LLM context", mlog.Err(err))
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

			return json.Unmarshal(argumentsBytes, target)
		}

		// Call the MCP tool resolver
		result, err := resolver(mcpContext, argsGetter)
		if err != nil {
			r.logger.Debug("LLM tool resolver failed", mlog.Err(err))
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
func (r *MattermostToolRegistry) createMCPToolContext(ctx context.Context) (*MCPToolContext, error) {
	client, err := r.authProvider.GetAuthenticatedMattermostClient(ctx)
	if err != nil {
		return nil, err
	}

	return &MCPToolContext{
		Client: client,
	}, nil
}
