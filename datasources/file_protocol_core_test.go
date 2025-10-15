// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileProtocol_GetType(t *testing.T) {
	protocol := NewFileProtocol(nil)

	if protocol.GetType() != FileProtocolType {
		t.Errorf("Expected FileProtocolType, got %s", protocol.GetType())
	}
}

func TestFileProtocol_SetAuth(t *testing.T) {
	protocol := NewFileProtocol(nil)
	auth := AuthConfig{
		Type: AuthTypeNone,
		Key:  "",
	}

	protocol.SetAuth(auth)

	if protocol.auth.Type != AuthTypeNone {
		t.Errorf("Expected auth type %s, got %s", AuthTypeNone, protocol.auth.Type)
	}
}

func TestFileProtocol_ExtractSearchTerms(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tests := []struct {
		name     string
		topic    string
		expected []string
	}{
		{
			name:     "simple terms",
			topic:    "mobile notifications bug",
			expected: []string{"mobile", "notifications", "bug"},
		},
		{
			name:     "with stop words",
			topic:    "the bug in the mobile app",
			expected: []string{"bug", "mobile", "app"},
		},
		{
			name:     "mixed case",
			topic:    "Mobile Performance Issues",
			expected: []string{"mobile", "performance", "issues"},
		},
		{
			name:     "single word",
			topic:    "channels",
			expected: []string{"channels"},
		},
		{
			name:     "complex boolean query - market research",
			topic:    "(market OR competitive OR analysis OR trends OR research) AND (enterprise OR commercial)",
			expected: []string{"market", "competitive", "analysis", "trends", "research", "enterprise", "commercial"},
		},
		{
			name:     "complex boolean query - customer feedback",
			topic:    "(customer OR sales OR feedback OR community OR insights) AND (requirements OR requests OR voting)",
			expected: []string{"customer", "sales", "feedback", "community", "insights", "requirements", "requests", "voting"},
		},
		{
			name:     "complex boolean query with NOT",
			topic:    "(mobile OR web) AND (bug OR issue) AND NOT obsolete",
			expected: []string{"mobile", "web", "bug", "issue", "obsolete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := protocol.extractSearchTerms(tt.topic)
			if len(terms) != len(tt.expected) {
				t.Errorf("Expected %d terms, got %d: %v", len(tt.expected), len(terms), terms)
			}

			termMap := make(map[string]bool)
			for _, term := range terms {
				termMap[term] = true
			}
			for _, expected := range tt.expected {
				if !termMap[expected] {
					t.Errorf("Expected term %s not found in result: %v", expected, terms)
				}
			}
		})
	}
}

