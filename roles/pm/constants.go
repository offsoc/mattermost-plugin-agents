// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

// Intent detection keywords
const (
	// Task creation keywords
	KeywordCreateTask = "create task"
	KeywordNewTask    = "new task"
	KeywordNeedsTo    = "needs to"
	KeywordAssign     = "assign"

	// Status query keywords
	KeywordStatus    = "status"
	KeywordProgress  = "progress"
	KeywordBlocking  = "blocking"
	KeywordWorkingOn = "working on"

	// Task update keywords
	KeywordUpdate   = "update"
	KeywordChange   = "change"
	KeywordMove     = "move"
	KeywordReassign = "reassign"

	// Meeting/standup keywords
	KeywordStandup     = "standup"
	KeywordMeeting     = "meeting"
	KeywordActionItems = "action items"

	// Weak PM signal keywords
	KeywordBug     = "bug"
	KeywordIssue   = "issue"
	KeywordFeature = "feature"
	KeywordJira    = "jira"
)

// Prompt template names for PM role
const (
	PromptPmFeatureGapAnalysisSystem  = "pm_feature_gap_analysis_system"
	PromptPmMarketResearchSystem      = "pm_market_research_system"
	PromptPmMeetingActionItemsSystem  = "pm_meeting_action_items_system"
	PromptPmStandupFacilitationSystem = "pm_standup_facilitation_system"
	PromptPmStandupSummarySystem      = "pm_standup_summary_system"
	PromptPmStatusReportSystem        = "pm_status_report_system"
	PromptPmStrategicAlignmentSystem  = "pm_strategic_alignment_system"
	PromptPmTaskCreationSystem        = "pm_task_creation_system"
	PromptPmTaskUpdateSystem          = "pm_task_update_system"
)
