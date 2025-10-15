// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
)

// ToolCompileMarketResearch resolves the CompileMarketResearch tool
func (p *Provider) ToolCompileMarketResearch(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args CompileMarketResearchArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool CompileMarketResearch: %w", err)
	}

	// Normalize feature names
	normalizedFeatures := make([]string, len(args.PrimaryFeatures))
	for i, feature := range args.PrimaryFeatures {
		normalizedFeatures[i] = shared.NormalizeFeatureName(feature)
	}
	args.PrimaryFeatures = normalizedFeatures

	return p.service.CompileMarketResearch(llmContext, args, p.metadataProvider)
}

// ToolAnalyzeFeatureGaps resolves the AnalyzeFeatureGaps tool
func (p *Provider) ToolAnalyzeFeatureGaps(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args AnalyzeFeatureGapsArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool AnalyzeFeatureGaps: %w", err)
	}

	// Normalize feature names
	normalizedFeatures := make([]string, len(args.PrimaryFeatures))
	for i, feature := range args.PrimaryFeatures {
		normalizedFeatures[i] = shared.NormalizeFeatureName(feature)
	}
	args.PrimaryFeatures = normalizedFeatures

	return p.service.AnalyzeFeatureGaps(llmContext, args, p.metadataProvider)
}

// ToolAnalyzeStrategicAlignment resolves the AnalyzeStrategicAlignment tool
func (p *Provider) ToolAnalyzeStrategicAlignment(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args AnalyzeStrategicAlignmentArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool AnalyzeStrategicAlignment: %w", err)
	}

	// Normalize feature names
	normalizedFeatures := make([]string, len(args.PrimaryFeatures))
	for i, feature := range args.PrimaryFeatures {
		normalizedFeatures[i] = shared.NormalizeFeatureName(feature)
	}
	args.PrimaryFeatures = normalizedFeatures

	return p.service.AnalyzeStrategicAlignment(llmContext, args)
}
