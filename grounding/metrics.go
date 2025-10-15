// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"fmt"
	"strings"
)

// CalculateGroundingScore computes multi-dimensional grounding score
func CalculateGroundingScore(
	response string,
	citations []Citation,
	thresholds Thresholds,
	hasToolResults bool,
) Result {
	result := Result{
		TotalCitations:  len(citations),
		Citations:       citations,
		CitationsByType: make(map[CitationType]int),
	}

	for _, citation := range citations {
		result.CitationsByType[citation.Type]++
	}

	wordCount := len(strings.Fields(response))
	if wordCount > 0 {
		result.CitationDensity = float64(result.TotalCitations) / float64(wordCount) * 100
	}

	result.MetadataUsage = AnalyzeMetadataUsage(citations)

	validCount := 0
	invalidCount := 0
	groundedCount := 0
	ungroundedValidCount := 0
	ungroundedBrokenCount := 0
	fabricatedCount := 0
	apiErrorCount := 0
	hasValidatedCitations := false

	for _, citation := range citations {
		if citation.Type == CitationMetadata {
			continue
		}

		switch citation.ValidationStatus {
		case ValidationGrounded:
			groundedCount++
			validCount++
			hasValidatedCitations = true
		case ValidationUngroundedValid:
			ungroundedValidCount++
			validCount++
			hasValidatedCitations = true
		case ValidationUngroundedBroken:
			ungroundedBrokenCount++
			invalidCount++
			hasValidatedCitations = true
		case ValidationFabricated:
			fabricatedCount++
			invalidCount++
			hasValidatedCitations = true
		case ValidationAPIError:
			apiErrorCount++
			hasValidatedCitations = true
		case ValidationNotChecked, "":
			// Skip unvalidated citations - don't count as valid or invalid
		default:
			if citation.IsValid {
				validCount++
				hasValidatedCitations = true
			} else {
				invalidCount++
				hasValidatedCitations = true
			}
		}
	}

	result.ValidCitations = validCount
	result.InvalidCitations = invalidCount
	result.GroundedCitations = groundedCount
	result.UngroundedValidURLs = ungroundedValidCount
	result.UngroundedBrokenURLs = ungroundedBrokenCount
	result.FabricatedCitations = fabricatedCount
	result.APIErrors = apiErrorCount

	// Count metadata claims
	totalClaims := 0
	accurateClaims := 0
	inaccurateClaims := 0

	for _, citation := range citations {
		for _, claim := range citation.MetadataClaims {
			totalClaims++
			if claim.IsAccurate {
				accurateClaims++
			} else {
				inaccurateClaims++
			}
		}
	}

	result.TotalClaims = totalClaims
	result.AccurateClaims = accurateClaims
	result.InaccurateClaims = inaccurateClaims

	if totalClaims > 0 {
		result.ClaimAccuracyRate = float64(accurateClaims) / float64(totalClaims)
	}

	nonMetadataCitations := result.TotalCitations - result.CitationsByType[CitationMetadata]
	if nonMetadataCitations > 0 {
		result.ValidCitationRate = float64(validCount) / float64(nonMetadataCitations)
		result.FabricationRate = float64(fabricatedCount) / float64(nonMetadataCitations)
	}

	citationScore := calculateCitationScore(result, thresholds, hasValidatedCitations)
	metadataScore := calculateMetadataScore(result.MetadataUsage, thresholds)

	result.OverallScore = (thresholds.CitationWeight * citationScore) +
		(thresholds.MetadataWeight * metadataScore)

	if hasValidatedCitations {
		result.Pass = result.ValidCitationRate >= thresholds.MinCitationRate
	} else {
		result.Pass = false
	}

	reasoningParts := []string{
		fmt.Sprintf("Citations: %d (%.1f per 100 words)", result.TotalCitations, result.CitationDensity),
	}

	if hasValidatedCitations {
		if groundedCount > 0 || ungroundedValidCount > 0 || ungroundedBrokenCount > 0 || fabricatedCount > 0 || apiErrorCount > 0 {
			reasoningParts = append(reasoningParts,
				fmt.Sprintf("Grounded: %d, Ungrounded-valid: %d, Ungrounded-broken: %d, Fabricated: %d, API-errors: %d",
					groundedCount, ungroundedValidCount, ungroundedBrokenCount, fabricatedCount, apiErrorCount))
		}
		reasoningParts = append(reasoningParts,
			fmt.Sprintf("Valid rate: %.1f%%", result.ValidCitationRate*100))
		if fabricatedCount > 0 {
			reasoningParts = append(reasoningParts,
				fmt.Sprintf("Fabrication rate: %.1f%%", result.FabricationRate*100))
		}
	}

	// Add claim accuracy if we have claims
	if totalClaims > 0 {
		reasoningParts = append(reasoningParts,
			fmt.Sprintf("Claims: %d accurate, %d inaccurate (%.1f%% accuracy)",
				accurateClaims, inaccurateClaims, result.ClaimAccuracyRate*100))
	}

	reasoningParts = append(reasoningParts,
		fmt.Sprintf("Metadata fields: %d", countMetadataFields(result.MetadataUsage)),
		fmt.Sprintf("Score: %.2f", result.OverallScore))

	result.Reasoning = strings.Join(reasoningParts, ", ")

	if !result.Pass {
		if !hasValidatedCitations {
			result.Reasoning += " [FAIL: No validation performed - tool results or API clients required]"
		} else if result.ValidCitationRate < thresholds.MinCitationRate {
			result.Reasoning += fmt.Sprintf(" [FAIL: %.1f%% valid rate < threshold %.1f%%]",
				result.ValidCitationRate*100, thresholds.MinCitationRate*100)
		}
	}

	return result
}

func calculateCitationScore(result Result, thresholds Thresholds, hasValidatedCitations bool) float64 {
	nonMetadataCitations := result.TotalCitations - result.CitationsByType[CitationMetadata]

	// No citations at all = 0 score
	if nonMetadataCitations == 0 {
		return 0.0
	}

	// If citations were validated, use quality (valid rate)
	if hasValidatedCitations {
		return result.ValidCitationRate
	}

	// If not validated (no tool results or API), return 0 - validation should always be available
	return 0.0
}

func calculateMetadataScore(usage MetadataUsage, thresholds Thresholds) float64 {
	// If no fields used, score is 0
	if usage.TotalFields == 0 {
		return 0.0
	}

	// Normalize score: assume max 4-5 metadata fields for most roles
	maxExpectedFields := 5.0
	normalizedScore := float64(usage.TotalFields) / maxExpectedFields
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	}
	return normalizedScore
}

func countMetadataFields(usage MetadataUsage) int {
	return usage.TotalFields
}
