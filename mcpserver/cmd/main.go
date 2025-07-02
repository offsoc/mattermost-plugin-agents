// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-plugin-ai/mcpserver"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func main() {
	// Parse command line flags
	var (
		serverURL = flag.String("server-url", "", "Mattermost server URL (required)")
		token     = flag.String("token", "", "Personal Access Token (required)")
		debug     = flag.Bool("debug", false, "Enable debug logging")
		logFile   = flag.String("logfile", "", "Path to log file (logs to file in addition to stderr)")
		devMode   = flag.Bool("dev", false, "Enable development mode with additional tools for setting up test data")
		version   = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Fprintf(os.Stderr, "Mattermost MCP Server v0.1.0\n")
		os.Exit(0)
	}

	// Create logger with debug and file logging options
	// This automatically configures std log redirection
	logger, err := mcpserver.CreateLoggerWithOptions(*debug, *logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Check required parameters
	if *serverURL == "" {
		// Try environment variable
		*serverURL = os.Getenv("MM_SERVER_URL")
		if *serverURL == "" {
			logger.Error("server URL is required (use -server-url or MM_SERVER_URL environment variable)")
			logger.Flush() // Ensure logs are written before exit
			os.Exit(1)
		}
	}

	// Check for PAT token
	if *token == "" {
		*token = os.Getenv("MM_ACCESS_TOKEN")
	}
	if *token == "" {
		logger.Error("personal access token is required (use -token or MM_ACCESS_TOKEN environment variable)")
		logger.Flush() // Ensure logs are written before exit
		os.Exit(1)
	}

	logger.Debug("starting mattermost mcp server",
		mlog.String("server_url", *serverURL),
		mlog.String("transport", "stdio"),
		mlog.String("auth_mode", "PAT"),
	)

	if *devMode {
		logger.Info("development mode enabled", mlog.Bool("dev_mode", *devMode))
	}

	// Create Mattermost MCP server with STDIO transport and PAT authentication
	mcpServer, err := mcpserver.NewMattermostStdioMCPServer(*serverURL, *token,
		mcpserver.WithLogger(logger),
		mcpserver.WithDevMode(*devMode),
	)
	if err != nil {
		logger.Error("failed to create MCP server", mlog.Err(err))
		logger.Flush() // Ensure logs are written before exit
		os.Exit(1)
	}

	// Start the MCP server
	if err := mcpServer.Serve(); err != nil {
		logger.Error("server error", mlog.Err(err))
		logger.Flush() // Ensure logs are written before exit
		os.Exit(1)
	}
}
