// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/auth"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/types"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostStdioMCPServer wraps MattermostMCPServer for STDIO transport
type MattermostStdioMCPServer struct {
	*MattermostMCPServer
	config StdioConfig
}

// NewStdioServer creates a new STDIO transport MCP server
func NewStdioServer(config StdioConfig, logger *mlog.Logger) (*MattermostStdioMCPServer, error) {
	if config.MMServerURL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	if config.PersonalAccessToken == "" {
		return nil, fmt.Errorf("personal access token cannot be empty")
	}

	if logger == nil {
		var err error
		logger, err = createDefaultLogger()
		if err != nil {
			return nil, fmt.Errorf("failed to create default logger: %w", err)
		}
	}

	mattermostServer := &MattermostStdioMCPServer{
		MattermostMCPServer: &MattermostMCPServer{
			logger: logger,
			config: config,
		},
		config: config,
	}

	// Create authentication provider
	mattermostServer.authProvider = auth.NewTokenAuthenticationProvider(config.GetMMServerURL(), config.GetMMInternalServerURL(), config.PersonalAccessToken, logger)

	// Create MCP server
	mattermostServer.mcpServer = server.NewMCPServer(
		"mattermost-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// Validate token at startup for STDIO
	if err := mattermostServer.authProvider.ValidateAuth(context.Background()); err != nil {
		return nil, fmt.Errorf("startup token validation failed: %w", err)
	}

	// Register tools with local access mode
	mattermostServer.registerTools(types.AccessModeLocal)

	return mattermostServer, nil
}

// Serve starts the STDIO MCP server
func (s *MattermostStdioMCPServer) Serve() error {
	return s.serveStdio()
}

// serveStdio starts the server using stdio transport
func (s *MattermostMCPServer) serveStdio() error {
	errorLogger := log.New(&mlogWriter{logger: s.logger}, "", 0)
	return server.ServeStdio(s.mcpServer, server.WithErrorLogger(errorLogger))
}
