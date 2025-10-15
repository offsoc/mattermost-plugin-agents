// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

// Config contains Dev bot-specific configuration
type Config struct {
	Enabled  bool            `json:"enabled"`
	MockMode *MockModeConfig `json:"mockMode,omitempty"`
}

// MockModeConfig controls whether Dev tools use mock data
type MockModeConfig struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory"`
}
