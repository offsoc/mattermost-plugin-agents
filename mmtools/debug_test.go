// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
)

func TestDebugCommonDataSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping debug test in short mode")
	}

	// Create a very simple config for debugging
	debugConfig := &datasources.Config{
		Sources: []datasources.SourceConfig{
			{
				Name:     datasources.SourceMattermostDocs,
				Enabled:  true,
				Protocol: datasources.HTTPProtocolType,
				Endpoints: map[string]string{
					datasources.EndpointBaseURL: datasources.MattermostDocsURL,
					datasources.EndpointAdmin:   datasources.DocsAdminPath,
				},
				Auth:           datasources.AuthConfig{Type: datasources.AuthTypeNone},
				Sections:       []string{datasources.SectionAdmin},
				MaxDocsPerCall: 1,
				RateLimit: datasources.RateLimitConfig{
					RequestsPerMinute: 10,
					BurstSize:         2,
					Enabled:           true,
				},
			},
		},
		AllowedDomains: []string{"docs.mattermost.com"},
		CacheTTL:       datasources.DefaultCacheTTL,
	}

	client := datasources.NewClient(debugConfig, nil)
	defer client.Close()

	t.Logf("Testing with config: %+v", debugConfig.Sources[0])

	// Try to fetch documents
	docs, err := client.FetchFromSource(context.Background(), datasources.SourceMattermostDocs, "admin", 1)

	t.Logf("Fetch result: err=%v, docs=%v", err, docs)
	if err != nil {
		t.Logf("Error details: %v", err)
		// Don't fail the test, just log for debugging
		return
	}

	if len(docs) > 0 {
		t.Logf("Success! Got %d documents", len(docs))
		t.Logf("First doc: %+v", docs[0])
	} else {
		t.Logf("Got 0 documents - this may indicate an issue with the HTTP protocol implementation")
	}
}

func TestDebugHTTPProtocolDirect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping debug test in short mode")
	}

	// Test the HTTP protocol directly with a real HTTP client
	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpProtocol := datasources.NewHTTPProtocol(httpClient, nil)

	sourceConfig := datasources.SourceConfig{
		Name:     datasources.SourceMattermostDocs,
		Protocol: datasources.HTTPProtocolType,
		Endpoints: map[string]string{
			datasources.EndpointBaseURL: datasources.MattermostDocsURL,
			datasources.EndpointAdmin:   datasources.DocsAdminPath,
		},
		Auth:           datasources.AuthConfig{Type: datasources.AuthTypeNone},
		Sections:       []string{datasources.SectionAdmin},
		MaxDocsPerCall: 1,
	}

	request := datasources.ProtocolRequest{
		Source:   sourceConfig,
		Topic:    "admin",
		Sections: []string{datasources.SectionAdmin},
		Limit:    1,
	}

	t.Logf("Testing HTTP protocol directly with request: %+v", request)

	// Let's manually test the URL construction first
	baseURL := sourceConfig.Endpoints[datasources.EndpointBaseURL]
	adminPath := sourceConfig.Endpoints[datasources.EndpointAdmin]
	fullURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(adminPath, "/")
	t.Logf("Constructed URL: %s", fullURL)

	// Test if we can reach the URL directly
	resp, err := httpClient.Get(fullURL)
	if err != nil {
		t.Logf("Direct URL fetch failed: %v", err)
	} else {
		t.Logf("Direct URL fetch succeeded: status=%d", resp.StatusCode)
		resp.Body.Close()
	}

	// Test some known working URLs
	testURLs := []string{
		"https://docs.mattermost.com/",
		"https://docs.mattermost.com/about/",
		"https://docs.mattermost.com/install/",
		"https://docs.mattermost.com/administration/",
	}

	for _, testURL := range testURLs {
		resp, testErr := httpClient.Get(testURL)
		if testErr != nil {
			t.Logf("Test URL %s failed: %v", testURL, testErr)
		} else {
			t.Logf("Test URL %s status: %d", testURL, resp.StatusCode)
			resp.Body.Close()
		}
	}

	docs, err := httpProtocol.Fetch(context.Background(), request)

	t.Logf("Direct HTTP protocol result: err=%v, docs count=%d", err, len(docs))
	if err != nil {
		t.Logf("HTTP Protocol error: %v", err)
	}

	if len(docs) > 0 {
		t.Logf("Direct HTTP success! First doc: %+v", docs[0])
	}
}
