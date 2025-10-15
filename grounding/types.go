// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

// CitationType represents different types of citations found in responses
type CitationType string

const (
	CitationJiraTicket   CitationType = "jira_ticket"
	CitationGitHub       CitationType = "github"
	CitationURL          CitationType = "url"
	CitationProductBoard CitationType = "productboard"
	CitationZendesk      CitationType = "zendesk"
	CitationUserVoice    CitationType = "uservoice"
	CitationMetadata     CitationType = "metadata"
)

// ValidationStatus represents how a citation was validated
type ValidationStatus string

const (
	ValidationGrounded         ValidationStatus = "grounded"          // Citation found in tool results
	ValidationUngroundedValid  ValidationStatus = "ungrounded_valid"  // Not in tool results but verified via API
	ValidationUngroundedBroken ValidationStatus = "ungrounded_broken" // Not in tool results and URL is broken
	ValidationFabricated       ValidationStatus = "fabricated"        // Does not exist in external system (404)
	ValidationAPIError         ValidationStatus = "api_error"         // API verification failed
	ValidationNotChecked       ValidationStatus = "not_checked"       // Validation not performed
)

// MetadataClaim represents a metadata claim about an entity
type MetadataClaim struct {
	Field        string // Field name (e.g., "priority", "segments", "severity")
	ClaimedValue string // What LLM claimed (e.g., "high", "enterprise")
	ActualValue  string // What the reference index has (empty if not found)
	IsAccurate   bool   // Does claimed match actual?
}

// Citation represents a single extracted reference from a response
type Citation struct {
	Type             CitationType
	Value            string
	LineNumber       int
	Context          string
	IsValid          bool
	ValidationStatus ValidationStatus
	HTTPStatusCode   int

	// ValidationDetails provides human-readable info about validation result
	ValidationDetails string
	// VerifiedViaAPI indicates whether external API was called for verification
	VerifiedViaAPI bool
	// MetadataClaims contains extracted metadata claims for this citation
	MetadataClaims []MetadataClaim
}

// MetadataUsage tracks usage of structured metadata fields (role-agnostic)
type MetadataUsage struct {
	// FieldCounts maps field names to their usage count
	// Example: {"priority": 5, "segments": 3, "severity": 2}
	FieldCounts map[string]int

	// TotalFields is the number of unique fields used
	TotalFields int
}

// Result contains multi-dimensional grounding scores
type Result struct {
	TotalCitations       int
	CitationsByType      map[CitationType]int
	CitationDensity      float64
	MetadataUsage        MetadataUsage
	ValidCitations       int
	InvalidCitations     int
	ValidCitationRate    float64
	GroundedCitations    int
	UngroundedValidURLs  int
	UngroundedBrokenURLs int
	FabricatedCitations  int
	APIErrors            int
	FabricationRate      float64
	TotalClaims          int
	AccurateClaims       int
	InaccurateClaims     int
	ClaimAccuracyRate    float64
	OverallScore         float64
	Pass                 bool
	Reasoning            string
	Citations            []Citation
}

// Thresholds defines pass/fail criteria
type Thresholds struct {
	MinCitationRate   float64 // Minimum percentage of valid citations (0.0-1.0)
	MinMetadataFields int     // Minimum metadata fields to reference
	CitationWeight    float64 // Weight for citation score in overall score (0.0-1.0)
	MetadataWeight    float64 // Weight for metadata score in overall score (0.0-1.0)
}

// ReferenceIndex contains exact-match references from tool results
// Eliminates false positives from substring matching
type ReferenceIndex struct {
	GitHubIssues    map[string]*GitHubRef     // "mattermost/mattermost#19234" or "#19234"
	JiraTickets     map[string]*JiraRef       // "MM-12345"
	ConfluencePages map[string]*ConfluenceRef // URL as key
	ZendeskTickets  map[string]*ZendeskRef    // "12345"
	URLs            map[string]bool           // Normalized URLs
}

// GitHubRef - Reference from tool results with metadata
type GitHubRef struct {
	Owner    string // May be empty if not in metadata
	Repo     string // May be empty if not in metadata
	Number   int
	Sources  []string     // Which tool results mentioned this (for debugging)
	Metadata RoleMetadata // Role-specific metadata (PM, Dev, etc.)
}

// JiraRef - Reference from tool results with metadata
type JiraRef struct {
	Key      string // "MM-12345"
	Project  string // "MM"
	Number   int    // 12345
	Sources  []string
	Metadata RoleMetadata // Role-specific metadata (PM, Dev, etc.)
}

// ConfluenceRef - Minimal reference from tool results
type ConfluenceRef struct {
	Space   string
	PageID  string
	URL     string
	Sources []string
}

// ZendeskRef - Minimal reference from tool results
type ZendeskRef struct {
	TicketID string
	URL      string
	Sources  []string
}
