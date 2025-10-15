// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package types

import "github.com/mattermost/mattermost-plugin-ai/llm"

// ToolDefinition defines a tool with its safety level
type ToolDefinition struct {
	Tool     llm.Tool
	SafeOnly bool // true = safe for channels, false = DM only
}

// ToolMetadata defines tool capabilities and external documentation support
type ToolMetadata struct {
	Name                 string
	Description          string
	SupportedDataSources []string
	IntentKeywords       []string
}
