// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"testing"
)

func TestModelInfo(t *testing.T) {
	// Test that ModelInfo can be created and has the expected fields
	model := ModelInfo{
		ID:          "test-model-id",
		DisplayName: "Test Model",
	}

	if model.ID != "test-model-id" {
		t.Errorf("Expected ID to be 'test-model-id', got '%s'", model.ID)
	}

	if model.DisplayName != "Test Model" {
		t.Errorf("Expected DisplayName to be 'Test Model', got '%s'", model.DisplayName)
	}
}
