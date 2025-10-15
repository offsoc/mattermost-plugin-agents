// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/config"
)

// ConfigValidator validates Dev role configuration
type ConfigValidator struct{}

// RoleName returns the name of the role this validator handles
func (v *ConfigValidator) RoleName() string {
	return "dev"
}

// ValidateRoleConfig validates Dev configuration
func (v *ConfigValidator) ValidateRoleConfig(rawConfig json.RawMessage) error {
	var devConfig Config
	if err := json.Unmarshal(rawConfig, &devConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Dev config: %w", err)
	}

	// Dev config validation (currently no specific validation needed)
	return nil
}

// RegisterValidator registers the Dev configuration validator
// This should be called when the Dev role service is initialized
func RegisterValidator() {
	config.RegisterRoleValidator(&ConfigValidator{})
}
