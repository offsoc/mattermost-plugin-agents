// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testPMMetadata is a simple implementation of RoleMetadata for testing PM bot
// Avoids import cycle with grounding/roles/pm
type testPMMetadata struct {
	priority    string
	segments    []string
	categories  []string
	competitive string
}

func (m *testPMMetadata) GetFieldNames() []string {
	var fields []string
	if m.priority != "" {
		fields = append(fields, "priority")
	}
	if len(m.segments) > 0 {
		fields = append(fields, "segments")
	}
	if len(m.categories) > 0 {
		fields = append(fields, "categories")
	}
	if m.competitive != "" {
		fields = append(fields, "competitive")
	}
	return fields
}

func (m *testPMMetadata) GetFieldValue(fieldName string) []string {
	switch fieldName {
	case "priority":
		if m.priority != "" {
			return []string{m.priority}
		}
	case "segments":
		return m.segments
	case "categories":
		return m.categories
	case "competitive":
		if m.competitive != "" {
			return []string{m.competitive}
		}
	}
	return nil
}

func (m *testPMMetadata) ValidateClaim(field, value string) bool {
	values := m.GetFieldValue(field)
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (m *testPMMetadata) GetExtractionPatterns() ExtractionPatterns {
	return ExtractionPatterns{
		InlineFieldPattern: `(?i)(Priority|Segments?|Categories|Competitive):\s*(.+)`,
		ValuePatterns: map[string]string{
			"priority": `(high|medium|low|critical)`,
			"segments": `(enterprise|smb|federal|government|mid-market|startup)`,
		},
		FieldAliases: map[string]string{
			"segment":  "segments",
			"category": "categories",
		},
	}
}

func TestExtractMetadataClaims_InlinePattern_PM(t *testing.T) {
	response := "Based on MM-12345 (Priority: high | Segments: enterprise, federal | Categories: authentication), we need to address this issue."

	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
	}

	pmMeta := &testPMMetadata{
		priority:   "high",
		segments:   []string{"enterprise", "federal"},
		categories: []string{"authentication"},
	}

	extractor := NewClaimExtractor(pmMeta)
	result := extractor.ExtractMetadataClaims(response, citations)

	assert.Equal(t, 1, len(result))
	assert.Greater(t, len(result[0].MetadataClaims), 0, "Should extract claims")

	foundPriority := false
	foundSegments := 0
	foundCategories := false

	for _, claim := range result[0].MetadataClaims {
		switch claim.Field {
		case "priority":
			foundPriority = true
			assert.Equal(t, "high", claim.ClaimedValue)
		case "segments":
			foundSegments++
			assert.Contains(t, []string{"enterprise", "federal"}, claim.ClaimedValue)
		case "categories":
			foundCategories = true
			assert.Equal(t, "authentication", claim.ClaimedValue)
		}
	}

	assert.True(t, foundPriority, "Should extract priority")
	assert.Equal(t, 2, foundSegments, "Should extract both segments")
	assert.True(t, foundCategories, "Should extract category")
}

func TestExtractMetadataClaims_NaturalLanguage_PM(t *testing.T) {
	response := "MM-12345 is high priority. The enterprise issue MM-67890 is also relevant."

	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
		{Type: CitationJiraTicket, Value: "MM-67890"},
	}

	pmMeta := &testPMMetadata{
		priority: "high",
		segments: []string{"enterprise"},
	}

	extractor := NewClaimExtractor(pmMeta)
	result := extractor.ExtractMetadataClaims(response, citations)

	assert.Equal(t, 2, len(result))

	foundPriority := false
	for _, claim := range result[0].MetadataClaims {
		if claim.Field == "priority" && claim.ClaimedValue == "high" {
			foundPriority = true
		}
	}
	assert.True(t, foundPriority, "Should extract priority from 'is high priority' pattern")

	foundSegment := false
	for _, claim := range result[1].MetadataClaims {
		if claim.Field == "segments" && claim.ClaimedValue == "enterprise" {
			foundSegment = true
		}
	}
	assert.True(t, foundSegment, "Should extract segment from 'enterprise issue MM-67890' pattern")
}

func TestValidateMetadataClaims_AccurateClaims_PM(t *testing.T) {
	pmMeta := &testPMMetadata{
		priority:    "high",
		segments:    []string{"enterprise", "federal"},
		categories:  []string{"authentication", "performance"},
		competitive: "slack",
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {
				Key:      "MM-12345",
				Metadata: pmMeta,
			},
		},
	}

	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "priority", ClaimedValue: "high"},
				{Field: "segments", ClaimedValue: "enterprise"},
				{Field: "categories", ClaimedValue: "authentication"},
			},
		},
	}

	result := ValidateMetadataClaims(citations, refIndex)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, 3, len(result[0].MetadataClaims))

	for _, claim := range result[0].MetadataClaims {
		assert.True(t, claim.IsAccurate, "All claims should be accurate: %v", claim)
		assert.NotEmpty(t, claim.ActualValue, "ActualValue should be populated")
	}
}

