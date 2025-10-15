// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package evals

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/grounding"
)

// EvaluateGroundingWithLogging performs grounding validation with eval framework logging
func EvaluateGroundingWithLogging(
	evalT *EvalT,
	response string,
	toolResults []string,
	metadata []*datasources.EntityMetadata,
	apiClients *grounding.APIClients,
	thresholds grounding.Thresholds,
	logPrefix string,
) grounding.Result {
	evalT.Logf("%s: === GROUNDING VALIDATION ===", logPrefix)

	citations := grounding.ExtractCitations(response)
	wordCount := len(strings.Fields(response))
	evalT.Logf("%s: Response: %d words, %d citations", logPrefix, wordCount, len(citations))

	citationsByType := make(map[grounding.CitationType][]string)
	for _, citation := range citations {
		displayValue := citation.Value
		if citation.Type == grounding.CitationMetadata && citation.Context != "" {
			displayValue = citation.Context
		}
		citationsByType[citation.Type] = append(citationsByType[citation.Type], displayValue)
	}

	maxCitationsToShow := 3
	for citationType, values := range citationsByType {
		evalT.Logf("%s:   - %s: %d", logPrefix, citationType, len(values))

		if citationType == grounding.CitationMetadata {
			uniqueValues := make(map[string]bool)
			for _, v := range values {
				uniqueValues[v] = true
			}
			shown := 0
			for v := range uniqueValues {
				if shown >= maxCitationsToShow {
					remaining := len(uniqueValues) - maxCitationsToShow
					if remaining > 0 {
						evalT.Logf("%s:     ... and %d more unique metadata field(s)", logPrefix, remaining)
					}
					break
				}
				evalT.Logf("%s:     • %s", logPrefix, v)
				shown++
			}
		} else {
			for i, value := range values {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:     ... and %d more", logPrefix, len(values)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:     • %s", logPrefix, value)
			}
		}
	}

	// Build reference index if we have tool results or metadata
	var refIndex *grounding.ReferenceIndex
	if len(toolResults) > 0 || metadata != nil {
		evalT.Logf("%s: Validating against %d tool results", logPrefix, len(toolResults))

		refIndex = BuildReferenceIndex(toolResults, metadata)

		// Log index stats
		evalT.Logf("%s: Built reference index: %d GitHub, %d Jira, %d Confluence, %d Zendesk, %d URLs",
			logPrefix,
			len(refIndex.GitHubIssues),
			len(refIndex.JiraTickets),
			len(refIndex.ConfluencePages),
			len(refIndex.ZendeskTickets),
			len(refIndex.URLs))

		// Validate against reference index first
		citations = grounding.ValidateCitations(citations, refIndex)

		validCount := 0
		for _, citation := range citations {
			if citation.IsValid {
				validCount++
			}
		}
		if len(citations) > 0 {
			evalT.Logf("%s: Validation result: %d grounded / %d total (%.1f%%)",
				logPrefix, validCount, len(citations),
				float64(validCount)/float64(len(citations))*100)
		}
	} else {
		// No tool results - create empty reference index for API verification
		refIndex = &grounding.ReferenceIndex{
			GitHubIssues:    make(map[string]*grounding.GitHubRef),
			JiraTickets:     make(map[string]*grounding.JiraRef),
			ConfluencePages: make(map[string]*grounding.ConfluenceRef),
			ZendeskTickets:  make(map[string]*grounding.ZendeskRef),
			URLs:            make(map[string]bool),
		}
	}

	// Perform API verification for ungrounded citations (works even without tool results)
	if apiClients != nil {
		evalT.Logf("%s: API clients provided - will verify ungrounded citations via external APIs", logPrefix)
		ctx := context.Background()
		citations = grounding.ValidateCitationsWithAPI(ctx, citations, refIndex, apiClients)

		// Count API-verified citations
		apiVerifiedCount := 0
		for _, citation := range citations {
			if citation.VerifiedViaAPI {
				apiVerifiedCount++
			}
		}
		if apiVerifiedCount > 0 {
			evalT.Logf("%s: API verification completed: %d citations verified via external APIs", logPrefix, apiVerifiedCount)
		}
	}

	urlCitationCount := 0
	for _, citation := range citations {
		if citation.Type == grounding.CitationURL &&
			(citation.ValidationStatus == grounding.ValidationNotChecked || citation.ValidationStatus == "") {
			urlCitationCount++
		}
	}

	if urlCitationCount > 0 {
		evalT.Logf("%s: Checking accessibility of %d ungrounded URLs", logPrefix, urlCitationCount)
		citations = grounding.ValidateURLAccessibility(citations)

		ungroundedValid := 0
		ungroundedBroken := 0
		for _, citation := range citations {
			switch citation.ValidationStatus {
			case grounding.ValidationUngroundedValid:
				ungroundedValid++
			case grounding.ValidationUngroundedBroken:
				ungroundedBroken++
			}
		}
		evalT.Logf("%s: URL accessibility: %d accessible, %d broken",
			logPrefix, ungroundedValid, ungroundedBroken)
	}

	hasToolResults := len(toolResults) > 0 || metadata != nil
	result := grounding.CalculateGroundingScore(response, citations, thresholds, hasToolResults)

	// Log metadata usage (role-agnostic)
	metadataFieldCount := result.MetadataUsage.TotalFields
	fieldsSummary := ""
	if len(result.MetadataUsage.FieldCounts) > 0 {
		fields := []string{}
		for field, count := range result.MetadataUsage.FieldCounts {
			fields = append(fields, fmt.Sprintf("%s=%d", field, count))
		}
		fieldsSummary = fmt.Sprintf(" (%s)", strings.Join(fields, ", "))
	}
	evalT.Logf("%s: Metadata usage: %d unique fields%s [informative only, does not affect pass/fail]",
		logPrefix,
		metadataFieldCount,
		fieldsSummary)

	citationBreakdown := make(map[grounding.ValidationStatus][]string)
	metadataCitations := []string{}
	apiVerifiedCount := 0
	for _, citation := range citations {
		if citation.Type == grounding.CitationMetadata {
			displayValue := citation.Value
			if citation.Context != "" {
				displayValue = citation.Context
			}
			metadataCitations = append(metadataCitations, displayValue)
			continue
		}
		if citation.VerifiedViaAPI {
			apiVerifiedCount++
		}
		status := citation.ValidationStatus
		if status == "" {
			status = grounding.ValidationNotChecked
		}
		citationBreakdown[status] = append(citationBreakdown[status], citation.Value)
	}

	if apiVerifiedCount > 0 {
		evalT.Logf("%s: API verification: %d citations checked via external APIs", logPrefix, apiVerifiedCount)
	}

	hasNonMetadataCitations := len(citationBreakdown) > 0
	if hasNonMetadataCitations || len(metadataCitations) > 0 {
		evalT.Logf("%s: === VALIDATION RESULTS ===", logPrefix)
	}

	if len(metadataCitations) > 0 {
		if hasNonMetadataCitations {
			evalT.Logf("%s: Metadata citations (%d): Always considered grounded", logPrefix, len(metadataCitations))
		} else {
			uniqueMetadata := make(map[string]bool)
			for _, v := range metadataCitations {
				uniqueMetadata[v] = true
			}
			evalT.Logf("%s: All %d citations are metadata (always grounded): %d unique field(s)",
				logPrefix, len(metadataCitations), len(uniqueMetadata))
			shown := 0
			for v := range uniqueMetadata {
				if shown >= maxCitationsToShow {
					break
				}
				evalT.Logf("%s:   ✓ %s", logPrefix, v)
				shown++
			}
			if len(uniqueMetadata) > maxCitationsToShow {
				evalT.Logf("%s:   ... and %d more field(s)", logPrefix, len(uniqueMetadata)-maxCitationsToShow)
			}
		}
	}

	if len(citationBreakdown) > 0 {
		if vals, ok := citationBreakdown[grounding.ValidationGrounded]; ok {
			evalT.Logf("%s: Grounded citations (%d): Found in tool results", logPrefix, len(vals))
			for i, v := range vals {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:   ... and %d more", logPrefix, len(vals)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:   ✓ %s", logPrefix, v)
			}
		}
		if vals, ok := citationBreakdown[grounding.ValidationUngroundedValid]; ok {
			evalT.Logf("%s: Ungrounded but accessible (%d): Not in tool results, but URL verified", logPrefix, len(vals))
			for i, v := range vals {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:   ... and %d more", logPrefix, len(vals)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:   ⚠ %s", logPrefix, v)
			}
		}
		if vals, ok := citationBreakdown[grounding.ValidationUngroundedBroken]; ok {
			evalT.Logf("%s: Ungrounded and broken (%d): Not in tool results, URL inaccessible", logPrefix, len(vals))
			for i, v := range vals {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:   ... and %d more", logPrefix, len(vals)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:   ✗ %s", logPrefix, v)
			}
		}
		if vals, ok := citationBreakdown[grounding.ValidationFabricated]; ok {
			evalT.Logf("%s: Fabricated (%d): Does not exist in external system", logPrefix, len(vals))
			for i, v := range vals {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:   ... and %d more", logPrefix, len(vals)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:   ✗✗ %s", logPrefix, v)
			}
		}
		if vals, ok := citationBreakdown[grounding.ValidationAPIError]; ok {
			evalT.Logf("%s: API errors (%d): Could not verify via external API", logPrefix, len(vals))
			for i, v := range vals {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:   ... and %d more", logPrefix, len(vals)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:   ⚠ %s", logPrefix, v)
			}
		}
		if vals, ok := citationBreakdown[grounding.ValidationNotChecked]; ok {
			// Count by type for clearer reporting
			typeCount := make(map[grounding.CitationType]int)
			for _, citation := range citations {
				if citation.ValidationStatus == grounding.ValidationNotChecked || citation.ValidationStatus == "" {
					if citation.Type != grounding.CitationMetadata {
						typeCount[citation.Type]++
					}
				}
			}

			typeSummary := ""
			if len(typeCount) > 0 {
				parts := []string{}
				if count, ok := typeCount[grounding.CitationGitHub]; ok {
					parts = append(parts, fmt.Sprintf("%d GitHub", count))
				}
				if count, ok := typeCount[grounding.CitationJiraTicket]; ok {
					parts = append(parts, fmt.Sprintf("%d Jira", count))
				}
				if count, ok := typeCount[grounding.CitationURL]; ok {
					parts = append(parts, fmt.Sprintf("%d URL", count))
				}
				if count, ok := typeCount[grounding.CitationZendesk]; ok {
					parts = append(parts, fmt.Sprintf("%d Zendesk", count))
				}
				typeSummary = " (" + strings.Join(parts, ", ") + ")"
			}

			evalT.Logf("%s: Not checked (%d)%s: No tool results available to validate against", logPrefix, len(vals), typeSummary)
			for i, v := range vals {
				if i >= maxCitationsToShow {
					evalT.Logf("%s:   ... and %d more", logPrefix, len(vals)-maxCitationsToShow)
					break
				}
				evalT.Logf("%s:   ? %s", logPrefix, v)
			}
		}
	}

	if len(metadataCitations) == 0 && len(citationBreakdown) == 0 {
		evalT.Logf("%s: No citations found to validate", logPrefix)
	}

	evalT.Logf("%s: === SCORING ===", logPrefix)
	evalT.Logf("%s: Criteria: MinValidRate=%.1f%%, CitationWeight=%.2f, MetadataWeight=%.2f",
		logPrefix,
		thresholds.MinCitationRate*100,
		thresholds.CitationWeight,
		thresholds.MetadataWeight)

	nonMetadataCitations := result.TotalCitations - result.CitationsByType[grounding.CitationMetadata]
	hasValidatedCitations := result.ValidCitations > 0 || result.InvalidCitations > 0

	citationScore, metadataScore := calculateDisplayScores(result, thresholds, nonMetadataCitations, hasValidatedCitations)

	if hasValidatedCitations {
		// Log breakdown of how valid rate was calculated
		evalT.Logf("%s: Valid rate calculation: %d valid / %d total = %.1f%% (valid = grounded:%d + ungrounded-valid:%d, invalid = fabricated:%d + broken:%d + api-error:%d)",
			logPrefix,
			result.ValidCitations,
			result.ValidCitations+result.InvalidCitations,
			result.ValidCitationRate*100,
			result.GroundedCitations,
			result.UngroundedValidURLs,
			result.FabricatedCitations,
			result.UngroundedBrokenURLs,
			result.APIErrors)

		evalT.Logf("%s: Citation score: %.2f (count=%d, valid_rate=%.1f%%/%.1f%%)",
			logPrefix,
			citationScore,
			nonMetadataCitations,
			result.ValidCitationRate*100,
			thresholds.MinCitationRate*100)
	} else {
		evalT.Logf("%s: Citation score: %.2f (count=%d, not validated - no tool results or API verification)",
			logPrefix,
			citationScore,
			nonMetadataCitations)
	}

	metadataFieldsUsed := countMetadataFieldsFromUsage(result.MetadataUsage)
	evalT.Logf("%s: Metadata score: %.2f (fields=%d/5 max)",
		logPrefix,
		metadataScore,
		metadataFieldsUsed)
	evalT.Logf("%s: Overall score: %.2f = (%.2f × %.2f) + (%.2f × %.2f)",
		logPrefix,
		result.OverallScore,
		thresholds.CitationWeight,
		citationScore,
		thresholds.MetadataWeight,
		metadataScore)

	if result.Pass {
		evalT.Logf("%s: GROUNDING PASS - %s", logPrefix, result.Reasoning)
	} else {
		evalT.Logf("%s: GROUNDING FAIL - %s", logPrefix, result.Reasoning)
	}

	return result
}

