// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"

	jira "github.com/andygrunwald/go-jira"
)

// extractJiraMetadata extracts all metadata from Jira issues
// Note: Named with "New" suffix to avoid conflict with existing function during migration
func extractJiraMetadata(issue jira.Issue) EntityMetadata {
	if issue.Fields == nil {
		return EntityMetadata{}
	}

	var textParts []string
	textParts = append(textParts, issue.Fields.Summary, issue.Fields.Description)
	textParts = append(textParts, issue.Fields.Labels...)

	if issue.Fields.Comments != nil {
		for _, comment := range issue.Fields.Comments.Comments {
			textParts = append(textParts, comment.Body)
		}
	}

	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
	}

	// Priority from Jira fields
	if issue.Fields.Priority != nil {
		switch strings.ToLower(issue.Fields.Priority.Name) {
		case "highest", "blocker", "critical":
			pmMeta.Priority = pm.PriorityHigh
		case "high", "major":
			pmMeta.Priority = pm.PriorityHigh
		case "medium", "normal":
			pmMeta.Priority = pm.PriorityMedium
		case "low", "minor", "trivial":
			pmMeta.Priority = pm.PriorityLow
		}
	}
	if pmMeta.Priority == "" {
		statusName := ""
		if issue.Fields.Status != nil {
			statusName = issue.Fields.Status.Name
		}
		pmMeta.Priority = pm.EstimatePriority(issue.Fields.Summary+" "+issue.Fields.Description, statusName)
	}

	meta := EntityMetadata{
		EntityType:   EntityTypeJira,
		EntityID:     issue.Key,
		RoleMetadata: pmMeta,
	}

	// Entity linking (role-agnostic)
	meta.GitHubIssues, meta.GitHubPRs = extractGitHubReferences(allText)
	meta.JiraTickets = extractJiraReferences(allText)
	meta.Commits = extractCommitReferences(allText)
	meta.ModifiedFiles = extractFileReferences(allText)
	meta.ConfluencePages = extractConfluenceReferences(allText)
	meta.MattermostLinks = extractMattermostReferences(allText)

	return meta
}
