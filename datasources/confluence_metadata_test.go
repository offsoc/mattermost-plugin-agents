// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractConfluenceMetadata(t *testing.T) {
	tests := []struct {
		name                string
		page                ConfluenceContent
		textContent         string
		expectedSegments    []pm.CustomerSegment
		expectedCategories  []pm.TechnicalCategory
		expectedCompetitive pm.Competitor
		expectedPriority    pm.Priority
		expectedJiraCount   int
		expectedGitHubCount int
	}{
		{
			name: "High security deployment with compliance",
			page: ConfluenceContent{
				Title: "High Security Deployment Requirements",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "SECURE", Name: "Security"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "secure"},
							{Name: "classified"},
						},
					},
				},
			},
			textContent:        "Requirements for secure tactical deployment with CMMC compliance and IL4 certification",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFederal, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium, // No high/low priority keywords, defaults to Medium
		},
		{
			name: "Enterprise SSO integration guide",
			page: ConfluenceContent{
				Title: "Enterprise SSO Integration with SAML",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "ENT", Name: "Enterprise"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "enterprise"},
							{Name: "authentication"},
						},
					},
				},
			},
			textContent:        "Configure SAML and LDAP for enterprise E20 deployments with high availability",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryIntegrations, pm.CategoryPlugins, pm.CategoryAuthentication}, // "integration" in title matches integrations and plugins
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "Mobile app UX specification",
			page: ConfluenceContent{
				Title: "UX Spec: Mobile App Push Notifications",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "DES", Name: "Design"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "ux-spec"},
							{Name: "mobile"},
						},
					},
				},
			},
			textContent:        "Design specifications for push notifications on iOS and Android mobile applications",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryChannels, pm.CategoryMobile}, // "notifications" matches channels category
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "Healthcare HIPAA compliance",
			page: ConfluenceContent{
				Title: "HIPAA Compliance Requirements for Healthcare",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "HEALTH", Name: "Healthcare"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "healthcare"},
							{Name: "hipaa"},
						},
					},
				},
			},
			textContent:        "Compliance requirements for healthcare organizations handling patient data under HIPAA regulations",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentHealthcare, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium, // No high/low priority keywords, defaults to Medium
		},
		{
			name: "Slack migration competitive context",
			page: ConfluenceContent{
				Title: "Migrating from Slack to Mattermost",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "MIG", Name: "Migration"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "migration"},
							{Name: "slack"},
						},
					},
				},
			},
			textContent:         "Step-by-step guide for migrating teams from Slack workspaces to Mattermost channels",
			expectedCompetitive: pm.CompetitorSlack,
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name: "Teams comparison competitive context",
			page: ConfluenceContent{
				Title: "Mattermost vs Teams Feature Comparison",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "COMP", Name: "Competitive"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "competitive"},
							{Name: "teams"},
						},
					},
				},
			},
			textContent:         "Feature comparison between Mattermost and Teams for enterprise deployments",
			expectedSegments:    []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCompetitive: pm.CompetitorTeams, // "teams" unambiguously matches Teams competitor
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name: "Playbooks workflow automation",
			page: ConfluenceContent{
				Title: "Playbooks for Incident Response Workflows",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "PLAY", Name: "Playbooks"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "playbooks"},
							{Name: "workflow"},
						},
					},
				},
			},
			textContent:        "Automated incident response playbooks with workflow triggers and status updates",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPlaybooks},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "Database performance tuning",
			page: ConfluenceContent{
				Title: "Database Performance Optimization Guide",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "DB", Name: "Database"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "database"},
							{Name: "performance"},
						},
					},
				},
			},
			textContent:        "Performance tuning for PostgreSQL and MySQL databases with indexing strategies",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryChannels, pm.CategoryPerformance, pm.CategoryDatabase}, // "indexing" might match channels
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "DevOps Kubernetes deployment",
			page: ConfluenceContent{
				Title: "Kubernetes Operator Deployment for DevOps",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "DEVOPS", Name: "DevOps"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "devops"},
							{Name: "kubernetes"},
						},
					},
				},
			},
			textContent:      "Deploy Mattermost on Kubernetes using the operator for DevOps teams with CI/CD integration",
			expectedSegments: []pm.CustomerSegment{pm.SegmentDevOps},
			expectedPriority: pm.PriorityMedium,
		},
		{
			name: "Calls plugin integration",
			page: ConfluenceContent{
				Title: "Calls Plugin Configuration Guide",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "CALLS", Name: "Calls"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "calls"},
							{Name: "plugin"},
						},
					},
				},
			},
			textContent:        "Configure the Calls plugin for video conferencing and screen sharing",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCalls, pm.CategoryPlugins},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "Boards kanban workflow",
			page: ConfluenceContent{
				Title: "Boards Kanban View Configuration",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "BOARDS", Name: "Boards"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "boards"},
							{Name: "kanban"},
						},
					},
				},
			},
			textContent:        "Configure kanban board views and card templates for project management",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryBoards},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "Page with Jira references",
			page: ConfluenceContent{
				Title: "Feature Implementation Plan - See MM-12345",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "PLAN", Name: "Planning"},
			},
			textContent:       "This feature is tracked in MM-12345 and relates to MM-98765. See Jira for details.",
			expectedJiraCount: 2,
			expectedPriority:  pm.PriorityMedium,
		},
		{
			name: "Page with GitHub references",
			page: ConfluenceContent{
				Title: "Code Review: mattermost/mattermost#12345",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "CODE", Name: "Code Review"},
			},
			textContent:         "This change is tracked in mattermost/mattermost#12345 and also fixes #678",
			expectedGitHubCount: 2,
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name: "Finance segment compliance",
			page: ConfluenceContent{
				Title: "Financial Services Compliance Requirements",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "FIN", Name: "Finance"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "finance"},
							{Name: "banking"},
						},
					},
				},
			},
			textContent:        "Banking regulations and financial services compliance for secure messaging",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFinance, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name: "SMB deployment guide",
			page: ConfluenceContent{
				Title: "Deployment Guide for Small Business",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "SMB", Name: "Small Business"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "smb"},
							{Name: "startup"},
						},
					},
				},
			},
			textContent:      "Quick start guide for small business and startup teams",
			expectedSegments: []pm.CustomerSegment{pm.SegmentSMB},
			expectedPriority: pm.PriorityMedium, // No low priority keywords, defaults to Medium
		},
		{
			name: "Channels organization best practices",
			page: ConfluenceContent{
				Title: "Channel Organization Best Practices",
				Space: struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				}{Key: "GUIDE", Name: "Guides"},
				Metadata: struct {
					Labels struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					} `json:"labels"`
				}{
					Labels: struct {
						Results []struct {
							Name string `json:"name"`
						} `json:"results"`
					}{
						Results: []struct {
							Name string `json:"name"`
						}{
							{Name: "channels"},
						},
					},
				},
			},
			textContent:        "Best practices for organizing channels and managing channel permissions",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryChannels},
			expectedPriority:   pm.PriorityMedium, // No low priority keywords, defaults to Medium
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractConfluenceMetadata(tt.page, tt.textContent)

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

