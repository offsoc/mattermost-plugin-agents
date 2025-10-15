// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package datasources

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// TestConfluence_RawAPIResponse tests what Confluence API actually returns before filtering
func TestConfluence_RawAPIResponse(t *testing.T) {
	token := os.Getenv("MM_AI_CONFLUENCE_DOCS_TOKEN")
	if token == "" {
		t.Skip("MM_AI_CONFLUENCE_DOCS_TOKEN not set")
	}

	protocol := NewConfluenceProtocol(&http.Client{}, nil)
	protocol.SetAuth(AuthConfig{Type: AuthTypeAPIKey, Key: token})

	// Temporarily disable universal scorer by setting it to nil
	protocol.universalScorer = nil

	source := SourceConfig{
		Name:     SourceConfluenceDocs,
		Protocol: ConfluenceProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: ConfluenceURL,
			EndpointSpaces:  "CLOUD",
		},
	}

	fmt.Printf("\n=== Configuration ===\n")
	fmt.Printf("Confluence URL: %s\n", ConfluenceURL)
	fmt.Printf("Space: CLOUD\n")
	fmt.Printf("Token length: %d chars\n", len(token))

	// Build CQL manually to show what's being queried
	cql := "space = CLOUD AND type = page ORDER BY lastModified DESC"
	fmt.Printf("CQL (empty topic): %s\n", cql)
	fmt.Printf("Full URL: %s/rest/api/content/search?cql=%s&limit=5\n", ConfluenceURL, cql)

	// Test 1: Empty topic (should return recent docs)
	request1 := ProtocolRequest{
		Source: source,
		Topic:  "",
		Limit:  5,
	}

	docs1, err := protocol.Fetch(context.Background(), request1)
	if err != nil {
		t.Fatalf("Error with empty topic: %v", err)
	}

	fmt.Printf("\n=== Test 1: Empty topic ===\n")
	fmt.Printf("Returned %d documents (without filtering):\n", len(docs1))
	for i, doc := range docs1 {
		fmt.Printf("%d. %s (content len=%d)\n", i+1, doc.Title, len(doc.Content))
	}

	if len(docs1) == 0 {
		t.Log("WARNING: Confluence returns 0 docs even with empty topic - likely auth or config issue")
	}

	// Test 2: "channels" topic
	request2 := ProtocolRequest{
		Source: source,
		Topic:  "channels",
		Limit:  5,
	}

	docs2, err := protocol.Fetch(context.Background(), request2)
	if err != nil {
		t.Fatalf("Error with 'channels' topic: %v", err)
	}

	fmt.Printf("\n=== Test 2: 'channels' topic ===\n")
	fmt.Printf("Returned %d documents (without filtering):\n", len(docs2))
	for i, doc := range docs2 {
		fmt.Printf("%d. %s (content len=%d)\n", i+1, doc.Title, len(doc.Content))
	}

	if len(docs2) == 0 && len(docs1) > 0 {
		t.Log("WARNING: Confluence has docs but 'channels' query returns 0 - likely CQL syntax or search term issue")
	}
}
