// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testDevMetadata is a simple implementation of RoleMetadata for testing Dev bot
// Avoids import cycle with grounding/roles/dev
type testDevMetadata struct {
	severity   string
	issueType  string
	components []string
	languages  []string
	complexity string
}

func (m *testDevMetadata) GetFieldNames() []string {
	var fields []string
	if m.severity != "" {
		fields = append(fields, "severity")
	}
	if m.issueType != "" {
		fields = append(fields, "issue_type")
	}
	if len(m.components) > 0 {
		fields = append(fields, "components")
	}
	if len(m.languages) > 0 {
		fields = append(fields, "languages")
	}
	if m.complexity != "" {
		fields = append(fields, "complexity")
	}
	return fields
}

func (m *testDevMetadata) GetFieldValue(fieldName string) []string {
	switch fieldName {
	case "severity":
		if m.severity != "" {
			return []string{m.severity}
		}
	case "issue_type":
		if m.issueType != "" {
			return []string{m.issueType}
		}
	case "components":
		return m.components
	case "languages":
		return m.languages
	case "complexity":
		if m.complexity != "" {
			return []string{m.complexity}
		}
	}
	return nil
}

func (m *testDevMetadata) ValidateClaim(field, value string) bool {
	values := m.GetFieldValue(field)
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (m *testDevMetadata) GetExtractionPatterns() ExtractionPatterns {
	return ExtractionPatterns{
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
		},
	}
}

func TestExtractMetadataClaims_InlinePattern_Dev(t *testing.T) {
	response := "Based on MM-12345 (Severity: critical | Components: api, database | Languages: go), we need to address this issue."

	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
	}

	devMeta := &testDevMetadata{
		severity:   "critical",
		components: []string{"api", "database"},
		languages:  []string{"go"},
	}

	extractor := NewClaimExtractor(devMeta)
	result := extractor.ExtractMetadataClaims(response, citations)

	assert.Equal(t, 1, len(result))
	assert.Greater(t, len(result[0].MetadataClaims), 0, "Should extract claims")

	foundSeverity := false
	foundComponents := 0
	foundLanguages := false

	for _, claim := range result[0].MetadataClaims {
		switch claim.Field {
		case "severity":
			foundSeverity = true
			assert.Equal(t, "critical", claim.ClaimedValue)
		case "components":
			foundComponents++
			assert.Contains(t, []string{"api", "database"}, claim.ClaimedValue)
		case "languages":
			foundLanguages = true
			assert.Equal(t, "go", claim.ClaimedValue)
		}
	}

	assert.True(t, foundSeverity, "Should extract severity")
	assert.Equal(t, 2, foundComponents, "Should extract both components")
	assert.True(t, foundLanguages, "Should extract language")
}

func TestExtractMetadataClaims_NaturalLanguage_Dev(t *testing.T) {
	response := "MM-12345 is critical severity. The bug issue MM-67890 is also relevant."

	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
		{Type: CitationJiraTicket, Value: "MM-67890"},
	}

	devMeta := &testDevMetadata{
		severity:  "critical",
		issueType: "bug",
	}

	extractor := NewClaimExtractor(devMeta)
	result := extractor.ExtractMetadataClaims(response, citations)

	assert.Equal(t, 2, len(result))

	foundSeverity := false
	for _, claim := range result[0].MetadataClaims {
		if claim.Field == "severity" && claim.ClaimedValue == "critical" {
			foundSeverity = true
		}
	}
	assert.True(t, foundSeverity, "Should extract severity from 'is critical severity' pattern")

	foundIssueType := false
	for _, claim := range result[1].MetadataClaims {
		if claim.Field == "issue_type" && claim.ClaimedValue == "bug" {
			foundIssueType = true
		}
	}
	assert.True(t, foundIssueType, "Should extract issue type from 'bug issue MM-67890' pattern")
}

