// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dev

// Severity represents technical severity level (for DevBot)
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityTrivial  Severity = "trivial"
)

// IssueType represents the type of work item (for DevBot)
type IssueType string

const (
	IssueTypeBug           IssueType = "bug"
	IssueTypeFeature       IssueType = "feature"
	IssueTypeRefactor      IssueType = "refactor"
	IssueTypeTechDebt      IssueType = "tech_debt"
	IssueTypeDocumentation IssueType = "documentation"
	IssueTypeTest          IssueType = "test"
	IssueTypeInfra         IssueType = "infrastructure"
)

// Component represents code component/path (for DevBot)
type Component string

const (
	ComponentServerAPI        Component = "server/api"
	ComponentServerChannels   Component = "server/channels"
	ComponentServerPlugins    Component = "server/plugins"
	ComponentServerAuth       Component = "server/auth"
	ComponentServerWebsocket  Component = "server/websocket"
	ComponentServerDatabase   Component = "server/database"
	ComponentWebappComponents Component = "webapp/components"
	ComponentWebappChannels   Component = "webapp/channels"
	ComponentWebappPlugins    Component = "webapp/plugins"
	ComponentMobileIOS        Component = "mobile/ios"
	ComponentMobileAndroid    Component = "mobile/android"
	ComponentMobileNetworking Component = "mobile/networking"
	ComponentDesktop          Component = "desktop"
	ComponentPlaybooks        Component = "playbooks"
	ComponentBoards           Component = "boards"
	ComponentCalls            Component = "calls"
)

// Language represents programming language (for DevBot)
type Language string

const (
	LanguageGo         Language = "go"
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguageSwift      Language = "swift"
	LanguageKotlin     Language = "kotlin"
	LanguageJava       Language = "java"
	LanguageObjectiveC Language = "objective-c"
	LanguageSQL        Language = "sql"
	LanguagePython     Language = "python"
)

// Complexity represents estimated work complexity (for DevBot)
type Complexity string

const (
	ComplexitySimple  Complexity = "simple"
	ComplexityMedium  Complexity = "medium"
	ComplexityComplex Complexity = "complex"
)

// FormatSeverityLabel creates a label string for severity
func FormatSeverityLabel(severity Severity) string {
	return "severity:" + string(severity)
}

// FormatIssueTypeLabel creates a label string for issue type
func FormatIssueTypeLabel(issueType IssueType) string {
	return "type:" + string(issueType)
}

// FormatComponentLabel creates a label string for component
func FormatComponentLabel(component Component) string {
	return "component:" + string(component)
}

// FormatLanguageLabel creates a label string for language
func FormatLanguageLabel(language Language) string {
	return "lang:" + string(language)
}

// FormatComplexityLabel creates a label string for complexity
func FormatComplexityLabel(complexity Complexity) string {
	return "complexity:" + string(complexity)
}
