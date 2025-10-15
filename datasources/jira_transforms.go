// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-ai/chunking"
)

// convertIssueToDoc converts a Jira issue to a Doc, extracting summary, description,
// status, assignee, priority, type, and comments into structured metadata with recency labels
func (j *JiraProtocol) convertIssueToDoc(issue jira.Issue, sourceName string) *Doc {
	if issue.Fields == nil {
		return nil
	}

	meta := extractJiraMetadata(issue)

	// Format content with inline metadata in title
	content := j.formatIssueContentWithMetadata(issue, meta)

	// Log content details for debugging
	if j.pluginAPI != nil {
		descLen := 0
		if issue.Fields.Description != "" {
			descLen = len(issue.Fields.Description)
		}
		commentCount := 0
		if issue.Fields.Comments != nil && issue.Fields.Comments.Comments != nil {
			commentCount = len(issue.Fields.Comments.Comments)
		}
		j.pluginAPI.LogDebug("Jira issue transformed",
			"issue_key", issue.Key,
			"content_length", len(content),
			"description_length", descLen,
			"comment_count", commentCount)
	}

	if strings.TrimSpace(issue.Fields.Summary) == "" {
		contentLines := strings.Split(content, "\n")
		hasContent := false
		for _, line := range contentLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") &&
				!strings.HasPrefix(trimmed, "---") &&
				!strings.HasPrefix(trimmed, "- **") &&
				!strings.HasPrefix(trimmed, "**[") &&
				!strings.HasPrefix(trimmed, "(") &&
				!strings.HasPrefix(trimmed, "**Details:**") &&
				!strings.Contains(trimmed, issue.Key) {
				hasContent = true
				break
			}
		}
		if !hasContent {
			return nil
		}
	}

	section := "issue"
	if issue.Fields.Type.Name != "" {
		section = strings.ToLower(issue.Fields.Type.Name)
	}

	issueURL := ""
	if issue.Self != "" {
		lastRestIdx := strings.LastIndex(issue.Self, "/rest")
		if lastRestIdx > 0 {
			issueURL = fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(issue.Self[:lastRestIdx], "/"), issue.Key)
		}
	}
	if issueURL == "" {
		issueURL = fmt.Sprintf("jira://issue/%s", issue.Key)
	}

	labels := buildLabelsFromMetadata(meta)

	if issue.Fields.Status != nil {
		labels = append(labels, fmt.Sprintf("status:%s", issue.Fields.Status.Name))
	}
	if issue.Fields.Assignee != nil {
		labels = append(labels, fmt.Sprintf("assignee:%s", issue.Fields.Assignee.DisplayName))
	}
	if issue.Fields.Type.Name != "" {
		labels = append(labels, fmt.Sprintf("type:%s", issue.Fields.Type.Name))
	}

	for _, label := range issue.Fields.Labels {
		labels = append(labels, label)
		labels = append(labels, fmt.Sprintf("jira:%s", label))
	}

	daysCreated := DaysSince(j.formatTime(issue.Fields.Created))
	if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_created")
	}
	daysUpdated := DaysSince(j.formatTime(issue.Fields.Updated))
	if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
		labels = append(labels, recencyLabel+"_updated")
	}

	doc := &Doc{
		Title:        fmt.Sprintf("[%s] %s", issue.Key, issue.Fields.Summary),
		Content:      content,
		URL:          issueURL,
		Section:      section,
		Source:       sourceName,
		LastModified: j.formatTime(issue.Fields.Updated),
		CreatedDate:  j.formatTime(issue.Fields.Created),
		Labels:       labels,
	}

	if issue.Fields.Creator != nil {
		doc.Author = issue.Fields.Creator.DisplayName
	}

	return doc
}

// formatIssueContent creates comprehensive content from Jira issue
// formatIssueContentWithMetadata formats Jira issue with metadata inline in title
func (j *JiraProtocol) formatIssueContentWithMetadata(issue jira.Issue, meta EntityMetadata) string {
	var builder strings.Builder

	// Title with inline metadata: **[MM-12345] Title** (Priority: high | Segments: enterprise | Categories: mobile)
	builder.WriteString("**[")
	builder.WriteString(issue.Key)
	builder.WriteString("] ")
	builder.WriteString(issue.Fields.Summary)
	builder.WriteString("**")

	metadataStr := formatEntityMetadata(meta)
	if metadataStr != "" {
		builder.WriteString(" ")
		builder.WriteString(metadataStr)
	}
	builder.WriteString("\n\n")

	// Description section with full text (not truncated)
	if issue.Fields.Description != "" {
		builder.WriteString("**Description:**\n")
		cleanDescription := j.htmlProcessor.ExtractStructuredText(issue.Fields.Description)
		if strings.TrimSpace(cleanDescription) == "" {
			cleanDescription = issue.Fields.Description
		}
		builder.WriteString(cleanDescription)
		builder.WriteString("\n\n")
	}

	// Issue details in compact format
	builder.WriteString("**Details:** ")
	var details []string

	if issue.Fields.Type.Name != "" {
		details = append(details, fmt.Sprintf("Type: %s", issue.Fields.Type.Name))
	}
	if issue.Fields.Status != nil {
		details = append(details, fmt.Sprintf("Status: %s", issue.Fields.Status.Name))
	}
	if issue.Fields.Assignee != nil {
		details = append(details, fmt.Sprintf("Assignee: %s", issue.Fields.Assignee.DisplayName))
	}

	builder.WriteString(strings.Join(details, " | "))
	builder.WriteString("\n\n")

	detailedMeta := formatEntityMetadataDetailed(meta)
	if detailedMeta != "" {
		builder.WriteString(detailedMeta)
	}

	return builder.String()
}

