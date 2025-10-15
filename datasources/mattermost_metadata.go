// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// extractMattermostMetadata extracts all metadata from Mattermost posts
func extractMattermostMetadata(post MattermostPost) EntityMetadata {
	allText := post.Message

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(allText),
		Categories:  pm.ExtractTechnicalCategories(allText),
		Competitive: pm.ExtractCompetitiveContext(allText),
		Priority:    inferMattermostPostPriority(post),
	}

	meta := EntityMetadata{
		EntityType:   "mattermost",
		EntityID:     post.ID,
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

// inferMattermostPostPriority determines priority based on post engagement
func inferMattermostPostPriority(post MattermostPost) pm.Priority {
	score := 0

	if post.IsPinned {
		score += 5
	}

	switch {
	case post.ReplyCount >= 20:
		score += 3
	case post.ReplyCount >= 10:
		score += 2
	case post.ReplyCount >= 5:
		score++
	}

	switch {
	case score >= 7:
		return pm.PriorityHigh
	case score >= 4:
		return pm.PriorityMedium
	default:
		return pm.PriorityLow
	}
}
