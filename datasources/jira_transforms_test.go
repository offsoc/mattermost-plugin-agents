// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
	"time"

	"github.com/andygrunwald/go-jira"
	pm "github.com/mattermost/mattermost-plugin-ai/datasources/segments/pm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJiraProtocol_convertIssueToDoc(t *testing.T) {
	protocol := &JiraProtocol{
		htmlProcessor: NewHTMLProcessor(),
	}

	tests := []struct {
		name          string
		issue         jira.Issue
		sourceName    string
		expectedTitle string
		expectNil     bool
		checkContent  func(t *testing.T, content string)
		checkLabels   func(t *testing.T, labels []string)
		checkURL      func(t *testing.T, url string)
		checkSection  string
	}{
		{
			name: "Complete issue with all fields",
			issue: jira.Issue{
				Key:  "MM-12345",
				Self: "https://mattermost.atlassian.net/rest/api/2/issue/12345",
				Fields: &jira.IssueFields{
					Summary:     "Fix mobile authentication bug",
					Description: "Mobile app crashes on iOS for enterprise customers with SSO enabled",
					Status: &jira.Status{
						Name: "In Progress",
					},
					Type: jira.IssueType{
						Name: "Bug",
					},
					Priority: &jira.Priority{
						Name: "High",
					},
					Assignee: &jira.User{
						DisplayName: "John Doe",
					},
					Creator: &jira.User{
						DisplayName: "Jane Smith",
					},
					Labels:  []string{"mobile", "authentication", "enterprise"},
					Created: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
					Updated: jira.Time(time.Date(2025, 1, 10, 15, 30, 0, 0, time.UTC)),
				},
			},
			sourceName:    "jira_mattermost",
			expectedTitle: "[MM-12345] Fix mobile authentication bug",
			expectNil:     false,
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "**[MM-12345] Fix mobile authentication bug**")
				assert.Contains(t, content, "**Description:**")
				assert.Contains(t, content, "Mobile app crashes")
				assert.Contains(t, content, "**Details:**")
				assert.Contains(t, content, "Type: Bug")
				assert.Contains(t, content, "Status: In Progress")
				assert.Contains(t, content, "Assignee: John Doe")
			},
			checkLabels: func(t *testing.T, labels []string) {
				assert.Contains(t, labels, "mobile")
				assert.Contains(t, labels, "authentication")
				assert.Contains(t, labels, "enterprise")
				assert.Contains(t, labels, "jira:mobile")
				assert.Contains(t, labels, "jira:authentication")
				assert.Contains(t, labels, "jira:enterprise")
				assert.Contains(t, labels, "status:In Progress")
				assert.Contains(t, labels, "assignee:John Doe")
				assert.Contains(t, labels, "type:Bug")
			},
			checkURL: func(t *testing.T, url string) {
				assert.Equal(t, "https://mattermost.atlassian.net/browse/MM-12345", url)
			},
			checkSection: "bug",
		},
		{
			name: "Minimal issue",
			issue: jira.Issue{
				Key: "MM-99999",
				Fields: &jira.IssueFields{
					Summary: "Minimal issue",
					Type: jira.IssueType{
						Name: "Task",
					},
					Created: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
					Updated: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
				},
			},
			sourceName:    "jira_test",
			expectedTitle: "[MM-99999] Minimal issue",
			expectNil:     false,
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "**[MM-99999] Minimal issue**")
			},
			checkSection: "task",
		},
		{
			name: "Issue with nil fields",
			issue: jira.Issue{
				Key:    "MM-00000",
				Fields: nil,
			},
			sourceName: "jira_test",
			expectNil:  true,
		},
		{
			name: "Issue with empty summary and no content",
			issue: jira.Issue{
				Key: "MM-11111",
				Fields: &jira.IssueFields{
					Summary:     "",
					Description: "",
					Type: jira.IssueType{
						Name: "Task",
					},
					Created: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
					Updated: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
				},
			},
			sourceName: "jira_test",
			expectNil:  true,
		},
		{
			name: "Issue with metadata extraction",
			issue: jira.Issue{
				Key: "MM-22222",
				Fields: &jira.IssueFields{
					Summary:     "Enterprise mobile feature request",
					Description: "Need mobile app support for high-security deployments with SAML authentication",
					Type: jira.IssueType{
						Name: "Story",
					},
					Created: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
					Updated: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
				},
			},
			sourceName:    "jira_test",
			expectedTitle: "[MM-22222] Enterprise mobile feature request",
			expectNil:     false,
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "Priority:")
				assert.Contains(t, content, "Segments:")
				assert.Contains(t, content, "Categories:")
			},
			checkLabels: func(t *testing.T, labels []string) {
				hasSegmentLabel := false
				hasCategoryLabel := false
				for _, label := range labels {
					if strings.HasPrefix(label, "segment:") {
						hasSegmentLabel = true
					}
					if strings.HasPrefix(label, "category:") {
						hasCategoryLabel = true
					}
				}
				assert.True(t, hasSegmentLabel, "Should have at least one segment label")
				assert.True(t, hasCategoryLabel, "Should have at least one category label")
			},
		},
		{
			name: "Issue with fallback URL",
			issue: jira.Issue{
				Key:  "MM-33333",
				Self: "",
				Fields: &jira.IssueFields{
					Summary: "Test issue",
					Type: jira.IssueType{
						Name: "Bug",
					},
					Created: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
					Updated: jira.Time(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)),
				},
			},
			sourceName:    "jira_test",
			expectedTitle: "[MM-33333] Test issue",
			expectNil:     false,
			checkURL: func(t *testing.T, url string) {
				assert.Equal(t, "jira://issue/MM-33333", url)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := protocol.convertIssueToDoc(tt.issue, tt.sourceName)

			if tt.expectNil {
				assert.Nil(t, doc, "Expected nil document")
				return
			}

			require.NotNil(t, doc, "Expected non-nil document")
			assert.Equal(t, tt.expectedTitle, doc.Title)
			assert.Equal(t, tt.sourceName, doc.Source)

			if tt.checkContent != nil {
				tt.checkContent(t, doc.Content)
			}

			if tt.checkLabels != nil {
				tt.checkLabels(t, doc.Labels)
			}

			if tt.checkURL != nil {
				tt.checkURL(t, doc.URL)
			}

			if tt.checkSection != "" {
				assert.Equal(t, tt.checkSection, doc.Section)
			}

			assert.NotEmpty(t, doc.CreatedDate)
			assert.NotEmpty(t, doc.LastModified)
		})
	}
}

