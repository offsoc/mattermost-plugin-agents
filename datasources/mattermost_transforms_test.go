// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMattermostTransformer_ConvertPostsToDoc(t *testing.T) {
	transformer := NewMattermostTransformer()
	baseURL := "https://community.mattermost.com"
	sourceName := "mattermost_community"

	posts := map[string]MattermostPost{
		"post1": {
			ID:         "post1",
			Message:    "This is a test post about mobile authentication for enterprise customers",
			CreateAt:   time.Now().Unix() * 1000,
			EditAt:     0,
			UserID:     "user1",
			IsPinned:   true,
			ReplyCount: 15,
		},
		"post2": {
			ID:         "post2",
			Message:    "Another post about database performance",
			CreateAt:   time.Now().Unix() * 1000,
			EditAt:     time.Now().Unix() * 1000,
			UserID:     "user2",
			IsPinned:   false,
			ReplyCount: 5,
		},
		"post3": {
			ID:         "post3",
			Message:    "Reply to previous post",
			CreateAt:   time.Now().Unix() * 1000,
			UserID:     "user3",
			RootID:     "post2",
			ReplyCount: 0,
		},
	}

	order := []string{"post1", "post2", "post3"}

	docs := transformer.ConvertPostsToDoc(posts, order, baseURL, sourceName, "")

	require.NotEmpty(t, docs)

	for _, doc := range docs {
		assert.NotEmpty(t, doc.Title)
		assert.NotEmpty(t, doc.Content)
		assert.NotEmpty(t, doc.URL)
		assert.Equal(t, "posts", doc.Section)
		assert.Equal(t, sourceName, doc.Source)
		assert.NotEmpty(t, doc.Labels)
		assert.NotEmpty(t, doc.Author)
		assert.NotEmpty(t, doc.CreatedDate)
		assert.NotEmpty(t, doc.LastModified)
	}

	doc1Found := false
	for _, doc := range docs {
		if strings.Contains(doc.URL, "post1") {
			doc1Found = true
			assert.Contains(t, doc.Labels, "pinned")
			assert.Contains(t, doc.Labels, "has_replies")
			break
		}
	}
	assert.True(t, doc1Found, "Should find post1 in converted docs")
}

func TestMattermostTransformer_FilterAndConvertPosts(t *testing.T) {
	transformer := NewMattermostTransformer()
	baseURL := "https://community.mattermost.com"
	sourceName := "mattermost_community"
	teamName := "core"

	channel := &MattermostChannel{
		ID:   "channel1",
		Name: "developers",
	}

	posts := []MattermostPost{
		{
			ID:         "post1",
			Message:    "Discussion about mobile authentication and SSO",
			CreateAt:   time.Now().Unix() * 1000,
			UserID:     "user1",
			IsPinned:   true,
			ReplyCount: 20,
		},
		{
			ID:         "post2",
			Message:    "Database performance optimization tips",
			CreateAt:   time.Now().Unix() * 1000,
			UserID:     "user2",
			IsPinned:   false,
			ReplyCount: 10,
		},
		{
			ID:       "post3",
			Message:  "Off-topic message about lunch",
			CreateAt: time.Now().Unix() * 1000,
			UserID:   "user3",
		},
		{
			ID:       "post4",
			Message:  "This is a deleted post",
			CreateAt: time.Now().Unix() * 1000,
			DeleteAt: time.Now().Unix() * 1000,
			UserID:   "user4",
		},
		{
			ID:       "post5",
			Type:     "system_join_channel",
			Message:  "User joined the channel",
			CreateAt: time.Now().Unix() * 1000,
			UserID:   "user5",
		},
	}

	tests := []struct {
		name         string
		topic        string
		expectedMin  int
		checkLabels  func(t *testing.T, doc Doc)
		checkContent func(t *testing.T, doc Doc)
	}{
		{
			name:        "No topic filter",
			topic:       "",
			expectedMin: 3,
			checkLabels: func(t *testing.T, doc Doc) {
				assert.Contains(t, doc.Labels, "channel:developers")
				assert.Contains(t, doc.Labels, "team:core")
			},
			checkContent: func(t *testing.T, doc Doc) {
				assert.Contains(t, doc.Content, "From ~developers")
				assert.Contains(t, doc.Content, "in core team")
			},
		},
		{
			name:        "Topic filter for authentication",
			topic:       "authentication",
			expectedMin: 1,
			checkLabels: func(t *testing.T, doc Doc) {
				if strings.Contains(doc.Content, "authentication") {
					assert.Contains(t, doc.Labels, "pinned")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := transformer.FilterAndConvertPosts(posts, tt.topic, channel, baseURL, sourceName, "community", teamName)

			assert.GreaterOrEqual(t, len(docs), tt.expectedMin)

			for _, doc := range docs {
				assert.NotContains(t, doc.Content, "deleted post")
				assert.NotContains(t, doc.Content, "system_")

				if tt.checkLabels != nil {
					tt.checkLabels(t, doc)
				}
				if tt.checkContent != nil {
					tt.checkContent(t, doc)
				}
			}
		})
	}
}

func TestMattermostTransformer_FormatPostContent(t *testing.T) {
	transformer := NewMattermostTransformer()

	tests := []struct {
		name         string
		post         MattermostPost
		checkContent func(t *testing.T, content string)
	}{
		{
			name: "Basic post",
			post: MattermostPost{
				Message:  "This is a test message",
				CreateAt: time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC).Unix() * 1000,
			},
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "Posted:")
				assert.Contains(t, content, "This is a test message")
				assert.NotContains(t, content, "Edited:")
				assert.NotContains(t, content, "Replies:")
			},
		},
		{
			name: "Edited post with replies",
			post: MattermostPost{
				Message:    "Edited message",
				CreateAt:   time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC).Unix() * 1000,
				EditAt:     time.Date(2025, 1, 15, 15, 45, 0, 0, time.UTC).Unix() * 1000,
				ReplyCount: 5,
			},
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "Posted:")
				assert.Contains(t, content, "Edited:")
				assert.Contains(t, content, "Replies: 5")
				assert.Contains(t, content, "Edited message")
			},
		},
		{
			name: "Long post gets truncated",
			post: MattermostPost{
				Message:  strings.Repeat("x", 1500),
				CreateAt: time.Now().Unix() * 1000,
			},
			checkContent: func(t *testing.T, content string) {
				assert.LessOrEqual(t, len(content), 1050)
				assert.Contains(t, content, "...")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := transformer.FormatPostContent(tt.post)
			require.NotEmpty(t, content)
			tt.checkContent(t, content)
		})
	}
}

