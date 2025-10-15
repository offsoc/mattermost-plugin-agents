// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractMattermostMetadata(t *testing.T) {
	tests := []struct {
		name                string
		post                MattermostPost
		expectedSegments    []pm.CustomerSegment
		expectedCategories  []pm.TechnicalCategory
		expectedCompetitive pm.Competitor
		expectedPriority    pm.Priority
		expectedJiraCount   int
		expectedGitHubCount int
	}{
		{
			name: "Mobile app issue for enterprise",
			post: MattermostPost{
				Message:    "Mobile app crashes on iOS for enterprise customers with SSO enabled",
				ReplyCount: 15,
				IsPinned:   false,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryMobile, pm.CategoryPerformance, pm.CategoryAuthentication}, // "crashes" matches performance
			expectedPriority:   pm.PriorityLow,                                                                               // 0 (not pinned) + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "Federal authentication high priority pinned",
			post: MattermostPost{
				Message:    "SAML authentication failing for high-security deployment - urgent fix needed",
				ReplyCount: 25,
				IsPinned:   true,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryAuthentication},
			expectedPriority:   pm.PriorityHigh, // 5 (pinned) + 3 (replies>=20) = 8 >= 7 = High
		},
		{
			name: "Playbooks workflow discussion",
			post: MattermostPost{
				Message:    "Playbooks workflow automation for incident response needs improvement",
				ReplyCount: 8,
				IsPinned:   false,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPlaybooks},
			expectedPriority:   pm.PriorityLow, // 0 + 1 (replies>=5) = 1 < 4 = Low
		},
		{
			name: "Microsoft competitive context",
			post: MattermostPost{
				Message:    "Comparing Microsoft collaboration features with Mattermost capabilities",
				ReplyCount: 12,
				IsPinned:   false,
			},
			expectedCompetitive: pm.CompetitorMicrosoftAny, // "Microsoft" extracts as Microsoft
			expectedPriority:    pm.PriorityLow,            // 0 + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "Slack comparison",
			post: MattermostPost{
				Message:    "How does Mattermost compare to Slack for large teams?",
				ReplyCount: 10,
				IsPinned:   false,
			},
			expectedCompetitive: pm.CompetitorSlack,
			expectedPriority:    pm.PriorityLow, // 0 + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "Healthcare HIPAA compliance",
			post: MattermostPost{
				Message:    "HIPAA compliance requirements for healthcare organizations using Mattermost",
				ReplyCount: 18,
				IsPinned:   false,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentHealthcare, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityLow, // 0 + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "DevOps Kubernetes integration",
			post: MattermostPost{
				Message:    "Kubernetes operator deployment for DevOps teams - best practices?",
				ReplyCount: 22,
				IsPinned:   false,
			},
			expectedSegments: []pm.CustomerSegment{pm.SegmentDevOps},
			expectedPriority: pm.PriorityLow, // 0 + 3 (replies>=20) = 3 < 4 = Low
		},
		{
			name: "Database performance critical pinned",
			post: MattermostPost{
				Message:    "Database performance issues causing slowdowns in large enterprise deployment",
				ReplyCount: 30,
				IsPinned:   true,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryDatabase, pm.CategoryPerformance},
			expectedPriority:   pm.PriorityHigh, // 5 (pinned) + 3 (replies>=20) = 8 >= 7 = High
		},
		{
			name: "Low priority generic question",
			post: MattermostPost{
				Message:    "How do I change my profile picture?",
				ReplyCount: 2,
				IsPinned:   false,
			},
			expectedPriority: pm.PriorityLow, // 0 + 0 = 0 < 4 = Low
		},
		{
			name: "Plugin integration issue with Jira reference",
			post: MattermostPost{
				Message:    "Plugin failing to load after upgrade - see MM-12345 for details",
				ReplyCount: 14,
				IsPinned:   false,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPerformance, pm.CategoryPlugins}, // "upgrade" matches performance, removed integrations
			expectedJiraCount:  1,
			expectedPriority:   pm.PriorityLow, // 0 + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "GitHub issue reference",
			post: MattermostPost{
				Message:    "Bug reported in mattermost/mattermost#12345 and #678 needs attention",
				ReplyCount: 11,
				IsPinned:   false,
			},
			expectedGitHubCount: 2,
			expectedPriority:    pm.PriorityLow, // 0 + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "Calls feature discussion",
			post: MattermostPost{
				Message:    "Calls plugin screen sharing not working properly",
				ReplyCount: 16,
				IsPinned:   false,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCalls, pm.CategoryPlugins},
			expectedPriority:   pm.PriorityLow, // 0 + 2 (replies>=10) = 2 < 4 = Low
		},
		{
			name: "Boards kanban feature",
			post: MattermostPost{
				Message:    "Boards kanban view customization options",
				ReplyCount: 7,
				IsPinned:   false,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryBoards},
			expectedPriority:   pm.PriorityLow, // 0 + 1 (replies>=5) = 1 < 4 = Low
		},
		{
			name: "Finance segment discussion",
			post: MattermostPost{
				Message:    "Financial services compliance and banking regulations for Mattermost deployment",
				ReplyCount: 20,
				IsPinned:   false,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFinance, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityLow, // 0 + 3 (replies>=20) = 3 < 4 = Low
		},
		{
			name: "SMB segment discussion",
			post: MattermostPost{
				Message:    "Setup guide for small business and startup teams",
				ReplyCount: 5,
				IsPinned:   false,
			},
			expectedSegments: []pm.CustomerSegment{pm.SegmentSMB},
			expectedPriority: pm.PriorityLow, // 0 + 1 (replies>=5) = 1 < 4 = Low
		},
		{
			name: "Channels organization pinned",
			post: MattermostPost{
				Message:    "Channel organization and management best practices for large teams",
				ReplyCount: 12,
				IsPinned:   true,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryChannels},
			expectedPriority:   pm.PriorityHigh, // 5 (pinned) + 2 (replies>=10) = 7 >= 7 = High
		},
		{
			name: "Pinned post with minimal engagement",
			post: MattermostPost{
				Message:    "Important announcement about upcoming maintenance",
				ReplyCount: 3,
				IsPinned:   true,
			},
			expectedPriority: pm.PriorityMedium, // 5 (pinned) + 0 = 5 >= 4 = Medium
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractMattermostMetadata(tt.post)

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
				assert.Equal(t, tt.expectedPriority, pmMeta.Priority, "Priority should match")
			}

			if tt.expectedJiraCount > 0 {
				assert.Len(t, meta.JiraTickets, tt.expectedJiraCount, "Jira ticket count should match")
			}

			if tt.expectedGitHubCount > 0 {
				actualGitHubCount := len(meta.GitHubIssues) + len(meta.GitHubPRs)
				assert.Equal(t, tt.expectedGitHubCount, actualGitHubCount, "GitHub reference count should match")
			}
		})
	}
}

