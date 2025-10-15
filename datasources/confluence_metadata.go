// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

func extractConfluenceMetadata(page ConfluenceContent, textContent string) EntityMetadata {
	var textParts []string
	var labelParts []string
	textParts = append(textParts, page.Title, textContent)
	for _, label := range page.Metadata.Labels.Results {
		textParts = append(textParts, label.Name)
		labelParts = append(labelParts, label.Name)
	}

	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:   pm.ExtractCustomerSegments(textParts...),
		Categories: pm.ExtractTechnicalCategories(textParts...),
	}

	// Check labels + title first (higher priority), then fall back to all text
	pmMeta.Competitive = pm.ExtractCompetitiveContext(append([]string{page.Title}, labelParts...)...)
	if pmMeta.Competitive == "" {
		pmMeta.Competitive = pm.ExtractCompetitiveContext(textParts...)
	}

	// Priority estimation based on content signals and space
	searchText := strings.ToLower(page.Title + " " + textContent)
	pmMeta.Priority = pm.EstimatePriority(searchText, page.Space.Key)

	meta := EntityMetadata{
		EntityType:   EntityTypeConfluence,
		EntityID:     page.ID,
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
