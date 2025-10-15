// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"
	"time"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractDiscourseMetadata(t *testing.T) {
	// Helper to generate recent dates for testing priority calculation
	recentDate := time.Now().Add(-2 * 24 * time.Hour).Format(time.RFC3339)  // 2 days ago (recent activity bonus)
	weekAgoDate := time.Now().Add(-8 * 24 * time.Hour).Format(time.RFC3339) // 8 days ago
	oldDate := "2024-11-01T10:00:00Z"                                       // Old date (no recency bonus)

	tests := []struct {
		name                string
		topic               DiscourseTopic
		expectedSegments    []pm.CustomerSegment
		expectedCategories  []pm.TechnicalCategory
		expectedCompetitive pm.Competitor
		expectedPriority    pm.Priority
		expectedJiraCount   int
		expectedGitHubCount int
	}{
		{
			name: "Mobile high engagement enterprise topic",
			topic: DiscourseTopic{
				Title:        "Mobile app performance issues for enterprise customers",
				Tags:         []string{"mobile", "performance", "enterprise"},
				PostsCount:   30,
				Views:        550,
				LikeCount:    22,
				CreatedAt:    weekAgoDate,
				LastPostedAt: recentDate,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryMobile, pm.CategoryPerformance},
			expectedPriority:   pm.PriorityHigh, // 3 (likes>=20) + 3 (views>=500) + 2 (posts>=30) + 2 (recent) = 10 >= 7 = High
		},
		{
			name: "Federal customer authentication with SAML",
			topic: DiscourseTopic{
				Title:        "SAML authentication failing for classified deployment",
				Tags:         []string{"authentication", "saml", "federal"},
				PostsCount:   15,
				Views:        300,
				LikeCount:    12,
				CreatedAt:    weekAgoDate,
				LastPostedAt: recentDate,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFederal, pm.SegmentEnterprise}, // "government" matches enterprise too
			expectedCategories: []pm.TechnicalCategory{pm.CategoryAuthentication},
			expectedPriority:   pm.PriorityHigh, // 2 (likes>=10) + 2 (views>=200) + 1 (posts>=10) + 2 (recent) = 7 >= 7 = High
		},
		{
			name: "Playbooks feature request",
			topic: DiscourseTopic{
				Title:        "Playbooks workflow automation for incident response",
				Tags:         []string{"playbooks", "workflow", "automation"},
				PostsCount:   8,
				Views:        180,
				LikeCount:    6,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate, // Old, no recent bonus
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPlaybooks},
			expectedPriority:   pm.PriorityLow, // 1 (likes>=5) + 1 (views>=100) + 0 (posts<10) + 0 (old) = 2 < 4 = Low
		},
		{
			name: "Competitive context - Microsoft comparison",
			topic: DiscourseTopic{
				Title:        "Comparing Microsoft collaboration tools with Mattermost",
				Tags:         []string{"comparison", "microsoft"},
				PostsCount:   25,
				Views:        450,
				LikeCount:    18,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedCompetitive: pm.CompetitorMicrosoftAny, // "Microsoft" is detected
			expectedPriority:    pm.PriorityMedium,         // 2 (likes>=10) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 5 >= 4 = Medium
		},
		{
			name: "Competitive context - Slack comparison",
			topic: DiscourseTopic{
				Title:        "Slack vs Mattermost feature comparison",
				Tags:         []string{"comparison", "slack"},
				PostsCount:   20,
				Views:        400,
				LikeCount:    15,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedCompetitive: pm.CompetitorSlack,
			expectedPriority:    pm.PriorityMedium, // 2 (likes>=10) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 5 >= 4 = Medium
		},
		{
			name: "Healthcare compliance discussion",
			topic: DiscourseTopic{
				Title:        "HIPAA compliance requirements for healthcare organizations",
				Tags:         []string{"healthcare", "compliance", "hipaa"},
				PostsCount:   12,
				Views:        220,
				LikeCount:    8,
				CreatedAt:    weekAgoDate,
				LastPostedAt: recentDate,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentHealthcare, pm.SegmentEnterprise}, // "organizations" matches enterprise
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium, // 1 (likes>=5) + 2 (views>=200) + 1 (posts>=10) + 2 (recent) = 6 >= 4 = Medium
		},
		{
			name: "DevOps Kubernetes integration",
			topic: DiscourseTopic{
				Title:        "Kubernetes operator deployment for DevOps teams",
				Tags:         []string{"kubernetes", "devops", "deployment"},
				PostsCount:   18,
				Views:        350,
				LikeCount:    10,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedSegments: []pm.CustomerSegment{pm.SegmentDevOps},
			// "kubernetes" and "deployment" don't match standard categories
			expectedPriority: pm.PriorityMedium, // 2 (likes>=10) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 5 >= 4 = Medium
		},
		{
			name: "Database performance with multiple categories",
			topic: DiscourseTopic{
				Title:        "Database performance tuning for large enterprise deployments",
				Tags:         []string{"database", "performance", "enterprise"},
				PostsCount:   35,
				Views:        600,
				LikeCount:    25,
				CreatedAt:    weekAgoDate,
				LastPostedAt: recentDate,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryDatabase, pm.CategoryPerformance},
			expectedPriority:   pm.PriorityHigh, // 3 (likes>=20) + 3 (views>=500) + 2 (posts>=30) + 2 (recent) = 10 >= 7 = High
		},
		{
			name: "Low priority generic discussion",
			topic: DiscourseTopic{
				Title:        "General questions about Mattermost",
				Tags:         []string{},
				PostsCount:   3,
				Views:        50,
				LikeCount:    1,
				CreatedAt:    oldDate,
				LastPostedAt: oldDate,
			},
			expectedPriority: pm.PriorityLow, // 0 + 0 + 0 + 0 = 0 < 4 = Low
		},
		{
			name: "Plugin integration issue",
			topic: DiscourseTopic{
				Title:        "Plugin failing to load after upgrade",
				Tags:         []string{"plugin", "integration"},
				PostsCount:   22,
				Views:        380,
				LikeCount:    14,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPerformance, pm.CategoryIntegrations, pm.CategoryPlugins}, // "upgrade" matches performance
			expectedPriority:   pm.PriorityMedium,                                                                           // 2 (likes>=10) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 5 >= 4 = Medium
		},
		{
			name: "Topic with Jira references",
			topic: DiscourseTopic{
				Title:        "Bug report: See MM-12345 for details",
				Tags:         []string{"bug"},
				PostsCount:   10,
				Views:        150,
				LikeCount:    5,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedJiraCount: 1,
			expectedPriority:  pm.PriorityLow, // 1 (likes>=5) + 1 (views>=100) + 1 (posts>=10) + 0 (old) = 3 < 4 = Low
		},
		{
			name: "Topic with GitHub references",
			topic: DiscourseTopic{
				Title:        "Related to mattermost/mattermost#12345 and #678",
				Tags:         []string{},
				PostsCount:   8,
				Views:        120,
				LikeCount:    4,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedGitHubCount: 2,
			expectedPriority:    pm.PriorityLow, // 0 + 1 (views>=100) + 0 + 0 = 1 < 4 = Low
		},
		{
			name: "Calls feature discussion",
			topic: DiscourseTopic{
				Title:        "Calls plugin screen sharing not working",
				Tags:         []string{"calls", "plugin"},
				PostsCount:   16,
				Views:        280,
				LikeCount:    11,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCalls, pm.CategoryPlugins},
			expectedPriority:   pm.PriorityMedium, // 2 (likes>=10) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 5 >= 4 = Medium
		},
		{
			name: "Boards kanban feature",
			topic: DiscourseTopic{
				Title:        "Boards kanban view customization",
				Tags:         []string{"boards", "kanban"},
				PostsCount:   12,
				Views:        200,
				LikeCount:    7,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryBoards},
			expectedPriority:   pm.PriorityMedium, // 1 (likes>=5) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 4 >= 4 = Medium
		},
		{
			name: "Finance/Banking segment",
			topic: DiscourseTopic{
				Title:        "Financial services compliance and banking regulations",
				Tags:         []string{"finance", "banking", "compliance"},
				PostsCount:   14,
				Views:        250,
				LikeCount:    9,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFinance, pm.SegmentEnterprise}, // "services" matches enterprise
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium, // 1 (likes>=5) + 2 (views>=200) + 1 (posts>=10) + 0 (old) = 4 >= 4 = Medium
		},
		{
			name: "SMB segment discussion",
			topic: DiscourseTopic{
				Title:        "Setup guide for small business and startup teams",
				Tags:         []string{"smb", "small-business", "startup"},
				PostsCount:   9,
				Views:        140,
				LikeCount:    5,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedSegments: []pm.CustomerSegment{pm.SegmentSMB},
			expectedPriority: pm.PriorityLow, // 1 (likes>=5) + 1 (views>=100) + 0 (posts<10) + 0 (old) = 2 < 4 = Low
		},
		{
			name: "Channels feature with medium engagement",
			topic: DiscourseTopic{
				Title:        "Channel organization and management best practices",
				Tags:         []string{"channels"},
				PostsCount:   11,
				Views:        190,
				LikeCount:    6,
				CreatedAt:    weekAgoDate,
				LastPostedAt: oldDate,
			},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryChannels},
			expectedPriority:   pm.PriorityLow, // 1 (likes>=5) + 1 (views>=100) + 1 (posts>=10) + 0 (old) = 3 < 4 = Low
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractDiscourseMetadata(tt.topic)

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

func TestExtractDiscourseMetadata_EntityLinking(t *testing.T) {
	topic := DiscourseTopic{
		Title: `Complex discussion with multiple references:
- Jira ticket: MM-12345
- GitHub issue: mattermost/mattermost#100
- Another issue: #200
- Commit: fixed a3f5c2d
- File: server/api/auth.go:45
- Confluence: https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345
- Mattermost: https://community.mattermost.com/core/pl/abc123def456`,
		Tags:         []string{"bug", "enterprise"},
		PostsCount:   15,
		Views:        250,
		LikeCount:    10,
		CreatedAt:    "2024-12-01T10:00:00Z",
		LastPostedAt: "2024-12-05T15:00:00Z",
	}

	meta := extractDiscourseMetadata(topic)

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

func TestInferDiscourseTopicPriority(t *testing.T) {
	recentDate := time.Now().Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	weekAgoDate := time.Now().Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	oldDate := "2024-11-01T10:00:00Z"

	tests := []struct {
		name             string
		topic            DiscourseTopic
		expectedPriority pm.Priority
	}{
		{
			name: "High priority - high engagement all metrics",
			topic: DiscourseTopic{
				PostsCount:   35,
				Views:        600,
				LikeCount:    25,
				LastPostedAt: recentDate,
			},
			expectedPriority: pm.PriorityHigh, // 3 + 3 + 2 + 2 = 10 >= 7
		},
		{
			name: "High priority - very high likes and views",
			topic: DiscourseTopic{
				PostsCount:   10,
				Views:        550,
				LikeCount:    22,
				LastPostedAt: recentDate,
			},
			expectedPriority: pm.PriorityHigh, // 3 + 3 + 1 + 2 = 9 >= 7
		},
		{
			name: "High priority - moderate engagement with recent activity",
			topic: DiscourseTopic{
				PostsCount:   15,
				Views:        250,
				LikeCount:    12,
				LastPostedAt: recentDate,
			},
			expectedPriority: pm.PriorityHigh, // 2 + 2 + 1 + 2 = 7 >= 7 = High
		},
		{
			name: "Medium priority - high posts but lower views",
			topic: DiscourseTopic{
				PostsCount:   30,
				Views:        150,
				LikeCount:    5,
				LastPostedAt: weekAgoDate,
			},
			expectedPriority: pm.PriorityMedium, // 1 + 1 + 2 + 1 = 5 >= 4
		},
		{
			name: "Low priority - minimal engagement",
			topic: DiscourseTopic{
				PostsCount:   3,
				Views:        50,
				LikeCount:    1,
				LastPostedAt: oldDate,
			},
			expectedPriority: pm.PriorityLow, // 0 + 0 + 0 + 0 = 0 < 4
		},
		{
			name: "Low priority - no engagement metrics",
			topic: DiscourseTopic{
				PostsCount:   1,
				Views:        10,
				LikeCount:    0,
				LastPostedAt: oldDate,
			},
			expectedPriority: pm.PriorityLow, // 0 + 0 + 0 + 0 = 0 < 4
		},
		{
			name: "Medium priority - recent activity bonus",
			topic: DiscourseTopic{
				PostsCount:   8,
				Views:        120,
				LikeCount:    4,
				LastPostedAt: recentDate,
			},
			expectedPriority: pm.PriorityLow, // 0 + 1 + 0 + 2 = 3 < 4, actually Low
		},
		{
			name: "High priority - extreme visibility",
			topic: DiscourseTopic{
				PostsCount:   5,
				Views:        800,
				LikeCount:    15,
				LastPostedAt: recentDate,
			},
			expectedPriority: pm.PriorityHigh, // 2 + 3 + 0 + 2 = 7 >= 7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := inferDiscourseTopicPriority(tt.topic)
			assert.Equal(t, tt.expectedPriority, priority, "Priority should match expected value")
		})
	}
}
