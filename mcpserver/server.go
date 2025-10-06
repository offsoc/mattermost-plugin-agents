// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/auth"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/tools"
	"github.com/mattermost/mattermost-plugin-ai/mcpserver/types"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostMCPServer provides a high-level interface for creating an MCP server
// with Mattermost-specific tools and authentication
type MattermostMCPServer struct {
	mcpServer    *server.MCPServer
	authProvider auth.AuthenticationProvider
	logger       *mlog.Logger
	config       ServerConfig
}

// registerTools registers all tools using the tool provider
func (s *MattermostMCPServer) registerTools(accessMode types.AccessMode) {
	toolProvider := tools.NewMattermostToolProvider(s.authProvider, s.logger, s.config.GetMMServerURL(), s.config.GetMMInternalServerURL(), s.config.GetDevMode(), accessMode)
	toolProvider.ProvideTools(s.mcpServer)
}

// GetMCPServer returns the underlying MCP server for testing purposes
func (s *MattermostMCPServer) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}
