// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package grounding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCitations_ExactMatch(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
		{Type: CitationJiraTicket, Value: "MM-99999"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"},
			"MM-67890": {Key: "MM-67890"},
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.True(t, validated[0].IsValid, "MM-12345 should be valid")
	assert.False(t, validated[1].IsValid, "MM-99999 should be invalid")
}

func TestValidateCitations_NoFalsePositives(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-123"},
	}

	// Index contains MM-12345 but NOT MM-123
	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"},
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.False(t, validated[0].IsValid, "MM-123 should NOT match MM-12345 (no substring matching)")
	assert.Equal(t, ValidationNotChecked, validated[0].ValidationStatus)
}

func TestValidateCitations_CaseSensitiveKeys(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
		{Type: CitationJiraTicket, Value: "mm-12345"},
	}

	// Index contains exact key "MM-12345"
	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"},
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.True(t, validated[0].IsValid, "Exact case should match")
	// Note: lowercase will NOT match - exact match only
	// The BuildReferenceIndex should normalize keys from metadata
	// For now, this is expected behavior: exact key matching
	assert.False(t, validated[1].IsValid, "Different case should NOT match (exact matching)")
}

func TestValidateCitations_EmptyIndex(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets:     make(map[string]*JiraRef),
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.False(t, validated[0].IsValid, "Should be invalid with empty index")
	assert.Equal(t, ValidationNotChecked, validated[0].ValidationStatus)
}

