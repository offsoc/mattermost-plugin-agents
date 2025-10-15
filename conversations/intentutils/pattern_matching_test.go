// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package intentutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesPattern_SingleWord(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "test", Weight: 1.0, WordBoundary: false},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Exact match",
			message:       "test",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Contains word",
			message:       "This is a test message",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Part of larger word without boundary",
			message:       "testing the system",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "No match",
			message:       "example message",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_WordBoundary(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "test", Weight: 1.0, WordBoundary: true},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Exact word match",
			message:       "test",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Word with boundaries",
			message:       "This is a test message",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Part of larger word - should NOT match",
			message:       "testing the system",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
		{
			name:          "Word at start",
			message:       "test this feature",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Word at end",
			message:       "run the test",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_Phrase(t *testing.T) {
	patterns := []KeywordPattern{
		{Phrase: "create task", Weight: 1.5, WordBoundary: false},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Exact phrase",
			message:       "create task",
			shouldMatch:   true,
			expectedScore: 1.5,
		},
		{
			name:          "Phrase in sentence",
			message:       "Please create task for this bug",
			shouldMatch:   true,
			expectedScore: 1.5,
		},
		{
			name:          "No match",
			message:       "create new item",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_PhraseWithWordBoundary(t *testing.T) {
	patterns := []KeywordPattern{
		{Phrase: "bug fix", Weight: 1.0, WordBoundary: true},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Exact phrase with boundaries",
			message:       "Need a bug fix",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Phrase at start",
			message:       "bug fix required",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Part of larger phrase - should NOT match",
			message:       "debugging fixation",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_RegexPattern(t *testing.T) {
	patterns := []KeywordPattern{
		{Pattern: `\b(create|add|new)\s+(task|issue|ticket)\b`, Weight: 2.0},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Create task",
			message:       "create task for feature",
			shouldMatch:   true,
			expectedScore: 2.0,
		},
		{
			name:          "Add issue",
			message:       "add issue to backlog",
			shouldMatch:   true,
			expectedScore: 2.0,
		},
		{
			name:          "New ticket",
			message:       "new ticket created",
			shouldMatch:   true,
			expectedScore: 2.0,
		},
		{
			name:          "No match",
			message:       "delete task",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_MultiplePatterns(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "urgent", Weight: 1.0, WordBoundary: true},
		{Word: "critical", Weight: 1.5, WordBoundary: true},
		{Word: "blocker", Weight: 2.0, WordBoundary: true},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64 // maxScore
	}{
		{
			name:          "Matches lowest weight",
			message:       "This is urgent",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Matches medium weight",
			message:       "Critical bug found",
			shouldMatch:   true,
			expectedScore: 1.5,
		},
		{
			name:          "Matches highest weight",
			message:       "Blocker preventing release",
			shouldMatch:   true,
			expectedScore: 2.0,
		},
		{
			name:          "Multiple matches - returns max",
			message:       "Urgent and critical blocker",
			shouldMatch:   true,
			expectedScore: 2.0,
		},
		{
			name:          "No matches",
			message:       "Normal priority task",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_WeightAccumulation(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "bug", Weight: 0.5, WordBoundary: true},
		{Word: "fix", Weight: 0.5, WordBoundary: true},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Single keyword",
			message:       "Found a bug",
			shouldMatch:   true,
			expectedScore: 0.5,
		},
		{
			name:          "Both keywords - max is returned",
			message:       "Bug fix needed",
			shouldMatch:   true,
			expectedScore: 0.5, // maxScore, not totalScore
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_CaseInsensitive(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "urgent", Weight: 1.0, WordBoundary: true},
	}

	tests := []struct {
		name        string
		message     string
		shouldMatch bool
	}{
		{
			name:        "Lowercase",
			message:     "urgent issue",
			shouldMatch: true,
		},
		{
			name:        "Uppercase",
			message:     "URGENT issue",
			shouldMatch: true,
		},
		{
			name:        "Mixed case",
			message:     "UrGeNt issue",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
		})
	}
}

func TestMatchesPattern_EmptyMessage(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "test", Weight: 1.0},
	}

	matched, score := MatchesPattern("", patterns)
	assert.False(t, matched)
	assert.Equal(t, 0.0, score)
}

func TestMatchesPattern_EmptyPatterns(t *testing.T) {
	patterns := []KeywordPattern{}

	matched, score := MatchesPattern("test message", patterns)
	assert.False(t, matched)
	assert.Equal(t, 0.0, score)
}

func TestMatchesPattern_WhitespaceHandling(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "test", Weight: 1.0, WordBoundary: true},
	}

	tests := []struct {
		name        string
		message     string
		shouldMatch bool
	}{
		{
			name:        "Leading whitespace",
			message:     "  test message",
			shouldMatch: true,
		},
		{
			name:        "Trailing whitespace",
			message:     "test message  ",
			shouldMatch: true,
		},
		{
			name:        "Multiple spaces",
			message:     "this    test    message",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
		})
	}
}

func TestMatchesPattern_SpecialCharacters(t *testing.T) {
	patterns := []KeywordPattern{
		{Phrase: "bug #123", Weight: 1.0, WordBoundary: true},
	}

	tests := []struct {
		name          string
		message       string
		shouldMatch   bool
		expectedScore float64
	}{
		{
			name:          "Special chars in phrase",
			message:       "Found bug #123",
			shouldMatch:   true,
			expectedScore: 1.0,
		},
		{
			name:          "Different number",
			message:       "Found bug #456",
			shouldMatch:   false,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, score := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestMatchesPattern_UnicodeCharacters(t *testing.T) {
	patterns := []KeywordPattern{
		{Word: "café", Weight: 1.0},
	}

	tests := []struct {
		name        string
		message     string
		shouldMatch bool
	}{
		{
			name:        "Unicode match",
			message:     "Visit the café",
			shouldMatch: true,
		},
		{
			name:        "No match",
			message:     "Visit the cafe",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
		})
	}
}

func TestMatchesPattern_RegexEdgeCases(t *testing.T) {
	patterns := []KeywordPattern{
		{Pattern: `^\d+$`, Weight: 1.0}, // Only digits
	}

	tests := []struct {
		name        string
		message     string
		shouldMatch bool
	}{
		{
			name:        "Only digits",
			message:     "12345",
			shouldMatch: true,
		},
		{
			name:        "Digits with text",
			message:     "Issue 12345",
			shouldMatch: false,
		},
		{
			name:        "No digits",
			message:     "test",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := MatchesPattern(tt.message, patterns)
			assert.Equal(t, tt.shouldMatch, matched)
		})
	}
}
