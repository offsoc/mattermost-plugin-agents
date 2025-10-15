// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"github.com/mattermost/mattermost-plugin-ai/mmtools/types"
)

// pmToolMetadataRegistry contains PM tool metadata only
var pmToolMetadataRegistry = map[string]types.ToolMetadata{
	ToolNameCompileMarketResearch: {
		Name:                 ToolNameCompileMarketResearch,
		Description:          "Compile market research insights from conversations and external documentation",
		SupportedDataSources: []string{"mattermost_docs", "mattermost_handbook", "mattermost_forum", "mattermost_blog", "mattermost_newsroom", "github_repos", "community_forum", "mattermost_hub", "confluence_docs", "plugin_marketplace", "jira_docs", "productboard_features", "zendesk_tickets"},
		IntentKeywords:       []string{"market", "competitive", "competitor", "analysis", "trends", "trend", "research", "comparison", "benchmarking", "strategy", "positioning", "vision", "business", "discussion", "feedback", "community", "insights", "industry", "announcements", "alternatives", "alternative", "vs", "evaluation", "customer", "sales", "ecosystem", "third-party", "user needs", "voting", "requests", "requirements", "opportunity", "growth", "adoption", "usage", "alignment"},
	},
	ToolNameAnalyzeFeatureGaps: {
		Name:                 ToolNameAnalyzeFeatureGaps,
		Description:          "Analyze feature gaps using customer feedback and external documentation",
		SupportedDataSources: []string{"mattermost_docs", "mattermost_handbook", "mattermost_forum", "mattermost_blog", "mattermost_newsroom", "github_repos", "community_forum", "mattermost_hub", "confluence_docs", "plugin_marketplace", "jira_docs", "productboard_features", "zendesk_tickets"},
		IntentKeywords:       []string{"gaps", "limitations", "limitation", "missing", "not supported", "capabilities", "features", "functionality", "requirements", "needs", "issues", "issue", "feedback", "user feedback", "requests", "customer request", "feature request", "problems", "problem", "suggestions", "improvements", "specifications", "roadmap", "development", "announcements", "plans", "enhancement", "parity", "doesn't support", "can't", "workaround", "pain points", "customer objections", "deal blockers", "adoption barriers", "feature specs", "priorities", "customer needs", "acceptance criteria", "user stories", "alternatives", "extensions", "integrations", "plugins", "ideas", "bugs", "bug", "enhancements", "competitive", "competitor", "vs"},
	},
	ToolNameAnalyzeStrategicAlignment: {
		Name:                 ToolNameAnalyzeStrategicAlignment,
		Description:          "Analyze strategic alignment of features with company vision, apply PM frameworks, and provide stakeholder-balanced recommendations",
		SupportedDataSources: []string{"mattermost_docs", "mattermost_handbook", "mattermost_forum", "mattermost_blog", "mattermost_newsroom", "github_repos", "community_forum", "mattermost_hub", "confluence_docs", "plugin_marketplace", "jira_docs", "productboard_features", "zendesk_tickets"},
		IntentKeywords:       []string{"vision", "mission", "strategy", "goals", "objectives", "roadmap", "framework", "alignment", "okr", "priorities", "north star", "strategic plan", "strategic", "prioritization", "planning", "announcements", "milestone", "epic", "priority", "business case", "ecosystem", "stakeholder"},
	},
}

// GetToolMetadata returns metadata for a specific PM tool
func GetToolMetadata(toolName string) (types.ToolMetadata, bool) {
	metadata, exists := pmToolMetadataRegistry[toolName]
	return metadata, exists
}

// GetSupportedDataSources returns all data sources supported by a PM tool
func GetSupportedDataSources(toolName string) []string {
	if metadata, exists := pmToolMetadataRegistry[toolName]; exists {
		return metadata.SupportedDataSources
	}
	return []string{}
}