func TestValidateMetadataClaims_AccurateClaims_Dev(t *testing.T) {
	devMeta := &testDevMetadata{
		severity:   "critical",
		issueType:  "bug",
		components: []string{"api", "database"},
		languages:  []string{"go", "typescript"},
		complexity: "high",
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {
				Key:      "MM-12345",
				Metadata: devMeta,
			},
		},
	}

	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "severity", ClaimedValue: "critical"},
				{Field: "components", ClaimedValue: "api"},
				{Field: "languages", ClaimedValue: "go"},
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

func TestValidateMetadataClaims_InaccurateClaims_Dev(t *testing.T) {
	devMeta := &testDevMetadata{
		severity:   "minor",
		issueType:  "feature",
		components: []string{"webapp"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {
				Key:      "MM-12345",
				Metadata: devMeta,
			},
		},
	}

	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "severity", ClaimedValue: "critical"},
				{Field: "issue_type", ClaimedValue: "bug"},
				{Field: "components", ClaimedValue: "api"},
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

	severityClaim := result[0].MetadataClaims[0]
	assert.Equal(t, "severity", severityClaim.Field)
	assert.Equal(t, "critical", severityClaim.ClaimedValue)
	assert.Equal(t, "minor", severityClaim.ActualValue)
	assert.False(t, severityClaim.IsAccurate)
}

func TestValidateCitationsWithClaims_EndToEnd_Dev(t *testing.T) {
	response := "Based on MM-12345 (Severity: critical | Components: api), we should prioritize this fix."

	citations := ExtractCitations(response)

	var jiraCitations []Citation
	for _, c := range citations {
		if c.Type == CitationJiraTicket {
			jiraCitations = append(jiraCitations, c)
		}
	}

	devMeta := &testDevMetadata{
		severity:   "critical",
		components: []string{"api"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {
				Key:      "MM-12345",
				Metadata: devMeta,
			},
		},
	}

	result := ValidateCitationsWithClaims(response, jiraCitations, refIndex, devMeta)

	assert.Equal(t, 1, len(result))

	assert.True(t, result[0].IsValid)
	assert.Equal(t, ValidationGrounded, result[0].ValidationStatus)

	assert.Greater(t, len(result[0].MetadataClaims), 0)

	for _, claim := range result[0].MetadataClaims {
		assert.True(t, claim.IsAccurate, "Claims should be accurate: %v", claim)
	}
}

func TestCalculateGroundingScore_WithClaims_Dev(t *testing.T) {
	response := "Based on MM-12345 (Severity: critical | Components: api)"

	citations := []Citation{
		{
			Type:             CitationJiraTicket,
			Value:            "MM-12345",
			ValidationStatus: ValidationGrounded,
			IsValid:          true,
			MetadataClaims: []MetadataClaim{
				{Field: "severity", ClaimedValue: "critical", ActualValue: "critical", IsAccurate: true},
				{Field: "components", ClaimedValue: "api", ActualValue: "api", IsAccurate: true},
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

func TestCalculateGroundingScore_WithInaccurateClaims_Dev(t *testing.T) {
	response := "Based on MM-12345 (Severity: critical)"

	citations := []Citation{
		{
			Type:             CitationJiraTicket,
			Value:            "MM-12345",
			ValidationStatus: ValidationGrounded,
			IsValid:          true,
			MetadataClaims: []MetadataClaim{
				{Field: "severity", ClaimedValue: "critical", ActualValue: "minor", IsAccurate: false},
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

func TestCalculateGroundingScore_MetadataScore_Dev(t *testing.T) {
	response := "Based on MM-12345 (Severity: critical | Components: api | Languages: go)"

	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "severity", ClaimedValue: "critical"},
				{Field: "components", ClaimedValue: "api"},
				{Field: "languages", ClaimedValue: "go"},
			},
		},
	}

	thresholds := DefaultThresholds()
	result := CalculateGroundingScore(response, citations, thresholds, false)

	assert.Equal(t, 3, result.MetadataUsage.TotalFields, "Should have 3 unique metadata fields")
	assert.Equal(t, 1, result.MetadataUsage.FieldCounts["severity"])
	assert.Equal(t, 1, result.MetadataUsage.FieldCounts["components"])
	assert.Equal(t, 1, result.MetadataUsage.FieldCounts["languages"])
	assert.Greater(t, result.OverallScore, 0.0)
}
