// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/stretchr/testify/assert"
)

func TestBuildMetadataContext(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected string
	}{
		{
			name:     "empty labels",
			labels:   []string{},
			expected: "",
		},
		{
			name:     "single segment",
			labels:   []string{"segment:federal"},
			expected: "*[Segments: federal]*",
		},
		{
			name:     "multiple segments",
			labels:   []string{"segment:federal", "segment:enterprise"},
			expected: "*[Segments: federal, enterprise]*",
		},
		{
			name:     "full metadata",
			labels:   []string{"segment:federal", "priority:high", "competitive:slack", "category:ai"},
			expected: "*[Segments: federal | Priority: high | Competitive: vs slack | Categories: ai]*",
		},
		{
			name:     "priority medium is hidden",
			labels:   []string{"segment:federal", "priority:medium"},
			expected: "*[Segments: federal]*",
		},
		{
			name:     "cross-references included",
			labels:   []string{"segment:federal", "jira:MM-12345", "github:#456"},
			expected: "*[Segments: federal | Refs: jira:MM-12345, github:#456]*",
		},
		{
			name:     "too many cross-refs truncated to 2",
			labels:   []string{"jira:MM-1", "jira:MM-2", "jira:MM-3"},
			expected: "*[Refs: jira:MM-1, jira:MM-2]*",
		},
		{
			name:     "competitive context",
			labels:   []string{"competitive:slack", "competitive:teams"},
			expected: "*[Competitive: vs slack, teams]*",
		},
		{
			name:     "multiple categories",
			labels:   []string{"category:ai", "category:mobile"},
			expected: "*[Categories: ai, mobile]*",
		},
		{
			name:     "priority high only",
			labels:   []string{"priority:high"},
			expected: "*[Priority: high]*",
		},
		{
			name:     "priority low only",
			labels:   []string{"priority:low"},
			expected: "*[Priority: low]*",
		},
		{
			name:     "complex real-world example",
			labels:   []string{"segment:federal", "segment:healthcare", "priority:high", "category:plugins", "category:authentication", "competitive:slack", "jira:MM-54321"},
			expected: "*[Segments: federal, healthcare | Priority: high | Competitive: vs slack | Categories: plugins, authentication | Refs: jira:MM-54321]*",
		},
		{
			name:     "only non-metadata labels",
			labels:   []string{"type:feature", "state:delivered"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildMetadataContext(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddDocumentListWithMetadata(t *testing.T) {
	rb := NewReportBuilder()

	docs := []datasources.Doc{
		{
			Title:   "Test Feature",
			URL:     "https://example.com/feature",
			Content: "Feature description here",
			Labels:  []string{"segment:federal", "priority:high", "competitive:slack"},
		},
	}

	rb.AddDocumentList("Test Source", docs, 100, "...")
	output := rb.String()

	assert.Contains(t, output, "**Test Source:**")
	assert.Contains(t, output, "[Test Feature](https://example.com/feature)")
	assert.Contains(t, output, "*[Segments: federal | Priority: high | Competitive: vs slack]*")
	assert.Contains(t, output, "Feature description here")
}

func TestAddDocumentListWithoutMetadata(t *testing.T) {
	rb := NewReportBuilder()

	docs := []datasources.Doc{
		{
			Title:   "Test Feature",
			URL:     "https://example.com/feature",
			Content: "Feature description here",
			Labels:  []string{},
		},
	}

	rb.AddDocumentList("Test Source", docs, 100, "...")
	output := rb.String()

	assert.Contains(t, output, "Feature description here")
	assert.NotContains(t, output, "*[")
}

func TestAddDocumentListMultipleDocsWithMetadata(t *testing.T) {
	rb := NewReportBuilder()

	docs := []datasources.Doc{
		{
			Title:   "AI Feature",
			URL:     "https://example.com/ai",
			Content: "AI description",
			Labels:  []string{"segment:federal", "priority:high", "category:ai"},
		},
		{
			Title:   "Mobile Feature",
			URL:     "https://example.com/mobile",
			Content: "Mobile description",
			Labels:  []string{"segment:enterprise", "category:mobile"},
		},
		{
			Title:   "No Metadata Feature",
			URL:     "https://example.com/plain",
			Content: "Plain description",
			Labels:  []string{},
		},
	}

	rb.AddDocumentList("ProductBoard Features", docs, 100, "...")
	output := rb.String()

	assert.Contains(t, output, "*[Segments: federal | Priority: high | Categories: ai]*")
	assert.Contains(t, output, "*[Segments: enterprise | Categories: mobile]*")
	assert.Contains(t, output, "AI description")
	assert.Contains(t, output, "Mobile description")
	assert.Contains(t, output, "Plain description")

	lines := strings.Split(output, "\n")
	metadataLineCount := 0
	for _, line := range lines {
		if strings.Contains(line, "*[") && strings.Contains(line, "]*") {
			metadataLineCount++
		}
	}
	assert.Equal(t, 2, metadataLineCount, "Should have exactly 2 metadata lines (first two docs)")
}

func TestAddDocumentListWithExcerptTruncation(t *testing.T) {
	rb := NewReportBuilder()

	longContent := strings.Repeat("This is a very long description that should be truncated. ", 10)
	docs := []datasources.Doc{
		{
			Title:   "Test Feature",
			URL:     "https://example.com/feature",
			Content: longContent,
			Labels:  []string{"segment:federal"},
		},
	}

	rb.AddDocumentList("Test Source", docs, 50, "...")
	output := rb.String()

	assert.Contains(t, output, "*[Segments: federal]*")
	assert.Contains(t, output, "...")
	assert.Less(t, len(output), len(longContent)+200)
}
