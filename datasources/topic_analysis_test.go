// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"strings"
	"testing"
)

func TestBuildExpandedSearchTerms(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	tests := []struct {
		name     string
		topic    string
		maxTerms int
		expected []string
		minTerms int // Minimum expected terms (for cases where synonyms may vary)
	}{
		{
			name:     "empty topic",
			topic:    "",
			maxTerms: 10,
			expected: nil,
		},
		{
			name:     "zero max terms uses default",
			topic:    "mobile",
			maxTerms: 0,
			expected: []string{"mobile", "ios", "android", "react native", "react-native"},
			minTerms: 3, // At least original + some synonyms
		},
		{
			name:     "negative max terms uses default",
			topic:    "mobile",
			maxTerms: -1,
			expected: []string{"mobile", "ios", "android", "react native", "react-native"},
			minTerms: 3,
		},
		{
			name:     "single word with synonyms",
			topic:    "mobile",
			maxTerms: 5,
			expected: []string{"mobile", "ios", "android", "react native", "react-native"},
			minTerms: 3,
		},
		{
			name:     "multi-word topic",
			topic:    "mobile app development",
			maxTerms: 8,
			expected: []string{"mobile app development", "mobile", "app", "development", "react native", "react-native", "ios", "android"},
			minTerms: 4, // Original phrase + individual words + some synonyms
		},
		{
			name:     "topic with no synonyms",
			topic:    "unknown topic xyz",
			maxTerms: 10,
			expected: []string{"unknown topic xyz", "unknown", "topic", "xyz"},
			minTerms: 4,
		},
		{
			name:     "max terms limits output",
			topic:    "mobile",
			maxTerms: 2,
			expected: []string{"mobile"},
			minTerms: 1,
		},
		{
			name:     "AI/ML terms expansion",
			topic:    "ai assistant",
			maxTerms: 10,
			expected: []string{"ai assistant", "ai", "assistant", "artificial intelligence", "large language model", "chatbot", "copilot", "agents"},
			minTerms: 4,
		},
		{
			name:     "enterprise terms expansion",
			topic:    "enterprise security",
			maxTerms: 8,
			expected: []string{"enterprise security", "enterprise", "security", "sso", "ldap", "compliance", "authentication", "authorization"},
			minTerms: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.BuildExpandedSearchTerms(tt.topic, tt.maxTerms)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			// Check minimum number of terms
			if len(result) < tt.minTerms {
				t.Errorf("Expected at least %d terms, got %d: %v", tt.minTerms, len(result), result)
			}

			// Check max terms limit is respected
			maxLimit := tt.maxTerms
			if maxLimit <= 0 {
				maxLimit = 10 // Default
			}
			if len(result) > maxLimit {
				t.Errorf("Expected at most %d terms, got %d: %v", maxLimit, len(result), result)
			}

			// Check that original topic or its words are included
			if tt.topic != "" {
				topicLower := strings.ToLower(tt.topic)
				topicWords := strings.Fields(topicLower)

				foundOriginal := false
				for _, term := range result {
					if term == topicLower {
						foundOriginal = true
						break
					}
					// Check if any original words are present
					for _, word := range topicWords {
						if term == word {
							foundOriginal = true
							break
						}
					}
					if foundOriginal {
						break
					}
				}

				if !foundOriginal {
					t.Errorf("Expected result to contain original topic '%s' or its words, got %v", tt.topic, result)
				}
			}

			// Check for duplicates
			seen := make(map[string]bool)
			for _, term := range result {
				if seen[term] {
					t.Errorf("Found duplicate term '%s' in result: %v", term, result)
				}
				seen[term] = true
			}

			// Check that all terms are non-empty
			for _, term := range result {
				if term == "" {
					t.Errorf("Found empty term in result: %v", result)
				}
			}
		})
	}
}

func TestBuildExpandedSearchTermsPriority(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	// Test that full phrase gets highest priority
	result := analyzer.BuildExpandedSearchTerms("mobile app", 10)

	if len(result) == 0 {
		t.Fatal("Expected non-empty result")
	}

	// First term should be the full phrase
	if result[0] != "mobile app" {
		t.Errorf("Expected first term to be full phrase 'mobile app', got '%s'", result[0])
	}

	// Check that original words appear early in the list
	mobileIndex := -1
	appIndex := -1

	for i, term := range result {
		if term == "mobile" {
			mobileIndex = i
		}
		if term == "app" {
			appIndex = i
		}
	}

	// Original words should be found and should appear relatively early
	if mobileIndex == -1 {
		t.Errorf("Expected to find 'mobile' in expanded terms")
	}
	if appIndex == -1 {
		t.Errorf("Expected to find 'app' in expanded terms")
	}

	// Both original words should appear in first half of results
	halfLength := len(result) / 2
	if mobileIndex > halfLength {
		t.Errorf("Expected 'mobile' to appear in first half of results, found at index %d out of %d", mobileIndex, len(result))
	}
	if appIndex > halfLength {
		t.Errorf("Expected 'app' to appear in first half of results, found at index %d out of %d", appIndex, len(result))
	}
}

