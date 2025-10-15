// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/conversations/intentutils"
	"github.com/stretchr/testify/assert"
)

func TestIntent_Matches_TaskCreation(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		shouldMatch    bool
		minConfidence  float64
	}{
		{
			name:           "Direct 'create task' phrase",
			message:        "Please create task for this bug fix",
			expectedPrompt: PromptPmTaskCreationSystem,
			shouldMatch:    true,
			minConfidence:  0.7,
		},
		{
			name:           "New task phrase",
			message:        "We need a new task for the mobile feature",
			expectedPrompt: PromptPmTaskCreationSystem,
			shouldMatch:    true,
			minConfidence:  0.7,
		},
		{
			name:           "Assign keyword",
			message:        "Can you assign this to John?",
			expectedPrompt: PromptPmTaskCreationSystem,
			shouldMatch:    true,
			minConfidence:  0.6,
		},
		{
			name:           "Create ticket pattern",
			message:        "Create a ticket for the authentication issue",
			expectedPrompt: PromptPmTaskCreationSystem,
			shouldMatch:    true,
			minConfidence:  0.7,
		},
		{
			name:           "Need to implement pattern",
			message:        "We need to implement SSO support",
			expectedPrompt: PromptPmTaskCreationSystem,
			shouldMatch:    true,
			minConfidence:  0.4,
		},
		{
			name:           "Assign to pattern",
			message:        "Assign to the backend team",
			expectedPrompt: PromptPmTaskCreationSystem,
			shouldMatch:    true,
			minConfidence:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			if tt.shouldMatch {
				assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
				assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
				assert.Greater(t, confidence, intentutils.NoConfidence, "Should have confidence > 0")
			} else {
				assert.NotEqual(t, tt.expectedPrompt, prompt, "Should not match")
			}
		})
	}
}

func TestIntent_Matches_StrategicAlignment(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Vision keyword",
			message:        "What's our product vision for 2025?",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.5,
		},
		{
			name:           "Strategic alignment",
			message:        "How does this align with our strategic goals?",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.5,
		},
		{
			name:           "Prioritization",
			message:        "We need to prioritize the mobile features",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.5,
		},
		{
			name:           "Trade-offs",
			message:        "What are the trade-offs for implementing this now?",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.5,
		},
		{
			name:           "OKRs framework",
			message:        "Can we use RICE to prioritize these features?",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Stakeholder keyword",
			message:        "Need to discuss with stakeholders",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Technical debt",
			message:        "Should we address technical debt now?",
			expectedPrompt: PromptPmStrategicAlignmentSystem,
			minConfidence:  0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, confidence := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Prompt mismatch")
			assert.GreaterOrEqual(t, confidence, tt.minConfidence, "Confidence too low")
			assert.Greater(t, confidence, intentutils.NoConfidence, "Should have confidence > 0")
		})
	}
}

