// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

// Entity type constants
const (
	EntityTypeJira       = "jira"
	EntityTypeGitHub     = "github"
	EntityTypeConfluence = "confluence"
	EntityTypeZendesk    = "zendesk"
)

// EntityMetadata contains all extracted metadata for a document
type EntityMetadata struct {
	// Entity identification (which entity does this metadata describe?)
	EntityType string // Use EntityType* constants
	EntityID   string // "MM-12345", "mattermost/mattermost#19234", etc.

	// Role-specific metadata (PM, Dev, etc.)
	RoleMetadata RoleMetadata

	// Cross-references found within this entity's content
	GitHubIssues    []GitHubReference
	GitHubPRs       []GitHubReference
	JiraTickets     []JiraReference
	ConfluencePages []ConfluenceReference
	ModifiedFiles   []FileReference
	Commits         []CommitReference
	ZendeskTickets  []ZendeskReference
	MattermostLinks []MattermostReference
}

// GitHubReferenceType represents the type of GitHub reference
type GitHubReferenceType int

const (
	GHIssue GitHubReferenceType = iota
	GHPullRequest
)

// GitHubReference represents a GitHub issue or PR
type GitHubReference struct {
	Type   GitHubReferenceType
	Number int
	Owner  string // Optional: extracted from full URLs
	Repo   string // Optional: extracted from full URLs
	URL    string // Full URL if available
}

// JiraReference represents a Jira ticket
type JiraReference struct {
	Key     string // MM-12345
	Project string // MM
	Number  int    // 12345
	URL     string // Full URL if available
}

// FileReference represents a code file or path
type FileReference struct {
	Path      string // server/api/auth.go
	StartLine int    // Optional: 0 if not specified
	EndLine   int    // Optional: 0 if not specified
	Component string // Inferred: server/api
	Language  string // Inferred: go
}

// CommitReference represents a Git commit
type CommitReference struct {
	SHA      string // Full or short SHA
	Owner    string // Optional: from full URLs
	Repo     string // Optional: from full URLs
	URL      string // Full URL if available
	ShortSHA string // First 7 chars
}

// ConfluenceReference represents a Confluence page
type ConfluenceReference struct {
	Space  string // TEAM
	PageID string // 123456
	Title  string // Optional: if extractable
	URL    string // Full URL
}

// ZendeskReference represents a Zendesk ticket
type ZendeskReference struct {
	TicketID string
	URL      string
}

// MattermostReference represents a Mattermost permalink
type MattermostReference struct {
	TeamName  string
	ChannelID string
	PostID    string
	URL       string
}
