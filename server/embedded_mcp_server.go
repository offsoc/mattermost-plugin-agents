// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"

	"github.com/mattermost/mattermost-plugin-ai/mcpserver"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EmbeddedMCPServer manages the lifecycle of an embedded MCP server within the plugin
// This provides in-memory communication between the plugin and MCP server, eliminating
// the need for OAuth flows and network communication
type EmbeddedMCPServer struct {
	server *mcpserver.MattermostInMemoryMCPServer
	logger pluginapi.LogService
}

// NewEmbeddedMCPServer creates a new embedded MCP server instance
func NewEmbeddedMCPServer(pluginAPI *pluginapi.Client, logger pluginapi.LogService) (*EmbeddedMCPServer, error) {
	// Get site URL from plugin configuration
	siteURL := ""
	if config := pluginAPI.Configuration.GetConfig(); config != nil && config.ServiceSettings.SiteURL != nil {
		siteURL = *config.ServiceSettings.SiteURL
	}

	if siteURL == "" {
		return nil, errors.New("site URL not configured, cannot initialize embedded MCP server")
	}

	// Create configuration for in-memory transport
	config := mcpserver.InMemoryConfig{
		BaseConfig: mcpserver.BaseConfig{
			MMServerURL: siteURL,
			// For embedded server, internal URL can be the same as external
			MMInternalServerURL: siteURL,
			DevMode:             false,
		},
	}

	// Create a logger adapter that routes MCP server logs through the plugin's logging system
	// This is now a simple pass-through since both use the same interface
	mcpLogger := NewPluginAPILoggerAdapter(logger)

	// Create the in-memory MCP server
	server, err := mcpserver.NewInMemoryServer(config, mcpLogger)
	if err != nil {
		return nil, err
	}

	embeddedServer := &EmbeddedMCPServer{
		server: server,
		logger: logger,
	}

	return embeddedServer, nil
}

// CreateClientTransport creates a new in-memory transport for a client connection
func (e *EmbeddedMCPServer) CreateClientTransport(userID, token string) (*mcp.InMemoryTransport, error) {
	// Create the connection through the server
	clientTransport, err := e.server.CreateConnectionForUser(userID, token)
	if err != nil {
		return nil, err
	}

	e.logger.Debug("Created client transport for embedded MCP server",
		"user_id", userID)

	return clientTransport, nil
}
