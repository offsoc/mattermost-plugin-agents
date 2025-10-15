// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPProtocolContentExtraction(t *testing.T) {
	httpClient := &http.Client{}
	protocol := NewHTTPProtocol(httpClient, nil)
	ctx := context.Background()

	testCases := []struct {
		name        string
		url         string
		topic       string
		source      string
		expectText  string
		expectTitle bool
	}{
		{
			name:        "Mattermost Docs - Mobile deployment",
			url:         "https://docs.mattermost.com/deployment-guide/mobile/mobile-app-deployment.html",
			topic:       "mobile app deployment",
			source:      "mattermost_docs",
			expectText:  "mobile app",
			expectTitle: true,
		},
		{
			name:        "Mattermost Handbook - Company info",
			url:         "https://handbook.mattermost.com/",
			topic:       "company handbook",
			source:      "mattermost_handbook",
			expectText:  "mattermost",
			expectTitle: true,
		},
		{
			name:        "Mattermost Blog - Recent posts",
			url:         "https://mattermost.com/blog/",
			topic:       "blog posts",
			source:      "mattermost_blog",
			expectText:  "mattermost",
			expectTitle: true,
		},
		{
			name:        "Mattermost Newsroom - Press releases",
			url:         "https://mattermost.com/newsroom/",
			topic:       "newsroom",
			source:      "mattermost_newsroom",
			expectText:  "mattermost",
			expectTitle: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := protocol.fetchSingleDoc(ctx, tc.url, tc.topic, tc.source)

			// Some URLs might be protected or have content quality issues
			// Skip the test if we can't fetch the document, but log it
			if doc == nil {
				t.Skipf("Could not fetch document from %s - may be protected or content quality filtered", tc.url)
				return
			}

			assert.NotEmpty(t, doc.Content, "Document should have content")
			assert.Contains(t, doc.URL, tc.url, "URL should match expected")
			assert.Equal(t, tc.source, doc.Source, "Source should match expected")

			if tc.expectTitle {
				assert.NotEmpty(t, doc.Title, "Document should have a title")
			}

			// Content should contain meaningful information (case-insensitive)
			assert.Contains(t, strings.ToLower(doc.Content), strings.ToLower(tc.expectText), "Content should contain expected text")
			assert.Greater(t, len(doc.Content), 100, "Content should be substantial")

			// Verify navigation text is filtered out
			assert.NotContains(t, doc.Content, "Toggle", "Navigation text should be filtered")
			assert.NotContains(t, doc.Content, "Edit this page", "Navigation text should be filtered")
		})
	}
}

func TestHTTPProtocolRedirectHandling(t *testing.T) {
	httpClient := &http.Client{}
	protocol := NewHTTPProtocol(httpClient, nil)

	// Test old URL that redirects
	ctx := context.Background()
	doc := protocol.fetchSingleDoc(ctx, "https://docs.mattermost.com/deploy/mobile-overview.html", "mobile deployment", "mattermost_docs")

	require.NotNil(t, doc, "Document should be fetched successfully after redirect")
	assert.NotEmpty(t, doc.Content, "Document should have content after following redirect")

	// The redirect should follow to the correct target - check that we got actual content
	assert.Contains(t, doc.Content, "mobile", "Content should contain mobile-related information")
	assert.Greater(t, len(doc.Content), 50, "Content should be substantial after redirect")
}

func TestHTTPDatasourceConfiguration(t *testing.T) {
	config := CreateDefaultConfig()

	// Test that all HTTP-based datasources are properly configured
	expectedHTTPSources := map[string]string{
		"mattermost_docs":     "https://docs.mattermost.com",
		"mattermost_handbook": "https://handbook.mattermost.com",
		"mattermost_blog":     "https://mattermost.com/blog",
		"mattermost_newsroom": "https://mattermost.com/newsroom",
		"plugin_marketplace":  "https://integrations.mattermost.com",
	}

	for sourceName, expectedURL := range expectedHTTPSources {
		t.Run(sourceName, func(t *testing.T) {
			var found bool
			var sourceConfig SourceConfig

			for _, source := range config.Sources {
				if source.Name == sourceName {
					found = true
					sourceConfig = source
					break
				}
			}

			require.True(t, found, "Source %s should be configured", sourceName)
			assert.Equal(t, HTTPProtocolType, sourceConfig.Protocol, "Source %s should use HTTP protocol", sourceName)
			assert.Equal(t, expectedURL, sourceConfig.Endpoints["base_url"], "Source %s should have correct base URL", sourceName)
			assert.False(t, sourceConfig.Enabled, "Source %s should be disabled by default", sourceName)
			assert.True(t, sourceConfig.RateLimit.Enabled, "Source %s should have rate limiting enabled", sourceName)
		})
	}
}

func TestContentQualityFiltering(t *testing.T) {
	protocol := NewHTTPProtocol(nil, nil)

	tests := []struct {
		name     string
		title    string
		content  string
		expected bool
	}{
		{"Empty content", "Title", "", false},
		{"Short content", "Title", "ab", false},
		{"Navigation heavy content", "Navigation", "Toggle sidebar edit this page home menu search toggle previous next home menu search", false},
		{"Portal page with many links", "Home", "Welcome <a href='/mobile'>Mobile</a> <a href='/web'>Web</a> <a href='/api'>API</a> <a href='/server'>Server</a> <a href='/desktop'>Desktop</a> <a href='/admin'>Admin</a> Quick links", false},
		{"Valid content", "Mobile Guide", "The mobile app deployment process involves configuring push notifications, setting up EMM integration, and distributing the applications to end users. This comprehensive guide covers all aspects of mobile deployment.", true},
		{"Valid paragraph", "Features", "Mattermost provides native mobile applications for iOS and Android platforms. These applications support real-time messaging, file sharing, and integrated workflows.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.htmlProcessor.IsContentQualityAcceptable(tt.title, tt.content)
			assert.Equal(t, tt.expected, result, "Content quality assessment should match expected result")
		})
	}
}

func TestHTTPProtocol_ComplexBooleanQueries(t *testing.T) {
	source := SourceConfig{
		Name:     SourceMattermostDocs,
		Protocol: HTTPProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL:  MattermostDocsURL,
			SectionAdmin:     "/guides/administration.html",
			SectionMobile:    "/guides/mobile.html",
			SectionAPI:       "/guides/integration.html",
			SectionGeneral:   "/deployment-guide/deployment-guide-index.html",
			SectionChannels:  "/product-overview/faq-enterprise.html",
			SectionDeveloper: "/use-case-guide/integrated-security-operations.html",
		},
		Sections: []string{SectionAdmin, SectionMobile, SectionAPI, SectionGeneral, SectionChannels, SectionDeveloper},
		Auth:     AuthConfig{Type: AuthTypeNone},
	}

	setupFunc := func() error {
		t.Log("Testing HTTP protocol with Mattermost docs - complex boolean queries")
		return nil
	}

	protocol := NewHTTPProtocol(&http.Client{}, nil)
	VerifyProtocolMattermostDocsBooleanQuery(t, protocol, source, setupFunc)
}
