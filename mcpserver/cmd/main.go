// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"os"

	"github.com/mattermost/mattermost-plugin-ai/mcpserver"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

var (
	serverURL string
	token     string
	debug     bool
	logFile   string
	devMode   bool
	transport string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mattermost-mcp-server",
		Short: "Mattermost Model Context Protocol (MCP) Server",
		Long: `A Model Context Protocol (MCP) server that provides tools for interacting with Mattermost.

The server supports reading posts, searching, creating content, and managing teams/channels.
Authentication is handled via Personal Access Tokens (PAT).`,
		Version: version,
		RunE:    runServer,
	}

	// Define flags
	rootCmd.Flags().StringVarP(&serverURL, "server-url", "s", "", "Mattermost server URL (required, or set MM_SERVER_URL env var)")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "Personal Access Token (required, or set MM_ACCESS_TOKEN env var)")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to log file (logs to file in addition to stderr)")
	rootCmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode with additional tools for setting up test data")
	rootCmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type (currently only stdio is supported)")

	// Note: We don't mark flags as required since they can also come from environment variables

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	// Create logger with debug and file logging options
	// This automatically configures std log redirection
	logger, err := mcpserver.CreateLoggerWithOptions(debug, logFile)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Check for environment variables if flags not provided
	if serverURL == "" {
		serverURL = os.Getenv("MM_SERVER_URL")
		if serverURL == "" {
			logger.Error("server URL is required (use --server-url or MM_SERVER_URL environment variable)")
			logger.Flush()
			return fmt.Errorf("server URL is required")
		}
	}

	if token == "" {
		token = os.Getenv("MM_ACCESS_TOKEN")
		if token == "" {
			logger.Error("personal access token is required (use --token or MM_ACCESS_TOKEN environment variable)")
			logger.Flush()
			return fmt.Errorf("personal access token is required")
		}
	}

	// Validate transport type
	if transport != "stdio" {
		logger.Error("invalid transport type", mlog.String("transport", transport))
		logger.Flush()
		return fmt.Errorf("invalid transport type: %s (currently only 'stdio' is supported)", transport)
	}

	logger.Debug("starting mattermost mcp server",
		mlog.String("server_url", serverURL),
		mlog.String("transport", transport),
		mlog.String("auth_mode", "PAT"),
	)

	if devMode {
		logger.Info("development mode enabled", mlog.Bool("dev_mode", devMode))
	}

	// Create Mattermost MCP server based on transport type
	var mcpServer *mcpserver.MattermostMCPServer

	switch transport {
	case "stdio":
		mcpServer, err = mcpserver.NewMattermostStdioMCPServer(serverURL, token,
			mcpserver.WithLogger(logger),
			mcpserver.WithDevMode(devMode),
		)
	default:
		logger.Error("unsupported transport type", mlog.String("transport", transport))
		logger.Flush()
		return fmt.Errorf("unsupported transport type: %s", transport)
	}
	if err != nil {
		logger.Error("failed to create MCP server", mlog.Err(err))
		logger.Flush()
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Start the MCP server
	if err := mcpServer.Serve(); err != nil {
		logger.Error("server error", mlog.Err(err))
		logger.Flush()
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
