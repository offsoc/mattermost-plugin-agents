// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestFileProtocol_ComplexBooleanQueries(t *testing.T) {
	filePath := "assets/productboard-features.json"

	// Check if file exists and has content
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		t.Skip("ProductBoard data file not found")
	}
	if err == nil && fileInfo.Size() == 0 {
		t.Skip("ProductBoard data file is empty (placeholder file)")
	}

	protocol := NewFileProtocol(nil)

	source := SourceConfig{
		Name:     SourceProductBoardFeatures,
		Protocol: FileProtocolType,
		Endpoints: map[string]string{
			EndpointFilePath: "productboard-features.json",
		},
		Auth: AuthConfig{Type: AuthTypeNone},
	}

	setupFunc := func() error {
		t.Log("Testing File protocol with ProductBoard features - complex boolean queries")
		return nil
	}

	VerifyProtocolProductBoardBooleanQuery(t, protocol, source, setupFunc)
}

func TestFileProtocol_ComplexBooleanQueries_ZendeskTickets(t *testing.T) {
	source := SourceConfig{
		Name:     SourceZendeskTickets,
		Protocol: FileProtocolType,
		Endpoints: map[string]string{
			EndpointFilePath: "zendesk_tickets.txt",
		},
		Auth:     AuthConfig{Type: AuthTypeNone},
		Sections: []string{SectionGeneral},
	}

	filePath := "assets/" + source.Endpoints[EndpointFilePath]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Zendesk data file not available (data source disabled)")
	}

	protocol := NewFileProtocol(nil)
	ctx := context.Background()

	testCases := []struct {
		name          string
		query         string
		expectedMin   int
		shouldContain string
	}{
		{
			name:          "Bug with scroll",
			query:         "bug AND scroll",
			expectedMin:   1,
			shouldContain: "scroll",
		},
		{
			name:          "Enterprise OR tier_1",
			query:         "enterprise OR tier_1",
			expectedMin:   3,
			shouldContain: "",
		},
		{
			name:          "Complex: (bug OR issue) AND (scroll OR thread)",
			query:         "(bug OR issue) AND (scroll OR thread)",
			expectedMin:   1,
			shouldContain: "",
		},
		{
			name:          "Quoted phrase with AND",
			query:         "\"100k_customer\" AND mysql",
			expectedMin:   2,
			shouldContain: "mysql",
		},
		{
			name:          "NOT operator",
			query:         "scroll AND NOT japanese",
			expectedMin:   1,
			shouldContain: "scroll",
		},
		{
			name:          "Multi-word quoted phrase",
			query:         "\"mobile app\"",
			expectedMin:   1,
			shouldContain: "mobile app",
		},
		{
			name:          "Quoted phrase with OR",
			query:         "\"mobile app\" OR autoscrolling",
			expectedMin:   1,
			shouldContain: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.query,
				Sections: source.Sections,
				Limit:    10,
			}

			docs, err := protocol.Fetch(ctx, request)
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			if len(docs) < tc.expectedMin {
				t.Errorf("Expected at least %d docs, got %d for query: %s", tc.expectedMin, len(docs), tc.query)
			}

			if tc.shouldContain != "" && len(docs) > 0 {
				found := false
				searchText := docs[0].Title + " " + docs[0].Content
				if containsIgnoreCase(searchText, tc.shouldContain) {
					found = true
				}
				if !found {
					t.Errorf("Expected first doc to contain '%s' for query: %s", tc.shouldContain, tc.query)
				}
			}

			if len(docs) > 0 {
				queryNode, err := ParseBooleanQuery(tc.query)
				if err == nil {
					searchText := docs[0].Title + " " + docs[0].Content
					if !EvaluateBoolean(queryNode, searchText) {
						t.Errorf("First doc does not match boolean query: %s\nTitle: %s", tc.query, docs[0].Title)
					}
				}
			}

			t.Logf("Query '%s' returned %d docs", tc.name, len(docs))
		})
	}
}

func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

func TestFileProtocol_ComplexBooleanQueries_UserVoice(t *testing.T) {
	filePath := "assets/uservoice_suggestions.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("UserVoice data file not available (data source disabled)")
	}

	protocol := NewFileProtocol(nil)

	source := SourceConfig{
		Name:     SourceFeatureRequests,
		Protocol: FileProtocolType,
		Endpoints: map[string]string{
			EndpointFilePath: "uservoice_suggestions.json",
		},
		Auth:     AuthConfig{Type: AuthTypeNone},
		Sections: []string{SectionFeatureRequests},
	}

	ctx := context.Background()

	testCases := []struct {
		name          string
		query         string
		expectedMin   int
		shouldContain string
	}{
		{
			name:          "Simple AND query",
			query:         "thread AND quote",
			expectedMin:   1,
			shouldContain: "thread",
		},
		{
			name:          "OR query",
			query:         "channel OR thread",
			expectedMin:   3,
			shouldContain: "",
		},
		{
			name:          "Complex nested query",
			query:         "(reply OR quote) AND (thread OR message)",
			expectedMin:   1,
			shouldContain: "",
		},
		{
			name:          "Multi-word quoted phrase",
			query:         "\"collapsed threads\"",
			expectedMin:   1,
			shouldContain: "collapsed threads",
		},
		{
			name:          "NOT operator",
			query:         "quote AND NOT kubernetes",
			expectedMin:   1,
			shouldContain: "quote",
		},
		{
			name:          "Quoted phrase with AND",
			query:         "\"quote reply\" AND button",
			expectedMin:   1,
			shouldContain: "quote",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := ProtocolRequest{
				Source:   source,
				Topic:    tc.query,
				Sections: source.Sections,
				Limit:    10,
			}

			docs, err := protocol.Fetch(ctx, request)
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			if len(docs) < tc.expectedMin {
				t.Errorf("Expected at least %d docs, got %d for query: %s", tc.expectedMin, len(docs), tc.query)
			}

			if tc.shouldContain != "" && len(docs) > 0 {
				found := false
				searchText := docs[0].Title + " " + docs[0].Content
				if containsIgnoreCase(searchText, tc.shouldContain) {
					found = true
				}
				if !found {
					t.Errorf("Expected first doc to contain '%s' for query: %s", tc.shouldContain, tc.query)
				}
			}

			if len(docs) > 0 {
				queryNode, err := ParseBooleanQuery(tc.query)
				if err == nil {
					searchText := docs[0].Title + " " + docs[0].Content
					if !EvaluateBoolean(queryNode, searchText) {
						t.Errorf("First doc does not match boolean query: %s\nTitle: %s", tc.query, docs[0].Title)
					}
				}
			}

			t.Logf("Query '%s' returned %d docs", tc.name, len(docs))
		})
	}
}
