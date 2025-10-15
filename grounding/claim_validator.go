// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"strings"
)

// ValidateMetadataClaims validates metadata claims against the reference index
// Returns updated citations with IsAccurate field populated
func ValidateMetadataClaims(citations []Citation, refIndex *ReferenceIndex) []Citation {
	result := make([]Citation, len(citations))
	copy(result, citations)

	for i, citation := range result {
		if len(citation.MetadataClaims) == 0 {
			continue
		}

		var metadata RoleMetadata
		switch citation.Type {
		case CitationJiraTicket:
			if ref := refIndex.JiraTickets[citation.Value]; ref != nil {
				metadata = ref.Metadata
			}
		case CitationGitHub:
			if ref := refIndex.GitHubIssues[citation.Value]; ref != nil {
				metadata = ref.Metadata
			}
		}

		if metadata == nil {
			// Entity not in reference index or no metadata, can't validate
			continue
		}

		for j, claim := range result[i].MetadataClaims {
			result[i].MetadataClaims[j] = validateClaim(claim, metadata)
		}
	}

	return result
}

// validateClaim validates a single metadata claim against the reference metadata
func validateClaim(claim MetadataClaim, metadata RoleMetadata) MetadataClaim {
	claim.ActualValue = ""
	claim.IsAccurate = false

	actualValues := metadata.GetFieldValue(claim.Field)
	if len(actualValues) == 0 {
		// Field not present in actual metadata
		return claim
	}

	claim.ActualValue = strings.Join(actualValues, ", ")

	claimedLower := strings.ToLower(claim.ClaimedValue)
	for _, actual := range actualValues {
		if strings.ToLower(actual) == claimedLower {
			claim.IsAccurate = true
			break
		}
	}

	return claim
}
