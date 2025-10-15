// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/types"
)

// Provider implements the PM role tool provider
type Provider struct {
	httpClient       *http.Client
	service          *Service
	metadataProvider shared.MetadataProvider
}

// NewProvider creates a new PM tool provider
// Service and metadataProvider should be created by the parent (MMToolProvider)
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

// GetToolDefinitions returns all PM-specific tools
func (p *Provider) GetToolDefinitions() []types.ToolDefinition {
	var toolDefs []types.ToolDefinition

	// Only provide tools if service is initialized
	if p.service == nil {
		return toolDefs
	}

	if p.httpClient != nil {
		toolDefs = append(toolDefs, p.getJiraToolDefinitions()...)
	}

	// PM-specific analysis tools
	toolDefs = append(toolDefs, p.getPMAnalysisToolDefinitions()...)

	return toolDefs
}

// getJiraToolDefinitions returns Jira-related PM tools
func (p *Provider) getJiraToolDefinitions() []types.ToolDefinition {
	return []types.ToolDefinition{}
}

// getPMAnalysisToolDefinitions returns PM analysis tools
func (p *Provider) getPMAnalysisToolDefinitions() []types.ToolDefinition {
	return []types.ToolDefinition{
		{
			Tool: llm.Tool{
				Name:        "CompileMarketResearch",
				Description: "Market intelligence engine that SEARCHES community forums, UserVoice, documentation sites, and internal channels for competitive analysis and market insights. CALL THIS TOOL when queries involve: (1) Searching forums or community discussions for market feedback, (2) Finding competitor mentions or customer feedback across data sources, (3) Gathering feature requests from UserVoice or forums, (4) Business context analysis (market positioning, competitive gaps). This tool automatically searches relevant data sources based on your query. Examples: 'search forums for feature X feedback', 'find customer requests about Y', 'what are users saying about Z in forums'",
				Schema:      llm.NewJSONSchemaFromStruct[CompileMarketResearchArgs](),
				Resolver:    p.ToolCompileMarketResearch,
			},
			SafeOnly: true,
		},
		{
			Tool: llm.Tool{
				Name:        "AnalyzeFeatureGaps",
				Description: "Feature gap analysis engine that SEARCHES community forums, UserVoice, customer feedback channels, and documentation for missing capabilities and customer needs. CALL THIS TOOL when queries involve: (1) Searching forums for feature requests or pain points, (2) Finding what features customers are asking for in UserVoice or community discussions, (3) Identifying missing capabilities from customer feedback, (4) Gap analysis between current features and customer needs. This tool automatically searches and analyzes data from forums, UserVoice, and other feedback sources. Examples: 'search forums for analytics requests', 'find reporting dashboard feedback in community', 'what features are users requesting'",
				Schema:      llm.NewJSONSchemaFromStruct[AnalyzeFeatureGapsArgs](),
				Resolver:    p.ToolAnalyzeFeatureGaps,
			},
			SafeOnly: true,
		},
		{
			Tool: llm.Tool{
				Name:        "AnalyzeStrategicAlignment",
				Description: "Strategic decision support engine for PM analysis including vision alignment, prioritization frameworks, and strategic decisions. CALL THIS TOOL when queries involve: (1) Strategic alignment questions (how does X align with company vision/goals/mission), (2) PM framework application (RICE, OKR, prioritization), (3) Feature/product strategy decisions, (4) Stakeholder-driven scenarios with business context. Use for strategic questions like 'how does this align with...', 'should we prioritize...', 'what's the strategic impact...' DO NOT use for pure educational questions like 'What is RICE?' Examples: ✅ 'How does feature X align with our mission?' → USE TOOL ✅ 'Should we prioritize A or B?' → USE TOOL ❌ 'How does RICE framework work?' → DON'T USE TOOL",
				Schema:      llm.NewJSONSchemaFromStruct[AnalyzeStrategicAlignmentArgs](),
				Resolver:    p.ToolAnalyzeStrategicAlignment,
			},
			SafeOnly: true,
		},
	}
}

// MatchesBot checks if this provider should be used for the given bot
func (p *Provider) MatchesBot(bot *bots.Bot) bool {
	if bot == nil {
		return false
	}

	botName := strings.ToLower(bot.GetConfig().Name)
	botDisplayName := strings.ToLower(bot.GetConfig().DisplayName)

	return strings.Contains(botName, "pm") ||
		strings.Contains(botName, "project") ||
		strings.Contains(botName, "product") ||
		strings.Contains(botDisplayName, "pm") ||
		strings.Contains(botDisplayName, "project") ||
		strings.Contains(botDisplayName, "product")
}

// GetToolMetadata returns metadata for a PM tool
func (p *Provider) GetToolMetadata(toolName string) (types.ToolMetadata, bool) {
	return GetToolMetadata(toolName)
}

// GetSupportedDataSources returns data sources for a PM tool
func (p *Provider) GetSupportedDataSources(toolName string) []string {
	return GetSupportedDataSources(toolName)
}