func TestExtractConfluenceMetadata_EntityLinking(t *testing.T) {
	page := ConfluenceContent{
		Title: `Technical Design Document with References`,
		Space: struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		}{Key: "ENG", Name: "Engineering"},
		Metadata: struct {
			Labels struct {
				Results []struct {
					Name string `json:"name"`
				} `json:"results"`
			} `json:"labels"`
		}{
			Labels: struct {
				Results []struct {
					Name string `json:"name"`
				} `json:"results"`
			}{
				Results: []struct {
					Name string `json:"name"`
				}{
					{Name: "technical-design"},
				},
			},
		},
	}

	textContent := `This design addresses the requirements in MM-12345 and MM-67890.

Related GitHub issues:
- mattermost/mattermost#100
- mattermost/focalboard#200
- #300

Code changes:
- server/api/auth.go:45
- webapp/components/login.tsx:123

Commits:
- Fixed in commit a3f5c2d
- Also merged abc123def

Related documentation:
- https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345
- https://community.mattermost.com/core/pl/abc123def456`

	meta := extractConfluenceMetadata(page, textContent)

	assert.Len(t, meta.JiraTickets, 2, "Should extract 2 Jira tickets")
	assert.Equal(t, "MM-12345", meta.JiraTickets[0].Key)
	assert.Equal(t, "MM-67890", meta.JiraTickets[1].Key)

	totalGitHub := len(meta.GitHubIssues) + len(meta.GitHubPRs)
	assert.GreaterOrEqual(t, totalGitHub, 3, "Should extract at least 3 GitHub issues")

	assert.Len(t, meta.ModifiedFiles, 2, "Should extract 2 file references")
	assert.Equal(t, "server/api/auth.go", meta.ModifiedFiles[0].Path)
	assert.Equal(t, 45, meta.ModifiedFiles[0].StartLine)

	assert.Len(t, meta.Commits, 2, "Should extract 2 commits")
	assert.Equal(t, "a3f5c2d", meta.Commits[0].SHA)

	assert.Len(t, meta.ConfluencePages, 1, "Should extract 1 Confluence page")
	assert.Equal(t, "ENG", meta.ConfluencePages[0].Space)

	assert.Len(t, meta.MattermostLinks, 1, "Should extract 1 Mattermost permalink")
	assert.Equal(t, "core", meta.MattermostLinks[0].TeamName)
}
