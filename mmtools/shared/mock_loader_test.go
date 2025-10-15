// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockLoader(t *testing.T) {
	// Create temporary directory for test mocks
	tempDir := t.TempDir()

	// Test disabled mock loader
	t.Run("disabled mock loader", func(t *testing.T) {
		loader := NewMockLoader(false, tempDir)
		response, found := loader.LoadMockResponse("GetJiraIssue")
		require.False(t, found)
		require.Empty(t, response)
	})

	// Test exact match
	t.Run("exact match", func(t *testing.T) {
		mockContent := "Mock Jira Issue Response"
		mockFile := filepath.Join(tempDir, "GetJiraIssue.txt")
		err := os.WriteFile(mockFile, []byte(mockContent), 0600)
		require.NoError(t, err)

		loader := NewMockLoader(true, tempDir)
		response, found := loader.LoadMockResponse("GetJiraIssue")
		require.True(t, found)
		require.Equal(t, mockContent, response)
	})

	// Test pattern match
	t.Run("pattern match", func(t *testing.T) {
		mockContent := "Mock Search Results"
		mockFile := filepath.Join(tempDir, "SearchJiraIssues_empty.txt")
		err := os.WriteFile(mockFile, []byte(mockContent), 0600)
		require.NoError(t, err)

		loader := NewMockLoader(true, tempDir)
		response, found := loader.LoadMockResponse("SearchJiraIssues")
		require.True(t, found)
		require.Equal(t, mockContent, response)
	})

	// Test no match
	t.Run("no match", func(t *testing.T) {
		loader := NewMockLoader(true, tempDir)
		response, found := loader.LoadMockResponse("NonExistentTool")
		require.False(t, found)
		require.Empty(t, response)
	})

	// Test list available mocks
	t.Run("list available mocks", func(t *testing.T) {
		loader := NewMockLoader(true, tempDir)
		mocks, err := loader.(*SimpleMockLoader).ListAvailableMocks()
		require.NoError(t, err)
		require.Contains(t, mocks, "GetJiraIssue")
		require.Contains(t, mocks, "SearchJiraIssues_empty")
	})
}
