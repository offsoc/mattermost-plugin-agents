// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
)

func TestDiscourseProtocol_GetType(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)
	if protocol.GetType() != DiscourseProtocolType {
		t.Errorf("Expected protocol type %s, got %s", DiscourseProtocolType, protocol.GetType())
	}
}

func TestDiscourseProtocol_SetAuth(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)

	// Test setting API key auth
	auth := AuthConfig{Type: AuthTypeAPIKey, Key: "test-api-key"}
	protocol.SetAuth(auth)

	if protocol.auth.Type != AuthTypeAPIKey {
		t.Errorf("Expected auth type %s, got %s", AuthTypeAPIKey, protocol.auth.Type)
	}
	if protocol.auth.Key != "test-api-key" {
		t.Errorf("Expected auth key 'test-api-key', got '%s'", protocol.auth.Key)
	}
}

func TestDiscourseProtocol_FormatDate(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid RFC3339 date",
			input:    "2023-01-15T10:30:00Z",
			expected: "2023-01-15 10:30",
		},
		{
			name:     "Empty date",
			input:    "",
			expected: "Unknown",
		},
		{
			name:     "Invalid date format",
			input:    "invalid-date",
			expected: "invalid-date",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := protocol.formatDate(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestDiscourseProtocol_StripHTML(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple paragraph",
			input:    "<p>Hello world</p>",
			expected: "\nHello world\n",
		},
		{
			name:     "Multiple tags",
			input:    "<div><strong>Bold</strong> and <em>italic</em> text</div>",
			expected: "Bold and italic text",
		},
		{
			name:     "Line breaks",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "Line breaks with spaces",
			input:    "Line 1<br />Line 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "No HTML",
			input:    "Plain text content",
			expected: "Plain text content",
		},
		{
			name:     "Script tag removal",
			input:    "<p>Content</p><script>alert('xss')</script><p>More content</p>",
			expected: "Content\n\nMore content",
		},
		{
			name:     "Style tag removal",
			input:    "<p>Content</p><style>.class { color: red; }</style><p>More content</p>",
			expected: "Content\n\nMore content",
		},
		{
			name:     "Noscript tag removal",
			input:    "<p>Content</p><noscript>No JS message</noscript><p>More content</p>",
			expected: "Content\n\nMore content",
		},
		{
			name:     "Tags with attributes",
			input:    `<div class="container"><span id="test">Text</span></div>`,
			expected: "Text",
		},
		{
			name:     "Inline CSS cleanup",
			input:    "<p>Text</p>color: red; font-size: 12px;",
			expected: "Text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := protocol.stripHTML(tc.input)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tc.expected)
			if result != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result)
			}
		})
	}
}

func TestDiscourseProtocol_FormatTopicContent(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)

	topic := DiscourseTopic{
		ID:           123,
		Title:        "Test Topic",
		PostsCount:   5,
		Views:        100,
		LikeCount:    10,
		CreatedAt:    "2023-01-15T10:30:00Z",
		LastPostedAt: "2023-01-16T15:45:00Z",
		Tags:         []string{"feature", "discussion"},
	}

	content := protocol.formatTopicContent(topic)

	// Check that all expected information is included
	expectedParts := []string{
		"Test Topic",
		"Posts: 5",
		"Views: 100",
		"Likes: 10",
		"2023-01-15 10:30",
		"2023-01-16 15:45",
		"feature, discussion",
		"Community discussion",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("Expected content to contain '%s', but it didn't. Content: %s", part, content)
		}
	}
}

func TestDiscourseProtocol_FormatPostContent(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)

	post := DiscoursePost{
		ID:         456,
		Username:   "testuser",
		PostNumber: 2,
		CreatedAt:  "2023-01-15T10:30:00Z",
		LikeCount:  3,
		Cooked:     "<p>This is a test post with <strong>some HTML</strong> content.</p>",
	}

	content := protocol.formatPostContent(post)

	// Check that all expected information is included
	expectedParts := []string{
		"Post #2",
		"testuser",
		"2023-01-15 10:30",
		"Likes: 3",
		"This is a test post with some HTML content.",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("Expected content to contain '%s', but it didn't. Content: %s", part, content)
		}
	}
}

func TestDiscourseProtocol_Close(t *testing.T) {
	protocol := NewDiscourseProtocol(nil, nil)

	// Should not error even without rate limiter
	err := protocol.Close()
	if err != nil {
		t.Errorf("Expected no error from Close(), got: %v", err)
	}

	// Create a rate limiter and test closing it
	protocol.rateLimiter = NewRateLimiter(10, 2)
	err = protocol.Close()
	if err != nil {
		t.Errorf("Expected no error from Close() with rate limiter, got: %v", err)
	}

	// Rate limiter should be nil after close
	if protocol.rateLimiter != nil {
		t.Error("Expected rate limiter to be nil after Close()")
	}
}

func TestConvertSearchResultsToDocs_WithStructuredLabels(t *testing.T) {
	protocol := &DiscourseProtocol{}

	searchResult := DiscourseSearchResult{
		Topics: []DiscourseTopic{
			{
				ID:           12345,
				Title:        "Jira plugin failing for enterprise customers",
				Slug:         "jira-plugin-failing-enterprise",
				PostsCount:   30,
				Views:        550,
				LikeCount:    28,
				CreatedAt:    "2025-09-10T10:00:00Z",
				LastPostedAt: "2025-09-29T12:00:00Z",
				Tags:         []string{"jira-plugin", "enterprise", "bug"},
			},
		},
		Posts: []DiscoursePost{},
	}

	docs := protocol.convertSearchResultsToDocs(searchResult, "https://community.mattermost.com", "bugs", "community_forum")

	if len(docs) != 1 {
		t.Fatalf("Expected 1 doc, got %d", len(docs))
	}

	doc := docs[0]

	// Check structured labels
	expectedLabels := []string{
		"priority:high",
		"segment:enterprise",
		"category:plugins",
		"high_engagement",
		"high_visibility",
		"tag:jira-plugin",
		"tag:enterprise",
		"tag:bug",
	}

	for _, expectedLabel := range expectedLabels {
		if !containsLabel(doc.Labels, expectedLabel) {
			t.Errorf("Expected label %s not found in doc labels: %v", expectedLabel, doc.Labels)
		}
	}
}
