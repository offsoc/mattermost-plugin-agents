// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"
)

func TestFormatJiraAuth(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		token    string
		expected string
	}{
		{
			name:     "both email and token provided",
			email:    "user@example.com",
			token:    "api-token-123",
			expected: "user@example.com:api-token-123",
		},
		{
			name:     "token only, email empty",
			email:    "",
			token:    "api-token-123",
			expected: "user@example.com:api-token-123",
		},
		{
			name:     "token already in email:token format",
			email:    "user@example.com",
			token:    "another@example.com:api-token-456",
			expected: "another@example.com:api-token-456",
		},
		{
			name:     "empty token",
			email:    "user@example.com",
			token:    "",
			expected: "",
		},
		{
			name:     "both empty",
			email:    "",
			token:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJiraAuth(tt.email, tt.token)
			if result != tt.expected {
				t.Errorf("FormatJiraAuth(%q, %q) = %q, want %q", tt.email, tt.token, result, tt.expected)
			}
		})
	}
}

func TestParseJiraAuth(t *testing.T) {
	tests := []struct {
		name          string
		authKey       string
		expectedEmail string
		expectedToken string
	}{
		{
			name:          "valid email:token format",
			authKey:       "user@example.com:api-token-123",
			expectedEmail: "user@example.com",
			expectedToken: "api-token-123",
		},
		{
			name:          "token only",
			authKey:       "api-token-123",
			expectedEmail: "user@example.com",
			expectedToken: "api-token-123",
		},
		{
			name:          "empty authKey",
			authKey:       "",
			expectedEmail: "",
			expectedToken: "",
		},
		{
			name:          "multiple colons",
			authKey:       "user@example.com:token:with:colons",
			expectedEmail: "user@example.com",
			expectedToken: "token:with:colons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, token := ParseJiraAuth(tt.authKey)
			if email != tt.expectedEmail {
				t.Errorf("ParseJiraAuth(%q) email = %q, want %q", tt.authKey, email, tt.expectedEmail)
			}
			if token != tt.expectedToken {
				t.Errorf("ParseJiraAuth(%q) token = %q, want %q", tt.authKey, token, tt.expectedToken)
			}
		})
	}
}

func TestJiraAuthRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		email string
		token string
	}{
		{
			name:  "standard case",
			email: "user@example.com",
			token: "api-token-123",
		},
		{
			name:  "email with plus",
			email: "user+test@example.com",
			token: "api-token-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatJiraAuth(tt.email, tt.token)
			parsedEmail, parsedToken := ParseJiraAuth(formatted)

			if parsedEmail != tt.email {
				t.Errorf("Roundtrip failed: email = %q, want %q", parsedEmail, tt.email)
			}
			if parsedToken != tt.token {
				t.Errorf("Roundtrip failed: token = %q, want %q", parsedToken, tt.token)
			}
		})
	}
}