// calculateDisplayScores computes citation and metadata scores for display logging
// These scores mirror the actual calculation in grounding.CalculateGroundingScore
func calculateDisplayScores(result grounding.Result, thresholds grounding.Thresholds, nonMetadataCitations int, hasValidatedCitations bool) (citationScore, metadataScore float64) {
	if nonMetadataCitations > 0 && hasValidatedCitations {
		citationScore = result.ValidCitationRate
	}

	metadataScore = float64(countMetadataFieldsFromUsage(result.MetadataUsage)) / 4.0
	return citationScore, metadataScore
}

// countMetadataFieldsFromUsage counts how many metadata fields are used
func countMetadataFieldsFromUsage(usage grounding.MetadataUsage) int {
	return usage.TotalFields
}

// EvaluateContentGrounding performs semantic content grounding validation with logging
// Validates if response content is semantically grounded in tool results
// This distinguishes baseline bots (no grounding) from enhanced bots (high grounding)
func EvaluateContentGrounding(
	evalT *EvalT,
	response string,
	toolResults []string,
	embedder grounding.ContentEmbedder,
	logPrefix string,
) *grounding.ContentGroundingResult {
	evalT.Logf("%s: === CONTENT GROUNDING VALIDATION ===", logPrefix)

	wordCount := len(strings.Fields(response))
	evalT.Logf("%s: Response: %d words, %d tool results", logPrefix, wordCount, len(toolResults))

	if len(toolResults) == 0 {
		evalT.Logf("%s: WARNING - No tool results provided, baseline bot expected to have low grounding", logPrefix)
	}

	// Use content grounding thresholds and options
	thresholds := grounding.DefaultContentGroundingThresholds()
	opts := grounding.DefaultContentGroundingOptions()

	ctx := context.Background()
	result, err := grounding.ValidateContentGrounding(
		ctx,
		response,
		toolResults,
		embedder,
		thresholds,
		opts,
	)

	if err != nil {
		evalT.Logf("%s: ERROR - Content grounding failed: %v", logPrefix, err)
		return nil
	}

	// Log results
	evalT.Logf("%s: === VALIDATION RESULTS ===", logPrefix)
	evalT.Logf("%s: Total claims: %d", logPrefix, result.TotalClaims)
	evalT.Logf("%s: Grounded: %d (%.1f%%)", logPrefix, result.GroundedCount,
		float64(result.GroundedCount)/float64(result.TotalClaims)*100)
	evalT.Logf("%s: Marginal: %d (%.1f%%)", logPrefix, result.MarginalCount,
		float64(result.MarginalCount)/float64(result.TotalClaims)*100)
	evalT.Logf("%s: Ungrounded: %d (%.1f%%)", logPrefix, result.UngroundedCount,
		float64(result.UngroundedCount)/float64(result.TotalClaims)*100)

	evalT.Logf("%s: === SCORING ===", logPrefix)
	evalT.Logf("%s: Grounding score: %.2f (%.1f%% grounded+marginal)", logPrefix,
		result.GroundingScore, result.GroundingScore*100)
	evalT.Logf("%s: Weighted score: %.2f", logPrefix, result.WeightedScore)

	if result.Pass {
		evalT.Logf("%s: CONTENT GROUNDING PASS - %s", logPrefix, result.Reasoning)
	} else {
		evalT.Logf("%s: CONTENT GROUNDING FAIL - %s", logPrefix, result.Reasoning)
	}

	return result
}

// ContentEmbedder interface for content grounding validation
// Defined here to avoid circular dependency with semantic package
type ContentEmbedder = grounding.ContentEmbedder