func TestMattermostTransformer_FormatPostContentWithChannel(t *testing.T) {
	transformer := NewMattermostTransformer()

	channel := &MattermostChannel{
		ID:   "channel1",
		Name: "developers",
	}

	tests := []struct {
		name         string
		post         MattermostPost
		teamName     string
		checkContent func(t *testing.T, content string)
	}{
		{
			name: "Post with team context",
			post: MattermostPost{
				Message:    "Test message",
				CreateAt:   time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC).Unix() * 1000,
				IsPinned:   true,
				ReplyCount: 10,
			},
			teamName: "core",
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "From ~developers")
				assert.Contains(t, content, "in core team")
				assert.Contains(t, content, "Posted:")
				assert.Contains(t, content, "📌 Pinned message")
				assert.Contains(t, content, "💬 10 replies")
				assert.Contains(t, content, "Test message")
			},
		},
		{
			name: "Post without team context",
			post: MattermostPost{
				Message:  "Simple message",
				CreateAt: time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC).Unix() * 1000,
			},
			teamName: "",
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "From ~developers")
				assert.NotContains(t, content, "in")
				assert.NotContains(t, content, "team")
				assert.Contains(t, content, "Simple message")
			},
		},
		{
			name: "Edited post",
			post: MattermostPost{
				Message:  "Edited content",
				CreateAt: time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC).Unix() * 1000,
				EditAt:   time.Date(2025, 1, 15, 15, 45, 0, 0, time.UTC).Unix() * 1000,
			},
			teamName: "core",
			checkContent: func(t *testing.T, content string) {
				assert.Contains(t, content, "Edited:")
			},
		},
		{
			name: "Long post gets truncated",
			post: MattermostPost{
				Message:  strings.Repeat("y", 1500),
				CreateAt: time.Now().Unix() * 1000,
			},
			teamName: "core",
			checkContent: func(t *testing.T, content string) {
				assert.LessOrEqual(t, len(content), 1100)
				assert.Contains(t, content, "...")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := transformer.FormatPostContentWithChannel(tt.post, channel, tt.teamName)
			require.NotEmpty(t, content)
			tt.checkContent(t, content)
		})
	}
}

func TestMattermostTransformer_GeneratePostTitle(t *testing.T) {
	transformer := NewMattermostTransformer()

	tests := []struct {
		name          string
		post          MattermostPost
		expectedTitle string
	}{
		{
			name: "Short message",
			post: MattermostPost{
				Message: "Short test message",
			},
			expectedTitle: "Short test message",
		},
		{
			name: "Long single line",
			post: MattermostPost{
				Message: strings.Repeat("x", 100),
			},
			expectedTitle: strings.Repeat("x", 50) + "...",
		},
		{
			name: "Multi-line message",
			post: MattermostPost{
				Message: "First line is the title\nSecond line\nThird line",
			},
			expectedTitle: "First line is the title",
		},
		{
			name: "Empty message",
			post: MattermostPost{
				Message: "",
			},
			expectedTitle: "Mattermost Post",
		},
		{
			name: "Message with only whitespace first line",
			post: MattermostPost{
				Message: "   \nSecond line",
			},
			expectedTitle: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := transformer.GeneratePostTitle(tt.post)
			assert.Equal(t, tt.expectedTitle, title)
		})
	}
}

