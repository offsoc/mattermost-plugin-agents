// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
)

func TestExtractGitHubReferences(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		expectedIssues  int
		expectedPRs     int
		checkFirstIssue *GitHubReference
		checkFirstPR    *GitHubReference
	}{
		{
			name:            "Full GitHub issue URL",
			text:            "See https://github.com/mattermost/mattermost/issues/123",
			expectedIssues:  1,
			expectedPRs:     0,
			checkFirstIssue: &GitHubReference{Type: GHIssue, Owner: "mattermost", Repo: "mattermost", Number: 123, URL: "https://github.com/mattermost/mattermost/issues/123"},
		},
		{
			name:           "Full GitHub PR URL",
			text:           "See https://github.com/mattermost/focalboard/pull/456",
			expectedIssues: 0,
			expectedPRs:    1,
			checkFirstPR:   &GitHubReference{Type: GHPullRequest, Owner: "mattermost", Repo: "focalboard", Number: 456, URL: "https://github.com/mattermost/focalboard/pull/456"},
		},
		{
			name:            "Cross-repo reference",
			text:            "Related to mattermost/focalboard#789",
			expectedIssues:  1,
			expectedPRs:     0,
			checkFirstIssue: &GitHubReference{Type: GHIssue, Owner: "mattermost", Repo: "focalboard", Number: 789, URL: "https://github.com/mattermost/focalboard/issues/789"},
		},
		{
			name:            "Short form issue reference",
			text:            "Fixed in #42",
			expectedIssues:  1,
			expectedPRs:     0,
			checkFirstIssue: &GitHubReference{Type: GHIssue, Number: 42},
		},
		{
			name:            "GitHub Enterprise URL",
			text:            "See https://github.enterprise.company.com/team/repo/issues/999",
			expectedIssues:  1,
			expectedPRs:     0,
			checkFirstIssue: &GitHubReference{Type: GHIssue, Owner: "team", Repo: "repo", Number: 999, URL: "https://github.enterprise.company.com/team/repo/issues/999"},
		},
		{
			name:            "HTTP protocol (air-gapped)",
			text:            "Bug in http://git.internal.corp/backend/api/issues/111",
			expectedIssues:  1,
			expectedPRs:     0,
			checkFirstIssue: &GitHubReference{Type: GHIssue, Owner: "backend", Repo: "api", Number: 111, URL: "http://git.internal.corp/backend/api/issues/111"},
		},
		{
			name:           "Mixed references",
			text:           "See #10, https://github.com/mattermost/mattermost/issues/20, and mattermost/focalboard#30",
			expectedIssues: 3,
			expectedPRs:    0,
		},
		{
			name:            "Avoid Zendesk conflict - GitHub 1-4 digits",
			text:            "#1234 is GitHub, not Zendesk",
			expectedIssues:  1,
			expectedPRs:     0,
			checkFirstIssue: &GitHubReference{Type: GHIssue, Number: 1234},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, prs := extractGitHubReferences(tt.text)
			assert.Len(t, issues, tt.expectedIssues, "Unexpected number of issues")
			assert.Len(t, prs, tt.expectedPRs, "Unexpected number of PRs")

			if tt.checkFirstIssue != nil && len(issues) > 0 {
				assert.Equal(t, tt.checkFirstIssue.Type, issues[0].Type)
				assert.Equal(t, tt.checkFirstIssue.Number, issues[0].Number)
				if tt.checkFirstIssue.Owner != "" {
					assert.Equal(t, tt.checkFirstIssue.Owner, issues[0].Owner)
				}
				if tt.checkFirstIssue.Repo != "" {
					assert.Equal(t, tt.checkFirstIssue.Repo, issues[0].Repo)
				}
				if tt.checkFirstIssue.URL != "" {
					assert.Equal(t, tt.checkFirstIssue.URL, issues[0].URL)
				}
			}

			if tt.checkFirstPR != nil && len(prs) > 0 {
				assert.Equal(t, tt.checkFirstPR.Type, prs[0].Type)
				assert.Equal(t, tt.checkFirstPR.Number, prs[0].Number)
				if tt.checkFirstPR.Owner != "" {
					assert.Equal(t, tt.checkFirstPR.Owner, prs[0].Owner)
				}
				if tt.checkFirstPR.Repo != "" {
					assert.Equal(t, tt.checkFirstPR.Repo, prs[0].Repo)
				}
				if tt.checkFirstPR.URL != "" {
					assert.Equal(t, tt.checkFirstPR.URL, prs[0].URL)
				}
			}
		})
	}
}

