// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"fmt"
	"strings"
)

// Metadata contains PM-specific metadata for entities
type Metadata struct {
	Segments    []CustomerSegment
	Categories  []TechnicalCategory
	Competitive Competitor
	Priority    Priority
}

// GetPriority returns the priority level for this entity as a string
func (m Metadata) GetPriority() string {
	// Map PM priority to standard string values
	switch m.Priority {
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	case PriorityCompleted:
		return "none" // Completed items are not prioritized
	default:
		return "none"
	}
}

// GetLabels returns search/filter labels for this entity
func (m Metadata) GetLabels() []string {
	labels := []string{}

	for _, segment := range m.Segments {
		labels = append(labels, FormatSegmentLabel(segment))
	}

	for _, category := range m.Categories {
		labels = append(labels, FormatCategoryLabel(category))
	}

	if m.Competitive != "" {
		labels = append(labels, FormatCompetitiveLabel(m.Competitive))
	}

	if m.Priority != "" {
		labels = append(labels, FormatPriorityLabel(m.Priority))
	}

	return labels
}

// Summary returns a human-readable summary of the metadata
// Format: "(Priority: high | Segments: enterprise, federal | Categories: mobile, performance)"
func (m Metadata) Summary() string {
	var sb strings.Builder
	var parts []string

	// Priority first (most important for PM decisions)
	if m.Priority != "" {
		parts = append(parts, fmt.Sprintf("Priority: %s", m.Priority))
	}

	// Customer segments (critical for targeting)
	if len(m.Segments) > 0 {
		segmentStrs := make([]string, len(m.Segments))
		for i, s := range m.Segments {
			segmentStrs[i] = string(s)
		}
		parts = append(parts, fmt.Sprintf("Segments: %s", strings.Join(segmentStrs, ", ")))
	}

	// Technical categories (useful for impact analysis)
	if len(m.Categories) > 0 {
		categoryStrs := make([]string, len(m.Categories))
		for i, c := range m.Categories {
			categoryStrs[i] = string(c)
		}
		parts = append(parts, fmt.Sprintf("Categories: %s", strings.Join(categoryStrs, ", ")))
	}

	// Competitive context (strategic insights)
	if m.Competitive != "" {
		parts = append(parts, fmt.Sprintf("Competitive: %s", m.Competitive))
	}

	// Only return metadata if we have something useful
	if len(parts) == 0 {
		return ""
	}

	sb.WriteString("(")
	sb.WriteString(strings.Join(parts, " | "))
	sb.WriteString(")")

	return sb.String()
}
