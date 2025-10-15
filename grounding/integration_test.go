// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package grounding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroundingValidation_CompleteFlow_WithToolResults(t *testing.T) {
	response := `
Based on the search results, here are the findings:
- MM-12345 addresses authentication
- See https://www.google.com for search
`

	// Build reference index with Jira ticket
	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"},
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	citations := ExtractCitations(response)
	assert.Equal(t, 2, len(citations), "Should extract Jira + URL")

	citations = ValidateCitations(citations, refIndex)

	citations = ValidateURLAccessibility(citations)

	jiraValid := false
	urlValidated := false
	for _, cit := range citations {
		if cit.Type == CitationJiraTicket && cit.ValidationStatus == ValidationGrounded {
			jiraValid = true
		}
		if cit.Type == CitationURL && cit.ValidationStatus == ValidationUngroundedValid {
			urlValidated = true
		}
	}

	assert.True(t, jiraValid, "Jira citation should be grounded")
	assert.True(t, urlValidated, "URL should be validated as accessible")
}

func TestGroundingValidation_CompleteFlow_NoToolResults(t *testing.T) {
	response := `
Here's what you should know:
- See https://www.google.com for search
`

	citations := ExtractCitations(response)
	assert.Equal(t, 1, len(citations), "Should extract 1 URL citation")

	citations = ValidateURLAccessibility(citations)

	assert.Equal(t, ValidationUngroundedValid, citations[0].ValidationStatus, "URL should be validated as accessible")
	assert.True(t, citations[0].HTTPStatusCode >= 200 && citations[0].HTTPStatusCode < 400, "Should have success status code")
}