func TestValidateMetadataClaims_InaccurateClaims_PM(t *testing.T) {
	pmMeta := &testPMMetadata{
		priority:   "low",
		segments:   []string{"smb"},
		categories: []string{"mobile"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {
				Key:      "MM-12345",
				Metadata: pmMeta,
			},
		},
	}

	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "priority", ClaimedValue: "high"},
				{Field: "segments", ClaimedValue: "enterprise"},
				{Field: "categories", ClaimedValue: "authentication"},
			},
		},
	}

	result := ValidateMetadataClaims(citations, refIndex)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, 3, len(result[0].MetadataClaims))

	for _, claim := range result[0].MetadataClaims {
		assert.False(t, claim.IsAccurate, "All claims should be inaccurate: %v", claim)
		assert.NotEmpty(t, claim.ActualValue, "ActualValue should show what it really is")
	}

	priorityClaim := result[0].MetadataClaims[0]
	assert.Equal(t, "priority", priorityClaim.Field)
	assert.Equal(t, "high", priorityClaim.ClaimedValue)
	assert.Equal(t, "low", priorityClaim.ActualValue)
	assert.False(t, priorityClaim.IsAccurate)
}

func TestValidateCitationsWithClaims_EndToEnd_PM(t *testing.T) {
	response := "Based on MM-12345 (Priority: high | Segments: enterprise), we should prioritize this fix."

	citations := ExtractCitations(response)

	var jiraCitations []Citation
	for _, c := range citations {
		if c.Type == CitationJiraTicket {
			jiraCitations = append(jiraCitations, c)
		}
	}

	pmMeta := &testPMMetadata{
		priority: "high",
		segments: []string{"enterprise"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {
				Key:      "MM-12345",
				Metadata: pmMeta,
			},
		},
	}

	result := ValidateCitationsWithClaims(response, jiraCitations, refIndex, pmMeta)

	assert.Equal(t, 1, len(result))

	assert.True(t, result[0].IsValid)
	assert.Equal(t, ValidationGrounded, result[0].ValidationStatus)

	assert.Greater(t, len(result[0].MetadataClaims), 0)

	for _, claim := range result[0].MetadataClaims {
		assert.True(t, claim.IsAccurate, "Claims should be accurate: %v", claim)
	}
}

func TestCalculateGroundingScore_WithClaims_PM(t *testing.T) {
	response := "Based on MM-12345 (Priority: high | Segments: enterprise)"

	citations := []Citation{
		{
			Type:             CitationJiraTicket,
			Value:            "MM-12345",
			ValidationStatus: ValidationGrounded,
			IsValid:          true,
			MetadataClaims: []MetadataClaim{
				{Field: "priority", ClaimedValue: "high", ActualValue: "high", IsAccurate: true},
				{Field: "segments", ClaimedValue: "enterprise", ActualValue: "enterprise", IsAccurate: true},
			},
		},
	}

	thresholds := DefaultThresholds()
	result := CalculateGroundingScore(response, citations, thresholds, true)

	assert.Equal(t, 1, result.TotalCitations)
	assert.Equal(t, 2, result.TotalClaims)
	assert.Equal(t, 2, result.AccurateClaims)
	assert.Equal(t, 0, result.InaccurateClaims)
	assert.Equal(t, 1.0, result.ClaimAccuracyRate)
	assert.Contains(t, result.Reasoning, "Claims: 2 accurate, 0 inaccurate")
	assert.Contains(t, result.Reasoning, "100.0% accuracy")
}

func TestCalculateGroundingScore_WithInaccurateClaims_PM(t *testing.T) {
	response := "Based on MM-12345 (Priority: high)"

	citations := []Citation{
		{
			Type:             CitationJiraTicket,
			Value:            "MM-12345",
			ValidationStatus: ValidationGrounded,
			IsValid:          true,
			MetadataClaims: []MetadataClaim{
				{Field: "priority", ClaimedValue: "high", ActualValue: "low", IsAccurate: false},
			},
		},
	}

	thresholds := DefaultThresholds()
	result := CalculateGroundingScore(response, citations, thresholds, true)

	assert.Equal(t, 1, result.TotalClaims)
	assert.Equal(t, 0, result.AccurateClaims)
	assert.Equal(t, 1, result.InaccurateClaims)
	assert.Equal(t, 0.0, result.ClaimAccuracyRate)
	assert.Contains(t, result.Reasoning, "Claims: 0 accurate, 1 inaccurate")
	assert.Contains(t, result.Reasoning, "0.0% accuracy")
}

func TestCalculateGroundingScore_MetadataScore_PM(t *testing.T) {
	response := "Based on MM-12345 (Priority: high | Segments: enterprise | Categories: performance)"

	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "priority", ClaimedValue: "high"},
				{Field: "segments", ClaimedValue: "enterprise"},
				{Field: "categories", ClaimedValue: "performance"},
			},
		},
	}

	thresholds := DefaultThresholds()
	result := CalculateGroundingScore(response, citations, thresholds, false)

	assert.Equal(t, 3, result.MetadataUsage.TotalFields, "Should have 3 unique metadata fields")
	assert.Equal(t, 1, result.MetadataUsage.FieldCounts["priority"])
	assert.Equal(t, 1, result.MetadataUsage.FieldCounts["segments"])
	assert.Equal(t, 1, result.MetadataUsage.FieldCounts["categories"])
	assert.Greater(t, result.OverallScore, 0.0)
}
