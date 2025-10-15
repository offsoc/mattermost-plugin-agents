// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

// IntentHelper implements rolebot.IntentHelper for dev bots
type IntentHelper struct{}

// DetectIntent analyzes a message and returns the appropriate dev intent prompt
func (h *IntentHelper) DetectIntent(message string) string {
	intentDetector := &Intent{}
	intent, _ := intentDetector.Matches(message)
	if intent == "" {
		intent = prompts.PromptDirectMessageQuestionSystem
	}
	return intent
}

// HasIntentChanged determines if the dev intent has changed enough to warrant a context switch
func (h *IntentHelper) HasIntentChanged(previousIntent, currentIntent string) bool {
	if previousIntent == currentIntent {
		return false
	}

	if previousIntent == prompts.PromptDirectMessageQuestionSystem || currentIntent == prompts.PromptDirectMessageQuestionSystem {
		return true
	}

	return true
}

// GetIntentDisplayName returns a human-readable name for a dev intent
func (h *IntentHelper) GetDisplayName(intent string) string {
	switch intent {
	case PromptDevCodeExplanationSystem:
		return "code explanation"
	case PromptDevDebuggingSystem:
		return "debugging"
	case PromptDevArchitectureSystem:
		return "architecture"
	case PromptDevAPIExamplesSystem:
		return "API examples"
	case PromptDevPRSummarySystem:
		return "PR summary"
	case prompts.PromptDirectMessageQuestionSystem:
		return "general conversation"
	default:
		return "development"
	}
}
