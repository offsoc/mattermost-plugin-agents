// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// Option defines a function that configures a MattermostMCPServer
type Option func(*MattermostMCPServer) error

const (
	defaultTimeout = 30 * time.Second
)

// MattermostMCPServer provides a high-level interface for creating an MCP server
// with Mattermost-specific tools and authentication
type MattermostMCPServer struct {
	mcpServer    *server.MCPServer
	authProvider AuthenticationProvider
	logger       *mlog.Logger
	config       Config
}

// NewMattermostStdioMCPServer creates a new Mattermost MCP server using STDIO transport with Personal Access Token authentication
func NewMattermostStdioMCPServer(serverURL, token string, opts ...Option) (*MattermostMCPServer, error) {
	// Validate required parameters
	if serverURL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("personal access token cannot be empty")
	}

	// Create default logger with reasonable configuration
	defaultLogger, err := createDefaultLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create default logger: %w", err)
	}

	// Initialize server with defaults
	mattermostServer := &MattermostMCPServer{
		logger: defaultLogger,
		config: Config{
			ServerURL:           serverURL,
			PersonalAccessToken: token,
			RequestTimeout:      defaultTimeout,
			Transport:           "stdio", // Always STDIO for this constructor
			DevMode:             false,
		},
	}

	// Apply all options
	for _, opt := range opts {
		if err := opt(mattermostServer); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Create PAT authentication provider (after options are applied so it uses the correct logger)
	mattermostServer.authProvider = NewTokenAuthenticationProvider(serverURL, token, mattermostServer.logger)

	// Create the mcp-go server
	mattermostServer.mcpServer = server.NewMCPServer(
		"mattermost-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithLogging(), // Enable logging capabilities
	)

	// For STDIO transport, always validate token at startup
	if _, err := mattermostServer.authProvider.ValidateAuth(context.Background()); err != nil {
		return nil, fmt.Errorf("startup token validation failed: %w", err)
	}

	// Register all Mattermost tools
	mattermostServer.registerMattermostTools()

	return mattermostServer, nil
}

// Serve starts the server using the configured transport
func (s *MattermostMCPServer) Serve() error {
	switch s.config.Transport {
	case "stdio":
		return s.serveStdio()
	case "http":
		return s.serveHTTP()
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport)
	}
}

// serveStdio starts the server using stdio transport
func (s *MattermostMCPServer) serveStdio() error {
	// Configure error logger to use our mlog logger
	errorLogger := log.New(&mlogWriter{logger: s.logger}, "", 0)

	return server.ServeStdio(s.mcpServer, server.WithErrorLogger(errorLogger))
}

// serveHTTP starts the server using HTTP transport
func (s *MattermostMCPServer) serveHTTP() error {
	// TODO: Implement HTTP/SSE transport for OAuth authentication
	// This will be implemented when OAuth support is added
	s.logger.Info("HTTP transport requested but not yet implemented")
	s.logger.Info("Future implementation will support OAuth authentication and StreamableHTTP")
	return fmt.Errorf("HTTP transport not yet implemented - will be added for OAuth support")
}

