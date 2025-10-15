// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestConfluenceProtocol_GetType(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)
	if protocol.GetType() != ConfluenceProtocolType {
		t.Errorf("Expected ConfluenceProtocolType, got %v", protocol.GetType())
	}
}

func TestConfluenceProtocol_SetAuth(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)
	auth := AuthConfig{Type: AuthTypeToken, Key: "user@example.com:token123"}

	protocol.SetAuth(auth)

	if protocol.auth.Type != AuthTypeToken || protocol.auth.Key != "user@example.com:token123" {
		t.Errorf("Auth not set correctly: %+v", protocol.auth)
	}
}

func TestConfluenceProtocol_GetSpaceKeysForSections(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Endpoints: map[string]string{
			EndpointSpaces: ConfluenceSpaces,
		},
	}

	spaceKeys := protocol.getSpaceKeysForSections(source, []string{SectionProductRequirements})
	expected := strings.Split(ConfluenceSpaces, ",")
	for i, key := range expected {
		expected[i] = strings.TrimSpace(key)
	}

	if len(spaceKeys) != len(expected) {
		t.Errorf("Expected %d space keys, got %d", len(expected), len(spaceKeys))
	}

	for i, key := range spaceKeys {
		if key != expected[i] {
			t.Errorf("Expected space key %s, got %s", expected[i], key)
		}
	}
}

func TestConfluenceProtocol_GetSpaceKeysForSections_Empty(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Endpoints: map[string]string{},
	}

	spaceKeys := protocol.getSpaceKeysForSections(source, []string{SectionProductRequirements})

	if spaceKeys != nil {
		t.Errorf("Expected nil space keys for missing endpoint, got %v", spaceKeys)
	}
}

func TestConfluenceProtocol_MapSpaceToSection(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tests := []struct {
		spaceKey string
		expected string
	}{
		{"PM-REQ", SectionProductRequirements},
		{"MARKET-RESEARCH", SectionMarketResearch},
		{"FEATURE-SPECS", SectionFeatureSpecs},
		{"ROADMAP-2024", SectionRoadmaps},
		{"COMPETITIVE-ANALYSIS", SectionCompetitiveAnalysis},
		{"CUSTOMER-FEEDBACK", SectionCustomerInsights},
		{"UNKNOWN-SPACE", SectionGeneral},
	}

	for _, test := range tests {
		result := protocol.mapSpaceToSection(test.spaceKey)
		if result != test.expected {
			t.Errorf("For space key %s, expected section '%s', got '%s'", test.spaceKey, test.expected, result)
		}
	}
}

func TestConfluenceProtocol_ExtractTextFromStorage(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	storageFormat := `<p>This is a <strong>test document</strong> with some content.</p>
	<h2>Features</h2>
	<ul>
		<li>Feature 1</li>
		<li>Feature 2</li>
	</ul>
	<ac:structured-macro ac:name="info">
		<ac:rich-text-body>
			<p>Important information here</p>
		</ac:rich-text-body>
	</ac:structured-macro>`

	text := protocol.htmlProcessor.ExtractStructuredText(storageFormat)

	// Check that text content is extracted and XML tags are removed
	if !strings.Contains(text, "test document") {
		t.Errorf("Expected text to contain 'test document', got: %s", text)
	}
	if !strings.Contains(text, "Feature 1") {
		t.Errorf("Expected text to contain 'Feature 1'")
	}
	if !strings.Contains(text, "Important information") {
		t.Errorf("Expected text to contain 'Important information'")
	}
	if strings.Contains(text, "<p>") || strings.Contains(text, "</p>") {
		t.Errorf("Text should not contain XML tags")
	}
	if strings.Contains(text, "ac:structured-macro") {
		t.Errorf("Text should not contain Confluence macro tags")
	}
}

func TestConfluenceProtocol_ConvertToDoc(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	confluenceContent := ConfluenceContent{
		ID:    "12345",
		Type:  "page",
		Title: "Product Requirements Document",
		Space: struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		}{
			Key:  "PM",
			Name: "Product Management",
		},
		Body: struct {
			View struct {
				Value          string `json:"value"`
				Representation string `json:"representation"`
			} `json:"view"`
		}{
			View: struct {
				Value          string `json:"value"`
				Representation string `json:"representation"`
			}{
				Value:          "<p>This document outlines the product requirements for the new feature.</p><h2>Overview</h2><p>The feature should provide enhanced user experience.</p>",
				Representation: "storage",
			},
		},
		Links: struct {
			Base string `json:"base"`
			Web  string `json:"webui"`
		}{
			Base: "https://mattermost.atlassian.net",
			Web:  "/wiki/spaces/PM/pages/2569109806/Get+the+most+out+of+your+team+space",
		},
	}

	doc := protocol.convertToDoc(confluenceContent, SourceConfluenceDocs, "test topic")

	if doc == nil {
		t.Fatal("Expected non-nil doc")
	}

	if doc.Title != "Product Requirements Document" {
		t.Errorf("Expected title 'Product Requirements Document', got '%s'", doc.Title)
	}

	if doc.Source != SourceConfluenceDocs {
		t.Errorf("Expected source '%s', got '%s'", SourceConfluenceDocs, doc.Source)
	}

	if doc.Section != SectionProductRequirements {
		t.Errorf("Expected section '%s', got '%s'", SectionProductRequirements, doc.Section)
	}

	expectedURL := "https://mattermost.atlassian.net/wiki/spaces/PM/pages/2569109806/Get+the+most+out+of+your+team+space"
	if doc.URL != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, doc.URL)
	}

	if !strings.Contains(doc.Content, "product requirements") {
		t.Errorf("Expected content to contain 'product requirements', got '%s'", doc.Content)
	}
}