func TestExtractJiraReferences(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedCount int
		checkFirstKey string
		checkFirstURL string
	}{
		{
			name:          "Jira key only",
			text:          "See MM-12345 for details",
			expectedCount: 1,
			checkFirstKey: "MM-12345",
		},
		{
			name:          "Jira URL",
			text:          "https://mattermost.atlassian.net/browse/MM-67890",
			expectedCount: 1,
			checkFirstKey: "MM-67890",
			checkFirstURL: "https://mattermost.atlassian.net/browse/MM-67890",
		},
		{
			name:          "Jira Data Center URL",
			text:          "https://jira.bigcorp.com/browse/PROJ-456",
			expectedCount: 1,
			checkFirstKey: "PROJ-456",
			checkFirstURL: "https://jira.bigcorp.com/browse/PROJ-456",
		},
		{
			name:          "Multiple Jira references",
			text:          "See MM-111, MM-222, and https://mattermost.atlassian.net/browse/MM-333",
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractJiraReferences(tt.text)
			assert.Len(t, refs, tt.expectedCount, "Unexpected number of Jira references")

			if tt.checkFirstKey != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkFirstKey, refs[0].Key)
			}

			if tt.checkFirstURL != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkFirstURL, refs[0].URL)
			}
		})
	}
}

func TestExtractCommitReferences(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedCount int
		checkFirstSHA string
	}{
		{
			name:          "Commit with keyword",
			text:          "Fixed in commit a3f5c2d",
			expectedCount: 1,
			checkFirstSHA: "a3f5c2d",
		},
		{
			name:          "Commit URL",
			text:          "https://github.com/mattermost/mattermost/commit/1a2b3c4d5e6f7890",
			expectedCount: 1,
			checkFirstSHA: "1a2b3c4d5e6f7890",
		},
		{
			name:          "Multiple commit keywords",
			text:          "Fixed abc1234, merged def5678",
			expectedCount: 2,
		},
		{
			name:          "Avoid false positives - no keyword",
			text:          "API key: deadbeef12345",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractCommitReferences(tt.text)
			assert.Len(t, refs, tt.expectedCount, "Unexpected number of commit references")

			if tt.checkFirstSHA != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkFirstSHA, refs[0].SHA)
			}
		})
	}
}

func TestExtractFileReferences(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedCount  int
		checkFirstPath string
		checkLanguage  string
		checkComponent string
		checkLine      int
	}{
		{
			name:           "Simple file path",
			text:           "Bug in server/api/auth.go",
			expectedCount:  1,
			checkFirstPath: "server/api/auth.go",
			checkLanguage:  "go",
			checkComponent: "server/api",
		},
		{
			name:           "File path with line number",
			text:           "See server/api/auth.go:45",
			expectedCount:  1,
			checkFirstPath: "server/api/auth.go",
			checkLine:      45,
		},
		{
			name:          "Flexible directories - shared, plugins, packages",
			text:          "Bug in shared/utils/logger.go and plugins/jira/sync.go and packages/client/api.ts",
			expectedCount: 3,
		},
		{
			name:          "TypeScript file",
			text:          "Issue in webapp/src/components/header.tsx",
			expectedCount: 1,
			checkLanguage: "typescript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractFileReferences(tt.text)
			assert.Len(t, refs, tt.expectedCount, "Unexpected number of file references")

			if tt.checkFirstPath != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkFirstPath, refs[0].Path)
			}

			if tt.checkLanguage != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkLanguage, refs[0].Language)
			}

			if tt.checkComponent != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkComponent, refs[0].Component)
			}

			if tt.checkLine > 0 && len(refs) > 0 {
				assert.Equal(t, tt.checkLine, refs[0].StartLine)
			}
		})
	}
}

func TestExtractConfluenceReferences(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedCount int
		checkSpace    string
		checkPageID   string
	}{
		{
			name:          "Confluence URL with page",
			text:          "https://mattermost.atlassian.net/wiki/spaces/ENG/pages/12345",
			expectedCount: 1,
			checkSpace:    "ENG",
			checkPageID:   "12345",
		},
		{
			name:          "Self-hosted Confluence",
			text:          "https://wiki.company.com/spaces/TEAM/pages/67890",
			expectedCount: 1,
			checkSpace:    "TEAM",
			checkPageID:   "67890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractConfluenceReferences(tt.text)
			assert.Len(t, refs, tt.expectedCount, "Unexpected number of Confluence references")

			if tt.checkSpace != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkSpace, refs[0].Space)
			}

			if tt.checkPageID != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkPageID, refs[0].PageID)
			}
		})
	}
}