// registerMattermostTools registers all Mattermost tools with the MCP server
func (s *MattermostMCPServer) registerMattermostTools() {
	// Register read_post tool
	readPostTool := mcp.NewTool("read_post",
		mcp.WithDescription("Read a specific post and its thread from Mattermost"),
		mcp.WithString("post_id", mcp.Description("The ID of the post to read"), mcp.Required()),
		mcp.WithBoolean("include_thread", mcp.Description("Whether to include the entire thread (default: true)")),
	)
	s.mcpServer.AddTool(readPostTool, s.createToolHandler("read_post"))

	// Register read_channel tool
	readChannelTool := mcp.NewTool("read_channel",
		mcp.WithDescription("Read recent posts from a Mattermost channel"),
		mcp.WithString("channel_id", mcp.Description("The ID of the channel to read from"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Number of posts to retrieve (default: 20, max: 100)")),
		mcp.WithString("since", mcp.Description("Only get posts since this timestamp (ISO 8601 format)")),
	)
	s.mcpServer.AddTool(readChannelTool, s.createToolHandler("read_channel"))

	// Register search_posts tool
	searchPostsTool := mcp.NewTool("search_posts",
		mcp.WithDescription("Search for posts in Mattermost"),
		mcp.WithString("query", mcp.Description("The search query"), mcp.Required()),
		mcp.WithString("team_id", mcp.Description("Optional team ID to limit search scope")),
		mcp.WithString("channel_id", mcp.Description("Optional channel ID to limit search to a specific channel")),
		mcp.WithNumber("limit", mcp.Description("Number of results to return (default: 20, max: 100)")),
	)
	s.mcpServer.AddTool(searchPostsTool, s.createToolHandler("search_posts"))

	// Register create_post tool
	createPostTool := mcp.NewTool("create_post",
		mcp.WithDescription("Create a new post in Mattermost"),
		mcp.WithString("channel_id", mcp.Description("The ID of the channel to post in"), mcp.Required()),
		mcp.WithString("message", mcp.Description("The message content"), mcp.Required()),
		mcp.WithString("root_id", mcp.Description("Optional root post ID for replies")),
	)
	s.mcpServer.AddTool(createPostTool, s.createToolHandler("create_post"))

	// Register create_channel tool
	createChannelTool := mcp.NewTool("create_channel",
		mcp.WithDescription("Create a new channel in Mattermost"),
		mcp.WithString("name", mcp.Description("The channel name (URL-friendly)"), mcp.Required()),
		mcp.WithString("display_name", mcp.Description("The channel display name"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Channel type: 'O' for public, 'P' for private"), mcp.Required()),
		mcp.WithString("team_id", mcp.Description("The team ID where the channel will be created"), mcp.Required()),
		mcp.WithString("purpose", mcp.Description("Optional channel purpose")),
		mcp.WithString("header", mcp.Description("Optional channel header")),
	)
	s.mcpServer.AddTool(createChannelTool, s.createToolHandler("create_channel"))

	// Register get_channel_info tool
	getChannelInfoTool := mcp.NewTool("get_channel_info",
		mcp.WithDescription("Get information about a channel. If you have a channel ID, use that for fastest lookup. If the user provides a human-readable name, try channel_display_name first (what users see in the UI), then channel_name (URL name) as fallback."),
		mcp.WithString("channel_id", mcp.Description("The exact channel ID (fastest, most reliable method)")),
		mcp.WithString("channel_display_name", mcp.Description("The human-readable display name users see (e.g. 'General Discussion')")),
		mcp.WithString("channel_name", mcp.Description("The URL-friendly channel name (e.g. 'general-discussion')")),
		mcp.WithString("team_id", mcp.Description("Team ID (required if using channel_name or channel_display_name)")),
	)
	s.mcpServer.AddTool(getChannelInfoTool, s.createToolHandler("get_channel_info"))

	// Register get_team_info tool
	getTeamInfoTool := mcp.NewTool("get_team_info",
		mcp.WithDescription("Get information about a team. If you have a team ID, use that for fastest lookup. If the user provides a human-readable name, try team_display_name first (what users see in the UI), then team_name (URL name) as fallback."),
		mcp.WithString("team_id", mcp.Description("The exact team ID (fastest, most reliable method)")),
		mcp.WithString("team_display_name", mcp.Description("The human-readable display name users see (e.g. 'Engineering Team')")),
		mcp.WithString("team_name", mcp.Description("The URL-friendly team name (e.g. 'engineering-team')")),
	)
	s.mcpServer.AddTool(getTeamInfoTool, s.createToolHandler("get_team_info"))

	// Register search_users tool
	searchUsersTool := mcp.NewTool("search_users",
		mcp.WithDescription("Search for existing users by username, email, or name"),
		mcp.WithString("term", mcp.Description("Search term (username, email, first name, or last name)"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 20, max: 100)")),
	)
	s.mcpServer.AddTool(searchUsersTool, s.createToolHandler("search_users"))

	// Register get_channel_members tool
	getChannelMembersTool := mcp.NewTool("get_channel_members",
		mcp.WithDescription("Get members of a channel with pagination support"),
		mcp.WithString("channel_id", mcp.Description("ID of the channel to get members for"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Number of members to return (default: 50, max: 200)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 0)")),
	)
	s.mcpServer.AddTool(getChannelMembersTool, s.createToolHandler("get_channel_members"))

	// Register get_team_members tool
	getTeamMembersTool := mcp.NewTool("get_team_members",
		mcp.WithDescription("Get members of a team with pagination support"),
		mcp.WithString("team_id", mcp.Description("ID of the team to get members for"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Number of members to return (default: 50, max: 200)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination (default: 0)")),
	)
	s.mcpServer.AddTool(getTeamMembersTool, s.createToolHandler("get_team_members"))

	// Register development tools if dev mode is enabled
	if s.config.DevMode {
		s.registerDevTools()
	}
}

// registerDevTools registers development-specific tools when dev mode is enabled
func (s *MattermostMCPServer) registerDevTools() {
	// Register create_user tool
	createUserTool := mcp.NewTool("create_user",
		mcp.WithDescription("Create a new user account (dev mode only)"),
		mcp.WithString("username", mcp.Description("Username for the new user"), mcp.Required()),
		mcp.WithString("email", mcp.Description("Email address for the new user"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password for the new user"), mcp.Required()),
		mcp.WithString("first_name", mcp.Description("First name of the user")),
		mcp.WithString("last_name", mcp.Description("Last name of the user")),
		mcp.WithString("nickname", mcp.Description("Nickname for the user")),
	)
	s.mcpServer.AddTool(createUserTool, s.createToolHandler("create_user"))

	// Register create_team tool
	createTeamTool := mcp.NewTool("create_team",
		mcp.WithDescription("Create a new team (dev mode only)"),
		mcp.WithString("name", mcp.Description("URL name for the team"), mcp.Required()),
		mcp.WithString("display_name", mcp.Description("Display name for the team"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Team type: 'O' for open, 'I' for invite only"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Team description")),
	)
	s.mcpServer.AddTool(createTeamTool, s.createToolHandler("create_team"))

	// Register add_user_to_team tool
	addUserToTeamTool := mcp.NewTool("add_user_to_team",
		mcp.WithDescription("Add a user to a team (dev mode only)"),
		mcp.WithString("user_id", mcp.Description("ID of the user to add"), mcp.Required()),
		mcp.WithString("team_id", mcp.Description("ID of the team to add user to"), mcp.Required()),
	)
	s.mcpServer.AddTool(addUserToTeamTool, s.createToolHandler("add_user_to_team"))

	// Register add_user_to_channel tool
	addUserToChannelTool := mcp.NewTool("add_user_to_channel",
		mcp.WithDescription("Add a user to a channel (dev mode only)"),
		mcp.WithString("user_id", mcp.Description("ID of the user to add"), mcp.Required()),
		mcp.WithString("channel_id", mcp.Description("ID of the channel to add user to"), mcp.Required()),
	)
	s.mcpServer.AddTool(addUserToChannelTool, s.createToolHandler("add_user_to_channel"))

	// Register create_post_as_user tool
	createPostAsUserTool := mcp.NewTool("create_post_as_user",
		mcp.WithDescription("Create a post as a specific user using username/password login. Use this tool in dev mode for creating realistic multi-user scenarios. Simply provide the username and password of created users."),
		mcp.WithString("username", mcp.Description("Username to login as"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password to login with"), mcp.Required()),
		mcp.WithString("channel_id", mcp.Description("The ID of the channel to post in"), mcp.Required()),
		mcp.WithString("message", mcp.Description("The message content"), mcp.Required()),
		mcp.WithString("root_id", mcp.Description("Optional root post ID for replies")),
		mcp.WithString("props", mcp.Description("Optional post properties (JSON string)")),
	)
	s.mcpServer.AddTool(createPostAsUserTool, s.createToolHandler("create_post_as_user"))
}

func (s *MattermostMCPServer) createToolHandler(toolName string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Apply configured timeout to the context
		ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
		defer cancel()

		// Get authenticated client (timeout is already applied in base context)
		client, err := s.authProvider.GetAuthenticatedMattermostClient(ctx)
		if err != nil {
			s.logger.Debug("Tool call failed",
				mlog.String("tool", toolName),
				mlog.Err(err))
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

		// Use our existing tool provider to execute the tool
		toolProvider := NewMattermostToolProvider(s.authProvider, s.logger)

		// Create dev tool provider for development tools
		devToolProvider := NewDevToolProvider(s.authProvider, s.logger, s.config.ServerURL)

		// Execute the tool using our existing implementation
		var result *mcp.CallToolResult
		switch toolName {
		case "read_post":
			result, err = toolProvider.readPost(ctx, client, request.Params.Arguments)
		case "read_channel":
			result, err = toolProvider.readChannel(ctx, client, request.Params.Arguments)
		case "search_posts":
			result, err = toolProvider.searchPosts(ctx, client, request.Params.Arguments)
		case "create_post":
			result, err = toolProvider.createPost(ctx, client, request.Params.Arguments)
		case "create_channel":
			result, err = toolProvider.createChannel(ctx, client, request.Params.Arguments)
		case "get_channel_info":
			result, err = toolProvider.getChannelInfo(ctx, client, request.Params.Arguments)
		case "get_team_info":
			result, err = toolProvider.getTeamInfo(ctx, client, request.Params.Arguments)
		case "search_users":
			result, err = toolProvider.searchUsers(ctx, client, request.Params.Arguments)
		case "get_channel_members":
			result, err = toolProvider.getChannelMembers(ctx, client, request.Params.Arguments)
		case "get_team_members":
			result, err = toolProvider.getTeamMembers(ctx, client, request.Params.Arguments)
		// Development tools (only available in dev mode)
		case "create_user":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: create_user tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.createUser(ctx, client, request.Params.Arguments)
		case "create_team":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: create_team tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.createTeam(ctx, client, request.Params.Arguments)
		case "add_user_to_team":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: add_user_to_team tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.addUserToTeam(ctx, client, request.Params.Arguments)
		case "add_user_to_channel":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: add_user_to_channel tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.addUserToChannel(ctx, client, request.Params.Arguments)
		case "create_post_as_user":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: create_post_as_user tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.createPostAsUser(ctx, request.Params.Arguments)
		default:
			s.logger.Debug("Tool call failed - unknown tool",
				mlog.String("tool", toolName))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: unknown tool: " + toolName,
					},
				},
				IsError: true,
			}, nil
		}

		if err != nil {
			s.logger.Debug("Tool call failed - execution error",
				mlog.String("tool", toolName),
				mlog.Err(err))
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

		// Log successful tool completion
		isError := result != nil && result.IsError
		if isError {
			s.logger.Debug("Tool call completed with error result",
				mlog.String("tool", toolName))
		} else {
			s.logger.Debug("Tool call completed successfully",
				mlog.String("tool", toolName))
		}

		// Result is already a *mcp.CallToolResult
		return result, nil
	}
}

// createDefaultLogger creates a logger with sensible defaults for the MCP server
func createDefaultLogger() (*mlog.Logger, error) {
	// Use the same configuration helper for consistency
	return CreateLoggerWithOptions(false, "") // No debug, no file logging
}

// CreateLoggerWithOptions creates a logger with debug and file logging options
// This function sets up a fully configured logger and enables std log redirection
func CreateLoggerWithOptions(enableDebug bool, logFile string) (*mlog.Logger, error) {
	logger, err := mlog.NewLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create new logger: %w", err)
	}

	// Start with default levels - Info and above for production use
	levels := []mlog.Level{mlog.LvlInfo, mlog.LvlWarn, mlog.LvlError}
	if enableDebug {
		// Prepend debug level to ensure it's first in the list
		levels = append([]mlog.Level{mlog.LvlDebug}, levels...)
	}

	cfg := make(mlog.LoggerConfiguration)

	// Console logging configuration
	cfg["console"] = mlog.TargetCfg{
		Type:          "console",
		Levels:        levels,
		Format:        "plain",
		FormatOptions: json.RawMessage(`{"enable_color": false, "delim": " "}`),
		Options:       json.RawMessage(`{"out": "stderr"}`),
		MaxQueueSize:  1000,
	}

	// Add file logging if requested
	if logFile != "" {
		cfg["file"] = mlog.TargetCfg{
			Type:         "file",
			Levels:       levels,
			Format:       "json", // JSON format for file logs (better for parsing)
			Options:      json.RawMessage(fmt.Sprintf(`{"compress": false, "filename": "%s"}`, logFile)),
			MaxQueueSize: 1000,
		}
	}

	err = logger.ConfigureTargets(cfg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to configure logger targets: %w", err)
	}

	// Enable std log redirection - this ensures third-party libraries
	// using Go's standard log package route through our structured logger
	logger.RedirectStdLog(mlog.LvlInfo) // Redirect std logs at Info level

	return logger, nil
}

// Option functions for configuring MattermostMCPServer

// WithLogger configures the server to use a specific logger
func WithLogger(logger *mlog.Logger) Option {
	return func(s *MattermostMCPServer) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}
		s.logger = logger
		return nil
	}
}

// WithDevMode enables or disables development mode (enables additional tools for testing)
func WithDevMode(enabled bool) Option {
	return func(s *MattermostMCPServer) error {
		s.config.DevMode = enabled
		return nil
	}
}

// WithRequestTimeout sets the timeout for requests to Mattermost
func WithRequestTimeout(timeout time.Duration) Option {
	return func(s *MattermostMCPServer) error {
		if timeout <= 0 {
			return fmt.Errorf("request timeout must be positive, got: %v", timeout)
		}
		s.config.RequestTimeout = timeout
		return nil
	}
}

// mlogWriter adapts *mlog.Logger to io.Writer for the mcp-go error logger
type mlogWriter struct {
	logger *mlog.Logger
}

func (w *mlogWriter) Write(p []byte) (n int, err error) {
	// Logger is guaranteed to be non-nil by constructor
	w.logger.Error(string(p))
	return len(p), nil
}

// CallToolForTest calls a tool handler directly for testing purposes
// This bypasses the MCP transport layer and calls the tool implementation directly
func (s *MattermostMCPServer) CallToolForTest(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Create a proper MCP CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = arguments

	// Get and call the tool handler
	toolHandler := s.createToolHandler(toolName)
	return toolHandler(ctx, request)
}