func TestBuildExpandedSearchTermsDeduplication(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	// Test topic that might generate duplicate terms
	result := analyzer.BuildExpandedSearchTerms("mobile mobile app", 15)

	// Count occurrences of "mobile"
	mobileCount := 0
	for _, term := range result {
		if term == "mobile" {
			mobileCount++
		}
	}

	if mobileCount > 1 {
		t.Errorf("Expected 'mobile' to appear only once, found %d times in %v", mobileCount, result)
	}
}

func TestBuildExpandedSearchTermsEmptyAndEdgeCases(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	tests := []struct {
		name     string
		topic    string
		maxTerms int
		expected []string
	}{
		{
			name:     "whitespace only topic",
			topic:    "   ",
			maxTerms: 10,
			expected: nil,
		},
		{
			name:     "single character",
			topic:    "a",
			maxTerms: 10,
			expected: []string{"a"},
		},
		{
			name:     "special characters",
			topic:    "mobile-app",
			maxTerms: 10,
			expected: []string{"mobile-app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.BuildExpandedSearchTerms(tt.topic, tt.maxTerms)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			if len(result) == 0 && len(tt.expected) > 0 {
				t.Errorf("Expected non-empty result, got empty")
				return
			}

			// Check that result contains expected terms (allowing for additional synonyms)
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find '%s' in result %v", expected, result)
				}
			}
		})
	}
}

func TestGetTopicSynonyms(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	tests := []struct {
		name     string
		keyword  string
		expected []string
	}{
		{
			name:     "mobile synonyms",
			keyword:  "mobile",
			expected: []string{"ios", "android", "react native", "react-native", "push notification"},
		},
		{
			name:     "ai synonyms",
			keyword:  "ai",
			expected: []string{"artificial intelligence", "machine learning", "ml", "llm", "large language model", "chatbot", "assistant", "copilot", "agents", "mm agents"},
		},
		{
			name:     "unknown keyword",
			keyword:  "unknownkeyword",
			expected: nil,
		},
		{
			name:     "empty keyword",
			keyword:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.GetTopicSynonyms(tt.keyword)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			// Check that all expected synonyms are present
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find synonym '%s' for keyword '%s', got %v", expected, tt.keyword, result)
				}
			}

			// Check that we got some synonyms
			if len(result) == 0 {
				t.Errorf("Expected non-empty synonyms for keyword '%s'", tt.keyword)
			}
		})
	}
}

func TestExtractTopicKeywords(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	tests := []struct {
		name     string
		topic    string
		expected []string
	}{
		{
			name:     "empty topic",
			topic:    "",
			expected: nil,
		},
		{
			name:     "single word",
			topic:    "mobile",
			expected: []string{"mobile"},
		},
		{
			name:     "multiple words",
			topic:    "mobile app development",
			expected: []string{"mobile", "app", "development", "mobile app development"},
		},
		{
			name:     "mixed case",
			topic:    "Mobile App Development",
			expected: []string{"mobile", "app", "development", "mobile app development"},
		},
		{
			name:     "with punctuation",
			topic:    "mobile, app & development!",
			expected: []string{"mobile,", "app", "&", "development!", "mobile, app & development!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.ExtractTopicKeywords(tt.topic)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			// Check that result has reasonable content
			if len(result) == 0 && len(tt.expected) > 0 {
				t.Errorf("Expected non-empty result for topic '%s'", tt.topic)
				return
			}

			// Check that individual words are present
			topicWords := strings.Fields(strings.ToLower(tt.topic))
			for _, word := range topicWords {
				found := false
				for _, keyword := range result {
					if keyword == word {
						found = true
						break
					}
				}
				if !found && word != "" {
					t.Errorf("Expected to find word '%s' in keywords %v for topic '%s'", word, result, tt.topic)
				}
			}

			// If topic has multiple words, full phrase should be included
			if len(topicWords) > 1 {
				fullPhrase := strings.ToLower(tt.topic)
				found := false
				for _, keyword := range result {
					if keyword == fullPhrase {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find full phrase '%s' in keywords %v", fullPhrase, result)
				}
			}
		})
	}
}
