// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"os"
	"testing"
)

// TestGitHubProtocol_ComplexBooleanQueries tests GitHub protocol with complex boolean search queries
func TestGitHubProtocol_ComplexBooleanQueries(t *testing.T) {
	// Get token from environment variable (GitHub uses special constant, see config.go:244)
	envVarName := "MM_AI_GITHUB_TOKEN"
	token := os.Getenv(envVarName)

	protocol := NewGitHubProtocol(token, nil)

	source := SourceConfig{
		Name:     SourceGitHubRepos,
		Protocol: GitHubAPIProtocolType,
		Endpoints: map[string]string{
			EndpointOwner: GitHubOwnerMattermost,
			EndpointRepos: "mattermost-server,mattermost-mobile",
		},
		Sections: []string{"issues"},
		Auth:     AuthConfig{Type: AuthTypeToken, Key: token},
	}

	setupFunc := func() error {
		// GitHub API requires token for authenticated requests
		if source.Auth.Key == "" {
			t.Logf("Skipping GitHub complex query test: %s environment variable not set", envVarName)
			t.Skip(envVarName + " not configured")
		}
		return nil
	}

	VerifyProtocolGenericBooleanQuery(t, protocol, source, setupFunc)
}
