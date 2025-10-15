// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractGitHubMetadata(t *testing.T) {
	tests := []struct {
		name                string
		issue               GitHubIssue
		comments            []GitHubIssueComment
		expectedSegments    []pm.CustomerSegment
		expectedCategories  []pm.TechnicalCategory
		expectedCompetitive pm.Competitor
		expectedJiraCount   int
		expectedFileCount   int
		expectedCommitCount int
	}{
		{
			name: "Issue with Jira reference and file path",
			issue: GitHubIssue{
				Title: "Fix auth bug",
				Body:  "See MM-12345 for details. Bug in server/api/auth.go:45",
				State: "open",
			},
			comments:          []GitHubIssueComment{},
			expectedJiraCount: 1,
			expectedFileCount: 1,
		},
		{
			name: "Issue with cross-repo references",
			issue: GitHubIssue{
				Title: "Sync feature with Focalboard",
				Body:  "Related to mattermost/focalboard#456 and facebook/react#18556. Also see #123",
				State: "open",
			},
			comments: []GitHubIssueComment{},
		},
		{
			name: "Issue with commit references",
			issue: GitHubIssue{
				Title: "Bug fixed",
				Body:  "Fixed commit a3f5c2d and merged abc123def456",
				State: "closed",
			},
			comments:            []GitHubIssueComment{},
			expectedCommitCount: 2,
		},
		{
			name: "Enterprise customer issue",
			issue: GitHubIssue{
				Title: "Enterprise authentication issue",
				Body:  "Customer needs SAML integration",
				State: "open",
				Labels: []struct {
					Name string `json:"name"`
				}{
					{Name: "enterprise"},
				},
			},
			comments:         []GitHubIssueComment{},
			expectedSegments: []pm.CustomerSegment{pm.SegmentEnterprise},
		},
		{
			name: "Competitive context",
			issue: GitHubIssue{
				Title: "Feature parity with Slack",
				Body:  "Users switching from Slack",
				State: "open",
			},
			comments:            []GitHubIssueComment{},
			expectedCompetitive: pm.CompetitorSlack,
		},
		{
			name: "Authentication category",
			issue: GitHubIssue{
				Title: "Authentication bug in login",
				Body:  "Authentication fails",
				State: "open",
			},
			comments:           []GitHubIssueComment{},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryAuthentication},
		},
		{
			name: "Issue with comments containing references",
			issue: GitHubIssue{
				Title: "Performance issue",
				Body:  "System is slow",
				State: "open",
			},
			comments: []GitHubIssueComment{
				{Body: "This is related to MM-999 and server/cache/redis.go:123"},
			},
			expectedJiraCount: 1,
			expectedFileCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractGitHubMetadata("mattermost", "mattermost", tt.issue, tt.comments)

			pmMeta, ok := meta.RoleMetadata.(pm.Metadata)
			assert.True(t, ok, "RoleMetadata should be PMMetadata")

			if len(tt.expectedSegments) > 0 {
				assert.ElementsMatch(t, tt.expectedSegments, pmMeta.Segments, "Segments should match")
			}

			if len(tt.expectedCategories) > 0 {
				assert.ElementsMatch(t, tt.expectedCategories, pmMeta.Categories, "Categories should match")
			}

			if tt.expectedCompetitive != "" {
				assert.Equal(t, tt.expectedCompetitive, pmMeta.Competitive, "Competitive context should match")
			}

			if tt.expectedJiraCount > 0 {
				assert.Len(t, meta.JiraTickets, tt.expectedJiraCount, "Jira ticket count should match")
			}

			if tt.expectedFileCount > 0 {
				assert.Len(t, meta.ModifiedFiles, tt.expectedFileCount, "File reference count should match")
			}

			if tt.expectedCommitCount > 0 {
				assert.Len(t, meta.Commits, tt.expectedCommitCount, "Commit reference count should match")
			}
		})
	}
}

func TestExtractGitHubMetadata_EntityLinking(t *testing.T) {
	issue := GitHubIssue{
		Title: "Complex issue with multiple references",
		Body:  `Related to MM-12345, #100, mattermost/focalboard#200, server/api/auth.go:45, fixed a3f5c2d, https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345, https://community.mattermost.com/core/pl/abc123def456`,
		State: "open",
	}

	meta := extractGitHubMetadata("mattermost", "mattermost", issue, []GitHubIssueComment{})

	assert.Len(t, meta.JiraTickets, 1, "Should extract Jira ticket")
	assert.Equal(t, "MM-12345", meta.JiraTickets[0].Key)

	assert.GreaterOrEqual(t, len(meta.GitHubIssues), 2, "Should extract GitHub issues")

	assert.Len(t, meta.ModifiedFiles, 1, "Should extract file reference")
	assert.Equal(t, "server/api/auth.go", meta.ModifiedFiles[0].Path)
	assert.Equal(t, 45, meta.ModifiedFiles[0].StartLine)

	assert.Len(t, meta.Commits, 1, "Should extract commit")
	assert.Equal(t, "a3f5c2d", meta.Commits[0].SHA)

	assert.Len(t, meta.ConfluencePages, 1, "Should extract Confluence page")
	assert.Equal(t, "ENG", meta.ConfluencePages[0].Space)

	assert.Len(t, meta.MattermostLinks, 1, "Should extract Mattermost permalink")
	assert.Equal(t, "core", meta.MattermostLinks[0].TeamName)
}