func TestIntent_Matches_StatusQueries(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Direct status query",
			message:        "What's the status of the mobile project?",
			expectedPrompt: PromptPmStatusReportSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Progress keyword",
			message:        "Show me progress on the authentication feature",
			expectedPrompt: PromptPmStatusReportSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Working on pattern",
			message:        "What are you working on?",
			expectedPrompt: PromptPmStatusReportSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Where are we pattern",
			message:        "Where are we with the mobile project?",
			expectedPrompt: PromptPmStatusReportSystem,
			minConfidence:  0.3,
		},
		{
			name:           "Blocker with context",
			message:        "The team is blocked on this task",
			expectedPrompt: PromptPmStatusReportSystem,
			minConfidence:  0.2,
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

func TestIntent_Matches_TaskUpdates(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Update task priority",
			message:        "Update the task priority to high",
			expectedPrompt: PromptPmTaskUpdateSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Modify task",
			message:        "Modify the task description",
			expectedPrompt: PromptPmTaskUpdateSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Move task to high priority",
			message:        "Move task to high priority",
			expectedPrompt: PromptPmTaskUpdateSystem,
			minConfidence:  0.3,
		},
		{
			name:           "Reassign task",
			message:        "Reassign this task to Sarah",
			expectedPrompt: PromptPmTaskUpdateSystem,
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

func TestIntent_Matches_ActionItems(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Direct action items",
			message:        "What are the action items from today's meeting?",
			expectedPrompt: PromptPmMeetingActionItemsSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Follow up after meeting",
			message:        "List the follow up from yesterday's meeting",
			expectedPrompt: PromptPmMeetingActionItemsSystem,
			minConfidence:  0.3,
		},
		{
			name:           "What did we decide",
			message:        "What did we decide in the planning meeting?",
			expectedPrompt: PromptPmMeetingActionItemsSystem,
			minConfidence:  0.3,
		},
		{
			name:           "Action items from meeting",
			message:        "What are the action items from the meeting?",
			expectedPrompt: PromptPmMeetingActionItemsSystem,
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

func TestIntent_Matches_MeetingFacilitation(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Start standup",
			message:        "Start the daily standup",
			expectedPrompt: PromptPmStandupFacilitationSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Daily standup",
			message:        "Time for daily standup",
			expectedPrompt: PromptPmStandupFacilitationSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Facilitate meeting",
			message:        "Can you facilitate the retrospective meeting?",
			expectedPrompt: PromptPmStandupFacilitationSystem,
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

func TestIntent_Matches_FeatureGapAnalysis(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Feature gaps",
			message:        "What are the feature gaps in our mobile app?",
			expectedPrompt: PromptPmFeatureGapAnalysisSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Missing features",
			message:        "What features are missing compared to Slack?",
			expectedPrompt: PromptPmFeatureGapAnalysisSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Competitive gaps",
			message:        "Identify competitive gaps vs Teams",
			expectedPrompt: PromptPmFeatureGapAnalysisSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Customer needs",
			message:        "What do customers need that we don't have?",
			expectedPrompt: PromptPmFeatureGapAnalysisSystem,
			minConfidence:  0.3,
		},
		{
			name:           "Deal blockers",
			message:        "What are the common deal blockers?",
			expectedPrompt: PromptPmFeatureGapAnalysisSystem,
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

func TestIntent_Matches_MarketResearch(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Market research",
			message:        "What does market research show about collaboration tools?",
			expectedPrompt: PromptPmMarketResearchSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Competitive landscape",
			message:        "Analyze the competitive landscape",
			expectedPrompt: PromptPmMarketResearchSystem,
			minConfidence:  0.4,
		},
		{
			name:           "Industry trends",
			message:        "What are the industry trends?",
			expectedPrompt: PromptPmMarketResearchSystem,
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

func TestIntent_Matches_WeakSignals(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name           string
		message        string
		expectedPrompt string
		minConfidence  float64
	}{
		{
			name:           "Bug mention",
			message:        "There's a bug in the mobile app",
			expectedPrompt: PromptPmTaskCreationSystem,
			minConfidence:  0.15,
		},
		{
			name:           "Feature request",
			message:        "We got a feature request from customer",
			expectedPrompt: PromptPmTaskCreationSystem,
			minConfidence:  0.15,
		},
		{
			name:           "Jira mention",
			message:        "There's a jira ticket for this",
			expectedPrompt: PromptPmTaskCreationSystem,
			minConfidence:  0.15,
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
			name:    "Random conversation",
			message: "I like pizza",
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

func TestIntent_Matches_PriorityOrder(t *testing.T) {
	intent := &Intent{}

	tests := []struct {
		name              string
		message           string
		expectedPrompt    string
		notExpectedPrompt string
	}{
		{
			name:              "Task creation beats weak signal",
			message:           "Create task for this bug",
			expectedPrompt:    PromptPmTaskCreationSystem,
			notExpectedPrompt: "",
		},
		{
			name:              "Strategic alignment beats weak signals",
			message:           "How do we prioritize this feature vs technical debt?",
			expectedPrompt:    PromptPmStrategicAlignmentSystem,
			notExpectedPrompt: PromptPmTaskCreationSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, _ := intent.Matches(tt.message)

			assert.Equal(t, tt.expectedPrompt, prompt, "Should match expected prompt")
			if tt.notExpectedPrompt != "" {
				assert.NotEqual(t, tt.notExpectedPrompt, prompt, "Should not match lower priority prompt")
			}
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
			message:        "CREATE TASK FOR THIS BUG",
			expectedPrompt: PromptPmTaskCreationSystem,
		},
		{
			name:           "Mixed case",
			message:        "CrEaTe TaSk FoR tHiS bUg",
			expectedPrompt: PromptPmTaskCreationSystem,
		},
		{
			name:           "Lowercase",
			message:        "create task for this bug",
			expectedPrompt: PromptPmTaskCreationSystem,
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
			message:        "   create task for bug",
			expectedPrompt: PromptPmTaskCreationSystem,
		},
		{
			name:           "Trailing whitespace",
			message:        "create task for bug   ",
			expectedPrompt: PromptPmTaskCreationSystem,
		},
		{
			name:           "Multiple spaces",
			message:        "create    task    for    bug",
			expectedPrompt: PromptPmTaskCreationSystem,
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
			name:           "High confidence task creation",
			message:        "Create a task for this feature",
			expectedPrompt: PromptPmTaskCreationSystem,
			minConfidence:  0.7,
			maxConfidence:  1.0,
		},
		{
			name:           "Medium confidence status",
			message:        "What's the status?",
			expectedPrompt: PromptPmStatusReportSystem,
			minConfidence:  0.3,
			maxConfidence:  0.9,
		},
		{
			name:           "Low confidence weak signal",
			message:        "There's a bug",
			expectedPrompt: PromptPmTaskCreationSystem,
			minConfidence:  0.1,
			maxConfidence:  0.6,
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