// formatIssueContent is the legacy format (keeping for backwards compatibility)
func (j *JiraProtocol) formatIssueContent(issue jira.Issue) string {
	var builder strings.Builder

	builder.WriteString("# ")
	builder.WriteString(issue.Fields.Summary)
	builder.WriteString("\n\n")

	if issue.Fields.Description != "" {
		builder.WriteString("## Description\n")
		cleanDescription := j.htmlProcessor.ExtractStructuredText(issue.Fields.Description)
		if strings.TrimSpace(cleanDescription) == "" {
			cleanDescription = issue.Fields.Description
		}
		builder.WriteString(cleanDescription)
		builder.WriteString("\n\n")
	}

	builder.WriteString("## Issue Details\n")
	builder.WriteString(fmt.Sprintf("- **Issue Key**: %s\n", issue.Key))

	if issue.Fields.Type.Name != "" {
		builder.WriteString(fmt.Sprintf("- **Type**: %s\n", issue.Fields.Type.Name))
	}

	if issue.Fields.Status != nil {
		builder.WriteString(fmt.Sprintf("- **Status**: %s\n", issue.Fields.Status.Name))
	}

	if issue.Fields.Priority != nil {
		builder.WriteString(fmt.Sprintf("- **Priority**: %s\n", issue.Fields.Priority.Name))
	}

	if issue.Fields.Assignee != nil {
		builder.WriteString(fmt.Sprintf("- **Assignee**: %s\n", issue.Fields.Assignee.DisplayName))
	}

	if issue.Fields.Reporter != nil {
		builder.WriteString(fmt.Sprintf("- **Reporter**: %s\n", issue.Fields.Reporter.DisplayName))
	}

	if len(issue.Fields.Labels) > 0 {
		builder.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(issue.Fields.Labels, ", ")))
	}

	if issue.Fields.Comments != nil && len(issue.Fields.Comments.Comments) > 0 {
		builder.WriteString("\n## Recent Comments\n")
		commentCount := len(issue.Fields.Comments.Comments)
		maxComments := MaxJiraCommentsToInclude
		startIdx := 0
		if commentCount > maxComments {
			startIdx = commentCount - maxComments
		}

		for i := startIdx; i < commentCount; i++ {
			comment := issue.Fields.Comments.Comments[i]
			if comment.Author.DisplayName != "" {
				builder.WriteString(fmt.Sprintf("**%s**: ", comment.Author.DisplayName))
			}
			cleanComment := j.htmlProcessor.ExtractStructuredText(comment.Body)
			if strings.TrimSpace(cleanComment) == "" {
				cleanComment = comment.Body
			}
			builder.WriteString(cleanComment)
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

// formatTime formats Jira time to RFC3339
func (j *JiraProtocol) formatTime(jiraTime jira.Time) string {
	if time.Time(jiraTime).IsZero() {
		return ""
	}
	return time.Time(jiraTime).Format(time.RFC3339)
}

// applyRelevanceScoring filters documents based on relevance scoring
func (j *JiraProtocol) applyRelevanceScoring(docs []Doc, topic, sourceName string) []Doc {
	var filtered []Doc

	for _, doc := range docs {
		if j.universalScorer.IsUniversallyAcceptable(doc.Content, doc.Title, sourceName, topic) {
			chunks := chunking.ChunkText(doc.Content, chunking.DefaultOptions())
			if len(chunks) > 0 {
				chunkStrings := make([]string, len(chunks))
				for i, chunk := range chunks {
					chunkStrings[i] = chunk.Content
				}
				doc.Content = j.topicAnalyzer.SelectBestChunkWithContext(chunkStrings, topic)
			}
			filtered = append(filtered, doc)
		}
	}

	return filtered
}
