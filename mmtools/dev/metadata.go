// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"github.com/mattermost/mattermost-plugin-ai/mmtools/types"
)

// devToolMetadataRegistry contains all Dev tool metadata
var devToolMetadataRegistry = map[string]types.ToolMetadata{
	ToolNameExplainCodePattern: {
		Name:                 ToolNameExplainCodePattern,
		Description:          "Explain how specific Mattermost patterns and features are implemented with code examples",
		SupportedDataSources: []string{"github_repos", "mattermost_docs", "confluence_docs", "mattermost_forum", "community_forum", "jira_docs"},
		IntentKeywords:       []string{"explain", "how does", "implementation", "code", "pattern", "plugin", "hook", "slash command", "webhook", "api", "example", "usage", "tutorial", "guide"},
	},
	ToolNameDebugIssue: {
		Name:                 ToolNameDebugIssue,
		Description:          "Help debug Mattermost-specific errors and issues with solutions from similar cases",
		SupportedDataSources: []string{"github_repos", "mattermost_forum", "community_forum", "confluence_docs", "mattermost_docs", "jira_docs"},
		IntentKeywords:       []string{"error", "debug", "issue", "problem", "failing", "broken", "fix", "troubleshoot", "crash", "exception", "websocket", "plugin", "api", "channel", "post", "not working", "doesn't work"},
	},
	ToolNameFindArchitecture: {
		Name:                 ToolNameFindArchitecture,
		Description:          "Locate and explain Mattermost architectural patterns, ADRs, and design decisions",
		SupportedDataSources: []string{"confluence_docs", "github_repos", "mattermost_docs", "mattermost_forum", "jira_docs"},
		IntentKeywords:       []string{"architecture", "design", "pattern", "structure", "adr", "system design", "how is", "what's the", "overview", "component", "module", "layer", "lifecycle"},
	},
	ToolNameGetAPIExamples: {
		Name:                 ToolNameGetAPIExamples,
		Description:          "Find Mattermost API and plugin hook usage examples from real code",
		SupportedDataSources: []string{"github_repos", "mattermost_docs", "plugin_marketplace", "community_forum"},
		IntentKeywords:       []string{"api", "example", "usage", "how to use", "how to call", "plugin api", "rest api", "websocket api", "CreatePost", "GetUser", "hook", "method"},
	},
	ToolNameSummarizePRs: {
		Name:                 ToolNameSummarizePRs,
		Description:          "Summarize recent Mattermost pull requests, releases, and code changes",
		SupportedDataSources: []string{"github_repos", "mattermost_blog", "mattermost_newsroom", "mattermost_docs"},
		IntentKeywords:       []string{"pull request", "pr", "recent changes", "commits", "what changed", "summarize", "summary", "updates", "releases", "new features", "bug fixes", "changelog"},
	},
}

// GetToolMetadata returns metadata for a specific Dev tool
func GetToolMetadata(toolName string) (types.ToolMetadata, bool) {
	metadata, exists := devToolMetadataRegistry[toolName]
	return metadata, exists
}

// GetSupportedDataSources returns all data sources supported by a Dev tool
func GetSupportedDataSources(toolName string) []string {
	if metadata, exists := devToolMetadataRegistry[toolName]; exists {
		return metadata.SupportedDataSources
	}
	return []string{}
}
