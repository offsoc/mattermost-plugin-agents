// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"fmt"
	"regexp"
	"strings"
)

// ClaimExtractor extracts metadata claims from LLM responses using role-specific patterns
type ClaimExtractor struct {
	patterns ExtractionPatterns
}

// NewClaimExtractor creates a claim extractor for a specific role
func NewClaimExtractor(roleMetadata RoleMetadata) *ClaimExtractor {
	if roleMetadata == nil {
		return &ClaimExtractor{patterns: ExtractionPatterns{}}
	}
	return &ClaimExtractor{
		patterns: roleMetadata.GetExtractionPatterns(),
	}
}

// ExtractMetadataClaims extracts metadata claims from LLM response
// Handles patterns like:
// - "MM-12345 (Priority: high | Segments: enterprise)"
// - "Based on #19234, which is high priority"
// - "The enterprise customer issue MM-12345"
func (e *ClaimExtractor) ExtractMetadataClaims(response string, citations []Citation) []Citation {
	result := make([]Citation, len(citations))
	copy(result, citations)

	result = e.extractInlineMetadata(response, result)
	result = e.extractValuePatterns(response, result)

	return result
}

// extractInlineMetadata extracts metadata from inline patterns like "MM-12345 (Priority: high | Segments: enterprise)"
func (e *ClaimExtractor) extractInlineMetadata(response string, citations []Citation) []Citation {
	if e.patterns.InlineFieldPattern == "" {
		return citations
	}

	// Pattern: "MM-12345 (metadata...)" or "#123 (metadata...)"
	citationPattern := regexp.MustCompile(`(?i)(MM-\d+|#\d+|[a-zA-Z0-9-]+/[a-zA-Z0-9-]+#\d+)\s*\(\s*([^)]+)\)`)

	matches := citationPattern.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		entityRef := match[1]
		metadataStr := match[2]

		citationIdx := findCitationByValue(citations, entityRef)
		if citationIdx == -1 {
			continue
		}

		claims := e.parseMetadataString(metadataStr)
		citations[citationIdx].MetadataClaims = append(citations[citationIdx].MetadataClaims, claims...)
	}

	return citations
}

// parseMetadataString parses inline metadata like "Priority: high | Segments: enterprise, smb"
func (e *ClaimExtractor) parseMetadataString(metadataStr string) []MetadataClaim {
	var claims []MetadataClaim

	if e.patterns.InlineFieldPattern == "" {
		return claims
	}

	// Split by |
	parts := regexp.MustCompile(`\s*[|]\s*`).Split(metadataStr, -1)

	fieldValuePattern := regexp.MustCompile(e.patterns.InlineFieldPattern)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		match := fieldValuePattern.FindStringSubmatch(part)
		if len(match) < 3 {
			continue
		}

		field := strings.ToLower(match[1])
		value := strings.TrimSpace(match[2])

		if canonical, ok := e.patterns.FieldAliases[field]; ok {
			field = canonical
		}

		values := strings.Split(value, ",")
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v != "" {
				claims = append(claims, MetadataClaim{
					Field:        field,
					ClaimedValue: strings.ToLower(v),
				})
			}
		}
	}

	return claims
}

// extractValuePatterns extracts claims using field-specific value patterns
// Example: "MM-12345 is high priority" using priority pattern "(high|medium|low)"
func (e *ClaimExtractor) extractValuePatterns(response string, citations []Citation) []Citation {
	for fieldName, valuePattern := range e.patterns.ValuePatterns {
		// Pattern: "MM-12345 is <value> <fieldName>"
		// Example: "MM-12345 is high priority"
		patternStr := fmt.Sprintf(`(?i)(MM-\d+|#\d+|[a-zA-Z0-9-]+/[a-zA-Z0-9-]+#\d+)\s+(?:is|has|shows?)\s+%s\s+%s`, valuePattern, fieldName)
		pattern := regexp.MustCompile(patternStr)

		matches := pattern.FindAllStringSubmatch(response, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			entityRef := match[1]
			value := strings.ToLower(match[2])

			citationIdx := findCitationByValue(citations, entityRef)
			if citationIdx == -1 {
				continue
			}

			hasClaimForField := false
			for _, claim := range citations[citationIdx].MetadataClaims {
				if claim.Field == fieldName {
					hasClaimForField = true
					break
				}
			}

			if !hasClaimForField {
				citations[citationIdx].MetadataClaims = append(citations[citationIdx].MetadataClaims, MetadataClaim{
					Field:        fieldName,
					ClaimedValue: value,
				})
			}
		}

		// Pattern: "<value> <fieldName> <entity>"
		// Example: "enterprise customer MM-12345" or "high priority issue #123"
		reversePatternStr := fmt.Sprintf(`(?i)(?:%s)\s+(?:issue|ticket|customer|item)\s+(MM-\d+|#\d+)`, valuePattern)
		reversePattern := regexp.MustCompile(reversePatternStr)

		reverseMatches := reversePattern.FindAllStringSubmatch(response, -1)
		for _, match := range reverseMatches {
			if len(match) < 3 {
				continue
			}

			value := strings.ToLower(match[1])
			entityRef := match[2]

			citationIdx := findCitationByValue(citations, entityRef)
			if citationIdx == -1 {
				continue
			}

			hasMatchingClaim := false
			for _, claim := range citations[citationIdx].MetadataClaims {
				if claim.Field == fieldName && strings.Contains(strings.ToLower(claim.ClaimedValue), value) {
					hasMatchingClaim = true
					break
				}
			}

			if !hasMatchingClaim {
				citations[citationIdx].MetadataClaims = append(citations[citationIdx].MetadataClaims, MetadataClaim{
					Field:        fieldName,
					ClaimedValue: value,
				})
			}
		}
	}

	return citations
}

// findCitationByValue finds a citation by its value (handles case-insensitive and short forms)
func findCitationByValue(citations []Citation, value string) int {
	valueLower := strings.ToLower(value)

	for i, citation := range citations {
		citationLower := strings.ToLower(citation.Value)

		// Exact match
		if citationLower == valueLower {
			return i
		}

		// Short form match: "#123" matches "owner/repo#123"
		if strings.HasPrefix(valueLower, "#") && strings.HasSuffix(citationLower, valueLower) {
			return i
		}

		// Reverse: "owner/repo#123" matches "#123"
		if strings.HasPrefix(citationLower, "#") && strings.HasSuffix(valueLower, citationLower) {
			return i
		}
	}

	return -1
}
