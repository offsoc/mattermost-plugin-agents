// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

// RoleMetadata provides role-specific metadata for grounding validation
// Each bot role (PM, Dev, etc.) implements this to define its metadata fields
type RoleMetadata interface {
	// GetFieldNames returns all field names this role tracks (e.g., ["priority", "segments"])
	GetFieldNames() []string

	// GetFieldValue returns all values for a given field (e.g., "segments" -> ["enterprise", "smb"])
	GetFieldValue(fieldName string) []string

	// ValidateClaim checks if a field/value pair is valid for this role
	ValidateClaim(field, value string) bool

	// GetExtractionPatterns returns regex patterns for extracting claims from LLM responses
	GetExtractionPatterns() ExtractionPatterns
}

// ExtractionPatterns defines patterns for extracting metadata claims from LLM responses
type ExtractionPatterns struct {
	// InlineFieldPattern matches "Field: value" patterns in parentheses
	// Example: "MM-12345 (Priority: high | Segments: enterprise)"
	InlineFieldPattern string

	// ValuePatterns maps field names to their extraction patterns
	// Example: "priority" -> `(high|medium|low|critical)`
	ValuePatterns map[string]string

	// FieldAliases maps alternative names to canonical field names
	// Example: "segment" -> "segments", "category" -> "categories"
	FieldAliases map[string]string
}

// MetadataConverter converts datasources metadata to grounding metadata
type MetadataConverter interface {
	// Convert transforms datasources.EntityMetadata.RoleMetadata to grounding.RoleMetadata
	Convert(datasourcesMetadata interface{}) RoleMetadata
}
