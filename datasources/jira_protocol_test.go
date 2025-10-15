// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/andygrunwald/go-jira"
)

func TestJiraProtocol_GetType(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)
	if protocol.GetType() != JiraProtocolType {
		t.Errorf("Expected JiraProtocolType, got %v", protocol.GetType())
	}
}

func TestJiraProtocol_SetAuth(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)
	auth := AuthConfig{Type: AuthTypeAPIKey, Key: "user@example.com:api-token-123"}

	protocol.SetAuth(auth)

	if protocol.auth.Type != AuthTypeAPIKey || protocol.auth.Key != "user@example.com:api-token-123" {
		t.Errorf("Auth not set correctly: %+v", protocol.auth)
	}
}

func TestJiraProtocol_BuildJQLQuery(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	tests := []struct {
		name          string
		topic         string
		sections      []string
		expectedJQL   []string
		unexpectedJQL []string
	}{
		{
			name:     "empty topic and sections",
			topic:    "",
			sections: []string{},
			expectedJQL: []string{
				"updated >= -30d",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "simple topic only",
			topic:    "mobile app bug",
			sections: []string{},
			expectedJQL: []string{
				"text ~",
				"summary ~ \"mobile app bug\"",
				"description ~ \"mobile app bug\"",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "sections only",
			topic:    "",
			sections: []string{"bug", "task"},
			expectedJQL: []string{
				"issueType in (\"Bug\", \"Task\")",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "topic and sections",
			topic:    "authentication",
			sections: []string{"bug"},
			expectedJQL: []string{
				"text ~",
				"summary ~ \"authentication\"",
				"description ~ \"authentication\"",
				"issueType in (\"Bug\")",
				"ORDER BY updated DESC",
			},
		},
		{
			name:     "multiple sections",
			topic:    "api",
			sections: []string{"bug", "task", "story"},
			expectedJQL: []string{
				"text ~",
				"issueType in (\"Bug\", \"Task\", \"Story\")",
				"ORDER BY updated DESC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.buildJQLQuery(tt.topic, tt.sections)

			for _, expected := range tt.expectedJQL {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected JQL to contain '%s', got '%s'", expected, result)
				}
			}

			for _, unexpected := range tt.unexpectedJQL {
				if strings.Contains(result, unexpected) {
					t.Errorf("Expected JQL not to contain '%s', got '%s'", unexpected, result)
				}
			}

			// All JQL queries should end with ordering
			if !strings.HasSuffix(result, "ORDER BY updated DESC") {
				t.Errorf("Expected JQL to end with ordering, got '%s'", result)
			}
		})
	}
}

func TestJiraProtocol_FormatIssueContent(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	// Mock Jira issue with various fields
	issue := jira.Issue{
		Key: "MM-12345",
		Fields: &jira.IssueFields{
			Summary:     "Authentication bug in mobile app",
			Description: "Users experiencing login issues with SSO authentication",
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
				DisplayName: "John Developer",
			},
			Reporter: &jira.User{
				DisplayName: "Jane Tester",
			},
			Labels: []string{"mobile", "authentication", "sso"},
			Comments: &jira.Comments{
				Comments: []*jira.Comment{
					{
						Author: jira.User{
							DisplayName: "Tech Lead",
						},
						Body: "Related to OAuth library update",
					},
					{
						Author: jira.User{
							DisplayName: "QA Engineer",
						},
						Body: "Reproducible on iOS and Android",
					},
				},
			},
		},
	}

	content := protocol.formatIssueContent(issue)

	// Check that content includes all important information
	if !strings.Contains(content, "MM-12345") {
		t.Errorf("Expected content to contain issue key")
	}
	if !strings.Contains(content, "Authentication bug in mobile app") {
		t.Errorf("Expected content to contain summary")
	}
	if !strings.Contains(content, "Users experiencing login issues") {
		t.Errorf("Expected content to contain description")
	}
	if !strings.Contains(content, "Bug") {
		t.Errorf("Expected content to contain issue type")
	}
	if !strings.Contains(content, "In Progress") {
		t.Errorf("Expected content to contain status")
	}
	if !strings.Contains(content, "High") {
		t.Errorf("Expected content to contain priority")
	}
	if !strings.Contains(content, "John Developer") {
		t.Errorf("Expected content to contain assignee")
	}
	if !strings.Contains(content, "Jane Tester") {
		t.Errorf("Expected content to contain reporter")
	}
	if !strings.Contains(content, "mobile, authentication, sso") {
		t.Errorf("Expected content to contain labels")
	}
	if !strings.Contains(content, "OAuth library") {
		t.Errorf("Expected content to contain comment text")
	}
	if !strings.Contains(content, "Recent Comments") {
		t.Errorf("Expected content to contain comments section")
	}

	// Check that content is well-structured with markdown
	if !strings.Contains(content, "# Authentication bug in mobile app") {
		t.Errorf("Expected content to have markdown title")
	}
	if !strings.Contains(content, "## Description") {
		t.Errorf("Expected content to have description section")
	}
	if !strings.Contains(content, "## Issue Details") {
		t.Errorf("Expected content to have issue details section")
	}
}

func TestJiraProtocol_ConvertIssueToDoc(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	now := time.Now()
	jiraTime := jira.Time(now)

	issue := jira.Issue{
		Key:  "MM-12345",
		Self: "https://mattermost.atlassian.net/rest/api/2/issue/123456",
		Fields: &jira.IssueFields{
			Summary:     "Mobile authentication bug",
			Description: "Authentication bug affecting mobile users",
			Type: jira.IssueType{
				Name: "Bug",
			},
			Status: &jira.Status{
				Name: "Open",
			},
			Creator: &jira.User{
				DisplayName: "Bug Reporter",
			},
			Labels:  []string{"mobile", "authentication"},
			Updated: jiraTime,
			Created: jiraTime,
		},
	}

	doc := protocol.convertIssueToDoc(issue, SourceJiraDocs)

	if doc == nil {
		t.Fatal("Expected non-nil doc")
	}

	expectedTitle := "[MM-12345] Mobile authentication bug"
	if doc.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, doc.Title)
	}

	if doc.Source != SourceJiraDocs {
		t.Errorf("Expected source '%s', got '%s'", SourceJiraDocs, doc.Source)
	}

	if doc.Section != "bug" {
		t.Errorf("Expected section 'bug', got '%s'", doc.Section)
	}

	expectedURL := "https://mattermost.atlassian.net/browse/MM-12345"
	if doc.URL != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, doc.URL)
	}

	if doc.Author != "Bug Reporter" {
		t.Errorf("Expected author 'Bug Reporter', got '%s'", doc.Author)
	}

	// Labels include extracted metadata (segments, categories, priority) + original Jira labels
	// Just verify the original Jira labels are present
	foundMobile := false
	foundAuth := false
	for _, label := range doc.Labels {
		if label == "mobile" {
			foundMobile = true
		}
		if label == "authentication" {
			foundAuth = true
		}
	}
	if !foundMobile || !foundAuth {
		t.Errorf("Expected labels to contain 'mobile' and 'authentication', got %v", doc.Labels)
	}

	if !strings.Contains(doc.Content, "Mobile authentication bug") {
		t.Errorf("Expected content to contain issue summary")
	}

	if !strings.Contains(doc.Content, "Authentication bug") {
		t.Errorf("Expected content to contain description")
	}
}