func TestConfluenceProtocol_ConvertToDoc_EmptyStorage(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	confluenceContent := ConfluenceContent{
		ID:    "12345",
		Title: "Empty Document",
		Body: struct {
			View struct {
				Value          string `json:"value"`
				Representation string `json:"representation"`
			} `json:"view"`
		}{
			View: struct {
				Value          string `json:"value"`
				Representation string `json:"representation"`
			}{
				Value: "",
			},
		},
	}

	doc := protocol.convertToDoc(confluenceContent, SourceConfluenceDocs, "test topic")

	if doc != nil {
		t.Errorf("Expected nil doc for empty storage, got %+v", doc)
	}
}

func TestConfluenceProtocol_Fetch_MissingBaseURL(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	source := SourceConfig{
		Name: SourceConfluenceDocs,
		Endpoints: map[string]string{
			EndpointSpaces: "MARKET",
		},
		RateLimit: RateLimitConfig{
			Enabled: false,
		},
	}

	request := ProtocolRequest{
		Source:   source,
		Topic:    "test",
		Sections: []string{SectionMarketResearch},
		Limit:    5,
	}

	docs, err := protocol.Fetch(context.Background(), request)
	if err != nil {
		t.Errorf("Expected no error for missing base URL, got %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("Expected 0 documents for missing base URL, got %d", len(docs))
	}
}

func TestConfluenceProtocol_EscapeCQLString(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no special characters",
			input:    "mobile app",
			expected: "mobile app",
		},
		{
			name:     "double quotes",
			input:    `search "mobile app"`,
			expected: `search \"mobile app\"`,
		},
		{
			name:     "backslash",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "plus and minus",
			input:    "mobile+app-development",
			expected: `mobile\+app\-development`,
		},
		{
			name:     "parentheses and braces",
			input:    "mobile(app){development}",
			expected: `mobile\(app\)\{development\}`,
		},
		{
			name:     "brackets",
			input:    "mobile[app]",
			expected: `mobile\[app\]`,
		},
		{
			name:     "special search characters",
			input:    "mobile*app?query~fuzzy",
			expected: `mobile\*app\?query\~fuzzy`,
		},
		{
			name:     "colon and forward slash",
			input:    "https://example.com/path",
			expected: `https\:\/\/example.com\/path`,
		},
		{
			name:     "multiple special characters",
			input:    `query+"mobile app"-[tag]`,
			expected: `query\+\"mobile app\"\-\[tag\]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.escapeCQLString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConfluenceProtocol_BuildExpandedCQL(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	tests := []struct {
		name        string
		spaceKey    string
		topic       string
		expectedCQL []string // Expected substrings in the CQL
	}{
		{
			name:     "empty topic",
			spaceKey: "TEST",
			topic:    "",
			expectedCQL: []string{
				"space = TEST",
				"type = page",
				"ORDER BY lastModified DESC",
			},
		},
		{
			name:     "simple topic",
			spaceKey: "TEST",
			topic:    "mobile",
			expectedCQL: []string{
				"space = TEST",
				"type = page",
				"title ~ \"mobile\"",
				"text ~ \"mobile\"",
				"ORDER BY lastModified DESC",
			},
		},
		{
			name:     "topic with special characters",
			spaceKey: "TEST",
			topic:    "mobile+app",
			expectedCQL: []string{
				"space = TEST",
				"type = page",
				"title ~ \"mobile\\+app\"",
				"text ~ \"mobile\\+app\"",
				"ORDER BY lastModified DESC",
			},
		},
		{
			name:     "topic with synonyms",
			spaceKey: "TEST",
			topic:    "ai",
			expectedCQL: []string{
				"space = TEST",
				"type = page",
				"title ~ \"ai\"",
				"text ~ \"ai\"",
				"ORDER BY lastModified DESC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.buildExpandedCQL(tt.spaceKey, tt.topic)

			for _, expected := range tt.expectedCQL {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected CQL to contain '%s', got '%s'", expected, result)
				}
			}

			// Check basic structure
			if !strings.HasPrefix(result, "space = "+tt.spaceKey) {
				t.Errorf("Expected CQL to start with space constraint, got '%s'", result)
			}

			if !strings.HasSuffix(result, "ORDER BY lastModified DESC") {
				t.Errorf("Expected CQL to end with ordering, got '%s'", result)
			}
		})
	}
}

func TestConfluenceProtocol_BuildExpandedCQL_CharacterBudget(t *testing.T) {
	protocol := NewConfluenceProtocol(&http.Client{}, nil)

	// Test with a topic that would generate a very long CQL to test budget management
	longTopic := "artificial intelligence machine learning deep learning neural networks"
	result := protocol.buildExpandedCQL("TEST", longTopic)

	// Should not be excessively long (budget management should kick in)
	if len(result) > 8000 {
		t.Errorf("Expected CQL to respect character budget, got length %d: %s", len(result), result)
	}

	// Should still contain basic structure
	if !strings.Contains(result, "space = TEST") {
		t.Errorf("Expected CQL to contain space constraint even with budget limits")
	}

	if !strings.Contains(result, "ORDER BY lastModified DESC") {
		t.Errorf("Expected CQL to contain ordering even with budget limits")
	}
}
