// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

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
			name:           "Task creation intent",
			message:        "Create a task for this feature",
			expectedIntent: PromptPmTaskCreationSystem,
		},
		{
			name:           "Status intent",
			message:        "What's the status of the project?",
			expectedIntent: PromptPmStatusReportSystem,
		},
		{
			name:           "Strategic alignment intent",
			message:        "How does this align with our strategy?",
			expectedIntent: PromptPmStrategicAlignmentSystem,
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
			previousIntent:  PromptPmTaskCreationSystem,
			currentIntent:   PromptPmTaskCreationSystem,
			expectedChanged: false,
		},
		{
			name:            "Different PM intents - changed",
			previousIntent:  PromptPmTaskCreationSystem,
			currentIntent:   PromptPmStatusReportSystem,
			expectedChanged: true,
		},
		{
			name:            "From default to PM intent - changed",
			previousIntent:  prompts.PromptDirectMessageQuestionSystem,
			currentIntent:   PromptPmTaskCreationSystem,
			expectedChanged: true,
		},
		{
			name:            "From PM intent to default - changed",
			previousIntent:  PromptPmTaskCreationSystem,
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
			name:         "Task creation",
			intent:       PromptPmTaskCreationSystem,
			expectedName: "task creation",
		},
		{
			name:         "Status reporting",
			intent:       PromptPmStatusReportSystem,
			expectedName: "status reporting",
		},
		{
			name:         "Task updates",
			intent:       PromptPmTaskUpdateSystem,
			expectedName: "task updates",
		},
		{
			name:         "Standup facilitation",
			intent:       PromptPmStandupFacilitationSystem,
			expectedName: "standup facilitation",
		},
		{
			name:         "Meeting action items",
			intent:       PromptPmMeetingActionItemsSystem,
			expectedName: "meeting action items",
		},
		{
			name:         "General conversation",
			intent:       prompts.PromptDirectMessageQuestionSystem,
			expectedName: "general conversation",
		},
		{
			name:         "Unknown intent - fallback",
			intent:       "some_unknown_intent",
			expectedName: "project management",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.GetDisplayName(tt.intent)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestIntentHelper_GetDisplayName_AllPrompts(t *testing.T) {
	helper := &IntentHelper{}

	allPrompts := []struct {
		prompt      string
		displayName string
	}{
		{PromptPmTaskCreationSystem, "task creation"},
		{PromptPmStatusReportSystem, "status reporting"},
		{PromptPmTaskUpdateSystem, "task updates"},
		{PromptPmStandupFacilitationSystem, "standup facilitation"},
		{PromptPmMeetingActionItemsSystem, "meeting action items"},
		{PromptPmStrategicAlignmentSystem, "project management"},
		{PromptPmFeatureGapAnalysisSystem, "project management"},
		{PromptPmMarketResearchSystem, "project management"},
	}

	for _, prompt := range allPrompts {
		t.Run(prompt.prompt, func(t *testing.T) {
			result := helper.GetDisplayName(prompt.prompt)
			assert.NotEmpty(t, result, "Display name should not be empty")
		})
	}
}
