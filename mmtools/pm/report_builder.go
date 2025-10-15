// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
)

// ReportBuilder helps construct reports with less string concatenation duplication
type ReportBuilder struct {
	content strings.Builder
}

// NewReportBuilder creates a new report builder
func NewReportBuilder() *ReportBuilder {
	return &ReportBuilder{}
}

// AddTitle adds a formatted title to the report
func (rb *ReportBuilder) AddTitle(template, value string) *ReportBuilder {
	rb.content.WriteString(fmt.Sprintf(template, value))
	return rb
}

// AddHeader adds a section header
func (rb *ReportBuilder) AddHeader(header string) *ReportBuilder {
	rb.content.WriteString(header)
	return rb
}

// AddBulletList adds a bulleted list of items
func (rb *ReportBuilder) AddBulletList(items []string) *ReportBuilder {
	for _, item := range items {
		rb.content.WriteString(fmt.Sprintf("- %s\n", item))
	}
	if len(items) > 0 {
		rb.content.WriteString("\n")
	}
	return rb
}

// AddDocumentList adds a list of documents with titles, URLs, and optional excerpts
func (rb *ReportBuilder) AddDocumentList(sourceName string, docs []datasources.Doc, maxExcerptLength int, ellipsisText string) *ReportBuilder {
	if len(docs) == 0 {
		return rb
	}

	rb.content.WriteString(fmt.Sprintf("**%s:**\n", sourceName))
	for _, doc := range docs {
		rb.content.WriteString(fmt.Sprintf("- [%s](%s)\n", doc.Title, doc.URL))

		metadataContext := buildMetadataContext(doc.Labels)
		if metadataContext != "" {
			rb.content.WriteString(fmt.Sprintf("  %s\n", metadataContext))
		}

		if doc.Content != "" {
			excerpt := doc.Content
			if len(excerpt) > maxExcerptLength {
				excerpt = excerpt[:maxExcerptLength] + ellipsisText
			}
			rb.content.WriteString(fmt.Sprintf("  %s\n", excerpt))
		}
	}
	rb.content.WriteString("\n")
	return rb
}

// AddText adds arbitrary text to the report
func (rb *ReportBuilder) AddText(text string) *ReportBuilder {
	rb.content.WriteString(text)
	return rb
}

// String returns the final report as a string
func (rb *ReportBuilder) String() string {
	return rb.content.String()
}

// buildMetadataContext extracts and formats metadata from Doc.Labels for LLM context
func buildMetadataContext(labels []string) string {
	if len(labels) == 0 {
		return ""
	}

	var segments []string
	var competitors []string
	var priority string
	var categories []string
	var crossRefs []string

	for _, label := range labels {
		switch {
		case strings.HasPrefix(label, "segment:"):
			segment := strings.TrimPrefix(label, "segment:")
			segments = append(segments, segment)
		case strings.HasPrefix(label, "competitive:"):
			competitor := strings.TrimPrefix(label, "competitive:")
			competitors = append(competitors, competitor)
		case strings.HasPrefix(label, "priority:"):
			priority = strings.TrimPrefix(label, "priority:")
		case strings.HasPrefix(label, "category:"):
			category := strings.TrimPrefix(label, "category:")
			categories = append(categories, category)
		case strings.HasPrefix(label, "jira:"), strings.HasPrefix(label, "github:"):
			crossRefs = append(crossRefs, label)
		}
	}

	var parts []string

	if len(segments) > 0 {
		parts = append(parts, fmt.Sprintf("Segments: %s", strings.Join(segments, ", ")))
	}

	if priority != "" && priority != "medium" {
		parts = append(parts, fmt.Sprintf("Priority: %s", priority))
	}

	if len(competitors) > 0 {
		parts = append(parts, fmt.Sprintf("Competitive: vs %s", strings.Join(competitors, ", ")))
	}

	if len(categories) > 0 {
		parts = append(parts, fmt.Sprintf("Categories: %s", strings.Join(categories, ", ")))
	}

	if len(crossRefs) > 0 {
		displayRefs := crossRefs
		if len(crossRefs) > 2 {
			displayRefs = crossRefs[:2]
		}
		parts = append(parts, fmt.Sprintf("Refs: %s", strings.Join(displayRefs, ", ")))
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("*[%s]*", strings.Join(parts, " | "))
}
