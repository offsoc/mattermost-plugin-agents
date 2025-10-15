// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"net/http"
	"os"
	"testing"
)

// TestJiraProtocol_ComplexBooleanQueries tests Jira protocol with complex boolean search queries
func TestJiraProtocol_ComplexBooleanQueries(t *testing.T) {
	protocol := NewJiraProtocol(&http.Client{}, nil)

	// Get token from environment variable (Jira has fallback logic, see config.go:365)
	token := os.Getenv("MM_AI_JIRA_TOKEN")
	if token == "" {
		token = os.Getenv("MM_AI_JIRA_DOCS_TOKEN")
	}

	// Format token as email:token if not already in that format
	email := os.Getenv("MM_AI_JIRA_EMAIL")
	formattedToken := FormatJiraAuth(email, token)

	source := SourceConfig{
		Name:     SourceJiraDocs,
		Protocol: JiraProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: JiraURL,
		},
		Auth: AuthConfig{Type: AuthTypeAPIKey, Key: formattedToken},
	}

	setupFunc := func() error {
		if source.Auth.Key == "" {
			t.Log("Skipping Jira complex query test: MM_AI_JIRA_TOKEN or MM_AI_JIRA_DOCS_TOKEN environment variable not set")
			t.Skip("Jira API token not configured")
		}
		protocol.SetAuth(source.Auth)
		return nil
	}

	VerifyProtocolJiraBooleanQuery(t, protocol, source, setupFunc)
}
