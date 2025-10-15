// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources/queryutils"
)

func (g *GitHubProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	result := &SyntaxValidationResult{
		OriginalQuery:    request.Topic,
		IsValidSyntax:    true,
		SyntaxErrors:     []string{},
		SupportsFeatures: []string{"AND", "OR", "quotes", "labels", "date ranges", "user filters"},
	}

	if request.Topic == "" {
		result.RecommendedQuery = "mobile"
		return result, nil
	}

	owner := request.Source.Endpoints["owner"]
	reposList := request.Source.Endpoints["repos"]
	if owner == "" || reposList == "" {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "GitHub search requires owner and repos configuration")
		result.RecommendedQuery = request.Topic
		return result, nil
	}

	repos := strings.Split(reposList, ",")
	if len(repos) == 0 {
		result.IsValidSyntax = false
		result.SyntaxErrors = append(result.SyntaxErrors, "No repositories configured for search")
		result.RecommendedQuery = request.Topic
		return result, nil
	}

	testRepo := strings.TrimSpace(repos[0])

	issueCount := g.testGitHubSearchQuery(ctx, owner, testRepo, request.Topic)
	result.TestResultCount = issueCount

	if issueCount == 0 {
		simplifiedQuery := g.simplifyGitHubQuery(request.Topic)
		simpleCount := g.testGitHubSearchQuery(ctx, owner, testRepo, simplifiedQuery)

		if simpleCount > 0 {
			result.IsValidSyntax = false
			result.SyntaxErrors = append(result.SyntaxErrors,
				fmt.Sprintf("Complex query returned 0 results, but simplified query returned %d results", simpleCount))
			result.RecommendedQuery = simplifiedQuery
		} else {
			verySimpleQuery := "mobile"
			verySimpleCount := g.testGitHubSearchQuery(ctx, owner, testRepo, verySimpleQuery)
			if verySimpleCount > 0 {
				result.IsValidSyntax = false
				result.SyntaxErrors = append(result.SyntaxErrors, "Query syntax may be too complex for GitHub search")
				result.RecommendedQuery = verySimpleQuery
			} else {
				result.RecommendedQuery = request.Topic
			}
		}
	} else {
		result.RecommendedQuery = request.Topic
	}

	return result, nil
}

func (g *GitHubProtocol) testGitHubSearchQuery(ctx context.Context, owner, repo, query string) int {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0
	}

	queryParams := req.URL.Query()
	queryParams.Add(QueryParamState, QueryParamAll)
	queryParams.Add(QueryParamPerPage, "1")
	req.URL.RawQuery = queryParams.Encode()

	g.addAuthHeaders(req)

	if waitErr := WaitRateLimiter(ctx, g.rateLimiter); waitErr != nil {
		return 0
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0
	}

	var issues []GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return 0
	}

	count := 0
	for _, issue := range issues {
		if g.isRelevantToTopic(issue.Title, issue.Body, query) {
			count++
		}
	}

	return count
}

func (g *GitHubProtocol) simplifyGitHubQuery(query string) string {
	return queryutils.SimplifyQueryToKeywords(query, 3, "mobile")
}
