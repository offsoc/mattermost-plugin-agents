// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package conversations

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/conversations/intentutils"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockIntent is a mock implementation of the Intent interface for testing
type MockIntent struct {
	name       string
	promptName string
	keywords   []string
	confidence float64
}

func NewMockIntent(name, promptName string, keywords []string, confidence float64) *MockIntent {
	return &MockIntent{
		name:       name,
		promptName: promptName,
		keywords:   keywords,
		confidence: confidence,
	}
}

func (m *MockIntent) Matches(message string) (string, float64) {
	msg := strings.ToLower(message)
	for _, keyword := range m.keywords {
		if strings.Contains(msg, strings.ToLower(keyword)) {
			return m.promptName, m.confidence
		}
	}
	return "", 0.0
}

// MockHighConfidenceIntent returns high confidence for testing
type MockHighConfidenceIntent struct {
	promptName string
	keyword    string
}

func (m *MockHighConfidenceIntent) Matches(message string) (string, float64) {
	if strings.Contains(strings.ToLower(message), strings.ToLower(m.keyword)) {
		return m.promptName, 0.9
	}
	return "", 0.0
}

// MockLowConfidenceIntent returns low confidence for testing threshold
type MockLowConfidenceIntent struct {
	promptName string
	keyword    string
}

func (m *MockLowConfidenceIntent) Matches(message string) (string, float64) {
	if strings.Contains(strings.ToLower(message), strings.ToLower(m.keyword)) {
		return m.promptName, 0.2
	}
	return "", 0.0
}

func TestNewIntentDetector(t *testing.T) {
	detector := NewIntentDetector()
	require.NotNil(t, detector)
	assert.NotNil(t, detector.intents)
	assert.Len(t, detector.intents, 0)
}

func TestIntentDetector_RegisterIntent(t *testing.T) {
	detector := NewIntentDetector()

	intent1 := NewMockIntent("test1", "prompt1", []string{"test"}, 0.8)
	intent2 := NewMockIntent("test2", "prompt2", []string{"example"}, 0.7)

	detector.RegisterIntent(intent1)
	assert.Len(t, detector.intents, 1)

	detector.RegisterIntent(intent2)
	assert.Len(t, detector.intents, 2)
}

func TestIntentDetector_DetectIntent_SingleMatch(t *testing.T) {
	detector := NewIntentDetector()

	intent := NewMockIntent("task_creation", "task_create_prompt", []string{"create task", "add task"}, 0.8)
	detector.RegisterIntent(intent)

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "Matches create task",
			message:  "Please create task for the bug fix",
			expected: "task_create_prompt",
		},
		{
			name:     "Matches add task",
			message:  "I need to add task to the sprint",
			expected: "task_create_prompt",
		},
		{
			name:     "No match - returns default",
			message:  "What's the weather today?",
			expected: prompts.PromptDirectMessageQuestionSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectIntent(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntentDetector_DetectIntent_MultipleIntents_HighestWins(t *testing.T) {
	detector := NewIntentDetector()

	lowIntent := NewMockIntent("low", "low_prompt", []string{"help"}, 0.5)
	medIntent := NewMockIntent("med", "med_prompt", []string{"help"}, 0.7)
	highIntent := NewMockIntent("high", "high_prompt", []string{"help"}, 0.9)

	detector.RegisterIntent(lowIntent)
	detector.RegisterIntent(medIntent)
	detector.RegisterIntent(highIntent)

	result := detector.DetectIntent("I need help with this")
	assert.Equal(t, "high_prompt", result, "Should select highest confidence intent")
}

func TestIntentDetector_DetectIntent_ThresholdFiltering(t *testing.T) {
	detector := NewIntentDetector()

	lowConfIntent := &MockLowConfidenceIntent{
		promptName: "low_conf_prompt",
		keyword:    "test",
	}
	detector.RegisterIntent(lowConfIntent)

	result := detector.DetectIntent("This is a test message")
	assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, result,
		"Should fallback to default when confidence (0.2) is below threshold (0.5)")
}

func TestIntentDetector_DetectIntent_AtThreshold(t *testing.T) {
	detector := NewIntentDetector()

	exactThresholdIntent := NewMockIntent("threshold", "threshold_prompt", []string{"threshold"}, intentutils.MinimumConfidenceThreshold)
	detector.RegisterIntent(exactThresholdIntent)

	result := detector.DetectIntent("Testing threshold behavior")
	assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, result,
		"Should NOT match when confidence equals threshold (requires > not >=)")
}