func TestExtractZendeskReferences(t *testing.T) {
	tests := []struct {
		name             string
		text             string
		expectedCount    int
		checkFirstTicket string
	}{
		{
			name:             "Zendesk URL",
			text:             "https://mattermost.zendesk.com/tickets/123456",
			expectedCount:    1,
			checkFirstTicket: "123456",
		},
		{
			name:             "Zendesk short form - 5+ digits",
			text:             "#12345",
			expectedCount:    1,
			checkFirstTicket: "12345",
		},
		{
			name:          "Avoid GitHub conflict - 1-4 digits not matched",
			text:          "#1234",
			expectedCount: 0,
		},
		{
			name:             "Custom Zendesk domain",
			text:             "https://support.company.com/tickets/999999",
			expectedCount:    1,
			checkFirstTicket: "999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractZendeskReferences(tt.text)
			assert.Len(t, refs, tt.expectedCount, "Unexpected number of Zendesk references")

			if tt.checkFirstTicket != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkFirstTicket, refs[0].TicketID)
			}
		})
	}
}

func TestExtractMattermostReferences(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedCount int
		checkTeam     string
		checkPostID   string
	}{
		{
			name:          "Mattermost permalink",
			text:          "https://community.mattermost.com/core/pl/abc123def456",
			expectedCount: 1,
			checkTeam:     "core",
			checkPostID:   "abc123def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractMattermostReferences(tt.text)
			assert.Len(t, refs, tt.expectedCount, "Unexpected number of Mattermost references")

			if tt.checkTeam != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkTeam, refs[0].TeamName)
			}

			if tt.checkPostID != "" && len(refs) > 0 {
				assert.Equal(t, tt.checkPostID, refs[0].PostID)
			}
		})
	}
}

func TestFormatEntityMetadata(t *testing.T) {
	meta := EntityMetadata{
		RoleMetadata: pm.Metadata{
			Segments:    []pm.CustomerSegment{pm.SegmentEnterprise, pm.SegmentSMB},
			Categories:  []pm.TechnicalCategory{pm.CategoryAuthentication, pm.CategoryPerformance},
			Competitive: pm.CompetitorSlack,
			Priority:    pm.PriorityHigh,
		},
		GitHubIssues: []GitHubReference{
			{Type: GHIssue, Number: 123, URL: "https://github.com/mattermost/mattermost/issues/123"},
		},
		JiraTickets: []JiraReference{
			{Key: "MM-456", Project: "MM", Number: 456},
		},
		ModifiedFiles: []FileReference{
			{Path: "server/api/auth.go", StartLine: 45, Language: "go", Component: "server/api"},
		},
	}

	result := formatEntityMetadata(meta)

	// New inline format: (Priority: high | Segments: enterprise, smb | Categories: authentication, performance | Competitive: slack | Refs: MM-456, #123)
	assert.Contains(t, result, "Priority: high")
	assert.Contains(t, result, "Segments: enterprise, smb")
	assert.Contains(t, result, "Categories: authentication, performance")
	assert.Contains(t, result, "Competitive: slack")
	assert.Contains(t, result, "Refs: MM-456, #123")
	assert.Contains(t, result, "(")
	assert.Contains(t, result, ")")
	assert.Contains(t, result, "|")

	// Test detailed metadata format (has URLs)
	detailedResult := formatEntityMetadataDetailed(meta)
	assert.Contains(t, detailedResult, "GitHub Issues: https://github.com/mattermost/mattermost/issues/123")
	assert.Contains(t, detailedResult, "Jira Tickets: MM-456")
	assert.Contains(t, detailedResult, "Modified Files: server/api/auth.go:45")
}

func TestBuildLabelsFromMetadata(t *testing.T) {
	meta := EntityMetadata{
		RoleMetadata: pm.Metadata{
			Segments:    []pm.CustomerSegment{pm.SegmentEnterprise},
			Categories:  []pm.TechnicalCategory{pm.CategoryAuthentication},
			Competitive: pm.CompetitorSlack,
			Priority:    pm.PriorityHigh,
		},
	}

	labels := buildLabelsFromMetadata(meta)

	assert.Contains(t, labels, "segment:enterprise")
	assert.Contains(t, labels, "category:authentication")
	assert.Contains(t, labels, "competitive:slack")
	assert.Contains(t, labels, "priority:high")
}
