// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/conversations/intentutils"
)

// Intent handles developer-related conversations
type Intent struct{}

// Matches analyzes a message for dev-related intent and returns the appropriate prompt
func (i *Intent) Matches(message string) (string, float64) {
	msg := strings.ToLower(strings.TrimSpace(message))

	// Check debugging FIRST - more specific than code explanation
	// Debugging intent - Tier 1 & 2: Comprehensive patterns for troubleshooting
	debugPatterns := []intentutils.KeywordPattern{
		// Tier 1: Direct error/problem statements (high confidence)
		{Pattern: `\b(error|exception|crash|fail(ing|ed|ure)?|broken|bug)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(getting\s+(a|an)|receiving\s+(a|an)|throwing\s+(a|an))\s+(error|exception)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(not\s+working|doesn['']?t\s+work|isn['']?t\s+working|won['']?t\s+work)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(debug|troubleshoot|diagnose|investigate)\b`, Weight: intentutils.WeightHigh},

		// Tier 1: "Not being called/triggered" - common plugin issue (high confidence)
		{Pattern: `\b(not\s+being\s+(called|triggered|executed|invoked|fired)|never\s+(runs?|executes?|triggers?))\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(hook|callback|handler|listener).*(not|never|isn['']?t|doesn['']?t)\b`, Weight: intentutils.WeightHigh},

		// Tier 1: Validation/configuration failures (high confidence)
		{Pattern: `\b(validation\s+(failing|failed|error)|invalid\s+(manifest|config|configuration))\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(manifest|config).*(fail|error|invalid|problem)\b`, Weight: intentutils.WeightHigh},

		// Tier 2: Specific technical error contexts (medium-high confidence)
		{Pattern: `\b(websocket|plugin|api|channel|post|user|database|query)\s+(error|issue|problem|failing|failed)\b`, Weight: intentutils.WeightMediumHigh},
		{Pattern: `\b(user\s+not\s+found|channel\s+not\s+found|permission\s+denied)\b`, Weight: intentutils.WeightMediumHigh},

		// Tier 2: General problem indicators (medium confidence)
		{Pattern: `\b(issue|problem|wrong|incorrect)\b`, Weight: intentutils.WeightMedium},
		{Pattern: `\b(why|what['']?s\s+wrong|how\s+to\s+fix)\b`, Weight: intentutils.WeightMedium},
	}
	if matched, score := intentutils.MatchesPattern(msg, debugPatterns); matched {
		confidence := intentutils.HighConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptDevDebuggingSystem, confidence
	}

	// Code explanation intent - Tier 1 & 2: Comprehensive patterns for implementation questions
	codeExplanationPatterns := []intentutils.KeywordPattern{
		// Tier 1: "How do I/to" implementation questions (high confidence)
		{Pattern: `\b(how\s+(do\s+i|to|can\s+i|should\s+i))\s+(implement|create|add|build|make|use|setup|configure|integrate|work\s+with)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(how\s+(do\s+i|to|can\s+i))\s+(get|access|call|invoke)\b`, Weight: intentutils.WeightHigh},

		// Tier 1: "Find/show examples" requests (high confidence)
		{Pattern: `\b(find|show|get|need|want|looking\s+for)\s+(examples?|sample|demo|code|documentation)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(examples?\s+of|sample\s+code|demo\s+of)\b`, Weight: intentutils.WeightHigh},

		// Tier 1: "Explain/understand" with technical topics (high confidence)
		{Pattern: `\b(explain|describe|tell\s+me\s+about|what\s+is|how\s+does)\b.*(workflow|system|process|architecture|permission|connection|communication)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(explain|understand|clarify)\b.*(code|implementation|pattern|function|method|api)\b`, Weight: intentutils.WeightHigh},

		// Tier 2: Mattermost-specific technical terms (medium-high confidence)
		{Pattern: `\b(plugin|slash\s+command|webhook|hook|interactive\s+dialog|ephemeral|channel|post|user|team)\b`, Weight: intentutils.WeightMediumHigh},
		{Pattern: `\b(mattermost\s+(api|plugin|server)|plugin\s+api)\b`, Weight: intentutils.WeightMediumHigh},

		// Tier 2: API/method questions (medium confidence)
		{Pattern: `\b(api|endpoint|method|function)\b`, Weight: intentutils.WeightMedium},
		{Pattern: `\b(where|find|locate)\s+(is|the|can\s+i\s+find)?\s*(code|implementation|example|documentation)\b`, Weight: intentutils.WeightMedium},

		// Tier 2: Programming action verbs (medium confidence)
		{Pattern: `\b(implement|create|add|build|register|execute|call|invoke)\b`, Weight: intentutils.WeightMedium},
	}
	if matched, score := intentutils.MatchesPattern(msg, codeExplanationPatterns); matched {
		confidence := intentutils.HighConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptDevCodeExplanationSystem, confidence
	}

	// Architecture intent
	architecturePatterns := []intentutils.KeywordPattern{
		{Pattern: `\b(architecture|design|pattern|structure|adr|system\s+design)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(how\s+(is|does)|what['']?s\s+the)\s+.*(architecture|design|structured)\b`, Weight: intentutils.WeightMediumHigh},
	}
	if matched, score := intentutils.MatchesPattern(msg, architecturePatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptDevArchitectureSystem, confidence
	}

	// API usage intent
	apiPatterns := []intentutils.KeywordPattern{
		{Pattern: `\b(api|example|usage|how\s+to\s+use|how\s+to\s+call)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(plugin\s+api|rest\s+api|websocket\s+api)\b`, Weight: intentutils.WeightMediumHigh},
	}
	if matched, score := intentutils.MatchesPattern(msg, apiPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptDevAPIExamplesSystem, confidence
	}

	// PR summary intent - Tier 1 & 2: Version changes and code updates
	prPatterns := []intentutils.KeywordPattern{
		// Tier 1: Direct PR/version/summarize questions (high confidence)
		{Pattern: `\b(summarize|summary).*(pr|pull\s+request|security|changes|commits|recent)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(pull\s+request|pr|recent\s+changes|commits|changelog)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(what\s+changed|changes\s+in|new\s+in|updates\s+in)\s+(version|v\d|release)\b`, Weight: intentutils.WeightHigh},

		// Tier 1: Version upgrade questions (high confidence)
		{Pattern: `\b(upgrade|migrat(e|ing)|updat(e|ing)|moving)\s+(from|to)\s+(v\d|version)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(breaking\s+changes|deprecated|migration\s+guide)\b`, Weight: intentutils.WeightHigh},

		// Tier 2: General update/release questions (medium confidence)
		{Pattern: `\b(recent|latest|new)\s+(update|release|version|feature)\b`, Weight: intentutils.WeightMedium},
		{Pattern: `\b(security|bug\s+fix).*(pr|update|patch|release)\b`, Weight: intentutils.WeightMedium},
	}
	if matched, score := intentutils.MatchesPattern(msg, prPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptDevPRSummarySystem, confidence
	}

	return "", intentutils.NoConfidence
}
