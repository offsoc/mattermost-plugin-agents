// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateGroundingScore_Phase1(t *testing.T) {
	response := "Based on MM-12345 and MM-67890, we analyzed the issue."
	citations := ExtractCitations(response)
	thresholds := DefaultThresholds()

	result := CalculateGroundingScore(response, citations, thresholds, false)

	assert.Equal(t, 2, result.TotalCitations)
	assert.False(t, result.Pass, "Should fail without validation")
}

func TestCalculateGroundingScore_Phase1Pass(t *testing.T) {
	response := "Based on MM-1234, MM-5678, MM-9012, see https://docs.mattermost.com"
	citations := ExtractCitations(response)
	thresholds := DefaultThresholds()

	result := CalculateGroundingScore(response, citations, thresholds, false)

	assert.GreaterOrEqual(t, result.TotalCitations, 3)
	assert.False(t, result.Pass, "Should fail without validation")
	assert.Contains(t, result.Reasoning, "No validation performed")
}

func TestCalculateGroundingScore_Phase2WithValidation(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345", ValidationStatus: ValidationGrounded, IsValid: true},
		{Type: CitationJiraTicket, Value: "MM-67890", ValidationStatus: ValidationGrounded, IsValid: true},
		{Type: CitationJiraTicket, Value: "MM-99999", ValidationStatus: ValidationFabricated, IsValid: false},
	}

	response := "Based on MM-12345, MM-67890, and MM-99999"
	thresholds := DefaultThresholds()

	result := CalculateGroundingScore(response, citations, thresholds, true)

	assert.Equal(t, 3, result.TotalCitations)
	assert.Equal(t, 2, result.ValidCitations)
	assert.Equal(t, 1, result.InvalidCitations)
	assert.InDelta(t, 0.667, result.ValidCitationRate, 0.01)
	assert.False(t, result.Pass, "Should fail: valid rate 66% < threshold 70%")
}

func TestCalculateGroundingScore_ThresholdBoundary(t *testing.T) {
	thresholds := Thresholds{
		MinCitationRate:   0.70,
		MinMetadataFields: 0,
		CitationWeight:    1.0,
		MetadataWeight:    0.0,
	}

	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-1", ValidationStatus: ValidationGrounded, IsValid: true},
		{Type: CitationJiraTicket, Value: "MM-2", ValidationStatus: ValidationGrounded, IsValid: true},
		{Type: CitationJiraTicket, Value: "MM-3", ValidationStatus: ValidationGrounded, IsValid: true},
	}

	result := CalculateGroundingScore("test", citations, thresholds, true)

	assert.True(t, result.Pass, "Should pass: 100% valid rate >= 70% threshold")
}

func TestCalculateGroundingScore_ZeroCitations(t *testing.T) {
	response := "2 + 2 = 4"
	citations := ExtractCitations(response)
	thresholds := DefaultThresholds()

	result := CalculateGroundingScore(response, citations, thresholds, false)

	assert.Equal(t, 0, result.TotalCitations)
	assert.False(t, result.Pass)
	assert.Contains(t, result.Reasoning, "No validation performed")
}

func TestCalculateGroundingScore_FabricatedCitations(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345", ValidationStatus: ValidationGrounded, IsValid: true},
		{Type: CitationJiraTicket, Value: "MM-67890", ValidationStatus: ValidationFabricated, IsValid: false},
		{Type: CitationGitHub, Value: "mattermost/mattermost#99999", ValidationStatus: ValidationFabricated, IsValid: false},
		{Type: CitationURL, Value: "https://example.com/real", ValidationStatus: ValidationUngroundedValid},
	}

	response := "Based on MM-12345, MM-67890, mattermost/mattermost#99999, and https://example.com/real"
	thresholds := DefaultThresholds()

	result := CalculateGroundingScore(response, citations, thresholds, true)

	assert.Equal(t, 4, result.TotalCitations)
	assert.Equal(t, 2, result.ValidCitations, "Grounded + ungrounded-valid citations are valid")
	assert.Equal(t, 2, result.InvalidCitations, "Two fabricated citations are invalid")
	assert.Equal(t, 2, result.FabricatedCitations, "Two fabricated citations detected")
	assert.Equal(t, 1, result.GroundedCitations)
	assert.Equal(t, 1, result.UngroundedValidURLs)
	assert.Equal(t, 0.5, result.FabricationRate, "50% fabrication rate (2 of 4)")
	assert.Contains(t, result.Reasoning, "Fabricated: 2")
	assert.Contains(t, result.Reasoning, "Fabrication rate: 50.0%")
}

func TestCalculateGroundingScore_APIErrors(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345", ValidationStatus: ValidationGrounded, IsValid: true},
		{Type: CitationJiraTicket, Value: "MM-67890", ValidationStatus: ValidationAPIError},
		{Type: CitationGitHub, Value: "#12345", ValidationStatus: ValidationAPIError},
	}

	response := "Based on MM-12345, MM-67890, and #12345"
	thresholds := DefaultThresholds()

	result := CalculateGroundingScore(response, citations, thresholds, true)

	assert.Equal(t, 3, result.TotalCitations)
	assert.Equal(t, 2, result.APIErrors, "Two API errors")
	assert.Equal(t, 0, result.FabricatedCitations, "No fabrications, just API errors")
	assert.Contains(t, result.Reasoning, "API-errors: 2")
}

func TestCalculateGroundingScore_BaselineUngroundedValid(t *testing.T) {
	citations := []Citation{
		{Type: CitationURL, Value: "https://forum.mattermost.com/search?q=test", ValidationStatus: ValidationUngroundedValid},
		{Type: CitationURL, Value: "https://mattermost.atlassian.net/browse/MM-789", ValidationStatus: ValidationUngroundedValid},
		{Type: CitationURL, Value: "https://forum.mattermost.com/t/broken/1", ValidationStatus: ValidationUngroundedBroken},
		{Type: CitationURL, Value: "https://forum.mattermost.com/t/broken/2", ValidationStatus: ValidationUngroundedBroken},
	}

	response := "Based on research at https://forum.mattermost.com/search?q=test and https://mattermost.atlassian.net/browse/MM-789"
	thresholds := DefaultThresholds()

	resultBaseline := CalculateGroundingScore(response, citations, thresholds, false)

	assert.Equal(t, 4, resultBaseline.TotalCitations)
	assert.Equal(t, 2, resultBaseline.ValidCitations, "In baseline (no tool results), ungrounded-valid should count as valid")
	assert.Equal(t, 2, resultBaseline.InvalidCitations, "Ungrounded-broken should count as invalid")
	assert.Equal(t, 0.5, resultBaseline.ValidCitationRate, "50% valid rate (2 of 4)")
	assert.False(t, resultBaseline.Pass, "Should fail: 50% < 70% threshold")

	resultWithTools := CalculateGroundingScore(response, citations, thresholds, true)

	assert.Equal(t, 4, resultWithTools.TotalCitations)
	assert.Equal(t, 2, resultWithTools.ValidCitations, "Ungrounded-valid citations always count as valid")
	assert.Equal(t, 2, resultWithTools.InvalidCitations, "Ungrounded-broken should count as invalid")
	assert.Equal(t, 0.5, resultWithTools.ValidCitationRate, "50% valid rate (2 ungrounded-valid of 4)")
	assert.False(t, resultWithTools.Pass, "Should fail: 50% < 70% threshold")
}
