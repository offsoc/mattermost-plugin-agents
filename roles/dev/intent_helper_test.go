// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/prompts"
	"github.com/stretchr/testify/assert"
)

func TestIntentHelper_DetectIntent(t *testing.T) {
	helper := &IntentHelper{}

	tests := []struct {
		name           string
		message        string
		expectedIntent string
	}{
		{
			name:           "Debugging intent",
			message:        "I'm getting an error in the plugin",
			expectedIntent: PromptDevDebuggingSystem,
		},
		{
			name:           "Code explanation intent",
			message:        "How do I implement a slash command?",
			expectedIntent: PromptDevCodeExplanationSystem,
		},
		{
			name:           "Architecture intent",
			message:        "What's the system design?",
			expectedIntent: PromptDevArchitectureSystem,
		},
		{
			name:           "No match - fallback to default",
			message:        "Hello, how are you?",
			expectedIntent: prompts.PromptDirectMessageQuestionSystem,
		},
		{
			name:           "Empty message - fallback to default",
			message:        "",
			expectedIntent: prompts.PromptDirectMessageQuestionSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.DetectIntent(tt.message)
			assert.Equal(t, tt.expectedIntent, result)
		})
	}
}

func TestIntentHelper_HasIntentChanged(t *testing.T) {
	helper := &IntentHelper{}

	tests := []struct {
		name            string
		previousIntent  string
		currentIntent   string
		expectedChanged bool
	}{
		{
			name:            "Same intent - no change",
			previousIntent:  PromptDevDebuggingSystem,
			currentIntent:   PromptDevDebuggingSystem,
			expectedChanged: false,
		},
		{
			name:            "Different Dev intents - changed",
			previousIntent:  PromptDevDebuggingSystem,
			currentIntent:   PromptDevCodeExplanationSystem,
			expectedChanged: true,
		},
		{
			name:            "From default to Dev intent - changed",
			previousIntent:  prompts.PromptDirectMessageQuestionSystem,
			currentIntent:   PromptDevDebuggingSystem,
			expectedChanged: true,
		},
		{
			name:            "From Dev intent to default - changed",
			previousIntent:  PromptDevDebuggingSystem,
			currentIntent:   prompts.PromptDirectMessageQuestionSystem,
			expectedChanged: true,
		},
		{
			name:            "Same default intent - no change",
			previousIntent:  prompts.PromptDirectMessageQuestionSystem,
			currentIntent:   prompts.PromptDirectMessageQuestionSystem,
			expectedChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.HasIntentChanged(tt.previousIntent, tt.currentIntent)
			assert.Equal(t, tt.expectedChanged, result)
		})
	}
}

func TestIntentHelper_GetDisplayName(t *testing.T) {
	helper := &IntentHelper{}

	tests := []struct {
		name         string
		intent       string
		expectedName string
	}{
		{
			name:         "Code explanation",
			intent:       PromptDevCodeExplanationSystem,
			expectedName: "code explanation",
		},
		{
			name:         "Debugging",
			intent:       PromptDevDebuggingSystem,
			expectedName: "debugging",
		},
		{
			name:         "Architecture",
			intent:       PromptDevArchitectureSystem,
			expectedName: "architecture",
		},
		{
			name:         "API examples",
			intent:       PromptDevAPIExamplesSystem,
			expectedName: "API examples",
		},
		{
			name:         "PR summary",
			intent:       PromptDevPRSummarySystem,
			expectedName: "PR summary",
		},
		{
			name:         "General conversation",
			intent:       prompts.PromptDirectMessageQuestionSystem,
			expectedName: "general conversation",
		},
		{
			name:         "Unknown intent - fallback",
			intent:       "some_unknown_intent",
			expectedName: "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.GetDisplayName(tt.intent)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}
