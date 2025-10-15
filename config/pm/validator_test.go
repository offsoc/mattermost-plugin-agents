// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/config"
)

func TestPMConfigValidator_ValidConfig(t *testing.T) {
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

func TestPMConfigValidator_RoleName(t *testing.T) {
	validator := &ConfigValidator{}
	if validator.RoleName() != "pm" {
		t.Errorf("Expected role name 'pm', got '%s'", validator.RoleName())
	}
}

func TestValidateRoleConfigs_WithPMConfig(t *testing.T) {
	RegisterValidator()

	validPMConfig := Config{
		Enabled: true,
	}

	pmConfigJSON, _ := json.Marshal(validPMConfig)

	roleConfigs := map[string]json.RawMessage{
		"pm": pmConfigJSON,
	}

	if err := config.ValidateRoleConfigs(roleConfigs); err != nil {
		t.Errorf("Expected valid PM config to pass validation, got error: %v", err)
	}
}
