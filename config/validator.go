// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import (
	"encoding/json"
	"fmt"
	"sync"
)

// RoleConfigValidator defines the interface for role-specific configuration validation
type RoleConfigValidator interface {
	// ValidateRoleConfig validates a role's configuration
	ValidateRoleConfig(rawConfig json.RawMessage) error
	// RoleName returns the name of the role this validator handles
	RoleName() string
}

// Global validator registry with mutex protection for thread safety
var (
	roleValidators   = make(map[string]RoleConfigValidator)
	roleValidatorsMu sync.RWMutex
)

// RegisterRoleValidator registers a validator for a specific role
// This should be called from role package init() functions
func RegisterRoleValidator(validator RoleConfigValidator) {
	roleValidatorsMu.Lock()
	defer roleValidatorsMu.Unlock()
	roleValidators[validator.RoleName()] = validator
}

// ValidateRoleConfigs validates all role configurations using registered validators
func ValidateRoleConfigs(roleConfigs map[string]json.RawMessage) error {
	if roleConfigs == nil {
		return nil
	}

	roleValidatorsMu.RLock()
	defer roleValidatorsMu.RUnlock()

	for roleName, rawConfig := range roleConfigs {
		if validator, exists := roleValidators[roleName]; exists {
			if err := validator.ValidateRoleConfig(rawConfig); err != nil {
				return fmt.Errorf("invalid %s configuration: %w", roleName, err)
			}
		}
	}

	return nil
}
