// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
)

func TestUniversalRelevanceScorer_IsUniversallyAcceptable(t *testing.T) {
	scorer := NewUniversalRelevanceScorer()

	tests := []struct {
		name     string
		content  string
		title    string
		source   string
		topic    string
		expected bool
	}{
		{
			name:     "empty content",
			content:  "",
			title:    "Test Title",
			source:   SourceGitHubRepos,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "content too short",
			content:  "Short",
			title:    "Test Title",
			source:   SourceGitHubRepos,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "good quality content with topic match",
			content:  "This is a comprehensive guide about mobile app development using React Native. It covers installation, setup, and best practices for building cross-platform applications.",
			title:    "Mobile Development Guide",
			source:   SourceMattermostDocs,
			topic:    "mobile",
			expected: true,
		},
		{
			name:     "good quality content without topic",
			content:  "This document provides detailed information about server configuration and deployment strategies. It includes step-by-step instructions and troubleshooting tips.",
			title:    "Server Configuration",
			source:   SourceMattermostDocs,
			topic:    "",
			expected: true,
		},
		{
			name:     "navigation-heavy content",
			content:  "Home | Next | Previous | Menu | Search | Login | Register | Toggle | Sidebar | Navigation | Home | Next | Previous | Menu | Search | Navigation",
			title:    "Navigation Page",
			source:   SourceGitHubRepos,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "content with no alphabetic characters",
			content:  "123456789 !@#$%^&*() 123456789 !@#$%^&*() 123456789 !@#$%^&*() 123456789 !@#$%^&*()",
			title:    "Test Title",
			source:   SourceGitHubRepos,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "error page content",
			content:  "This is an error page with some content that is longer than the minimum required length but should be filtered out.",
			title:    "404 Not Found",
			source:   SourceGitHubRepos,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "home page content",
			content:  "Welcome to our documentation home page. This page contains links and general information about our platform and services.",
			title:    "Home",
			source:   SourceMattermostDocs,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "GitHub TODO content",
			content:  "TODO: Implement mobile support for this feature. This is a placeholder and needs to be completed in future development cycles.",
			title:    "Mobile Support Task",
			source:   SourceGitHubRepos,
			topic:    "mobile",
			expected: false,
		},
		{
			name:     "community content with shorter threshold",
			content:  "Good discussion about mobile development practices in our community forum with valuable insights.",
			title:    "Mobile Discussion",
			source:   SourceCommunityForum,
			topic:    "mobile",
			expected: true,
		},
		{
			name:     "irrelevant content with topic",
			content:  "This document is about server administration and has nothing to do with the requested topic. It covers database management, user permissions, and system monitoring.",
			title:    "Server Administration",
			source:   SourceMattermostDocs,
			topic:    "mobile",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.IsUniversallyAcceptable(tt.content, tt.title, tt.source, tt.topic)
			if result != tt.expected {
				t.Errorf("IsUniversallyAcceptable() = %v, expected %v for content: %s", result, tt.expected, tt.content[:min(50, len(tt.content))])
			}
		})
	}
}

func TestUniversalRelevanceScorer_IsPlainTextQualityAcceptable(t *testing.T) {
	scorer := NewUniversalRelevanceScorer()

	tests := []struct {
		name     string
		content  string
		title    string
		expected bool
	}{
		{
			name:     "good quality content",
			content:  "This is a well-written document with substantial content that provides valuable information to users.",
			title:    "Quality Document",
			expected: true,
		},
		{
			name:     "navigation heavy content",
			content:  "home menu search login register toggle sidebar next previous navigation breadcrumb home menu search login",
			title:    "Navigation Page",
			expected: false,
		},
		{
			name:     "no alphabetic characters",
			content:  "123456789 !@#$%^&*() 123456789 !@#$%^&*() 123456789",
			title:    "Test",
			expected: false,
		},
		{
			name:     "index page title",
			content:  "This page contains useful information about our platform and services.",
			title:    "index",
			expected: false,
		},
		{
			name:     "error page title",
			content:  "This page contains useful information about troubleshooting common issues.",
			title:    "Error 404 - Page Not Found",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.isPlainTextQualityAcceptable(tt.content, tt.title)
			if result != tt.expected {
				t.Errorf("isPlainTextQualityAcceptable() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestUniversalRelevanceScorer_MeetsSourceQualityStandards(t *testing.T) {
	scorer := NewUniversalRelevanceScorer()

	tests := []struct {
		name     string
		content  string
		source   string
		expected bool
	}{
		{
			name:     "official docs with good length",
			content:  strings.Repeat("Official documentation content with detailed information and examples. ", 3),
			source:   SourceMattermostDocs,
			expected: true,
		},
		{
			name:     "official docs too short",
			content:  "Short official doc content.",
			source:   SourceMattermostDocs,
			expected: false,
		},
		{
			name:     "github content with TODO",
			content:  "TODO: This feature needs to be implemented in the next release cycle. It will provide enhanced functionality.",
			source:   SourceGitHubRepos,
			expected: false,
		},
		{
			name:     "github content without TODO",
			content:  "This GitHub issue describes a bug in the mobile application that affects user authentication.",
			source:   SourceGitHubRepos,
			expected: true,
		},
		{
			name:     "community content shorter threshold",
			content:  "Community discussion about best practices and helpful tips for developers.",
			source:   SourceCommunityForum,
			expected: true,
		},
		{
			name:     "default source standard",
			content:  "This content meets the default minimum length requirements for quality assessment.",
			source:   "unknown_source",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scorer.meetsSourceQualityStandards(tt.content, tt.source)
			if result != tt.expected {
				t.Errorf("meetsSourceQualityStandards() = %v, expected %v for source %s", result, tt.expected, tt.source)
			}
		})
	}
}

func TestUniversalRelevanceScorer_Integration(t *testing.T) {
	scorer := NewUniversalRelevanceScorer()

	// Test content that should pass all filters
	goodContent := "This comprehensive guide covers mobile application development using React Native framework. It includes detailed setup instructions, best practices for cross-platform development, and troubleshooting common issues that developers encounter when building mobile applications."

	result := scorer.IsUniversallyAcceptable(goodContent, "Mobile Development Guide", SourceMattermostDocs, "mobile")
	if !result {
		t.Error("Expected good quality content to pass universal acceptance")
	}

	// Test content that should fail quality checks
	poorContent := "home next previous menu"

	result = scorer.IsUniversallyAcceptable(poorContent, "Navigation", SourceMattermostDocs, "mobile")
	if result {
		t.Error("Expected poor quality content to fail universal acceptance")
	}

	// Test content that should fail topic relevance
	irrelevantContent := "This document covers database administration, server configuration, and system monitoring. It provides detailed instructions for setting up MySQL, PostgreSQL, and managing user permissions in enterprise environments."

	result = scorer.IsUniversallyAcceptable(irrelevantContent, "Database Administration", SourceMattermostDocs, "mobile")
	if result {
		t.Error("Expected topic-irrelevant content to fail universal acceptance")
	}
}
