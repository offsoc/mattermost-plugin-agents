// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// extractHTTPMetadata extracts all metadata from HTTP documentation pages
// Note: Named with "New" suffix to avoid conflict with existing function during migration
func extractHTTPMetadata(title, content, url string) EntityMetadata {
	textParts := []string{title, content}
	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
		Priority:    pm.EstimatePriority(title+" "+content, "documentation"),
	}

	meta := EntityMetadata{
		EntityType:   "http",
		EntityID:     url,
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
