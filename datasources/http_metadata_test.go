// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractHTTPMetadata(t *testing.T) {
	tests := []struct {
		name                string
		title               string
		content             string
		url                 string
		expectedSegments    []pm.CustomerSegment
		expectedCategories  []pm.TechnicalCategory
		expectedCompetitive pm.Competitor
		expectedPriority    pm.Priority
		expectedJiraCount   int
		expectedGitHubCount int
	}{
		{
			name:               "Mobile app documentation for enterprise",
			title:              "Mobile App Deployment for Enterprise Customers",
			content:            "Deploy Mattermost mobile apps for enterprise. Covers iOS and Android configuration with SSO.",
			url:                "https://docs.mattermost.com/admin/mobile-deployment.html",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryMobile, pm.CategoryAuthentication},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Federal SAML authentication",
			title:              "SAML Configuration for Federal Deployments",
			content:            "Configure SAML authentication for high-security environments. Covers FedRAMP requirements.",
			url:                "https://docs.mattermost.com/admin/federal-saml.html",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFederal, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryAuthentication}, // FedRAMP doesn't match compliance keyword
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Healthcare HIPAA compliance",
			title:              "HIPAA Compliance Guide for Healthcare",
			content:            "Ensure HIPAA compliance in healthcare deployments. Configure audit logs and encryption for medical data.",
			url:                "https://docs.mattermost.com/admin/healthcare-compliance.html",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentHealthcare, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "DevOps CI/CD integration",
			title:              "CI/CD Integration for DevOps Teams",
			content:            "Integrate Mattermost with CI/CD pipelines. Automate notifications and workflows for DevOps.",
			url:                "https://docs.mattermost.com/integrations/cicd.html",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentDevOps},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryIntegrations, pm.CategoryPlugins, pm.CategoryChannels, pm.CategoryPlaybooks}, // "workflows" matches multiple
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Database performance tuning",
			title:              "Database Performance Optimization",
			content:            "Optimize database performance for large enterprise deployments. Covers indexing and query optimization.",
			url:                "https://docs.mattermost.com/admin/database-performance.html",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryDatabase, pm.CategoryPerformance},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:                "Slack migration guide",
			title:               "Migrating from Slack",
			content:             "Complete guide for migrating from Slack to Mattermost. Import channels, users, and messages.",
			url:                 "https://docs.mattermost.com/admin/slack-migration.html",
			expectedCompetitive: pm.CompetitorSlack,
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name:                "Microsoft comparison",
			title:               "Comparing Mattermost with Microsoft Solutions",
			content:             "Feature comparison between Mattermost and Microsoft collaboration tools.",
			url:                 "https://docs.mattermost.com/about/comparisons.html",
			expectedCompetitive: pm.CompetitorMicrosoftAny,
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name:               "Playbooks automation",
			title:              "Automating Workflows with Playbooks",
			content:            "Create automated incident response workflows using Playbooks. Coordinate team responses.",
			url:                "https://docs.mattermost.com/playbooks/automation.html",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPlaybooks},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Calls feature setup",
			title:              "Setting Up Mattermost Calls",
			content:            "Configure Calls plugin for voice and video communication. Enable screen sharing.",
			url:                "https://docs.mattermost.com/calls/setup.html",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCalls, pm.CategoryPlugins},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Boards project management",
			title:              "Project Management with Boards",
			content:            "Use Boards for project management. Create kanban boards and task cards.",
			url:                "https://docs.mattermost.com/boards/overview.html",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryBoards},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Channels organization",
			title:              "Organizing Channels Effectively",
			content:            "Best practices for channel organization and management. Structure channels for large teams.",
			url:                "https://docs.mattermost.com/channels/organization.html",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryChannels},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:               "Finance sector deployment",
			title:              "Mattermost for Financial Services",
			content:            "Deploy Mattermost in financial services and banking. Meet compliance regulations.",
			url:                "https://docs.mattermost.com/solutions/finance.html",
			expectedSegments:   []pm.CustomerSegment{pm.SegmentFinance, pm.SegmentEnterprise},
			expectedCategories: []pm.TechnicalCategory{pm.CategoryCompliance},
			expectedPriority:   pm.PriorityMedium,
		},
		{
			name:             "SMB deployment guide",
			title:            "Deployment Guide for Small Business",
			content:          "Quick setup guide for small business and startup teams. Get started quickly.",
			url:              "https://docs.mattermost.com/guides/smb-deployment.html",
			expectedSegments: []pm.CustomerSegment{pm.SegmentSMB},
			expectedPriority: pm.PriorityMedium,
		},
		{
			name:              "Documentation with Jira reference",
			title:             "Authentication Fix Documentation",
			content:           "This issue was fixed in MM-12345. See the Jira ticket for implementation details.",
			url:               "https://docs.mattermost.com/admin/auth-fix.html",
			expectedJiraCount: 1,
			expectedPriority:  pm.PriorityMedium,
		},
		{
			name:                "Documentation with GitHub references",
			title:               "Performance Improvements",
			content:             "Performance improvements tracked in mattermost/mattermost#12345 and #678.",
			url:                 "https://docs.mattermost.com/about/performance.html",
			expectedGitHubCount: 2,
			expectedPriority:    pm.PriorityMedium,
		},
		{
			name:               "Plugin integration guide",
			title:              "Plugin Integration Best Practices",
			content:            "Best practices for integrating plugins. Covers plugin development and deployment.",
			url:                "https://docs.mattermost.com/plugins/integration.html",
			expectedCategories: []pm.TechnicalCategory{pm.CategoryPlugins, pm.CategoryIntegrations},
			expectedPriority:   pm.PriorityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := extractHTTPMetadata(tt.title, tt.content, tt.url)

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

func TestExtractHTTPMetadata_EntityLinking(t *testing.T) {
	title := "Troubleshooting Guide"
	content := `Comprehensive troubleshooting documentation with multiple references:
- Jira ticket: MM-12345 tracks the main issue
- GitHub issue: mattermost/mattermost#100 for the server fix
- Another issue: #200 for the webapp
- Commit: fixed a3f5c2d
- Modified file: server/api/auth.go:45
- Confluence page: https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345
- Community post: https://community.mattermost.com/core/pl/abc123def456`
	url := "https://docs.mattermost.com/troubleshooting/auth.html"

	meta := extractHTTPMetadata(title, content, url)

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
