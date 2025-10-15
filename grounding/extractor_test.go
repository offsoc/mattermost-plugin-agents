// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCitations_JiraTickets(t *testing.T) {
	response := "Based on MM-12345 and MM-67890, we found issues."
	citations := ExtractCitations(response)

	assert.Len(t, citations, 2)
	assert.Equal(t, CitationJiraTicket, citations[0].Type)
	assert.Equal(t, "MM-12345", citations[0].Value)
	assert.Equal(t, CitationJiraTicket, citations[1].Type)
	assert.Equal(t, "MM-67890", citations[1].Value)
}

func TestExtractCitations_JiraCaseInsensitive(t *testing.T) {
	response := "Found mm-12345 and Mm-67890"
	citations := ExtractCitations(response)

	assert.Len(t, citations, 2)
	assert.Equal(t, "MM-12345", citations[0].Value)
	assert.Equal(t, "MM-67890", citations[1].Value)
}

func TestExtractCitations_GitHub(t *testing.T) {
	response := "See mattermost/mattermost#12345 for details"
	citations := ExtractCitations(response)

	assert.Len(t, citations, 1)
	assert.Equal(t, CitationGitHub, citations[0].Type)
	assert.Equal(t, "mattermost/mattermost#12345", citations[0].Value)
}

func TestExtractCitations_URLs(t *testing.T) {
	response := "Check https://docs.mattermost.com/install and http://example.com"
	citations := ExtractCitations(response)

	assert.Len(t, citations, 2)
	assert.Equal(t, CitationURL, citations[0].Type)
	assert.Contains(t, citations[0].Value, "docs.mattermost.com")
	assert.Equal(t, CitationURL, citations[1].Type)
	assert.Contains(t, citations[1].Value, "example.com")
}

func TestExtractCitations_Metadata(t *testing.T) {
	// Metadata citations are no longer extracted by ExtractCitations
	// They are generated from metadata claims after claim extraction
	response := "Priority: high, Segments: enterprise, Categories: performance"
	citations := ExtractCitations(response)

	// Should not extract metadata as citations
	for _, c := range citations {
		assert.NotEqual(t, CitationMetadata, c.Type, "Metadata should not be extracted as citations")
	}
}

func TestExtractCitations_Empty(t *testing.T) {
	response := ""
	citations := ExtractCitations(response)

	assert.Empty(t, citations)
}

func TestExtractCitations_NoCitations(t *testing.T) {
	response := "This response has no citations at all."
	citations := ExtractCitations(response)

	assert.Empty(t, citations)
}

func TestExtractCitations_MultiplePerLine(t *testing.T) {
	response := "MM-12345 and MM-67890 and mattermost/mattermost#123"
	citations := ExtractCitations(response)

	assert.Len(t, citations, 3)
}

func TestAnalyzeMetadataUsage(t *testing.T) {
	// AnalyzeMetadataUsage now takes citations with metadata claims
	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
			MetadataClaims: []MetadataClaim{
				{Field: "priority", ClaimedValue: "high"},
				{Field: "segments", ClaimedValue: "enterprise"},
				{Field: "segments", ClaimedValue: "federal"},
				{Field: "categories", ClaimedValue: "performance"},
				{Field: "competitive", ClaimedValue: "slack"},
			},
		},
	}

	usage := AnalyzeMetadataUsage(citations)

	assert.Equal(t, 4, usage.TotalFields, "Should have 4 unique field types")
	assert.Equal(t, 1, usage.FieldCounts["priority"])
	assert.Equal(t, 2, usage.FieldCounts["segments"], "Two segment claims")
	assert.Equal(t, 1, usage.FieldCounts["categories"])
	assert.Equal(t, 1, usage.FieldCounts["competitive"])
}

func TestAnalyzeMetadataUsage_NoMetadata(t *testing.T) {
	citations := []Citation{
		{
			Type:  CitationJiraTicket,
			Value: "MM-12345",
		},
	}

	usage := AnalyzeMetadataUsage(citations)

	assert.Equal(t, 0, usage.TotalFields, "Should have no metadata fields")
	assert.Empty(t, usage.FieldCounts, "FieldCounts should be empty")
}

func TestExtractCitations_Deduplication(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectedCount int
		expectedType  CitationType
		expectedValue string
	}{
		{
			name:          "duplicate Jira tickets",
			response:      "MM-12345 is mentioned. Later MM-12345 appears again. And MM-12345 one more time.",
			expectedCount: 1,
			expectedType:  CitationJiraTicket,
			expectedValue: "MM-12345",
		},
		{
			name:          "duplicate GitHub references",
			response:      "See mattermost/mattermost#123 for details. Also check mattermost/mattermost#123 again.",
			expectedCount: 1,
			expectedType:  CitationGitHub,
			expectedValue: "mattermost/mattermost#123",
		},
		{
			name:          "duplicate URLs",
			response:      "Visit https://example.com for info. Link: https://example.com again.",
			expectedCount: 1,
			expectedType:  CitationURL,
			expectedValue: "https://example.com",
		},
		{
			name:          "mixed duplicates",
			response:      "MM-12345 and MM-67890 and MM-12345 again",
			expectedCount: 2,
			expectedType:  CitationJiraTicket,
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			citations := ExtractCitations(tt.response)

			typedCitations := []Citation{}
			for _, c := range citations {
				if c.Type == tt.expectedType {
					typedCitations = append(typedCitations, c)
				}
			}
			assert.Len(t, typedCitations, tt.expectedCount)
			if tt.expectedValue != "" && len(typedCitations) > 0 {
				assert.Equal(t, tt.expectedValue, typedCitations[0].Value)
			}
		})
	}
}

