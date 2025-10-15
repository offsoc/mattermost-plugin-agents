// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/llm"
)

// ToolExplainCodePattern resolves the ExplainCodePattern tool
func (p *Provider) ToolExplainCodePattern(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args ExplainCodePatternArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool ExplainCodePattern: %w", err)
	}

	return p.service.ExplainCodePattern(llmContext, args, p.metadataProvider)
}

// ToolDebugIssue resolves the DebugIssue tool
func (p *Provider) ToolDebugIssue(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args DebugIssueArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool DebugIssue: %w", err)
	}

	return p.service.DebugIssue(llmContext, args, p.metadataProvider)
}

// ToolFindArchitecture resolves the FindArchitecture tool
func (p *Provider) ToolFindArchitecture(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args FindArchitectureArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool FindArchitecture: %w", err)
	}

	return p.service.FindArchitecture(llmContext, args, p.metadataProvider)
}

// ToolGetAPIExamples resolves the GetAPIExamples tool
func (p *Provider) ToolGetAPIExamples(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args GetAPIExamplesArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool GetAPIExamples: %w", err)
	}

	return p.service.GetAPIExamples(llmContext, args, p.metadataProvider)
}

// ToolSummarizePRs resolves the SummarizePRs tool
func (p *Provider) ToolSummarizePRs(llmContext *llm.Context, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args SummarizePRsArgs
	if err := argsGetter(&args); err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool SummarizePRs: %w", err)
	}

	return p.service.SummarizePRs(llmContext, args, p.metadataProvider)
}
