// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

// IntentHelper implements rolebot.IntentHelper for PM bots
type IntentHelper struct{}

// DetectIntent analyzes a message and returns the appropriate PM intent prompt
func (h *IntentHelper) DetectIntent(message string) string {
	intentDetector := &Intent{}
	intent, _ := intentDetector.Matches(message)
	if intent == "" {
		intent = prompts.PromptDirectMessageQuestionSystem
	}
	return intent
}

// HasIntentChanged determines if the PM intent has changed enough to warrant a context switch
func (h *IntentHelper) HasIntentChanged(previousIntent, currentIntent string) bool {
	if previousIntent == currentIntent {
		return false
	}

	if previousIntent == prompts.PromptDirectMessageQuestionSystem || currentIntent == prompts.PromptDirectMessageQuestionSystem {
		return true
	}

	return true
}

// GetIntentDisplayName returns a human-readable name for a PM intent
func (h *IntentHelper) GetDisplayName(intent string) string {
	switch intent {
	case PromptPmTaskCreationSystem:
		return "task creation"
	case PromptPmStatusReportSystem:
		return "status reporting"
	case PromptPmTaskUpdateSystem:
		return "task updates"
	case PromptPmStandupFacilitationSystem:
		return "standup facilitation"
	case PromptPmMeetingActionItemsSystem:
		return "meeting action items"
	case prompts.PromptDirectMessageQuestionSystem:
		return "general conversation"
	default:
		return "project management"
	}
}
