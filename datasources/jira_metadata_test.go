// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	jira "github.com/andygrunwald/go-jira"
	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractJiraMetadata(t *testing.T) {
	tests := []struct {
		name                string
		issue               jira.Issue
		expectedSegments    []pm.CustomerSegment
		expectedCategories  []pm.TechnicalCategory
		expectedCompetitive pm.Competitor
		expectedPriority    pm.Priority
		expectedGitHubCount int
		expectedFileCount   int
	}{
		{
			name: "Issue with GitHub reference and file path",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Summary:     "Fix auth bug",
					Description: "See https://github.com/mattermost/mattermost/issues/123. Bug in server/api/auth.go:45",
					Priority: &jira.Priority{
						Name: "High",
					},
				},
			},
			expectedPriority:    pm.PriorityHigh,
			expectedGitHubCount: 1,
			expectedFileCount:   1,
		},
		{
			name: "Enterprise customer issue",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Summary:     "Enterprise SSO authentication",
					Description: "Customer needs SSO authentication support",
					Labels:      []string{"enterprise", "authentication"},
					Priority: &jira.Priority{
						Name: "Critical",
					},
				},
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryAuthentication},
			expectedPriority:   pm.PriorityHigh,
		},
		{
			name: "Competitive context",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Summary:     "Feature parity with Slack",
					Description: "Users switching from Slack expect this feature",
					Priority: &jira.Priority{
						Name: "Medium",
					},
				},
			},
			expectedCompetitive: pm.CompetitorSlack,
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name: "Issue with multiple entity references",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Summary: "Complex bug fix",
					Description: `
Related issues:
- GitHub: https://github.com/mattermost/mattermost/issues/100
- File: server/api/auth.go:45
- Commit: fixed a3f5c2d
- Jira: MM-999
`,
					Priority: &jira.Priority{
						Name: "High",
					},
				},
			},
			expectedPriority:    pm.PriorityHigh,
			expectedGitHubCount: 1,
			expectedFileCount:   1,
		},
		{
			name: "Issue with comments",
			issue: jira.Issue{
				Fields: &jira.IssueFields{
					Summary:     "Performance issue",
					Description: "System is slow",
					Priority: &jira.Priority{
						Name: "Low",
					},
					Comments: &jira.Comments{
						Comments: []*jira.Comment{
							{
								Body: "This is related to server/cache/redis.go:123 and commit abc1234",
							},
						},
					},
				},
			},
			expectedPriority:  pm.PriorityLow,
			expectedFileCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractJiraMetadata(tt.issue)

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

			if tt.expectedPriority != "" {
				assert.Equal(t, tt.expectedPriority, pmMeta.Priority, "pm.Priority should match")
			}

			if tt.expectedGitHubCount > 0 {
				assert.Len(t, meta.GitHubIssues, tt.expectedGitHubCount, "GitHub issue count should match")
			}

			if tt.expectedFileCount > 0 {
				assert.Len(t, meta.ModifiedFiles, tt.expectedFileCount, "File reference count should match")
			}
		})
	}
}

func TestExtractJiraMetadata_EntityLinking(t *testing.T) {
	issue := jira.Issue{
		Fields: &jira.IssueFields{
			Summary: "Complex issue with multiple references",
			Description: `
This issue is related to several things:
- GitHub issue: https://github.com/mattermost/mattermost/issues/100
- Another Jira: MM-12345
- File: server/api/auth.go:45
- Commit: fixed a3f5c2d
- Confluence: https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345
`,
			Priority: &jira.Priority{
				Name: "High",
			},
		},
	}

	meta := extractJiraMetadata(issue)

	assert.Len(t, meta.GitHubIssues, 1, "Should extract GitHub issue")
	assert.Equal(t, 100, meta.GitHubIssues[0].Number)

	assert.Len(t, meta.JiraTickets, 1, "Should extract Jira ticket")
	assert.Equal(t, "MM-12345", meta.JiraTickets[0].Key)

	assert.Len(t, meta.ModifiedFiles, 1, "Should extract file reference")
	assert.Equal(t, "server/api/auth.go", meta.ModifiedFiles[0].Path)
	assert.Equal(t, 45, meta.ModifiedFiles[0].StartLine)

	assert.Len(t, meta.Commits, 1, "Should extract commit")
	assert.Equal(t, "a3f5c2d", meta.Commits[0].SHA)

	assert.Len(t, meta.ConfluencePages, 1, "Should extract Confluence page")
	assert.Equal(t, "ENG", meta.ConfluencePages[0].Space)

	pmMeta, ok := meta.RoleMetadata.(pm.Metadata)
	assert.True(t, ok, "RoleMetadata should be PMMetadata")
	assert.Equal(t, pm.PriorityHigh, pmMeta.Priority, "Should extract priority")
}
