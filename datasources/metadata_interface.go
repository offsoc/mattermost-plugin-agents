// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

// RoleMetadata is the interface that all role-specific metadata must implement.
// This allows datasources to remain role-agnostic while providing structured
// metadata enrichment for different bot roles (PM, Dev, etc.).
//
// Each role (PM, Dev, Sales, Support, etc.) implements this interface with
// their own domain-specific fields while exposing a consistent API for
// common operations like priority assessment, label generation, and formatting.
type RoleMetadata interface {
	// GetPriority returns the priority level for this entity as a string.
	// Standard values: "high", "medium", "low", "none"
	// All roles should map their domain-specific signals to these standard values.
	GetPriority() string

	// GetLabels returns search/filter labels for this entity.
	// Labels are used for categorization, filtering, and search ranking.
	// Format: "key:value" (e.g., "segment:enterprise", "tech:go")
	GetLabels() []string

	// Summary returns a human-readable summary of the metadata.
	// This is used in LLM prompts for citation and context.
	// Format: "(Priority: high | key1: value1 | key2: value2)"
	Summary() string
}
