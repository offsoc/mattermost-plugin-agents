// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (g *GitHubProtocol) fetchRecentPRsWithFilters(ctx context.Context, owner, repo, topic string, limit int, sourceName string, filters *PRFilterOptions) []Doc {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []Doc{}
	}

	query := req.URL.Query()

	if filters != nil && filters.State != "" {
		query.Add(QueryParamState, filters.State)
	} else {
		query.Add(QueryParamState, QueryParamAll)
	}

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

	var pulls []GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&pulls); err != nil {
		return []Doc{}
	}

	var docs []Doc
	for _, pr := range pulls {
		if filters != nil {
			if filters.Since != nil || filters.Until != nil {
				updatedAt, err := time.Parse(time.RFC3339, pr.UpdatedAt)
				if err == nil {
					if filters.Since != nil && updatedAt.Before(*filters.Since) {
						continue
					}
					if filters.Until != nil && updatedAt.After(*filters.Until) {
						continue
					}
				}
			}
			if filters.Author != "" && pr.User.Login != filters.Author {
				continue
			}
			if len(filters.Labels) > 0 {
				hasAllLabels := true
				for _, requiredLabel := range filters.Labels {
					found := false
					for _, prLabel := range pr.Labels {
						if prLabel.Name == requiredLabel {
							found = true
							break
						}
					}
					if !found {
						hasAllLabels = false
						break
					}
				}
				if !hasAllLabels {
					continue
				}
			}
		}

		if !g.isRelevantToTopic(pr.Title, pr.Body, topic) || !g.topicAnalyzer.IsTopicRelevantContent(pr.Title+" "+pr.Body, topic) {
			continue
		}

		comments := g.fetchIssueComments(ctx, owner, repo, pr.Number, min(3, 10))

		meta := extractGitHubMetadata(owner, repo, pr, comments)

		content := g.formatGitHubPRWithMetadata(owner, repo, pr, comments, meta)
		if !g.universalScorer.IsUniversallyAcceptable(content, pr.Title, sourceName, topic) {
			continue
		}

		labels := buildLabelsFromMetadata(meta)

		doc := Doc{
			Title:        pr.Title,
			Content:      content,
			URL:          pr.HTMLURL,
			Section:      "pulls",
			Source:       sourceName,
			Labels:       labels,
			Author:       pr.User.Login,
			LastModified: pr.UpdatedAt,
		}
		docs = append(docs, doc)
	}

	return docs
}

func (g *GitHubProtocol) fetchRecentPRs(ctx context.Context, owner, repo, topic string, limit int, sourceName string) []Doc {
	return g.fetchRecentPRsWithFilters(ctx, owner, repo, topic, limit, sourceName, nil)
}

// formatGitHubPRWithMetadata formats GitHub PR with inline metadata
func (g *GitHubProtocol) formatGitHubPRWithMetadata(owner, repo string, pr GitHubIssue, comments []GitHubIssueComment, meta EntityMetadata) string {
	var builder strings.Builder

	// Title with inline metadata: **[owner/repo#123] PR Title** (Priority: high | Segments: enterprise)
	builder.WriteString(fmt.Sprintf("**[%s/%s#%d] %s**", owner, repo, pr.Number, pr.Title))

	metadataStr := formatEntityMetadata(meta)
	if metadataStr != "" {
		builder.WriteString(" ")
		builder.WriteString(metadataStr)
	}
	builder.WriteString("\n\n")

	// PR body - full content, not truncated
	if pr.Body != "" {
		builder.WriteString("**Description:**\n")
		builder.WriteString(pr.Body)
		builder.WriteString("\n\n")
	}

	// Details in compact format
	builder.WriteString(fmt.Sprintf("**Details:** State: %s | Author: %s", pr.State, pr.User.Login))
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
