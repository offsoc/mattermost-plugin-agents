// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type RateLimitInfo struct {
	Remaining int
	Limit     int
	Reset     time.Time
}

func extractRateLimitInfo(resp *http.Response) *RateLimitInfo {
	info := &RateLimitInfo{}

	if remainingStr := resp.Header.Get("X-RateLimit-Remaining"); remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			info.Remaining = remaining
		}
	}

	if limitStr := resp.Header.Get("X-RateLimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			info.Limit = limit
		}
	}

	if resetStr := resp.Header.Get("X-RateLimit-Reset"); resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			info.Reset = time.Unix(resetUnix, 0)
		}
	}

	return info
}

func (g *GitHubProtocol) searchCodeSingleAttempt(ctx context.Context, owner, repo, query, language string, limit int, sourceName string) []Doc {
	docs, wasRateLimited, rateInfo, err := g.searchCodeAttempt(ctx, owner, repo, query, language, limit, sourceName)

	if !wasRateLimited {
		if err != nil && g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub code search failed", "error", err.Error())
		}
		return docs
	}

	if g.pluginAPI != nil {
		g.pluginAPI.LogWarn("GitHub code search rate limited, skipping retries",
			"remaining", rateInfo.Remaining,
			"reset_at", rateInfo.Reset.Format(time.RFC3339),
			"repo", repo)
	}

	return []Doc{}
}

func (g *GitHubProtocol) searchCodeAttempt(ctx context.Context, owner, repo, query, language string, limit int, sourceName string) (docs []Doc, wasRateLimited bool, rateInfo *RateLimitInfo, err error) {
	searchQuery := g.buildGitHubSearchQuery(query)

	if repo != "" {
		searchQuery += fmt.Sprintf(" repo:%s/%s", owner, repo)
	}
	if language != "" {
		searchQuery += fmt.Sprintf(" language:%s", language)
	}

	url := "https://api.github.com/search/code"

	if g.circuitBreaker != nil && g.circuitBreaker.isOpen(url) {
		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub code search circuit breaker open, skipping request", "url", url)
		}
		return []Doc{}, false, nil, fmt.Errorf("circuit breaker open")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []Doc{}, false, nil, err
	}

	queryParams := req.URL.Query()
	queryParams.Add("q", searchQuery)
	queryParams.Add("per_page", fmt.Sprintf("%d", min(limit, maxCodeSearchResults)))
	req.URL.RawQuery = queryParams.Encode()

	g.addAuthHeaders(req)

	if g.rateLimiter != nil {
		if waitErr := g.rateLimiter.Wait(ctx); waitErr != nil {
			return []Doc{}, false, nil, waitErr
		}
	}

	resp, err := g.client.Do(req)
	if err != nil {
		if g.pluginAPI != nil {
			g.pluginAPI.LogWarn("GitHub code search failed", "error", err.Error())
		}
		return []Doc{}, false, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if g.circuitBreaker != nil && (resp.StatusCode == 403 || resp.StatusCode == 422 || resp.StatusCode >= 500) {
			g.circuitBreaker.recordFailure(url)
		}

		if resp.StatusCode == 403 {
			rateLimitInfo := extractRateLimitInfo(resp)

			if g.pluginAPI != nil {
				g.pluginAPI.LogWarn("GitHub code search rate limited",
					"status", resp.StatusCode,
					"query", searchQuery,
					"remaining", rateLimitInfo.Remaining,
					"reset_at", rateLimitInfo.Reset.Format(time.RFC3339))
			}

			return []Doc{}, true, rateLimitInfo, fmt.Errorf("rate limited")
		}

		if g.pluginAPI != nil {
			switch resp.StatusCode {
			case 422:
				g.pluginAPI.LogWarn("GitHub code search invalid query",
					"status", resp.StatusCode,
					"query", searchQuery)
			default:
				g.pluginAPI.LogWarn("GitHub code search HTTP error",
					"status", resp.StatusCode,
					"query", searchQuery)
			}
		}
		return []Doc{}, false, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	docs, parseErr := g.parseCodeSearchResponse(ctx, resp, sourceName)
	return docs, false, nil, parseErr
}
