// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/config"
)

func TestDevConfigValidator_ValidConfig(t *testing.T) {
	validator := &ConfigValidator{}

	validConfig := Config{
		Enabled: true,
	}

	rawConfig, err := json.Marshal(validConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := validator.ValidateRoleConfig(rawConfig); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestDevConfigValidator_RoleName(t *testing.T) {
	validator := &ConfigValidator{}
	if validator.RoleName() != "dev" {
		t.Errorf("Expected role name 'dev', got '%s'", validator.RoleName())
	}
}

func TestValidateRoleConfigs_WithDevConfig(t *testing.T) {
	RegisterValidator()

	validDevConfig := Config{
		Enabled: true,
	}

	devConfigJSON, _ := json.Marshal(validDevConfig)

	roleConfigs := map[string]json.RawMessage{
		"dev": devConfigJSON,
	}

	if err := config.ValidateRoleConfigs(roleConfigs); err != nil {
		t.Errorf("Expected valid Dev config to pass validation, got error: %v", err)
	}
}
