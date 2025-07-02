// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"time"
)

// Config represents the configuration for the MCP server
type Config struct {
	// Mattermost server URL (e.g., "https://mattermost.company.com")
	ServerURL string `json:"server_url"`

	// Personal Access Token for authentication
	PersonalAccessToken string `json:"personal_access_token"`

	// Timeout for MCP requests
	RequestTimeout time.Duration `json:"request_timeout"`

	// Transport type (currently only stdio is supported)
	Transport string `json:"transport"`

	// Development mode enables additional tools for setting up test data
	DevMode bool `json:"dev_mode"`
}