func TestJiraProtocol_ConvertIssueToDoc_EmptyFields(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	issue := jira.Issue{
		Key: "MM-12345",
		Fields: &jira.IssueFields{
			Summary: "", // Empty summary should be rejected
		},
	}

	doc := protocol.convertIssueToDoc(issue, SourceJiraDocs)

	// Should return nil due to empty summary and content
	if doc != nil {
		t.Errorf("Expected nil doc for empty summary and content, got %+v", doc)
	}

	// Test with short but non-empty summary - should be accepted
	issue.Fields.Summary = "Short"
	doc = protocol.convertIssueToDoc(issue, SourceJiraDocs)

	// Should be accepted now as JIRA issues with any summary are valid
	if doc == nil {
		t.Error("Expected doc for issue with short summary, got nil")
	}
}

func TestJiraProtocol_ConvertIssueToDoc_NilFields(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	issue := jira.Issue{
		Key:    "MM-12345",
		Fields: nil,
	}

	doc := protocol.convertIssueToDoc(issue, SourceJiraDocs)

	if doc != nil {
		t.Errorf("Expected nil doc for nil fields, got %+v", doc)
	}
}

func TestJiraProtocol_FormatTime(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	// Test with valid time
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	jiraTime := jira.Time(now)
	result := protocol.formatTime(jiraTime)
	expected := "2024-01-15T10:30:00Z"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with zero time
	zeroTime := jira.Time(time.Time{})
	result = protocol.formatTime(zeroTime)
	if result != "" {
		t.Errorf("Expected empty string for zero time, got '%s'", result)
	}
}

func TestJiraProtocol_Fetch_MissingBaseURL(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Name:      SourceJiraDocs,
		Endpoints: map[string]string{
			// Missing base URL
		},
		RateLimit: RateLimitConfig{
			Enabled: false,
		},
	}

	request := ProtocolRequest{
		Source:   source,
		Topic:    "test",
		Sections: []string{SectionBug},
		Limit:    5,
	}

	docs, err := protocol.Fetch(context.Background(), request)
	if err == nil {
		t.Error("Expected error for missing base URL")
	}
	if docs != nil {
		t.Errorf("Expected nil docs for missing base URL, got %d", len(docs))
	}
}

func TestJiraProtocol_Fetch_AuthenticationNotConfigured(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Name: SourceJiraDocs,
		Endpoints: map[string]string{
			EndpointBaseURL: "https://test.atlassian.net",
		},
		RateLimit: RateLimitConfig{
			Enabled: false,
		},
	}

	request := ProtocolRequest{
		Source:   source,
		Topic:    "test",
		Sections: []string{SectionBug},
		Limit:    5,
	}

	docs, err := protocol.Fetch(context.Background(), request)
	if err == nil {
		t.Error("Expected error for missing authentication")
	}
	if docs != nil {
		t.Errorf("Expected nil docs for missing authentication, got %d", len(docs))
	}
}

func TestJiraProtocol_Close(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	// Should not panic
	err := protocol.Close()
	if err != nil {
		t.Errorf("Unexpected error on close: %v", err)
	}

	// Should be able to close multiple times
	err = protocol.Close()
	if err != nil {
		t.Errorf("Unexpected error on second close: %v", err)
	}
}
