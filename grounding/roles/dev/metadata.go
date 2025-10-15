// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"strings"

	dsDev "github.com/mattermost/mattermost-plugin-ai/datasources/segments/dev"
	"github.com/mattermost/mattermost-plugin-ai/grounding"
)

// Metadata implements grounding.RoleMetadata for Dev bot
type Metadata struct {
	Severity   string
	IssueType  string
	Components []string
	Languages  []string
	Complexity string
}

// NewMetadata creates Dev grounding metadata from datasources Dev metadata
func NewMetadata(dsMetadata dsDev.Metadata) *Metadata {
	components := make([]string, len(dsMetadata.Components))
	for i, comp := range dsMetadata.Components {
		components[i] = string(comp)
	}

	languages := make([]string, len(dsMetadata.Languages))
	for i, lang := range dsMetadata.Languages {
		languages[i] = string(lang)
	}

	return &Metadata{
		Severity:   string(dsMetadata.Severity),
		IssueType:  string(dsMetadata.IssueType),
		Components: components,
		Languages:  languages,
		Complexity: string(dsMetadata.Complexity),
	}
}

// GetFieldNames returns Dev metadata field names
func (m *Metadata) GetFieldNames() []string {
	var fields []string
	if m.Severity != "" {
		fields = append(fields, "severity")
	}
	if m.IssueType != "" {
		fields = append(fields, "issue_type")
	}
	if len(m.Components) > 0 {
		fields = append(fields, "components")
	}
	if len(m.Languages) > 0 {
		fields = append(fields, "languages")
	}
	if m.Complexity != "" {
		fields = append(fields, "complexity")
	}
	return fields
}

// GetFieldValue returns values for a given Dev metadata field
func (m *Metadata) GetFieldValue(fieldName string) []string {
	switch strings.ToLower(fieldName) {
	case "severity":
		if m.Severity != "" {
			return []string{m.Severity}
		}
	case "issue_type", "issuetype", "type":
		if m.IssueType != "" {
			return []string{m.IssueType}
		}
	case "components", "component":
		return m.Components
	case "languages", "language":
		return m.Languages
	case "complexity":
		if m.Complexity != "" {
			return []string{m.Complexity}
		}
	}
	return nil
}

// ValidateClaim checks if a field/value pair is valid for Dev metadata
func (m *Metadata) ValidateClaim(field, value string) bool {
	values := m.GetFieldValue(field)
	valueLower := strings.ToLower(value)
	for _, v := range values {
		if strings.ToLower(v) == valueLower {
			return true
		}
	}
	return false
}

// GetExtractionPatterns returns regex patterns for extracting Dev metadata claims
func (m *Metadata) GetExtractionPatterns() grounding.ExtractionPatterns {
	return grounding.ExtractionPatterns{
		InlineFieldPattern: `(?i)(Severity|IssueType|Type|Components?|Languages?|Complexity):\s*(.+)`,
		ValuePatterns: map[string]string{
			"severity":   `(critical|major|minor|trivial)`,
			"issue_type": `(bug|feature|improvement|task)`,
			"complexity": `(high|medium|low)`,
		},
		FieldAliases: map[string]string{
			"component": "components",
			"language":  "languages",
			"type":      "issue_type",
			"issuetype": "issue_type",
		},
	}
}

// GetPriority maps severity to a priority string for interface compatibility
func (m *Metadata) GetPriority() string {
	// For dev bot, severity maps to priority
	switch strings.ToLower(m.Severity) {
	case "critical":
		return "high"
	case "major":
		return "medium"
	case "minor", "trivial":
		return "low"
	default:
		return "none"
	}
}
