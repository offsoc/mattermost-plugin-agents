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

func TestConfluenceProtocol_ComplexBooleanQueries(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tokenEnvVar := "MM_AI_CONFLUENCE_DOCS_TOKEN"
	token := os.Getenv(tokenEnvVar)

	source := SourceConfig{
		Name:     SourceConfluenceDocs,
		Protocol: ConfluenceProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: ConfluenceURL,
			EndpointSpaces:  ConfluenceSpaces,
		},
		Auth: AuthConfig{Type: AuthTypeAPIKey, Key: token},
	}

	setupFunc := func() error {
		if source.Auth.Key == "" {
			t.Logf("Skipping Confluence complex query test: %s environment variable not set", tokenEnvVar)
			t.Skip(tokenEnvVar + " not configured")
		}
		protocol.SetAuth(source.Auth)
		return nil
	}

	VerifyProtocolGenericBooleanQuery(t, protocol, source, setupFunc)
}
