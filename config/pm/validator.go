// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/config"
)

// ConfigValidator validates PM role configuration
type ConfigValidator struct{}

// RoleName returns the name of the role this validator handles
func (v *ConfigValidator) RoleName() string {
	return "pm"
}

// ValidateRoleConfig validates PM configuration
func (v *ConfigValidator) ValidateRoleConfig(rawConfig json.RawMessage) error {
	var pmConfig Config
	if err := json.Unmarshal(rawConfig, &pmConfig); err != nil {
		return fmt.Errorf("failed to unmarshal PM config: %w", err)
	}

	// PM config validation (currently no specific validation needed)
	return nil
}

// RegisterValidator registers the PM configuration validator
// This should be called when the PM role service is initialized
func RegisterValidator() {
	config.RegisterRoleValidator(&ConfigValidator{})
}
