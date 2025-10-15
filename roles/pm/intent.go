// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/conversations/intentutils"
)

// Intent handles project management related conversations
type Intent struct{}

// Matches analyzes a message for PM-related intent and returns the appropriate prompt
func (p *Intent) Matches(message string) (string, float64) {
	msg := strings.ToLower(strings.TrimSpace(message))

	// Task creation - highest confidence with sophisticated patterns
	taskCreationPatterns := []intentutils.KeywordPattern{
		{Phrase: KeywordCreateTask, Weight: intentutils.WeightHigh, WordBoundary: false},
		{Phrase: KeywordNewTask, Weight: intentutils.WeightHigh, WordBoundary: false},
		{Phrase: KeywordNeedsTo, Weight: intentutils.WeightMedium, WordBoundary: false},
		{Phrase: KeywordAssign, Weight: intentutils.WeightMediumHigh, WordBoundary: true}, // Word boundary to avoid "assignment"
		{Pattern: `\b(create|make|add)\s+(a\s+)?(task|ticket|issue)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(need|should|must)\s+(to\s+)?(create|make|do|fix|implement)\b`, Weight: intentutils.WeightMediumLow},
		{Pattern: `\b(assign\s+to|give\s+to|hand\s+over\s+to)\b`, Weight: intentutils.WeightMedium},
	}
	if matched, score := intentutils.MatchesPattern(msg, taskCreationPatterns); matched {
		confidence := intentutils.HighConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmTaskCreationSystem, confidence
	}

	// Strategic alignment, prioritization, and stakeholder trade-offs
	strategicPatterns := []intentutils.KeywordPattern{
		{Pattern: `\b(vision|mission|strategic\s+vision|product\s+vision)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(align(s|ment|ing)?|strategic\s+alignment|alignment\s+with)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(strateg(y|ic|ies)|strategic\s+fit|strategic\s+direction)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(prioriti[sz](e|ing|ation)|de[-\s]?prioriti[sz]e)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(trade[-\s]?offs?|trade[-\s]?off)\b`, Weight: intentutils.WeightHigh},
		{Pattern: `\b(rice|okrs?|framework|prioritization\s+framework|business\s+case)\b`, Weight: intentutils.WeightMediumHigh},
		{Pattern: `\b(stakeholders?|head\s+of\s+sales|sales\s+lead|engineering\s+lead|vp\s+of\s+engineering|cto)\b`, Weight: intentutils.WeightMediumHigh},
		{Pattern: `\b(technical\s+debt|maintenance\s+debt|long[-\s]?term\s+maintenance|complexity)\b`, Weight: intentutils.WeightMedium},
		{Pattern: `\b(present\s+a\s+case|make\s+the\s+case|justify|case\s+for|recommendation[s]?)\b`, Weight: intentutils.WeightMediumHigh},
		{Pattern: `\b(go[-\s]?to[-\s]?market|gtm)\b`, Weight: intentutils.WeightMediumLow},
	}
	if matched, score := intentutils.MatchesPattern(msg, strategicPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmStrategicAlignmentSystem, confidence
	}

	// Status queries with semantic patterns
	statusPatterns := []intentutils.KeywordPattern{
		{Phrase: KeywordStatus, Weight: 1.0, WordBoundary: true},
		{Phrase: KeywordProgress, Weight: 1.0, WordBoundary: true},
		{Phrase: KeywordBlocking, Weight: 0.9, WordBoundary: true},
		{Phrase: KeywordWorkingOn, Weight: 1.0, WordBoundary: false},
		{Pattern: `\b(what['']?s\s+the\s+status|status\s+update|progress\s+report)\b`, Weight: 1.0},
		{Pattern: `\b(how['']?s\s+(it\s+)?going|where\s+are\s+we|what['']?s\s+happening)\b`, Weight: 0.8},
		{Pattern: `\b(show\s+me|tell\s+me)\s+.*(progress|status|working\s+on)\b`, Weight: 0.9},
		{Pattern: `\b(blocked|blocker|impediment)\b`, Weight: 0.7},
	}
	if matched, score := intentutils.MatchesPattern(msg, statusPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmStatusReportSystem, confidence
	}

	// Task updates with action-oriented patterns
	updatePatterns := []intentutils.KeywordPattern{
		{Phrase: KeywordUpdate, Weight: 1.0, WordBoundary: true},
		{Phrase: KeywordChange, Weight: 0.8, WordBoundary: true},
		{Phrase: KeywordMove, Weight: 0.9, WordBoundary: true},
		{Phrase: KeywordReassign, Weight: 1.0, WordBoundary: false},
		{Pattern: `\b(update|modify|edit|change)\s+.*(task|ticket|issue|priority)\b`, Weight: 1.0},
		{Pattern: `\b(move\s+to|set\s+to|change\s+to)\s+(high|low|medium|done|in\s*progress)\b`, Weight: 0.9},
		{Pattern: `\b(reassign|transfer|handover)\s+(to|from)\b`, Weight: 1.0},
	}
	if matched, score := intentutils.MatchesPattern(msg, updatePatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmTaskUpdateSystem, confidence
	}

	// Action items with meeting context
	actionItemPatterns := []intentutils.KeywordPattern{
		{Phrase: KeywordActionItems, Weight: 1.0, WordBoundary: false},
		{Pattern: `\b(action\s+items?|follow\s*up\s+items?|todo\s+items?)\b`, Weight: 1.0},
		{Pattern: `\b(what\s+did\s+we\s+decide|next\s+steps|follow\s*up)\b`, Weight: 0.8},
		{Pattern: `\b(meeting\s+summary|recap\s+the\s+meeting)\b`, Weight: 0.7},
	}
	if matched, score := intentutils.MatchesPattern(msg, actionItemPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmMeetingActionItemsSystem, confidence
	}

	// Meeting/standup facilitation
	meetingPatterns := []intentutils.KeywordPattern{
		{Phrase: KeywordStandup, Weight: 1.0, WordBoundary: true},
		{Phrase: KeywordMeeting, Weight: 0.8, WordBoundary: true},
		{Pattern: `\b(start|begin|run)\s+(standup|meeting|daily)\b`, Weight: 1.0},
		{Pattern: `\b(daily\s+standup|team\s+standup|scrum\s+meeting)\b`, Weight: 1.0},
		{Pattern: `\b(facilitate|moderate)\s+.*(meeting|standup)\b`, Weight: 0.9},
	}
	if matched, score := intentutils.MatchesPattern(msg, meetingPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmStandupFacilitationSystem, confidence
	}

	// Feature gap analysis
	featureGapPatterns := []intentutils.KeywordPattern{
		{Pattern: `\b(feature\s+gaps?|gap\s+analysis)\b`, Weight: 1.0},
		{Pattern: `\bidentify\s+(any\s+)?gaps?\b`, Weight: 0.9},
		{Pattern: `\bgaps?\s+in\s+(our|current)\s+\w+`, Weight: 0.9},
		{Pattern: `\b(mobile|strategy)\s+(gaps?|limitations?)\b`, Weight: 0.8},
		{Pattern: `\bmissing\s+features?\b`, Weight: 1.0},
		{Pattern: `\bfeatures?\s+(are\s+)?missing\b`, Weight: 1.0},
		{Pattern: `\b(competitive\s+(gaps?|parity|analysis)|competitor\s+comparison)\b`, Weight: 1.0},
		{Pattern: `\b(what\s+(are\s+we|do\s+we)\s+missing|features\s+we\s+lack)\b`, Weight: 0.9},
		{Pattern: `\bwhat\s+features?\s+are\s+missing\b`, Weight: 0.9},
		{Pattern: `\b(customers?\s+(need|want|request)|customer\s+feedback)\b`, Weight: 0.8},
		{Pattern: `\b(deal\s+blockers?|adoption\s+barriers?|pain\s+points?)\b`, Weight: 0.9},
		{Pattern: `\b(vs\s+slack|vs\s+teams|compared\s+to|parity\s+with)\b`, Weight: 0.8},
		{Pattern: `\b(limitations?|constraints?|can['']?t\s+do|doesn['']?t\s+support)\b`, Weight: 0.7},
	}
	if matched, score := intentutils.MatchesPattern(msg, featureGapPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmFeatureGapAnalysisSystem, confidence
	}

	// Market research analysis
	marketResearchPatterns := []intentutils.KeywordPattern{
		{Pattern: `\b(market\s+(research|analysis|trends|landscape)|competitive\s+landscape)\b`, Weight: 1.0},
		{Pattern: `\b(market\s+opportunity|strategic\s+(insight|analysis))\b`, Weight: 1.0},
		{Pattern: `\b(competitor\s+(analysis|research)|competitive\s+intelligence)\b`, Weight: 0.9},
		{Pattern: `\b(industry\s+trends|market\s+positioning)\b`, Weight: 0.8},
		{Pattern: `\b(research\s+the\s+market|analyze\s+the\s+competition)\b`, Weight: 0.9},
	}
	if matched, score := intentutils.MatchesPattern(msg, marketResearchPatterns); matched {
		confidence := intentutils.MediumConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmMarketResearchSystem, confidence
	}

	// Weak PM signals with context awareness
	weakSignalPatterns := []intentutils.KeywordPattern{
		{Word: KeywordBug, Weight: 0.6, WordBoundary: true},
		{Word: KeywordIssue, Weight: 0.5, WordBoundary: true},
		{Word: KeywordFeature, Weight: 0.7, WordBoundary: true},
		{Word: KeywordJira, Weight: 0.8, WordBoundary: true},
		{Pattern: `\b(there['']?s\s+a\s+bug|found\s+a\s+bug|bug\s+report)\b`, Weight: 0.8},
		{Pattern: `\bbug\s+in\s+(the\s+)?\w+`, Weight: 0.9},
		{Pattern: `\b(new\s+feature|feature\s+request|enhancement)\b`, Weight: 0.7},
		{Pattern: `\b(jira\s+ticket|jira\s+issue|MM-\d+)\b`, Weight: 0.9},
		{Pattern: `\b(something['']?s\s+broken|not\s+working|error)\b`, Weight: 0.5},
	}
	if matched, score := intentutils.MatchesPattern(msg, weakSignalPatterns); matched {
		confidence := intentutils.LowConfidenceThreshold * score
		if confidence > intentutils.MaxConfidenceScore {
			confidence = intentutils.MaxConfidenceScore
		}
		return PromptPmTaskCreationSystem, confidence
	}

	return "", intentutils.NoConfidence
}