func TestFileProtocol_MatchesSearch(t *testing.T) {
	protocol := NewFileProtocol(nil)

	feature := ProductBoardFeature{
		Name:        "Mobile Push Notifications",
		Description: "Add support for push notifications on mobile devices",
		Type:        "feature",
		State:       "In Progress",
		Parent:      "Mobile App",
	}

	tests := []struct {
		name        string
		searchTerms []string
		expected    bool
	}{
		{
			name:        "matching terms",
			searchTerms: []string{"mobile", "notifications"},
			expected:    true,
		},
		{
			name:        "partial match",
			searchTerms: []string{"mobile", "calendar"},
			expected:    true,
		},
		{
			name:        "no match",
			searchTerms: []string{"channels", "sidebar"},
			expected:    false,
		},
		{
			name:        "empty search",
			searchTerms: []string{},
			expected:    true,
		},
		{
			name:        "single term match",
			searchTerms: []string{"mobile"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.matchesSearch(feature, tt.searchTerms)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFileProtocol_ValidateSearchSyntax(t *testing.T) {
	protocol := NewFileProtocol(nil)

	request := ProtocolRequest{
		Topic: "mobile notifications",
	}

	result, err := protocol.ValidateSearchSyntax(context.Background(), request)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !result.IsValidSyntax {
		t.Error("Expected valid syntax")
	}

	if result.OriginalQuery != request.Topic {
		t.Errorf("Expected original query %s, got %s", request.Topic, result.OriginalQuery)
	}

	if len(result.SupportsFeatures) == 0 {
		t.Error("Expected supported features to be listed")
	}
}

func TestFileProtocol_Fetch_MissingFilePath(t *testing.T) {
	protocol := NewFileProtocol(nil)

	request := ProtocolRequest{
		Source: SourceConfig{
			Name:      SourceProductBoardFeatures,
			Endpoints: map[string]string{},
		},
		Topic: "mobile",
		Limit: 5,
	}

	_, err := protocol.Fetch(context.Background(), request)

	if err == nil {
		t.Error("Expected error for missing file path")
	}

	if !contains(err.Error(), "file_path endpoint not configured") {
		t.Errorf("Expected file path error, got: %v", err)
	}
}

func TestFileProtocol_Fetch_WithTestData(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-features.json")

	testFeatures := []ProductBoardFeature{
		{
			Name:        "Mobile Push Notifications",
			Description: "Add support for push notifications on mobile devices. This feature enables real-time alerts for iOS and Android users when they receive new messages or mentions. The implementation includes background processing, notification grouping, and custom notification sounds to provide a comprehensive mobile experience.",
			Type:        "feature",
			State:       "In Progress",
			Owner:       "Alice",
			Parent:      "Mobile App",
		},
		{
			Name:        "Channel Sidebar Redesign",
			Description: "Redesign the channel sidebar for better user experience. This includes improved navigation, better visual hierarchy, drag and drop functionality for organizing channels, and enhanced search capabilities to help users find channels quickly.",
			Type:        "feature",
			State:       "Delivered",
			Owner:       "Bob",
		},
		{
			Name:        "Mobile Performance Improvements",
			Description: "Optimize mobile app performance through better caching strategies, lazy loading of content, reduced memory footprint, and improved network handling. These changes will reduce app startup time and improve responsiveness across all mobile devices.",
			Type:        "enhancement",
			State:       "Idea",
			Owner:       "Charlie",
		},
	}

	data, err := json.Marshal(testFeatures)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = os.WriteFile(testFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	request := ProtocolRequest{
		Source: SourceConfig{
			Name: SourceProductBoardFeatures,
			Endpoints: map[string]string{
				EndpointFilePath: testFile,
			},
		},
		Topic: "mobile",
		Limit: 10,
	}

	docs, err := protocol.Fetch(context.Background(), request)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one matching document")
	}

	for _, doc := range docs {
		if doc.Title == "" {
			t.Error("Document title should not be empty")
		}
		if doc.Source != SourceProductBoardFeatures {
			t.Errorf("Expected source %s, got %s", SourceProductBoardFeatures, doc.Source)
		}
	}
}

func TestFileProtocol_Fetch_RespectsLimit(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-features.json")

	testFeatures := make([]ProductBoardFeature, 10)
	for i := range testFeatures {
		testFeatures[i] = ProductBoardFeature{
			Name:        "Feature " + string(rune('A'+i)),
			Description: "Mobile feature description",
			Type:        "feature",
			State:       "Delivered",
		}
	}

	data, err := json.Marshal(testFeatures)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = os.WriteFile(testFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	request := ProtocolRequest{
		Source: SourceConfig{
			Name: SourceProductBoardFeatures,
			Endpoints: map[string]string{
				EndpointFilePath: testFile,
			},
		},
		Topic: "mobile",
		Limit: 3,
	}

	docs, err := protocol.Fetch(context.Background(), request)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(docs) > 3 {
		t.Errorf("Expected at most 3 documents, got %d", len(docs))
	}
}

func TestFileProtocol_Fetch_InvalidJSON(t *testing.T) {
	protocol := NewFileProtocol(nil)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.json")

	if err := os.WriteFile(testFile, []byte("invalid json{"), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	request := ProtocolRequest{
		Source: SourceConfig{
			Name: SourceProductBoardFeatures,
			Endpoints: map[string]string{
				EndpointFilePath: testFile,
			},
		},
		Topic: "mobile",
		Limit: 5,
	}

	_, err := protocol.Fetch(context.Background(), request)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !contains(err.Error(), "failed to parse JSON") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

func TestFileProtocol_Fetch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CreateDefaultConfig()
	config.EnableSource(SourceProductBoardFeatures)

	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	docs, err := client.FetchFromSource(ctx, SourceProductBoardFeatures, "mobile", 5)
	if err != nil {
		if contains(err.Error(), "no such file") {
			t.Skip("Skipping integration test - productboard-features.json not found")
		}
		t.Fatalf("Failed to fetch docs: %v", err)
	}

	if len(docs) == 0 {
		t.Log("Note: No documents matched search criteria")
	}

	for _, doc := range docs {
		if doc.Title == "" {
			t.Error("Document title should not be empty")
		}
		if doc.Source != SourceProductBoardFeatures {
			t.Errorf("Expected source %s, got %s", SourceProductBoardFeatures, doc.Source)
		}
		if doc.URL == "" {
			t.Error("Document URL should not be empty")
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