func TestIntentDetector_DetectIntent_JustAboveThreshold(t *testing.T) {
	detector := NewIntentDetector()

	aboveThresholdIntent := NewMockIntent("above", "above_prompt", []string{"above"}, 0.51)
	detector.RegisterIntent(aboveThresholdIntent)

	result := detector.DetectIntent("Testing above threshold")
	assert.Equal(t, "above_prompt", result,
		"Should match when confidence is just above threshold")
}

func TestIntentDetector_DetectIntent_NoIntentsRegistered(t *testing.T) {
	detector := NewIntentDetector()

	result := detector.DetectIntent("Any message")
	assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, result,
		"Should return default prompt when no intents are registered")
}

func TestIntentDetector_DetectIntent_EmptyMessage(t *testing.T) {
	detector := NewIntentDetector()

	intent := NewMockIntent("test", "test_prompt", []string{"test"}, 0.8)
	detector.RegisterIntent(intent)

	result := detector.DetectIntent("")
	assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, result,
		"Should return default prompt for empty message")
}

func TestIntentDetector_DetectIntent_ConflictResolution(t *testing.T) {
	detector := NewIntentDetector()

	intent1 := NewMockIntent("intent1", "prompt1", []string{"bug", "issue"}, 0.6)
	intent2 := NewMockIntent("intent2", "prompt2", []string{"bug", "critical"}, 0.8)

	detector.RegisterIntent(intent1)
	detector.RegisterIntent(intent2)

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "Both match on 'bug' - higher confidence wins",
			message:  "There's a bug in the system",
			expected: "prompt2",
		},
		{
			name:     "Only intent1 matches on 'issue'",
			message:  "There's an issue with deployment",
			expected: "prompt1",
		},
		{
			name:     "Only intent2 matches on 'critical'",
			message:  "This is critical",
			expected: "prompt2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectIntent(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntentDetector_DetectIntent_OrderIndependence(t *testing.T) {
	detector1 := NewIntentDetector()
	detector2 := NewIntentDetector()

	intentA := NewMockIntent("a", "prompt_a", []string{"test"}, 0.5)
	intentB := NewMockIntent("b", "prompt_b", []string{"test"}, 0.8)

	detector1.RegisterIntent(intentA)
	detector1.RegisterIntent(intentB)

	detector2.RegisterIntent(intentB)
	detector2.RegisterIntent(intentA)

	message := "This is a test message"
	result1 := detector1.DetectIntent(message)
	result2 := detector2.DetectIntent(message)

	assert.Equal(t, result1, result2,
		"Intent detection should be order-independent and select highest confidence")
	assert.Equal(t, "prompt_b", result1,
		"Should select intent B with higher confidence (0.8 > 0.5)")
}

func TestIntentDetector_DetectIntent_CaseSensitivity(t *testing.T) {
	detector := NewIntentDetector()

	intent := NewMockIntent("test", "test_prompt", []string{"URGENT"}, 0.8)
	detector.RegisterIntent(intent)

	tests := []struct {
		name    string
		message string
		matches bool
	}{
		{
			name:    "Lowercase matches",
			message: "This is urgent",
			matches: true,
		},
		{
			name:    "Uppercase matches",
			message: "This is URGENT",
			matches: true,
		},
		{
			name:    "Mixed case matches",
			message: "This is UrGeNt",
			matches: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectIntent(tt.message)
			if tt.matches {
				assert.Equal(t, "test_prompt", result)
			} else {
				assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, result)
			}
		})
	}
}

func TestIntentDetector_DetectIntent_MultipleKeywords(t *testing.T) {
	detector := NewIntentDetector()

	intent := NewMockIntent("status", "status_prompt", []string{"status", "update", "progress"}, 0.7)
	detector.RegisterIntent(intent)

	tests := []struct {
		name    string
		message string
		matches bool
	}{
		{
			name:    "Matches first keyword",
			message: "What's the status?",
			matches: true,
		},
		{
			name:    "Matches second keyword",
			message: "Give me an update",
			matches: true,
		},
		{
			name:    "Matches third keyword",
			message: "Show progress report",
			matches: true,
		},
		{
			name:    "No keyword match",
			message: "Random question here",
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectIntent(tt.message)
			if tt.matches {
				assert.Equal(t, "status_prompt", result)
			} else {
				assert.Equal(t, prompts.PromptDirectMessageQuestionSystem, result)
			}
		})
	}
}

func TestMinimumConfidenceThreshold_Value(t *testing.T) {
	assert.Equal(t, 0.5, intentutils.MinimumConfidenceThreshold,
		"Minimum confidence threshold should be 0.5")
}
