// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package rolebot

// IntentHelper provides utilities for working with intents
type IntentHelper interface {
	// GetDisplayName returns a human-readable name for an intent
	GetDisplayName(intent string) string

	// DetectIntent analyzes a message and returns the appropriate intent prompt name
	DetectIntent(message string) string

	// HasIntentChanged determines if the intent has changed enough to warrant a context switch
	HasIntentChanged(previousIntent, currentIntent string) bool
}
