// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
)

// extractDiscourseMetadata extracts all metadata from Discourse topics
// Note: Named with "New" suffix to avoid conflict with existing function during migration
func extractDiscourseMetadata(topic DiscourseTopic) EntityMetadata {
	var textParts []string
	textParts = append(textParts, topic.Title)
	textParts = append(textParts, topic.Tags...)

	allText := strings.Join(textParts, " ")

	pmMeta := pm.Metadata{
		Segments:    pm.ExtractCustomerSegments(textParts...),
		Categories:  pm.ExtractTechnicalCategories(textParts...),
		Competitive: pm.ExtractCompetitiveContext(textParts...),
		Priority:    inferDiscourseTopicPriority(topic),
	}

	meta := EntityMetadata{
		EntityType:   "discourse",
		EntityID:     fmt.Sprintf("%d", topic.ID),
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
