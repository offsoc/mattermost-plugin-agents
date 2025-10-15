// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ProtocolType defines the type of external data source protocol
type ProtocolType string

// DataSourceProtocol defines the interface for different external source protocols
type DataSourceProtocol interface {
	Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error)
	GetType() ProtocolType
	SetAuth(auth AuthConfig)
	ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error)
}

// Doc represents a document from an external source
type Doc struct {
	Title          string   `json:"title"`
	Content        string   `json:"content"` // Pre-chunked for LLM consumption
	URL            string   `json:"url"`
	Section        string   `json:"section"`
	Source         string   `json:"source"` // Which source this came from
	Author         string   `json:"author,omitempty"`
	LastModified   string   `json:"last_modified,omitempty"`
	Labels         []string `json:"labels,omitempty"`
	CreatedDate    string   `json:"created_date,omitempty"`
	AuthorityScore int      `json:"authority_score,omitempty"` // Source authority ranking (0-100)
}

// ProtocolRequest contains the parameters for a protocol fetch operation
type ProtocolRequest struct {
	Source   SourceConfig
	Topic    string
	Sections []string
	Limit    int
}

// AuthConfig defines authentication configuration for external sources
type AuthConfig struct {
	Type string `json:"type"` // "none", "api_key", "token"
	Key  string `json:"key"`  // API key or token value (supports env var fallback)
}

// RateLimitConfig defines rate limiting configuration per source
type RateLimitConfig struct {
	RequestsPerMinute int  `json:"requests_per_minute"` // Configurable rate limit
	BurstSize         int  `json:"burst_size"`          // Allow burst requests
	Enabled           bool `json:"enabled"`             // Enable/disable rate limiting
}

// SourceConfig defines configuration for an individual external source
type SourceConfig struct {
	Name           string            `json:"name"`            // "mattermost_docs", "github_repos"
	Enabled        bool              `json:"enabled"`         // Per-source enable/disable
	Protocol       ProtocolType      `json:"protocol"`        // Which protocol to use
	Endpoints      map[string]string `json:"endpoints"`       // Protocol-specific config
	Auth           AuthConfig        `json:"auth"`            // Authentication if needed
	Sections       []string          `json:"sections"`        // Available sections
	MaxDocsPerCall int               `json:"max_docs"`        // Document limit per call
	RateLimit      RateLimitConfig   `json:"rate_limit"`      // Configurable rate limiting
	ChannelMapping map[string]string `json:"channel_mapping"` // Section -> Channel name mapping (for Mattermost sources)
}

// Config defines the main configuration for external documentation
type Config struct {
	Sources           []SourceConfig `json:"sources"`
	AllowedDomains    []string       `json:"allowed_domains"`    // Security allowlist
	GitHubToken       string         `json:"github_token"`       // Optional GitHub access (supports env var fallback)
	CacheTTL          time.Duration  `json:"cache_ttl"`          // Default 24h
	FallbackDirectory string         `json:"fallback_directory"` // Directory for fallback data files
}

// SyntaxValidationResult contains the result of search syntax validation
type SyntaxValidationResult struct {
	OriginalQuery     string   `json:"original_query"`
	IsValidSyntax     bool     `json:"is_valid_syntax"`
	SyntaxErrors      []string `json:"syntax_errors,omitempty"`
	RecommendedQuery  string   `json:"recommended_query,omitempty"`
	TestResultCount   int      `json:"test_result_count"`
	ActualAPIResponse string   `json:"actual_api_response,omitempty"`
	SupportsFeatures  []string `json:"supports_features,omitempty"` // "AND", "OR", "quotes", "regex", etc.
}

// ParseAtlassianAuth parses Atlassian (Jira/Confluence) authentication credentials
// Supports multiple formats:
// 1. email:token format (e.g., "user@example.com:api-token-123")
// 2. Token only with email from endpoints map (fallback)
// 3. Bearer tokens (ATATT prefix) - returns empty email to signal Bearer auth
// Returns email (or empty for Bearer), token, and error
func ParseAtlassianAuth(authKey string, emailEndpoint string) (email, token string, err error) {
	if authKey == "" {
		return "", "", fmt.Errorf("authentication key is empty")
	}

	// Check for Confluence Bearer token format (ATATT prefix)
	if strings.HasPrefix(authKey, "ATATT") {
		return "", authKey, nil // Empty email signals Bearer auth
	}

	if strings.Contains(authKey, ":") {
		parts := strings.SplitN(authKey, ":", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}

	// Fallback: token only, try to get email from endpoint
	if emailEndpoint != "" {
		return emailEndpoint, authKey, nil
	}

	return "", "", fmt.Errorf("authentication requires email:token format or separate email configuration")
}

// DaysSince calculates the number of days since a given date string
// Supports multiple date formats: RFC3339, ISO8601, common date formats
// Returns -1 if the date cannot be parsed
func DaysSince(dateStr string) int {
	if dateStr == "" {
		return -1
	}

	// Try multiple date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return -1
	}

	duration := time.Since(parsedTime)
	days := int(duration.Hours() / 24)

	return days
}

// FormatRecencyLabel creates a recency label based on days since creation/update
func FormatRecencyLabel(days int) string {
	if days < 0 {
		return ""
	}

	switch {
	case days == 0:
		return "recency:today"
	case days == 1:
		return "recency:yesterday"
	case days <= 7:
		return "recency:this_week"
	case days <= 30:
		return "recency:this_month"
	case days <= 90:
		return "recency:this_quarter"
	case days <= 365:
		return "recency:this_year"
	default:
		return "recency:older"
	}
}
