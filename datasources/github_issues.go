// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (g *GitHubProtocol) fetchRecentIssues(ctx context.Context, owner, repo, topic string, limit int, sourceName string) []Doc {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []Doc{}
	}

	query := req.URL.Query()
	query.Add(QueryParamState, QueryParamAll)
	query.Add(QueryParamSort, "updated")
	query.Add("direction", "desc")
	query.Add(QueryParamPerPage, fmt.Sprintf("%d", limit))
	req.URL.RawQuery = query.Encode()

	g.addAuthHeaders(req)

	if waitErr := WaitRateLimiter(ctx, g.rateLimiter); waitErr != nil {
		return []Doc{}
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return []Doc{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return []Doc{}
	}

	var issues []GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return []Doc{}
	}

	var docs []Doc
	relevantIssues := 0
	skippedPRs := 0
	for _, issue := range issues {
		if isPullRequest(issue) {
			skippedPRs++
			continue
		}

		topicRelevant := g.isRelevantToTopic(issue.Title, issue.Body, topic)
		repoRelevant := g.topicAnalyzer.IsTopicRelevantContent(issue.Title+" "+issue.Body, topic)

		if !topicRelevant || !repoRelevant {
			continue
		}
		relevantIssues++

		comments := g.fetchIssueComments(ctx, owner, repo, issue.Number, min(3, 10))

		meta := extractGitHubMetadata(owner, repo, issue, comments)

		content := g.formatGitHubIssueWithMetadata(owner, repo, issue, comments, meta)
		if !g.universalScorer.IsUniversallyAcceptable(content, issue.Title, sourceName, topic) {
			continue
		}

		labels := buildLabelsFromMetadata(meta)

		labels = append(labels, fmt.Sprintf("issue:%d", issue.Number))
		labels = append(labels, fmt.Sprintf("state:%s", issue.State))

		for _, label := range issue.Labels {
			labels = append(labels, fmt.Sprintf("gh:%s", label.Name))
		}

		daysCreated := DaysSince(issue.CreatedAt)
		if recencyLabel := FormatRecencyLabel(daysCreated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_created")
		}
		daysUpdated := DaysSince(issue.UpdatedAt)
		if recencyLabel := FormatRecencyLabel(daysUpdated); recencyLabel != "" {
			labels = append(labels, recencyLabel+"_updated")
		}

		enhancedTitle := fmt.Sprintf("[%s/%s#%d] %s", owner, repo, issue.Number, issue.Title)

		doc := Doc{
			Title:        enhancedTitle,
			Content:      content,
			URL:          issue.HTMLURL,
			Section:      "issues",
			Source:       sourceName,
			Labels:       labels,
			Author:       issue.User.Login,
			CreatedDate:  issue.CreatedAt,
			LastModified: issue.UpdatedAt,
		}
		docs = append(docs, doc)
	}

	return docs
}

func (g *GitHubProtocol) fetchIssueComments(ctx context.Context, owner, repo string, number, limit int) []GitHubIssueComment {
	if limit <= 0 {
		return nil
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments?per_page=%d", owner, repo, number, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	g.addAuthHeaders(req)
	if waitErr := WaitRateLimiter(ctx, g.rateLimiter); waitErr != nil {
		return nil
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var comments []GitHubIssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil
	}
	return comments
}

// formatGitHubIssueWithMetadata formats GitHub issue with inline metadata
func (g *GitHubProtocol) formatGitHubIssueWithMetadata(owner, repo string, issue GitHubIssue, comments []GitHubIssueComment, meta EntityMetadata) string {
	var builder strings.Builder

	// Title with inline metadata: **[owner/repo#123] Title** (Priority: high | Segments: enterprise)
	builder.WriteString(fmt.Sprintf("**[%s/%s#%d] %s**", owner, repo, issue.Number, issue.Title))

	metadataStr := formatEntityMetadata(meta)
	if metadataStr != "" {
		builder.WriteString(" ")
		builder.WriteString(metadataStr)
	}
	builder.WriteString("\n\n")

	// Issue body - full content, not truncated
	if issue.Body != "" {
		builder.WriteString("**Description:**\n")
		builder.WriteString(issue.Body)
		builder.WriteString("\n\n")
	}

	// Details in compact format
	builder.WriteString(fmt.Sprintf("**Details:** State: %s | Author: %s", issue.State, issue.User.Login))

	if len(issue.Labels) > 0 {
		var labelNames []string
		for _, label := range issue.Labels {
			labelNames = append(labelNames, label.Name)
		}
		builder.WriteString(fmt.Sprintf(" | Labels: %s", strings.Join(labelNames, ", ")))
	}
	builder.WriteString("\n\n")

	// Top comments - keep full content
	if len(comments) > 0 {
		builder.WriteString("**Recent Comments:**\n")
		for i, c := range comments {
			if i >= 3 {
				break
			}
			builder.WriteString(fmt.Sprintf("- %s: %s\n", c.User.Login, c.Body))
		}
		builder.WriteString("\n")
	}

	detailedMeta := formatEntityMetadataDetailed(meta)
	if detailedMeta != "" {
		builder.WriteString(detailedMeta)
	}

	return builder.String()
}
