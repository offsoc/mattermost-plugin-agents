// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/types"
)

// Provider implements the DevBot role tool provider
type Provider struct {
	httpClient       *http.Client
	service          *Service
	metadataProvider shared.MetadataProvider
}

// NewProvider creates a new DevBot tool provider
func NewProvider(
	httpClient *http.Client,
	service *Service,
	metadataProvider shared.MetadataProvider,
) *Provider {
	return &Provider{
		httpClient:       httpClient,
		service:          service,
		metadataProvider: metadataProvider,
	}
}

// GetToolDefinitions returns all DevBot-specific tools
func (p *Provider) GetToolDefinitions() []types.ToolDefinition {
	var toolDefs []types.ToolDefinition

	if p.service == nil {
		return toolDefs
	}

	toolDefs = append(toolDefs, p.getDevToolDefinitions()...)

	if p.httpClient != nil {
		toolDefs = append(toolDefs, p.getJiraToolDefinitions()...)
	}

	return toolDefs
}

// getDevToolDefinitions returns developer-focused tools
func (p *Provider) getDevToolDefinitions() []types.ToolDefinition {
	return []types.ToolDefinition{
		{
			Tool: llm.Tool{
				Name:        ToolNameExplainCodePattern,
				Description: "Explain how specific Mattermost patterns and features are implemented with real code examples from the codebase. CALL THIS TOOL when queries involve: (1) Understanding Mattermost implementation patterns (plugin hooks, slash commands, websockets), (2) Finding code examples from Mattermost repos, (3) Learning how specific features work in server/webapp/mobile, (4) API usage patterns and best practices. This tool searches GitHub repos, documentation, and technical guides. Examples: 'how do plugin hooks work', 'show me slash command examples', 'explain websocket event handling'",
				Schema:      llm.NewJSONSchemaFromStruct[ExplainCodePatternArgs](),
				Resolver:    p.ToolExplainCodePattern,
			},
			SafeOnly: true,
		},
		{
			Tool: llm.Tool{
				Name:        ToolNameDebugIssue,
				Description: "Help debug Mattermost-specific errors and issues by searching for similar problems and solutions. CALL THIS TOOL when queries involve: (1) Debugging specific error messages or stack traces, (2) Finding solutions to known Mattermost issues, (3) Troubleshooting plugin or API problems, (4) Understanding error causes and fixes. This tool searches GitHub issues, forum discussions, and troubleshooting docs. Examples: 'websocket connection failed error', 'plugin activation fails', 'channel creation error'",
				Schema:      llm.NewJSONSchemaFromStruct[DebugIssueArgs](),
				Resolver:    p.ToolDebugIssue,
			},
			SafeOnly: true,
		},
		{
			Tool: llm.Tool{
				Name:        ToolNameFindArchitecture,
				Description: "Locate and explain Mattermost architectural patterns, ADRs, and system design. CALL THIS TOOL when queries involve: (1) Understanding system architecture and component design, (2) Finding Architecture Decision Records (ADRs), (3) Learning about internal structure and patterns, (4) Understanding how major systems work. This tool searches Confluence docs, architecture documentation, and design discussions. Examples: 'plugin lifecycle architecture', 'message routing system', 'permissions system design'",
				Schema:      llm.NewJSONSchemaFromStruct[FindArchitectureArgs](),
				Resolver:    p.ToolFindArchitecture,
			},
			SafeOnly: true,
		},
		{
			Tool: llm.Tool{
				Name:        ToolNameGetAPIExamples,
				Description: "Find real-world Mattermost API and plugin hook usage examples from actual code. CALL THIS TOOL when queries involve: (1) Learning how to use specific Mattermost APIs, (2) Finding plugin hook implementation examples, (3) Understanding API method usage patterns, (4) Seeing real code using REST API or Plugin API. This tool searches plugin code, API documentation, and example repositories. Examples: 'CreatePost API examples', 'MessageWillBePosted hook usage', 'GetUser API examples'",
				Schema:      llm.NewJSONSchemaFromStruct[GetAPIExamplesArgs](),
				Resolver:    p.ToolGetAPIExamples,
			},
			SafeOnly: true,
		},
		{
			Tool: llm.Tool{
				Name:        ToolNameSummarizePRs,
				Description: "Summarize recent Mattermost pull requests, releases, and code changes. CALL THIS TOOL when queries involve: (1) Understanding recent changes to Mattermost repos, (2) Finding what changed in recent releases, (3) Tracking feature additions or bug fixes, (4) Understanding PR activity and trends. This tool searches GitHub PRs, release notes, and changelogs. Examples: 'recent server PRs', 'what changed in mobile last month', 'recent API changes'",
				Schema:      llm.NewJSONSchemaFromStruct[SummarizePRsArgs](),
				Resolver:    p.ToolSummarizePRs,
			},
			SafeOnly: true,
		},
	}
}

// getJiraToolDefinitions returns Jira-related Dev tools
func (p *Provider) getJiraToolDefinitions() []types.ToolDefinition {
	return []types.ToolDefinition{}
}

// MatchesBot checks if this provider should be used for the given bot
func (p *Provider) MatchesBot(bot *bots.Bot) bool {
	if bot == nil {
		return false
	}

	botName := strings.ToLower(bot.GetConfig().Name)
	botDisplayName := strings.ToLower(bot.GetConfig().DisplayName)

	return strings.Contains(botName, "dev") ||
		strings.Contains(botName, "developer") ||
		strings.Contains(botDisplayName, "dev") ||
		strings.Contains(botDisplayName, "developer")
}

// GetToolMetadata returns metadata for a Dev tool
func (p *Provider) GetToolMetadata(toolName string) (types.ToolMetadata, bool) {
	return GetToolMetadata(toolName)
}

// GetSupportedDataSources returns data sources for a Dev tool
func (p *Provider) GetSupportedDataSources(toolName string) []string {
	return GetSupportedDataSources(toolName)
}
