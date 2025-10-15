// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import "context"

// APIClient interfaces - Defined in grounding to avoid datasources dependency
// Implementations live in evals package to maintain proper architecture

// GitHubAPIClient provides GitHub API operations for citation verification
type GitHubAPIClient interface {
	IssueExists(ctx context.Context, owner, repo string, number int) (bool, error)
}

// JiraAPIClient provides Jira API operations for citation verification
type JiraAPIClient interface {
	IssueExists(ctx context.Context, key string) (bool, error)
}

// ConfluenceAPIClient provides Confluence API operations for citation verification
type ConfluenceAPIClient interface {
	PageExists(ctx context.Context, space, pageID string) (bool, error)
}

// APIClients contains all API clients for external verification
type APIClients struct {
	GitHub     GitHubAPIClient
	Jira       JiraAPIClient
	Confluence ConfluenceAPIClient
}
