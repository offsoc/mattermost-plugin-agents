// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"fmt"
	"strings"
)

// Metadata contains Dev-specific metadata for entities
type Metadata struct {
	Severity   Severity
	IssueType  IssueType
	Components []Component
	Languages  []Language
	Complexity Complexity
}

// GetPriority returns the priority level for this entity as a string
func (m Metadata) GetPriority() string {
	switch m.Severity {
	case SeverityCritical:
		return "high"
	case SeverityHigh:
		return "high"
	case SeverityMedium:
		return "medium"
	case SeverityLow:
		return "low"
	case SeverityTrivial:
		return "low"
	default:
		return "none"
	}
}

// GetLabels returns search/filter labels for this entity
func (m Metadata) GetLabels() []string {
	labels := []string{}

	if m.Severity != "" {
		labels = append(labels, FormatSeverityLabel(m.Severity))
	}

	if m.IssueType != "" {
		labels = append(labels, FormatIssueTypeLabel(m.IssueType))
	}

	for _, component := range m.Components {
		labels = append(labels, FormatComponentLabel(component))
	}

	for _, language := range m.Languages {
		labels = append(labels, FormatLanguageLabel(language))
	}

	if m.Complexity != "" {
		labels = append(labels, FormatComplexityLabel(m.Complexity))
	}

	return labels
}

// Summary returns a human-readable summary of the metadata
func (m Metadata) Summary() string {
	var sb strings.Builder
	var parts []string

	if m.Severity != "" {
		parts = append(parts, fmt.Sprintf("Severity: %s", m.Severity))
	}

	if m.IssueType != "" {
		parts = append(parts, fmt.Sprintf("Type: %s", m.IssueType))
	}

	if len(m.Components) > 0 {
		componentStrs := make([]string, len(m.Components))
		for i, c := range m.Components {
			componentStrs[i] = string(c)
		}
		parts = append(parts, fmt.Sprintf("Components: %s", strings.Join(componentStrs, ", ")))
	}

	if len(m.Languages) > 0 {
		langStrs := make([]string, len(m.Languages))
		for i, l := range m.Languages {
			langStrs[i] = string(l)
		}
		parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(langStrs, ", ")))
	}

	if m.Complexity != "" {
		parts = append(parts, fmt.Sprintf("Complexity: %s", m.Complexity))
	}

	if len(parts) == 0 {
		return ""
	}

	sb.WriteString("(")
	sb.WriteString(strings.Join(parts, " | "))
	sb.WriteString(")")

	return sb.String()
}