func TestJiraProtocol_formatIssueContentWithMetadata(t *testing.T) {
	protocol := &JiraProtocol{
		htmlProcessor: NewHTMLProcessor(),
	}

	tests := []struct {
		name         string
		issue        jira.Issue
		meta         EntityMetadata
		checkContent func(t *testing.T, content string)
	}{
		{
			name: "Issue with full metadata",
			issue: jira.Issue{
				Key: "MM-12345",
				Fields: &jira.IssueFields{
					Summary:     "Test issue",
					Description: "This is a test description",
					Type: jira.IssueType{
						Name: "Bug",
					},
					Status: &jira.Status{
						Name: "Open",
					},
					Assignee: &jira.User{
						DisplayName: "John Doe",
					},
				},
			},
			meta: EntityMetadata{
				RoleMetadata: pm.Metadata{
					Priority:    pm.PriorityHigh,
					Segments:    []pm.CustomerSegment{pm.SegmentEnterprise},
					Categories:  []pm.TechnicalCategory{pm.CategoryMobile},
					Competitive: pm.CompetitorSlack,
				},
			},
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "**[MM-12345] Test issue**")
				assert.Contains(t, content, "Priority: high")
				assert.Contains(t, content, "Segments: enterprise")
				assert.Contains(t, content, "Categories: mobile")
				assert.Contains(t, content, "Competitive: slack")
				assert.Contains(t, content, "**Description:**")
				assert.Contains(t, content, "This is a test description")
				assert.Contains(t, content, "**Details:**")
				assert.Contains(t, content, "Type: Bug")
				assert.Contains(t, content, "Status: Open")
				assert.Contains(t, content, "Assignee: John Doe")
			},
		},
		{
			name: "Issue with entity links",
			issue: jira.Issue{
				Key: "MM-99999",
				Fields: &jira.IssueFields{
					Summary:     "Issue with links",
					Description: "Referenced in PR",
					Type: jira.IssueType{
						Name: "Task",
					},
				},
			},
			meta: EntityMetadata{
				RoleMetadata: pm.Metadata{
					Priority: pm.PriorityMedium,
				},
				GitHubPRs: []GitHubReference{
					{Owner: "mattermost", Repo: "mattermost", Number: 100},
				},
				Commits: []CommitReference{
					{SHA: "abc123", ShortSHA: "abc123"},
				},
			},
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "**[MM-99999] Issue with links**")
				assert.Contains(t, content, "Priority: medium")
				assert.Contains(t, content, "**Additional References:**")
				assert.Contains(t, content, "GitHub PRs: #100")
				assert.Contains(t, content, "Commits:")
			},
		},
		{
			name: "Issue with minimal metadata",
			issue: jira.Issue{
				Key: "MM-00000",
				Fields: &jira.IssueFields{
					Summary: "Minimal issue",
					Type: jira.IssueType{
						Name: "Task",
					},
				},
			},
			meta: EntityMetadata{},
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "**[MM-00000] Minimal issue**")
				assert.NotContains(t, content, "Priority:")
				assert.NotContains(t, content, "Segments:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := protocol.formatIssueContentWithMetadata(tt.issue, tt.meta)
			require.NotEmpty(t, content)
			tt.checkContent(t, content)
		})
	}
}

