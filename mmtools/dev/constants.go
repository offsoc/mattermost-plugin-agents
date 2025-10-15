// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

// Tool names
const (
	ToolNameExplainCodePattern = "ExplainCodePattern"
	ToolNameDebugIssue         = "DebugIssue"
	ToolNameFindArchitecture   = "FindArchitecture"
	ToolNameGetAPIExamples     = "GetAPIExamples"
	ToolNameSummarizePRs       = "SummarizePRs"
)

// Mattermost repositories
const (
	RepoMattermostServer    = "mattermost/mattermost"
	RepoMattermostWebapp    = "mattermost/mattermost-webapp"
	RepoMattermostMobile    = "mattermost/mattermost-mobile"
	RepoMattermostDesktop   = "mattermost/desktop"
	RepoPluginStarter       = "mattermost/mattermost-plugin-starter-template"
	RepoMattermostPluginAI  = "mattermost/mattermost-plugin-ai"
	RepoMattermostAPIClient = "mattermost/mattermost-api-reference"
)

// Component names
const (
	ComponentServer  = "server"
	ComponentWebapp  = "webapp"
	ComponentMobile  = "mobile"
	ComponentDesktop = "desktop"
	ComponentPlugins = "plugins"
)

// Programming languages
const (
	LanguageGo         = "go"
	LanguageReact      = "react"
	LanguageTypeScript = "typescript"
	LanguageJavaScript = "javascript"
	LanguageSwift      = "swift"
	LanguageKotlin     = "kotlin"
)

// Search term patterns for developer queries
const (
	SearchPatternAPIUsage      = "api usage"
	SearchPatternPluginHook    = "plugin hook"
	SearchPatternSlashCommand  = "slash command"
	SearchPatternWebsocket     = "websocket"
	SearchPatternError         = "error"
	SearchPatternDebug         = "debug"
	SearchPatternTroubleshoot  = "troubleshoot"
	SearchPatternArchitecture  = "architecture"
	SearchPatternDesignPattern = "design pattern"
	SearchPatternADR           = "adr"
	SearchPatternPullRequest   = "pull request"
	SearchPatternRelease       = "release"
)

// Content length limits
const (
	CodeExampleMaxLength = 3000
	MaxCodeSearchResults = 10
)

// Report section headers
const (
	HeaderCodeExamples     = "## Code Examples\n\n"
	HeaderImplementation   = "## Implementation Details\n\n"
	HeaderBestPractices    = "## Mattermost Best Practices\n\n"
	HeaderCommonIssues     = "## Common Issues\n\n"
	HeaderRelatedResources = "## Related Resources\n\n"
	HeaderArchitecture     = "## Architecture Overview\n\n"
	HeaderComponentDetails = "## Component Details\n\n"
	HeaderAPIReference     = "## API Reference\n\n"
	HeaderRecentChanges    = "## Recent Changes\n\n"
	HeaderBreakingChanges  = "## Breaking Changes\n\n"
	HeaderMigrationGuide   = "## Migration Guide\n\n"
)

// Report content templates
const (
	TemplateCodePatternTitle     = "# Code Pattern: %s\n\n"
	TemplateDebugIssueTitle      = "# Debug Issue: %s\n\n"
	TemplateArchitectureTitle    = "# Architecture: %s\n\n"
	TemplateAPIExamplesTitle     = "# API Examples: %s\n\n"
	TemplatePRSummaryTitle       = "# Recent Pull Requests: %s\n\n"
	TemplateCodeExample          = "**File:** `%s`\n\n```%s\n%s\n```\n\n"
	TemplateCommonDataSourcesRef = "📚 **Documentation:** [%s](%s)\n\n"
	TemplateGitHubIssueRef       = "🐛 **Related Issue:** [#%d - %s](%s)\n\n"
	TemplatePRRef                = "🔀 **Pull Request:** [#%d - %s](%s)\n\n"
)
