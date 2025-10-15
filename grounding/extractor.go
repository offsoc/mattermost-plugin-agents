// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"regexp"
	"strings"
)

var (
	jiraTicketRegex     = regexp.MustCompile(`(?i)\b(MM-\d+)\b`)
	githubRegex         = regexp.MustCompile(`(?i)\b([a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+#\d+)\b`)
	urlRegex            = regexp.MustCompile(`https?://[^\s\)]+`)
	productboardRegex   = regexp.MustCompile(`(?i)productboard://[a-zA-Z0-9_-]+`)
	zendeskRegex        = regexp.MustCompile(`(?i)zendesk://\d+`)
	uservoiceRegex      = regexp.MustCompile(`(?i)uservoice://\d+`)
	mattermostJiraRegex = regexp.MustCompile(`(?i)https?://mattermost\.atlassian\.net/browse/(MM-\d+)`)
)

// ExtractCitations extracts all citations from response text
func ExtractCitations(response string) []Citation {
	var citations []Citation
	lines := strings.Split(response, "\n")

	for lineNum, line := range lines {
		for _, match := range jiraTicketRegex.FindAllString(line, -1) {
			citations = append(citations, Citation{
				Type:       CitationJiraTicket,
				Value:      strings.ToUpper(match),
				LineNumber: lineNum + 1,
				Context:    truncateLine(line, 100),
			})
		}

		for _, match := range githubRegex.FindAllString(line, -1) {
			citations = append(citations, Citation{
				Type:       CitationGitHub,
				Value:      match,
				LineNumber: lineNum + 1,
				Context:    truncateLine(line, 100),
			})
		}

		for _, match := range urlRegex.FindAllString(line, -1) {
			if !strings.Contains(match, "github.com") && !strings.Contains(match, "productboard") {
				citations = append(citations, Citation{
					Type:       CitationURL,
					Value:      match,
					LineNumber: lineNum + 1,
					Context:    truncateLine(line, 100),
				})
			}
		}

		for _, match := range productboardRegex.FindAllString(line, -1) {
			citations = append(citations, Citation{
				Type:       CitationProductBoard,
				Value:      match,
				LineNumber: lineNum + 1,
				Context:    truncateLine(line, 100),
			})
		}

		for _, match := range zendeskRegex.FindAllString(line, -1) {
			citations = append(citations, Citation{
				Type:       CitationZendesk,
				Value:      match,
				LineNumber: lineNum + 1,
				Context:    truncateLine(line, 100),
			})
		}

		for _, match := range uservoiceRegex.FindAllString(line, -1) {
			citations = append(citations, Citation{
				Type:       CitationUserVoice,
				Value:      match,
				LineNumber: lineNum + 1,
				Context:    truncateLine(line, 100),
			})
		}

		// Metadata citations are no longer extracted here - they're generated from claims
	}

	return deduplicateCitations(citations)
}

func deduplicateCitations(citations []Citation) []Citation {
	seen := make(map[string]int)
	jiraTickets := make(map[string]bool)
	deduplicated := []Citation{}

	for _, citation := range citations {
		if citation.Type == CitationJiraTicket {
			jiraTickets[citation.Value] = true
		}

		key := string(citation.Type) + ":" + citation.Value
		if citation.Type == CitationMetadata {
			key = string(citation.Type) + ":" + citation.Context
		}

		if _, exists := seen[key]; !exists {
			seen[key] = len(deduplicated)
			deduplicated = append(deduplicated, citation)
		}
	}

	filtered := []Citation{}
	for _, citation := range deduplicated {
		if citation.Type == CitationURL {
			if ticketID := extractMattermostJiraTicketFromURL(citation.Value); ticketID != "" {
				if jiraTickets[ticketID] {
					continue
				}
			}
		}
		filtered = append(filtered, citation)
	}

	return filtered
}

func extractMattermostJiraTicketFromURL(urlStr string) string {
	matches := mattermostJiraRegex.FindStringSubmatch(urlStr)
	if len(matches) >= 2 {
		return strings.ToUpper(matches[1])
	}
	return ""
}

// AnalyzeMetadataUsage analyzes structured metadata field usage from citations
func AnalyzeMetadataUsage(citations []Citation) MetadataUsage {
	fieldCounts := make(map[string]int)

	for _, citation := range citations {
		for _, claim := range citation.MetadataClaims {
			fieldCounts[claim.Field]++
		}
	}

	return MetadataUsage{
		FieldCounts: fieldCounts,
		TotalFields: len(fieldCounts),
	}
}

func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen] + "..."
}