func TestExtractCitations_DeduplicationPreservesFirstOccurrence(t *testing.T) {
	response := `Line 1: MM-12345 first mention
Line 2: some text
Line 3: MM-12345 second mention`

	citations := ExtractCitations(response)

	jiraCitations := []Citation{}
	for _, c := range citations {
		if c.Type == CitationJiraTicket {
			jiraCitations = append(jiraCitations, c)
		}
	}

	assert.Len(t, jiraCitations, 1, "should only have one citation after deduplication")
	assert.Equal(t, 1, jiraCitations[0].LineNumber, "should preserve first occurrence line number")
	assert.Contains(t, jiraCitations[0].Context, "first mention", "should preserve first occurrence context")
}

func TestExtractCitations_JiraTicketAndURLDeduplication(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectedCount int
		expectedTypes []CitationType
		description   string
	}{
		{
			name:          "Jira ticket and URL should deduplicate",
			response:      "Based on MM-65759 and https://mattermost.atlassian.net/browse/MM-65759, we found issues.",
			expectedCount: 1,
			expectedTypes: []CitationType{CitationJiraTicket},
			description:   "URL should be removed when Jira ticket exists",
		},
		{
			name:          "Multiple Jira tickets with URLs",
			response:      "See MM-65759 (https://mattermost.atlassian.net/browse/MM-65759) and MM-65761 (https://mattermost.atlassian.net/browse/MM-65761)",
			expectedCount: 2,
			expectedTypes: []CitationType{CitationJiraTicket, CitationJiraTicket},
			description:   "Two Jira tickets, URLs should be removed",
		},
		{
			name:          "Mattermost Jira URL extracts ticket and removes URL",
			response:      "Check https://mattermost.atlassian.net/browse/MM-65759 for details",
			expectedCount: 1,
			expectedTypes: []CitationType{CitationJiraTicket},
			description:   "Ticket ID should be extracted from Mattermost Jira URL, URL removed",
		},
		{
			name:          "Mixed Jira and non-Jira URLs",
			response:      "See MM-65759, https://mattermost.atlassian.net/browse/MM-65759, and https://docs.mattermost.com",
			expectedCount: 2,
			expectedTypes: []CitationType{CitationJiraTicket, CitationURL},
			description:   "Only Jira URL should be removed, docs URL remains",
		},
		{
			name:          "Case insensitive Jira URL matching",
			response:      "Based on mm-65759 and https://mattermost.atlassian.net/browse/MM-65759",
			expectedCount: 1,
			expectedTypes: []CitationType{CitationJiraTicket},
			description:   "Case insensitive matching should work",
		},
		{
			name:          "Different Jira instances",
			response:      "Check MM-65759 and https://otherjira.atlassian.net/browse/MM-65759",
			expectedCount: 2,
			expectedTypes: []CitationType{CitationJiraTicket, CitationURL},
			description:   "Different Jira instances should be kept separate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			citations := ExtractCitations(tt.response)

			nonMetadataCitations := []Citation{}
			for _, c := range citations {
				if c.Type != CitationMetadata {
					nonMetadataCitations = append(nonMetadataCitations, c)
				}
			}

			assert.Len(t, nonMetadataCitations, tt.expectedCount, tt.description)

			if len(tt.expectedTypes) > 0 {
				for i, expectedType := range tt.expectedTypes {
					if i < len(nonMetadataCitations) {
						assert.Equal(t, expectedType, nonMetadataCitations[i].Type,
							"Citation %d should be type %s", i, expectedType)
					}
				}
			}
		})
	}
}

func TestExtractMattermostJiraTicketFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Standard Mattermost Jira URL",
			url:      "https://mattermost.atlassian.net/browse/MM-65759",
			expected: "MM-65759",
		},
		{
			name:     "HTTP Mattermost Jira URL",
			url:      "http://mattermost.atlassian.net/browse/MM-12345",
			expected: "MM-12345",
		},
		{
			name:     "Lowercase ticket in Mattermost URL",
			url:      "https://mattermost.atlassian.net/browse/mm-65759",
			expected: "MM-65759",
		},
		{
			name:     "Non-Jira URL",
			url:      "https://docs.mattermost.com/install",
			expected: "",
		},
		{
			name:     "GitHub URL",
			url:      "https://github.com/mattermost/mattermost/issues/12345",
			expected: "",
		},
		{
			name:     "Different Jira instance should NOT match",
			url:      "https://otherjira.atlassian.net/browse/MM-99999",
			expected: "",
		},
		{
			name:     "Short ticket number",
			url:      "https://mattermost.atlassian.net/browse/MM-1234",
			expected: "MM-1234",
		},
		{
			name:     "Long ticket number",
			url:      "https://mattermost.atlassian.net/browse/MM-123456",
			expected: "MM-123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMattermostJiraTicketFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