func TestMattermostTransformer_FormatTimestamp(t *testing.T) {
	transformer := NewMattermostTransformer()

	testTime := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
	expectedFormat := testTime.Local().Format("2006-01-02 15:04")

	tests := []struct {
		name      string
		timestamp int64
		expected  string
	}{
		{
			name:      "Valid timestamp",
			timestamp: testTime.Unix() * 1000,
			expected:  expectedFormat,
		},
		{
			name:      "Zero timestamp",
			timestamp: 0,
			expected:  "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.FormatTimestamp(tt.timestamp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMattermostTransformer_BuildPostURL(t *testing.T) {
	transformer := NewMattermostTransformer()

	tests := []struct {
		name     string
		baseURL  string
		teamName string
		postID   string
		expected string
	}{
		{
			name:     "With team name",
			baseURL:  "https://community.mattermost.com",
			teamName: "core",
			postID:   "post123",
			expected: "https://community.mattermost.com/core/pl/post123",
		},
		{
			name:     "Without team name",
			baseURL:  "https://community.mattermost.com",
			teamName: "",
			postID:   "post456",
			expected: "https://community.mattermost.com/pl/post456",
		},
		{
			name:     "Base URL with trailing slash",
			baseURL:  "https://community.mattermost.com/",
			teamName: "contributors",
			postID:   "post789",
			expected: "https://community.mattermost.com/contributors/pl/post789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := transformer.BuildPostURL(tt.baseURL, tt.teamName, tt.postID)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestMattermostTransformer_PostMatchesTopic(t *testing.T) {
	transformer := NewMattermostTransformer()

	tests := []struct {
		name        string
		post        MattermostPost
		topic       string
		shouldMatch bool
	}{
		{
			name: "Exact keyword match",
			post: MattermostPost{
				Message: "Discussion about mobile authentication issues",
			},
			topic:       "authentication",
			shouldMatch: true,
		},
		{
			name: "No match",
			post: MattermostPost{
				Message: "Discussion about database performance",
			},
			topic:       "mobile",
			shouldMatch: false,
		},
		{
			name: "Partial word match",
			post: MattermostPost{
				Message: "Testing the mobile app features",
			},
			topic:       "mobile",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := transformer.PostMatchesTopic(tt.post, tt.topic)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

func TestMattermostTransformer_FilterDocsByTopic(t *testing.T) {
	transformer := NewMattermostTransformer()

	docs := []Doc{
		{
			Title:   "Mobile Authentication",
			Content: "Guide for mobile app authentication with SSO",
		},
		{
			Title:   "Database Setup",
			Content: "How to set up the database for production",
		},
		{
			Title:   "Mobile Performance",
			Content: "Optimizing mobile app performance",
		},
	}

	tests := []struct {
		name         string
		topic        string
		expectedMin  int
		checkResults func(t *testing.T, filtered []Doc)
	}{
		{
			name:        "Empty topic returns all",
			topic:       "",
			expectedMin: 3,
			checkResults: func(t *testing.T, filtered []Doc) {
				assert.Len(t, filtered, 3)
			},
		},
		{
			name:        "Single keyword filter",
			topic:       "mobile",
			expectedMin: 2,
			checkResults: func(t *testing.T, filtered []Doc) {
				for _, doc := range filtered {
					searchText := strings.ToLower(doc.Title + " " + doc.Content)
					assert.Contains(t, searchText, "mobile")
				}
			},
		},
		{
			name:        "Boolean AND query",
			topic:       "mobile AND authentication",
			expectedMin: 1,
			checkResults: func(t *testing.T, filtered []Doc) {
				if len(filtered) > 0 {
					searchText := strings.ToLower(filtered[0].Title + " " + filtered[0].Content)
					assert.Contains(t, searchText, "mobile")
					assert.Contains(t, searchText, "authentication")
				}
			},
		},
		{
			name:        "Boolean OR query",
			topic:       "mobile OR database",
			expectedMin: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := transformer.FilterDocsByTopic(docs, tt.topic)
			assert.GreaterOrEqual(t, len(filtered), tt.expectedMin)
			if tt.checkResults != nil {
				tt.checkResults(t, filtered)
			}
		})
	}
}