func TestValidateCitations_MetadataAlwaysValid(t *testing.T) {
	citations := []Citation{
		{Type: CitationMetadata, Value: "metadata"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets:     make(map[string]*JiraRef),
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.True(t, validated[0].IsValid, "Metadata should always be valid")
	assert.Equal(t, ValidationGrounded, validated[0].ValidationStatus)
	assert.Equal(t, "Metadata always grounded", validated[0].ValidationDetails)
}

func TestValidateCitations_GitHub(t *testing.T) {
	citations := []Citation{
		{Type: CitationGitHub, Value: "mattermost/mattermost#12345"},
		{Type: CitationGitHub, Value: "#12345"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: make(map[string]*JiraRef),
		GitHubIssues: map[string]*GitHubRef{
			"mattermost/mattermost#12345": {Owner: "mattermost", Repo: "mattermost", Number: 12345},
			"#12345":                      {Number: 12345},
		},
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.True(t, validated[0].IsValid, "Full GitHub citation should be valid")
	assert.True(t, validated[1].IsValid, "Short GitHub citation should be valid")
}

func TestValidateCitations_ValidationStatus(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
		{Type: CitationURL, Value: "https://example.com"},
		{Type: CitationMetadata, Value: "metadata"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"},
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.Equal(t, ValidationGrounded, validated[0].ValidationStatus, "Jira citation should be grounded")
	assert.Equal(t, ValidationNotChecked, validated[1].ValidationStatus, "URL should be not checked")
	assert.Equal(t, ValidationGrounded, validated[2].ValidationStatus, "Metadata should be grounded")
}

func TestValidateURLAccessibility_ValidURL(t *testing.T) {
	citations := []Citation{
		{
			Type:             CitationURL,
			Value:            "https://www.google.com",
			ValidationStatus: ValidationNotChecked,
		},
	}

	validated := ValidateURLAccessibility(citations)

	assert.Equal(t, ValidationUngroundedValid, validated[0].ValidationStatus, "Accessible URL should be ungrounded-valid")
	assert.True(t, validated[0].HTTPStatusCode >= 200 && validated[0].HTTPStatusCode < 400, "Should have successful status code")
}

func TestValidateURLAccessibility_BrokenURL(t *testing.T) {
	citations := []Citation{
		{
			Type:             CitationURL,
			Value:            "https://this-domain-definitely-does-not-exist-12345.com",
			ValidationStatus: ValidationNotChecked,
		},
	}

	validated := ValidateURLAccessibility(citations)

	assert.Equal(t, ValidationUngroundedBroken, validated[0].ValidationStatus, "Broken URL should be ungrounded-broken")
}

func TestValidateURLAccessibility_SkipGroundedCitations(t *testing.T) {
	citations := []Citation{
		{
			Type:             CitationURL,
			Value:            "https://example.com",
			ValidationStatus: ValidationGrounded,
		},
	}

	validated := ValidateURLAccessibility(citations)

	assert.Equal(t, ValidationGrounded, validated[0].ValidationStatus, "Should skip already grounded citations")
	assert.Equal(t, 0, validated[0].HTTPStatusCode, "Should not make HTTP request for grounded citations")
}

func TestValidateURLAccessibility_SkipNonURLCitations(t *testing.T) {
	citations := []Citation{
		{
			Type:             CitationJiraTicket,
			Value:            "MM-12345",
			ValidationStatus: ValidationNotChecked,
		},
	}

	validated := ValidateURLAccessibility(citations)

	assert.Equal(t, ValidationNotChecked, validated[0].ValidationStatus, "Should skip non-URL citations")
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://docs.mattermost.com/", "docs.mattermost.com"},
		{"http://docs.mattermost.com", "docs.mattermost.com"},
		{"https://docs.mattermost.com/api/", "docs.mattermost.com/api"},
		{"HTTPS://DOCS.MATTERMOST.COM/API", "docs.mattermost.com/api"},
		{"https://example.com", "example.com"},
		{"https://example.com/path", "example.com/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateCitations_URLNormalization(t *testing.T) {
	citations := []Citation{
		{Type: CitationURL, Value: "https://docs.mattermost.com/api"},
		{Type: CitationURL, Value: "HTTP://DOCS.MATTERMOST.COM/API/"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets:     make(map[string]*JiraRef),
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs: map[string]bool{
			"docs.mattermost.com/api": true,
		},
	}

	validated := ValidateCitations(citations, refIndex)

	assert.True(t, validated[0].IsValid, "HTTPS URL should match")
	assert.True(t, validated[1].IsValid, "HTTP uppercase with trailing slash should match")
	assert.Equal(t, ValidationGrounded, validated[0].ValidationStatus)
	assert.Equal(t, "Found in reference index", validated[0].ValidationDetails)
}

func TestValidateCitations_ValidationDetails(t *testing.T) {
	citations := []Citation{
		{Type: CitationJiraTicket, Value: "MM-12345"},
		{Type: CitationJiraTicket, Value: "MM-99999"},
		{Type: CitationMetadata, Value: "metadata"},
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"},
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	validated := ValidateCitations(citations, refIndex)

	assert.Equal(t, "Found in reference index", validated[0].ValidationDetails)
	assert.Equal(t, "Not in tool results, needs API check", validated[1].ValidationDetails)
	assert.Equal(t, "Metadata always grounded", validated[2].ValidationDetails)
}

func TestExtractZendeskTicketID(t *testing.T) {
	tests := []struct {
		citation string
		expected string
	}{
		{"12345", "12345"},
		{"https://support.mattermost.com/hc/requests/12345", "12345"},
		{"https://support.mattermost.com/hc/en-us/requests/67890", "67890"},
	}

	for _, tt := range tests {
		t.Run(tt.citation, func(t *testing.T) {
			result := extractZendeskTicketID(tt.citation)
			if result != tt.expected {
				t.Errorf("extractZendeskTicketID(%q) = %q, want %q", tt.citation, result, tt.expected)
			}
		})
	}
}

func TestParseGitHubRef(t *testing.T) {
	tests := []struct {
		name        string
		ref         string
		wantOwner   string
		wantRepo    string
		wantNumber  int
		shouldError bool
	}{
		{
			name:        "Full form",
			ref:         "mattermost/mattermost#19234",
			wantOwner:   "mattermost",
			wantRepo:    "mattermost",
			wantNumber:  19234,
			shouldError: false,
		},
		{
			name:        "Short form",
			ref:         "#123",
			shouldError: true, // Cannot verify without owner/repo
		},
		{
			name:        "Invalid format",
			ref:         "invalid",
			shouldError: true,
		},
		{
			name:        "Missing number",
			ref:         "owner/repo#",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := parseGitHubRef(tt.ref)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %q, got nil", tt.ref)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %q: %v", tt.ref, err)
				}
				if owner != tt.wantOwner {
					t.Errorf("Owner: got %q, want %q", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("Repo: got %q, want %q", repo, tt.wantRepo)
				}
				if number != tt.wantNumber {
					t.Errorf("Number: got %d, want %d", number, tt.wantNumber)
				}
			}
		})
	}
}

// MockGitHubClient for testing
type MockGitHubClient struct {
	existsFn func(ctx context.Context, owner, repo string, num int) (bool, error)
}

func (m *MockGitHubClient) IssueExists(ctx context.Context, owner, repo string, num int) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, owner, repo, num)
	}
	return false, nil
}

// MockJiraClient for testing
type MockJiraClient struct {
	existsFn func(ctx context.Context, key string) (bool, error)
}

func (m *MockJiraClient) IssueExists(ctx context.Context, key string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, key)
	}
	return false, nil
}

