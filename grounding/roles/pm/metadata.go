// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"strings"

	dsPM "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/mattermost/mattermost-plugin-ai/grounding"
)

// Metadata implements grounding.RoleMetadata for PM bot
type Metadata struct {
	Priority    string
	Segments    []string
	Categories  []string
	Competitive string
}

// NewMetadata creates PM grounding metadata from datasources PM metadata
func NewMetadata(dsMetadata dsPM.Metadata) *Metadata {
	segments := make([]string, len(dsMetadata.Segments))
	for i, seg := range dsMetadata.Segments {
		segments[i] = string(seg)
	}

	categories := make([]string, len(dsMetadata.Categories))
	for i, cat := range dsMetadata.Categories {
		categories[i] = string(cat)
	}

	return &Metadata{
		Priority:    string(dsMetadata.Priority),
		Segments:    segments,
		Categories:  categories,
		Competitive: string(dsMetadata.Competitive),
	}
}

// GetFieldNames returns PM metadata field names
func (m *Metadata) GetFieldNames() []string {
	var fields []string
	if m.Priority != "" {
		fields = append(fields, "priority")
	}
	if len(m.Segments) > 0 {
		fields = append(fields, "segments")
	}
	if len(m.Categories) > 0 {
		fields = append(fields, "categories")
	}
	if m.Competitive != "" {
		fields = append(fields, "competitive")
	}
	return fields
}

// GetFieldValue returns values for a given PM metadata field
func (m *Metadata) GetFieldValue(fieldName string) []string {
	switch strings.ToLower(fieldName) {
	case "priority":
		if m.Priority != "" {
			return []string{m.Priority}
		}
	case "segments":
		return m.Segments
	case "categories":
		return m.Categories
	case "competitive":
		if m.Competitive != "" {
			return []string{m.Competitive}
		}
	}
	return nil
}

// ValidateClaim checks if a field/value pair is valid for PM metadata
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

// GetExtractionPatterns returns regex patterns for extracting PM metadata claims
func (m *Metadata) GetExtractionPatterns() grounding.ExtractionPatterns {
	return grounding.ExtractionPatterns{
		InlineFieldPattern: `(?i)(Priority|Segments?|Categories|Competitive):\s*(.+)`,
		ValuePatterns: map[string]string{
			"priority": `(high|medium|low|critical)`,
			"segments": `(enterprise|smb|federal|government|mid-market|startup)`,
		},
		FieldAliases: map[string]string{
			"segment":  "segments",
			"category": "categories",
			"categor":  "categories",
		},
	}
}
