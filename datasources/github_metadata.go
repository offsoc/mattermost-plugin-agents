// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// extractGitHubMetadata extracts all metadata from GitHub issues/PRs
// Note: Named with "New" suffix to avoid conflict with existing function during migration
func extractGitHubMetadata(owner, repo string, issue GitHubIssue, comments []GitHubIssueComment) EntityMetadata {
	var textParts []string
	textParts = append(textParts, issue.Title, issue.Body)

	for _, label := range issue.Labels {
		textParts = append(textParts, label.Name)
	}

	for _, comment := range comments {
		textParts = append(textParts, comment.Body)
	}

	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
		Priority:    pm.EstimatePriority(issue.Title+" "+issue.Body, issue.State),
	}

	meta := EntityMetadata{
		EntityType:   EntityTypeGitHub,
		EntityID:     string(rune(issue.Number)),
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
