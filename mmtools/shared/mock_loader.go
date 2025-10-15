// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SimpleMockLoader implements MockLoader interface with file-based mock responses
type SimpleMockLoader struct {
	enabled   bool
	directory string
}

// NewMockLoader creates a new mock loader instance
func NewMockLoader(enabled bool, directory string) MockLoader {
	return &SimpleMockLoader{
		enabled:   enabled,
		directory: directory,
	}
}

// LoadMockResponse loads a mock response from a file
func (m *SimpleMockLoader) LoadMockResponse(toolName string) (string, bool) {
	if !m.enabled || m.directory == "" {
		return "", false
	}

	// Try exact match: {ToolName}.txt
	exactPath := filepath.Join(m.directory, toolName+".txt")
	if content, err := os.ReadFile(exactPath); err == nil {
		return string(content), true
	}

	// Try pattern match: {ToolName}_*.txt
	pattern := filepath.Join(m.directory, toolName+"_*.txt")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", false
	}

	// Use first match
	if content, err := os.ReadFile(matches[0]); err == nil {
		return string(content), true
	}

	return "", false
}

// IsEnabled returns whether mock mode is enabled
func (m *SimpleMockLoader) IsEnabled() bool {
	return m.enabled
}

// ListAvailableMocks returns a list of available mock files
func (m *SimpleMockLoader) ListAvailableMocks() ([]string, error) {
	if !m.enabled || m.directory == "" {
		return nil, fmt.Errorf("mock mode not enabled")
	}

	pattern := filepath.Join(m.directory, "*.txt")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var mocks []string
	for _, match := range matches {
		basename := filepath.Base(match)
		name := strings.TrimSuffix(basename, ".txt")
		mocks = append(mocks, name)
	}

	return mocks, nil
}
