// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

// Config contains PM bot-specific configuration
type Config struct {
	Enabled        bool                `json:"enabled"`
	MockMode       *MockModeConfig     `json:"mockMode,omitempty"`
	ChannelTargets map[string][]string `json:"channelTargets,omitempty"`
}

// MockModeConfig controls whether PM tools use mock data
type MockModeConfig struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory"`
}

// DefaultFallbackDataDirectory is the default location for mock data
const DefaultFallbackDataDirectory = "./plugins/mattermost-ai/assets/fallback-data"
