// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcp

import (
	"fmt"
	"slices"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// ConversationToolPermissions stores which tools are auto-approved for a specific conversation
type ConversationToolPermissions struct {
	AutoApprovedTools []string `json:"auto_approved_tools"` // Tools that don't need approval
	UpdatedAt         int64    `json:"updated_at"`
}

func buildConversationPermissionKey(userID, rootPostID string) string {
	prefix := "mcp_tool_permission"
	return fmt.Sprintf("%s_%s_%s", prefix, userID, rootPostID)
}

// GetConversationPermissions loads the full permission set for a conversation
func GetConversationPermissions(client mmapi.Client, userID, rootPostID string) (*ConversationToolPermissions, error) {
	key := buildConversationPermissionKey(userID, rootPostID)

	var permissions ConversationToolPermissions
	err := client.KVGet(key, &permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation permissions from KV store: %w", err)
	}

	// If no permissions found, return empty set
	if permissions.AutoApprovedTools == nil {
		return &ConversationToolPermissions{
			AutoApprovedTools: []string{},
			UpdatedAt:         time.Now().Unix(),
		}, nil
	}

	return &permissions, nil
}

// IsToolAutoApproved checks if a specific tool is auto-approved for this conversation
func IsToolAutoApproved(client mmapi.Client, userID, rootPostID, toolName string) (bool, error) {
	permissions, err := GetConversationPermissions(client, userID, rootPostID)
	if err != nil {
		return false, err
	}

	return slices.Contains(permissions.AutoApprovedTools, toolName), nil
}

// AddAutoApproval adds a tool to the auto-approve list for this conversation
func AddAutoApproval(client mmapi.Client, userID, rootPostID, toolName string) error {
	permissions, err := GetConversationPermissions(client, userID, rootPostID)
	if err != nil {
		return err
	}

	// Check if already in list
	if slices.Contains(permissions.AutoApprovedTools, toolName) {
		// Already auto-approved, nothing to do
		return nil
	}

	// Add to list
	permissions.AutoApprovedTools = append(permissions.AutoApprovedTools, toolName)
	permissions.UpdatedAt = time.Now().Unix()

	// Save back to KV store
	key := buildConversationPermissionKey(userID, rootPostID)
	if err := client.KVSet(key, permissions); err != nil {
		return fmt.Errorf("failed to store conversation permissions in KV store: %w", err)
	}

	return nil
}

// RemoveAutoApproval removes a tool from the auto-approve list for this conversation
func RemoveAutoApproval(client mmapi.Client, userID, rootPostID, toolName string) error {
	permissions, err := GetConversationPermissions(client, userID, rootPostID)
	if err != nil {
		return err
	}

	found := false
	filtered := slices.DeleteFunc(permissions.AutoApprovedTools, func(t string) bool {
		if t == toolName {
			found = true
			return true
		}
		return false
	})

	if !found {
		// Tool wasn't in list, nothing to do
		return nil
	}

	permissions.AutoApprovedTools = filtered
	permissions.UpdatedAt = time.Now().Unix()

	// Save back to KV store
	key := buildConversationPermissionKey(userID, rootPostID)
	if err := client.KVSet(key, permissions); err != nil {
		return fmt.Errorf("failed to store conversation permissions in KV store: %w", err)
	}

	return nil
}

// GetAutoApprovedTools returns the list of auto-approved tools for a conversation
func GetAutoApprovedTools(client mmapi.Client, userID, rootPostID string) ([]string, error) {
	permissions, err := GetConversationPermissions(client, userID, rootPostID)
	if err != nil {
		return nil, err
	}

	return permissions.AutoApprovedTools, nil
}
