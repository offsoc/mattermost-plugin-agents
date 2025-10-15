// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package evals

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// GitHubAdapter implements grounding.GitHubAPIClient using GitHub REST API
type GitHubAdapter struct {
	token  string
	client *http.Client
}

// NewGitHubAdapter creates a GitHub API adapter
func NewGitHubAdapter(token string) *GitHubAdapter {
	return &GitHubAdapter{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// IssueExists checks if a GitHub issue or PR exists (200 vs 404)
func (a *GitHubAdapter) IssueExists(ctx context.Context, owner, repo string, number int) (bool, error) {
	if owner == "" || repo == "" || number == 0 {
		return false, fmt.Errorf("invalid GitHub reference: owner=%s, repo=%s, number=%d", owner, repo, number)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, number)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// JiraAdapter implements grounding.JiraAPIClient using Jira REST API
type JiraAdapter struct {
	baseURL string
	auth    string
	client  *http.Client
}

// NewJiraAdapter creates a Jira API adapter
// auth should be in format "email:token" or "Bearer token"
func NewJiraAdapter(baseURL, auth string) *JiraAdapter {
	return &JiraAdapter{
		baseURL: baseURL,
		auth:    auth,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// IssueExists checks if a Jira issue exists (200 vs 404)
func (a *JiraAdapter) IssueExists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("invalid Jira key: empty")
	}

	url := fmt.Sprintf("%s/rest/api/2/issue/%s", a.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	if a.auth != "" {
		req.Header.Set("Authorization", "Basic "+a.auth)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
