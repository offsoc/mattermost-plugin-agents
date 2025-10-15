// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	issuesSearchRateLimitPerMin = 30
	maxIssuesSearchResults      = 100
)

type GitHubIssuesSearchResult struct {
	TotalCount        int           `json:"total_count"`
	IncompleteResults bool          `json:"incomplete_results"`
	Items             []GitHubIssue `json:"items"`
}

func (g *GitHubProtocol) searchIssues(ctx context.Context, owner, repo, query string, limit int, sourceName string) []Doc {
	searchQuery := g.buildGitHubIssuesSearchQuery(query)

	if repo != "" {
		searchQuery += fmt.Sprintf(" repo:%s/%s", owner, repo)
	}

	url := "https://api.github.com/search/issues"

	if g.circuitBreaker != nil && g.circuitBreaker.isOpen(url) {
		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub issues search circuit breaker open, skipping request", "url", url)
		}
		return []Doc{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []Doc{}
	}

	queryParams := req.URL.Query()
	queryParams.Add("q", searchQuery)
	queryParams.Add(QueryParamPerPage, fmt.Sprintf("%d", min(limit, maxIssuesSearchResults)))
	queryParams.Add(QueryParamSort, "updated")
	queryParams.Add("order", "desc")
	req.URL.RawQuery = queryParams.Encode()

	g.addAuthHeaders(req)

	if g.rateLimiter != nil {
		if waitErr := g.rateLimiter.Wait(ctx); waitErr != nil {
			return []Doc{}
		}
	}

	resp, err := g.client.Do(req)
	if err != nil {
		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub issues search failed", "error", err.Error())
		}
		return []Doc{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if g.circuitBreaker != nil && (resp.StatusCode == 403 || resp.StatusCode == 422 || resp.StatusCode >= 500) {
			g.circuitBreaker.recordFailure(url)
		}

		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub issues search HTTP error",
				"status", resp.StatusCode,
				"query", searchQuery)
		}
		return []Doc{}
	}

	var searchResult GitHubIssuesSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub issues search decode error", "error", err.Error())
		}
		return []Doc{}
	}

	var docs []Doc
	for _, issue := range searchResult.Items {
		doc := Doc{
			Title:        issue.Title,
			Content:      issue.Body,
			URL:          issue.HTMLURL,
			Section:      "issue",
			Source:       sourceName,
			Labels:       g.buildIssueLabels(issue),
			CreatedDate:  issue.CreatedAt,
			LastModified: issue.UpdatedAt,
		}

		docs = append(docs, doc)
	}

	return docs
}

func (g *GitHubProtocol) buildIssueLabels(issue GitHubIssue) []string {
	labels := []string{
		fmt.Sprintf("number:#%d", issue.Number),
		fmt.Sprintf("state:%s", issue.State),
		fmt.Sprintf("author:%s", issue.User.Login),
	}

	for _, label := range issue.Labels {
		labels = append(labels, fmt.Sprintf("label:%s", label.Name))
	}

	return labels
}
