// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package conversations

import (
	"github.com/mattermost/mattermost-plugin-ai/conversations/intentutils"
	"github.com/mattermost/mattermost-plugin-ai/prompts"
)

// Intent represents a conversation intent that can be detected from user messages
type Intent interface {
	Matches(message string) (promptName string, confidence float64)
}

// IntentDetector manages multiple intents and selects the best match
type IntentDetector struct {
	intents []Intent
}

// NewIntentDetector creates a new intent detector with no registered intents
func NewIntentDetector() *IntentDetector {
	return &IntentDetector{
		intents: []Intent{},
	}
}

// RegisterIntent adds a new intent to the detector
func (id *IntentDetector) RegisterIntent(intent Intent) {
	id.intents = append(id.intents, intent)
}

// DetectIntent analyzes a message and returns the appropriate system prompt
func (id *IntentDetector) DetectIntent(message string) string {
	bestIntent := prompts.PromptDirectMessageQuestionSystem
	bestConfidence := 0.0

	for _, intent := range id.intents {
		if promptName, confidence := intent.Matches(message); confidence > bestConfidence && confidence > intentutils.MinimumConfidenceThreshold {
			bestIntent = promptName
			bestConfidence = confidence
		}
	}

	return bestIntent
}
