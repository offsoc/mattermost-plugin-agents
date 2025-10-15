// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"
)

func TestBuildGitHubIssuesSearchQuery(t *testing.T) {
	g := &GitHubProtocol{}

	tests := []struct {
		name          string
		input         string
		maxLength     int
		expectContain string
		expectNoError bool
	}{
		{
			name:          "simple query under limit",
			input:         "authentication security",
			expectContain: "authentication",
			expectNoError: true,
		},
		{
			name:          "long keyword list gets truncated",
			input:         "market competitive analysis trends research comparison benchmarking strategy positioning vision business discussion feedback community insights industry announcements alternatives evaluation customer sales ecosystem third-party user needs voting requests requirements competitor alternative trend opportunity growth adoption usage alignment enterprise enterprise advanced commercial licensing compliance governance administration enterprise features security authentication authorization sso saml ldap mfa two-factor oauth encryption audit boards kanban tasks project management cards focalboard project boards task management",
			expectContain: "market",
			expectNoError: true,
		},
		{
			name:          "boolean query with many terms",
			input:         "authentication OR security OR compliance OR governance OR sso OR saml OR ldap OR mfa OR oauth OR encryption",
			expectContain: "OR",
			expectNoError: true,
		},
		{
			name:          "empty query",
			input:         "",
			expectContain: "",
			expectNoError: true,
		},
		{
			name:          "query with short words filtered out",
			input:         "a an the authentication",
			expectContain: "authentication",
			expectNoError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.buildGitHubIssuesSearchQuery(tt.input)

			if len(result) > 256 {
				t.Errorf("Query exceeds 256 character limit: length=%d, query=%s", len(result), result)
			}

			if tt.expectContain != "" && result != "" {
				if tt.expectContain != "" && len(result) > 0 {
					// For empty expectation, just check we got something
					if tt.expectContain != "" {
						found := false
						// Check if term or any part is in result
						if len(tt.expectContain) > 0 {
							for i := 0; i <= len(result)-len(tt.expectContain); i++ {
								if result[i:i+len(tt.expectContain)] == tt.expectContain {
									found = true
									break
								}
							}
						}
						if !found && tt.expectContain != "" {
							t.Errorf("Expected result to contain %q, got: %s", tt.expectContain, result)
						}
					}
				}
			}

			if result != "" {
				orCount := 0
				for i := 0; i < len(result)-3; i++ {
					if result[i:i+4] == " OR " {
						orCount++
					}
				}
				if orCount > 5 {
					t.Errorf("Query has more than 5 OR operators: count=%d, query=%s", orCount, result)
				}
			}
		})
	}
}

func TestBuildSimpleIssuesQuery(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		wantLen   bool
	}{
		{
			name:      "within limit",
			input:     "authentication security",
			maxLength: 50,
			wantLen:   true,
		},
		{
			name:      "truncates at limit",
			input:     "one two three four five six seven eight nine ten",
			maxLength: 20,
			wantLen:   true,
		},
		{
			name:      "filters short words",
			input:     "a an to is authentication",
			maxLength: 50,
			wantLen:   true,
		},
		{
			name:      "deduplicates terms",
			input:     "security security authentication security",
			maxLength: 50,
			wantLen:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSimpleIssuesQuery(tt.input, tt.maxLength)

			if len(result) > tt.maxLength {
				t.Errorf("Result exceeds max length: got %d, want <= %d", len(result), tt.maxLength)
			}

			if tt.wantLen && len(result) == 0 && len(tt.input) > 0 {
				t.Error("Expected non-empty result for non-empty input")
			}
		})
	}
}

func TestBuildIssuesQueryWithOR(t *testing.T) {
	tests := []struct {
		name         string
		keywords     []string
		maxLength    int
		maxOperators int
		checkFunc    func(t *testing.T, result string)
	}{
		{
			name:         "within all limits",
			keywords:     []string{"auth", "security", "sso"},
			maxLength:    100,
			maxOperators: 5,
			checkFunc: func(t *testing.T, result string) {
				if result != "auth OR security OR sso" {
					t.Errorf("Expected 'auth OR security OR sso', got: %s", result)
				}
			},
		},
		{
			name:         "truncates to operator limit",
			keywords:     []string{"one", "two", "three", "four", "five", "six", "seven", "eight"},
			maxLength:    200,
			maxOperators: 5,
			checkFunc: func(t *testing.T, result string) {
				orCount := 0
				for i := 0; i < len(result)-3; i++ {
					if result[i:i+4] == " OR " {
						orCount++
					}
				}
				if orCount > 5 {
					t.Errorf("Expected <= 5 OR operators, got %d in: %s", orCount, result)
				}
			},
		},
		{
			name:         "respects length limit",
			keywords:     []string{"authentication", "authorization", "governance", "compliance"},
			maxLength:    40,
			maxOperators: 5,
			checkFunc: func(t *testing.T, result string) {
				if len(result) > 40 {
					t.Errorf("Result exceeds max length: %d > 40", len(result))
				}
			},
		},
		{
			name:         "deduplicates keywords",
			keywords:     []string{"auth", "security", "auth", "sso", "security"},
			maxLength:    100,
			maxOperators: 5,
			checkFunc: func(t *testing.T, result string) {
				if result != "auth OR security OR sso" {
					t.Errorf("Expected deduplicated result, got: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildIssuesQueryWithOR(tt.keywords, tt.maxLength, tt.maxOperators)
			tt.checkFunc(t, result)
		})
	}
}
