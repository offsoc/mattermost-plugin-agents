// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/conversations/intentutils"
	"github.com/stretchr/testify/assert"
)

func TestIntent_Matches_Debugging(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Direct error mention",
			message:        "I'm getting an error in the plugin API",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Exception thrown",
			message:        "The system is throwing an exception",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Not working",
			message:        "The websocket connection isn't working",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Debug request",
			message:        "Can you help debug this issue?",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Hook not being called",
			message:        "The MessageHasBeenPosted hook is not being called",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Validation failing",
			message:        "Plugin manifest validation is failing",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
			assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
		})
	}
}

func TestIntent_Matches_CodeExplanation(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "How do I implement",
			message:        "How do I implement a slash command?",
			expectedPrompt: PromptDevCodeExplanationSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Find examples",
			message:        "Can you find examples of interactive dialogs?",
			expectedPrompt: PromptDevCodeExplanationSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Explain workflow",
			message:        "Explain the message workflow in Mattermost",
			expectedPrompt: PromptDevCodeExplanationSystem,
			minConfidence:  0.7,
		},
		{
			name:           "Plugin API",
			message:        "How does the plugin API work?",
			expectedPrompt: PromptDevCodeExplanationSystem,
			minConfidence:  0.6,
		},
		{
			name:           "Implementation question",
			message:        "How to implement webhooks?",
			expectedPrompt: PromptDevCodeExplanationSystem,
			minConfidence:  0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
			assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
		})
	}
}

func TestIntent_Matches_Architecture(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Architecture ADR",
			message:        "Show me the ADR for this design",
			expectedPrompt: PromptDevArchitectureSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Design pattern",
			message:        "What design patterns are used?",
			expectedPrompt: PromptDevArchitectureSystem,
			minConfidence:  0.4,
		},
		{
			name:           "System structure",
			message:        "How is the system structured?",
			expectedPrompt: PromptDevArchitectureSystem,
			minConfidence:  0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
			assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
		})
	}
}

func TestIntent_Matches_PRSummary(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Summarize PR",
			message:        "Summarize the recent pull requests",
			expectedPrompt: PromptDevPRSummarySystem,
			minConfidence:  0.4,
		},
		{
			name:           "What changed in version",
			message:        "What's new in version 9.5?",
			expectedPrompt: PromptDevPRSummarySystem,
			minConfidence:  0.4,
		},
		{
			name:           "Breaking changes",
			message:        "Are there any breaking changes in the update?",
			expectedPrompt: PromptDevPRSummarySystem,
			minConfidence:  0.4,
		},
		{
			name:           "Latest release",
			message:        "What's in the latest release?",
			expectedPrompt: PromptDevPRSummarySystem,
			minConfidence:  0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
			assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
		})
	}
}

func TestIntent_Matches_PriorityOrder(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
	}{
		{
			name:           "Debugging beats code explanation",
			message:        "I'm getting an error when implementing the hook",
			expectedPrompt: PromptDevDebuggingSystem,
		},
		{
			name:           "Code explanation for valid how-to",
			message:        "How do I implement a new API endpoint?",
			expectedPrompt: PromptDevCodeExplanationSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, _ := intent.Matches(tt.message)
			assert.Equal(t, tt.expectedPrompt, prompt, "Should match expected prompt")
		})
	}
}

func TestIntent_Matches_NoMatch(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Generic greeting",
			message: "Hello, how are you?",
		},
		{
			name:    "Weather question",
			message: "What's the weather like?",
		},
		{
			name:    "Non-technical question",
			message: "What's for lunch today?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Empty(t, prompt, "Should not match any prompt")
			assert.Equal(t, intentutils.NoConfidence, confidence, "Confidence should be 0")
		})
	}
}

func TestIntent_Matches_CaseInsensitive(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
	}{
		{
			name:           "Uppercase",
			message:        "I'M GETTING AN ERROR",
			expectedPrompt: PromptDevDebuggingSystem,
		},
		{
			name:           "Mixed case",
			message:        "HoW dO i ImPlEmEnT a PlUgIn?",
			expectedPrompt: PromptDevCodeExplanationSystem,
		},
		{
			name:           "Lowercase",
			message:        "getting an error",
			expectedPrompt: PromptDevDebuggingSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Should match regardless of case")
			assert.Greater(t, confidence, intentutils.NoConfidence, "Should have confidence")
		})
	}
}

func TestIntent_Matches_WhitespaceHandling(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
	}{
		{
			name:           "Leading whitespace",
			message:        "   getting an error",
			expectedPrompt: PromptDevDebuggingSystem,
		},
		{
			name:           "Trailing whitespace",
			message:        "getting an error   ",
			expectedPrompt: PromptDevDebuggingSystem,
		},
		{
			name:           "Multiple spaces",
			message:        "getting    an    error",
			expectedPrompt: PromptDevDebuggingSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Should handle whitespace")
			assert.Greater(t, confidence, intentutils.NoConfidence, "Should have confidence")
		})
	}
}

func TestIntent_Matches_ConfidenceScores(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
		maxConfidence  float64
	}{
		{
			name:           "High confidence debugging",
			message:        "I'm getting an error",
			expectedPrompt: PromptDevDebuggingSystem,
			minConfidence:  0.7,
			maxConfidence:  1.0,
		},
		{
			name:           "High confidence code explanation",
			message:        "How do I implement a webhook?",
			expectedPrompt: PromptDevCodeExplanationSystem,
			minConfidence:  0.7,
			maxConfidence:  1.0,
		},
		{
			name:           "Medium confidence architecture",
			message:        "What's the architecture?",
			expectedPrompt: PromptDevArchitectureSystem,
			minConfidence:  0.3,
			maxConfidence:  0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
			assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
			assert.LessOrEqual(t, confidence, tt.maxConfidence, "Confidence too high")
		})
	}
}