func TestExtractMattermostMetadata_EntityLinking(t *testing.T) {
	post := MattermostPost{
		Message: `Complex post with multiple references:
- Jira ticket: MM-12345
- GitHub issue: mattermost/mattermost#100
- Another issue: #200
- Commit: fixed a3f5c2d
- File: server/api/auth.go:45
- Confluence: https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345
- Mattermost: https://community.mattermost.com/core/pl/abc123def456`,
		ReplyCount: 15,
		IsPinned:   false,
	}

	meta := extractMattermostMetadata(post)

	assert.Len(t, meta.JiraTickets, 1, "Should extract Jira ticket")
	assert.Equal(t, "MM-12345", meta.JiraTickets[0].Key)

	totalGitHub := len(meta.GitHubIssues) + len(meta.GitHubPRs)
	assert.GreaterOrEqual(t, totalGitHub, 2, "Should extract GitHub issues")

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

func TestInferMattermostPostPriority(t *testing.T) {
	tests := []struct {
		name             string
		post             MattermostPost
		expectedPriority pm.Priority
	}{
		{
			name: "High priority - pinned with high replies",
			post: MattermostPost{
				IsPinned:   true,
				ReplyCount: 25,
			},
			expectedPriority: pm.PriorityHigh, // 5 (pinned) + 3 (replies>=20) = 8 >= 7
		},
		{
			name: "High priority - pinned with medium replies",
			post: MattermostPost{
				IsPinned:   true,
				ReplyCount: 12,
			},
			expectedPriority: pm.PriorityHigh, // 5 (pinned) + 2 (replies>=10) = 7 >= 7
		},
		{
			name: "Medium priority - pinned with low replies",
			post: MattermostPost{
				IsPinned:   true,
				ReplyCount: 6,
			},
			expectedPriority: pm.PriorityMedium, // 5 (pinned) + 1 (replies>=5) = 6 >= 4
		},
		{
			name: "Medium priority - pinned with no replies",
			post: MattermostPost{
				IsPinned:   true,
				ReplyCount: 0,
			},
			expectedPriority: pm.PriorityMedium, // 5 (pinned) + 0 = 5 >= 4
		},
		{
			name: "Low priority - high replies not pinned",
			post: MattermostPost{
				IsPinned:   false,
				ReplyCount: 25,
			},
			expectedPriority: pm.PriorityLow, // 0 + 3 (replies>=20) = 3 < 4
		},
		{
			name: "Low priority - medium replies not pinned",
			post: MattermostPost{
				IsPinned:   false,
				ReplyCount: 12,
			},
			expectedPriority: pm.PriorityLow, // 0 + 2 (replies>=10) = 2 < 4
		},
		{
			name: "Low priority - no engagement",
			post: MattermostPost{
				IsPinned:   false,
				ReplyCount: 0,
			},
			expectedPriority: pm.PriorityLow, // 0 + 0 = 0 < 4
		},
		{
			name: "Low priority - minimal replies",
			post: MattermostPost{
				IsPinned:   false,
				ReplyCount: 3,
			},
			expectedPriority: pm.PriorityLow, // 0 + 0 = 0 < 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := inferMattermostPostPriority(tt.post)
			assert.Equal(t, tt.expectedPriority, priority, "Priority should match expected value")
		})
	}
}
