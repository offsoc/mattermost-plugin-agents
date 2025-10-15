// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

// ExplainCodePatternArgs defines the arguments for the ExplainCodePattern tool
type ExplainCodePatternArgs struct {
	CodePattern string `json:"code_pattern" jsonschema_description:"The Mattermost code pattern or feature to explain (e.g., 'plugin hooks', 'slash commands', 'websocket events', 'channel creation')"`
	Component   string `json:"component" jsonschema_description:"The Mattermost component: 'server', 'webapp', 'mobile', 'desktop', 'plugins'"`
	Language    string `json:"language,omitempty" jsonschema_description:"Programming language if relevant: 'go', 'react', 'typescript', 'javascript' (optional)"`
}

// DebugIssueArgs defines the arguments for the DebugIssue tool
type DebugIssueArgs struct {
	ErrorMessage string `json:"error_message" jsonschema_description:"The error message or issue description (e.g., 'websocket connection failed', 'plugin activation error')"`
	Component    string `json:"component" jsonschema_description:"The affected component: 'plugin API', 'websocket', 'channels', 'posts', 'users', 'permissions', 'authentication'"`
	Context      string `json:"context,omitempty" jsonschema_description:"Additional context like Mattermost version, reproduction steps, or environment details (optional)"`
}

// FindArchitectureArgs defines the arguments for the FindArchitecture tool
type FindArchitectureArgs struct {
	Topic string `json:"topic" jsonschema_description:"The architecture topic to explore (e.g., 'plugin lifecycle', 'message routing', 'permissions system', 'database schema')"`
	Scope string `json:"scope,omitempty" jsonschema_description:"The scope: 'server', 'webapp', 'mobile', 'full-stack' (optional, defaults to comprehensive)"`
}

// GetAPIExamplesArgs defines the arguments for the GetAPIExamples tool
type GetAPIExamplesArgs struct {
	APIName string `json:"api_name" jsonschema_description:"The Mattermost API or plugin hook name (e.g., 'CreatePost', 'GetUser', 'MessageWillBePosted', 'ServeHTTP')"`
	UseCase string `json:"use_case,omitempty" jsonschema_description:"Specific use case or variation (e.g., 'with attachments', 'bulk operations', 'error handling') (optional)"`
}

// SummarizePRsArgs defines the arguments for the SummarizePRs tool
type SummarizePRsArgs struct {
	Repository string `json:"repository,omitempty" jsonschema_description:"The Mattermost repository: 'mattermost/mattermost', 'mattermost/mattermost-webapp', 'mattermost/mattermost-mobile' (optional, defaults to main server repo)"`
	TimeRange  string `json:"time_range" jsonschema_description:"Time range for PRs: 'week', 'month', 'quarter' (default: 'week')"`
	Category   string `json:"category,omitempty" jsonschema_description:"PR category filter: 'features', 'bug_fixes', 'performance', 'api_changes', 'breaking_changes' (optional)"`
}