func TestJiraProtocol_formatTime(t *testing.T) {
	protocol := &JiraProtocol{}

	tests := []struct {
		name     string
		jiraTime jira.Time
		expected string
	}{
		{
			name:     "Valid time",
			jiraTime: jira.Time(time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)),
			expected: "2025-01-15T14:30:45Z",
		},
		{
			name:     "Zero time",
			jiraTime: jira.Time(time.Time{}),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.formatTime(tt.jiraTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJiraProtocol_formatIssueContent_Legacy(t *testing.T) {
	protocol := &JiraProtocol{
		htmlProcessor: NewHTMLProcessor(),
	}

	issue := jira.Issue{
		Key: "MM-12345",
		Fields: &jira.IssueFields{
			Summary:     "Legacy format test",
			Description: "This is a description",
			Type: jira.IssueType{
				Name: "Bug",
			},
			Status: &jira.Status{
				Name: "In Progress",
			},
			Priority: &jira.Priority{
				Name: "High",
			},
			Assignee: &jira.User{
				DisplayName: "John Doe",
			},
			Reporter: &jira.User{
				DisplayName: "Jane Smith",
			},
			Labels: []string{"test", "legacy"},
		},
	}

	content := protocol.formatIssueContent(issue)

	assert.Contains(t, content, "# Legacy format test")
	assert.Contains(t, content, "## Description")
	assert.Contains(t, content, "This is a description")
	assert.Contains(t, content, "## Issue Details")
	assert.Contains(t, content, "- **Issue Key**: MM-12345")
	assert.Contains(t, content, "- **Type**: Bug")
	assert.Contains(t, content, "- **Status**: In Progress")
	assert.Contains(t, content, "- **Priority**: High")
	assert.Contains(t, content, "- **Assignee**: John Doe")
	assert.Contains(t, content, "- **Reporter**: Jane Smith")
	assert.Contains(t, content, "- **Labels**: test, legacy")
}
