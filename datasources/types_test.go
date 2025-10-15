// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAtlassianAuth(t *testing.T) {
	testCases := []struct {
		name          string
		authKey       string
		emailEndpoint string
		expectEmail   string
		expectToken   string
		expectError   bool
	}{
		{
			name:        "email:token format",
			authKey:     "user@example.com:api-token-123",
			expectEmail: "user@example.com",
			expectToken: "api-token-123",
			expectError: false,
		},
		{
			name:          "token only with email endpoint",
			authKey:       "api-token-456",
			emailEndpoint: "admin@example.com",
			expectEmail:   "admin@example.com",
			expectToken:   "api-token-456",
			expectError:   false,
		},
		{
			name:        "Bearer token (ATATT prefix)",
			authKey:     "ATATT3xFfGF0yGhHJaScI4c_example_token",
			expectEmail: "", // Empty email signals Bearer auth
			expectToken: "ATATT3xFfGF0yGhHJaScI4c_example_token",
			expectError: false,
		},
		{
			name:        "token only without email endpoint - error",
			authKey:     "api-token-789",
			expectError: true,
		},
		{
			name:        "empty auth key - error",
			authKey:     "",
			expectError: true,
		},
		{
			name:        "malformed email:token (missing token) - error",
			authKey:     "user@example.com:",
			expectError: true,
		},
		{
			name:        "malformed email:token (missing email) - error",
			authKey:     ":api-token-123",
			expectError: true,
		},
		{
			name:        "email with plus sign",
			authKey:     "user+test@example.com:api-token-abc",
			expectEmail: "user+test@example.com",
			expectToken: "api-token-abc",
			expectError: false,
		},
		{
			name:          "token with colon-like chars using email endpoint",
			authKey:       "token::with::colons",
			emailEndpoint: "test@example.com",
			expectEmail:   "token",
			expectToken:   ":with::colons",
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			email, token, err := ParseAtlassianAuth(tc.authKey, tc.emailEndpoint)

			if tc.expectError {
				require.Error(t, err, "Expected error but got none")
				return
			}

			require.NoError(t, err, "Unexpected error: %v", err)
			assert.Equal(t, tc.expectEmail, email, "Email mismatch")
			assert.Equal(t, tc.expectToken, token, "Token mismatch")
		})
	}
}