func TestValidateCitationsWithAPI_Fabricated(t *testing.T) {
	citation := Citation{
		Type:  CitationGitHub,
		Value: "mattermost/mattermost#99999",
	}

	refIndex := &ReferenceIndex{
		JiraTickets:     make(map[string]*JiraRef),
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	mockGitHub := &MockGitHubClient{
		existsFn: func(ctx context.Context, owner, repo string, num int) (bool, error) {
			return false, nil // Simulates 404
		},
	}

	apiClients := &APIClients{
		GitHub: mockGitHub,
	}

	ctx := context.Background()
	results := ValidateCitationsWithAPI(ctx, []Citation{citation}, refIndex, apiClients)

	assert.Equal(t, ValidationFabricated, results[0].ValidationStatus)
	assert.False(t, results[0].IsValid)
	assert.True(t, results[0].VerifiedViaAPI)
	assert.Equal(t, "Does not exist in external system", results[0].ValidationDetails)
}

func TestValidateCitationsWithAPI_UngroundedValid(t *testing.T) {
	citation := Citation{
		Type:  CitationJiraTicket,
		Value: "MM-12345",
	}

	refIndex := &ReferenceIndex{
		JiraTickets:     make(map[string]*JiraRef), // Not in index
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	mockJira := &MockJiraClient{
		existsFn: func(ctx context.Context, key string) (bool, error) {
			return true, nil // Exists via API
		},
	}

	apiClients := &APIClients{
		Jira: mockJira,
	}

	ctx := context.Background()
	results := ValidateCitationsWithAPI(ctx, []Citation{citation}, refIndex, apiClients)

	assert.Equal(t, ValidationUngroundedValid, results[0].ValidationStatus)
	assert.True(t, results[0].IsValid)
	assert.True(t, results[0].VerifiedViaAPI)
	assert.Equal(t, "Verified via API", results[0].ValidationDetails)
}

func TestValidateCitationsWithAPI_SkipsGroundedCitations(t *testing.T) {
	citation := Citation{
		Type:  CitationJiraTicket,
		Value: "MM-12345",
	}

	refIndex := &ReferenceIndex{
		JiraTickets: map[string]*JiraRef{
			"MM-12345": {Key: "MM-12345"}, // In index
		},
		GitHubIssues:    make(map[string]*GitHubRef),
		ConfluencePages: make(map[string]*ConfluenceRef),
		ZendeskTickets:  make(map[string]*ZendeskRef),
		URLs:            make(map[string]bool),
	}

	mockJira := &MockJiraClient{
		existsFn: func(ctx context.Context, key string) (bool, error) {
			t.Error("API should not be called for grounded citations")
			return true, nil
		},
	}

	apiClients := &APIClients{
		Jira: mockJira,
	}

	ctx := context.Background()
	results := ValidateCitationsWithAPI(ctx, []Citation{citation}, refIndex, apiClients)

	assert.Equal(t, ValidationGrounded, results[0].ValidationStatus)
	assert.True(t, results[0].IsValid)
	assert.False(t, results[0].VerifiedViaAPI) // Should not call API
}
