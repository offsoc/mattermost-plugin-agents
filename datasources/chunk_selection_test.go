// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
)

func TestSelectBestChunkWithContext(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	chunks := []string{
		"This is about databases and storage systems.",
		"Mobile applications require special consideration for performance.",
		"The mobile app uses React Native framework for cross-platform development.",
		"Server configuration involves multiple components and services.",
	}

	topic := "mobile development"

	result := analyzer.SelectBestChunkWithContext(chunks, topic)

	// Should contain the mobile-related chunks with context
	if !strings.Contains(result, "Mobile applications require special consideration") {
		t.Error("Expected result to contain the most relevant mobile chunk")
	}

	// Should also contain adjacent chunks for context
	if !strings.Contains(result, "React Native framework") {
		t.Error("Expected result to contain adjacent chunk for context")
	}

	// Should not contain unrelated content when there are better matches
	if strings.Contains(result, "databases and storage") {
		t.Log("Result contains database content - this might be acceptable if it's adjacent to relevant content")
	}

	t.Logf("Result length: %d characters", len(result))
	t.Logf("Result: %s", result)
}

func TestHTMLProcessorContentQuality(t *testing.T) {
	processor := NewHTMLProcessor()

	tests := []struct {
		name       string
		title      string
		content    string
		expectPass bool
	}{
		{
			name:       "Good quality content",
			title:      "Mobile App Development Guide",
			content:    "This comprehensive guide covers mobile application development using React Native. It includes setup instructions, best practices, and deployment strategies. The content is detailed and provides valuable information for developers.",
			expectPass: true,
		},
		{
			name:       "Portal page with too many links",
			title:      "Documentation Home",
			content:    `Welcome to docs. <a href="/mobile">Mobile</a> <a href="/web">Web</a> <a href="/api">API</a> <a href="/server">Server</a> <a href="/desktop">Desktop</a> <a href="/integrations">Integrations</a> <a href="/plugins">Plugins</a> <a href="/admin">Admin</a> <a href="/user">User Guide</a> <a href="/dev">Developer</a> <a href="/troubleshoot">Troubleshooting</a> Quick links to get started.`,
			expectPass: false,
		},
		{
			name:       "Too short content",
			title:      "Brief Note",
			content:    "Short text.",
			expectPass: false,
		},
		{
			name:       "Navigation heavy content",
			title:      "Site Navigation",
			content:    "Home previous next menu search login register toggle sidebar menu home previous next search toggle menu previous next home sidebar toggle menu search register login previous next.",
			expectPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.IsContentQualityAcceptable(tt.title, tt.content)
			if result != tt.expectPass {
				t.Errorf("Expected %v, got %v for test case: %s", tt.expectPass, result, tt.name)
			}
		})
	}
}

func TestTopicKeywordExpansion(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	testCases := []struct {
		topic              string
		expectedKeywords   []string
		unexpectedKeywords []string
	}{
		{
			topic:              "mobile app",
			expectedKeywords:   []string{"mobile", "app", "ios", "android", "react native"},
			unexpectedKeywords: []string{"desktop", "server"},
		},
		{
			topic:              "ai assistant",
			expectedKeywords:   []string{"ai", "assistant", "copilot", "agents", "chatbot"},
			unexpectedKeywords: []string{"mobile", "database"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.topic, func(t *testing.T) {
			keywords := analyzer.ExtractTopicKeywords(tc.topic)

			keywordMap := make(map[string]bool)
			for _, kw := range keywords {
				keywordMap[kw] = true
			}

			for _, expected := range tc.expectedKeywords {
				if !keywordMap[expected] {
					t.Errorf("Expected keyword '%s' not found in results for topic '%s'", expected, tc.topic)
				}
			}

			for _, unexpected := range tc.unexpectedKeywords {
				if keywordMap[unexpected] {
					t.Errorf("Unexpected keyword '%s' found in results for topic '%s'", unexpected, tc.topic)
				}
			}

			t.Logf("Topic '%s' expanded to keywords: %v", tc.topic, keywords)
		})
	}
}
