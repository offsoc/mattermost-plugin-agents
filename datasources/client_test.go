// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"testing"
	"time"
)

// Helper function to count enabled sources in a config
func countEnabledSources(config *Config) int {
	count := 0
	for _, source := range config.Sources {
		if source.Enabled {
			count++
		}
	}
	return count
}

// Helper function to get list of enabled source names
func getEnabledSourceNames(config *Config) []string {
	names := []string{}
	for _, source := range config.Sources {
		if source.Enabled {
			names = append(names, source.Name)
		}
	}
	return names
}

// MockProtocol implements DataSourceProtocol for testing
type MockProtocol struct {
	protocolType ProtocolType
	docs         []Doc
	err          error
	fetchCalled  bool
	auth         AuthConfig
}

func (m *MockProtocol) Fetch(ctx context.Context, request ProtocolRequest) ([]Doc, error) {
	m.fetchCalled = true
	if m.err != nil {
		return nil, m.err
	}
	return m.docs, nil
}

func (m *MockProtocol) GetType() ProtocolType {
	return m.protocolType
}

func (m *MockProtocol) SetAuth(auth AuthConfig) {
	m.auth = auth
}

func (m *MockProtocol) ValidateSearchSyntax(ctx context.Context, request ProtocolRequest) (*SyntaxValidationResult, error) {
	return &SyntaxValidationResult{
		OriginalQuery:     request.Topic,
		IsValidSyntax:     true,
		SyntaxErrors:      []string{},
		RecommendedQuery:  request.Topic,
		TestResultCount:   1,
		ActualAPIResponse: "mock response",
		SupportsFeatures:  []string{"mock"},
	}, nil
}

func TestNewClient(t *testing.T) {
	config := CreateDefaultConfig()
	client := NewClient(config, nil)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config != config {
		t.Error("Client config not set correctly")
	}

	if client.cache == nil {
		t.Error("Client cache not initialized")
	}

	if len(client.protocols) == 0 {
		t.Error("Client protocols not initialized")
	}

	expectedEnabledCount := countEnabledSources(config)
	if len(client.sources) != expectedEnabledCount {
		t.Errorf("Expected %d enabled sources initially, got %d", expectedEnabledCount, len(client.sources))
	}
}

func TestNewClient_WithNilConfig(t *testing.T) {
	client := NewClient(nil, nil)

	if client == nil {
		t.Fatal("Expected non-nil client even with nil config")
	}

	if client.config == nil {
		t.Error("Client should have default config when nil provided")
	}
}

func TestClient_GetAvailableSources(t *testing.T) {
	config := CreateDefaultConfig()

	config.Sources[0].Enabled = true
	config.Sources[5].Enabled = true

	client := NewClient(config, nil)
	sources := client.GetAvailableSources()

	expectedSourceNames := getEnabledSourceNames(config)

	if len(sources) != len(expectedSourceNames) {
		t.Errorf("Expected %d available sources, got %d", len(expectedSourceNames), len(sources))
	}

	expectedMap := make(map[string]bool)
	for _, name := range expectedSourceNames {
		expectedMap[name] = true
	}

	for _, sourceName := range sources {
		if !expectedMap[sourceName] {
			t.Errorf("Unexpected source name: %s", sourceName)
		}
	}
}

func TestClient_IsSourceEnabled(t *testing.T) {
	config := CreateDefaultConfig()
	config.Sources[0].Enabled = true

	client := NewClient(config, nil)

	if !client.IsSourceEnabled("mattermost_docs") {
		t.Error("mattermost_docs should be enabled")
	}

	if client.IsSourceEnabled("github_repos") {
		t.Error("github_repos should be disabled")
	}

	if client.IsSourceEnabled("nonexistent") {
		t.Error("nonexistent source should not be enabled")
	}
}

func TestClient_GetSourceConfig(t *testing.T) {
	config := CreateDefaultConfig()
	config.Sources[0].Enabled = true

	client := NewClient(config, nil)

	sourceConfig, exists := client.GetSourceConfig("mattermost_docs")
	if !exists {
		t.Error("mattermost_docs config should exist")
	}
	if sourceConfig.Name != "mattermost_docs" {
		t.Errorf("Expected source name 'mattermost_docs', got '%s'", sourceConfig.Name)
	}

	_, exists = client.GetSourceConfig("nonexistent")
	if exists {
		t.Error("nonexistent source config should not exist")
	}
}

func TestClient_FetchFromSource_SourceNotFound(t *testing.T) {
	config := CreateDefaultConfig()
	client := NewClient(config, nil)

	docs, err := client.FetchFromSource(context.Background(), "nonexistent", "test", 5)
	if err == nil {
		t.Error("Expected error for nonexistent source")
	}
	if docs != nil {
		t.Error("Expected nil docs for nonexistent source")
	}
}

func TestClient_FetchFromSource_WithMockProtocol(t *testing.T) {
	config := CreateDefaultConfig()
	config.Sources[0].Enabled = true

	client := NewClient(config, nil)

	mockDocs := []Doc{
		{Title: "Test Doc", Content: "Test content", URL: "http://test.com", Source: "test"},
	}
	mockProtocol := &MockProtocol{
		protocolType: HTTPProtocolType,
		docs:         mockDocs,
	}
	client.protocols[HTTPProtocolType] = mockProtocol

	docs, err := client.FetchFromSource(context.Background(), "mattermost_docs", "test", 5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	if !mockProtocol.fetchCalled {
		t.Error("Mock protocol Fetch method should have been called")
	}

	if docs[0].Title != "Test Doc" {
		t.Errorf("Expected title 'Test Doc', got '%s'", docs[0].Title)
	}
}

func TestClient_FetchFromMultipleSources(t *testing.T) {
	config := CreateDefaultConfig()
	config.Sources[0].Enabled = true
	config.Sources[5].Enabled = true

	client := NewClient(config, nil)

	httpMock := &MockProtocol{
		protocolType: HTTPProtocolType,
		docs: []Doc{
			{Title: "HTTP Doc", Content: "HTTP content", Source: "http"},
		},
	}
	githubMock := &MockProtocol{
		protocolType: GitHubAPIProtocolType,
		docs: []Doc{
			{Title: "GitHub Doc", Content: "GitHub content", Source: "github"},
		},
	}

	client.protocols[HTTPProtocolType] = httpMock
	client.protocols[GitHubAPIProtocolType] = githubMock

	sourceNames := []string{"mattermost_docs", "github_repos"}
	results, err := client.FetchFromMultipleSources(context.Background(), sourceNames, "test", 5)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 source results, got %d", len(results))
	}

	if _, exists := results["mattermost_docs"]; !exists {
		t.Error("Expected mattermost_docs in results")
	}

	if _, exists := results["github_repos"]; !exists {
		t.Error("Expected github_repos in results")
	}
}

func TestClient_GetCacheStats(t *testing.T) {
	config := CreateDefaultConfig()
	client := NewClient(config, nil)

	stats := client.GetCacheStats()

	if stats == nil {
		t.Fatal("Expected non-nil cache stats")
	}

	expectedFields := []string{"cache_ttl_hours", "enabled_sources", "protocols", "config_domains"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected field '%s' in cache stats", field)
		}
	}

	expectedEnabledCount := countEnabledSources(config)
	if stats["enabled_sources"].(int) != expectedEnabledCount {
		t.Errorf("Expected %d enabled sources, got %v", expectedEnabledCount, stats["enabled_sources"])
	}

	if stats["protocols"].(int) <= 0 {
		t.Errorf("Expected positive protocol count, got %v", stats["protocols"])
	}
}

func TestClient_Close(t *testing.T) {
	config := CreateDefaultConfig()
	client := NewClient(config, nil)

	err := client.Close()
	if err != nil {
		t.Errorf("Unexpected error on close: %v", err)
	}
}

func TestClient_FetchFromSource_WithCache(t *testing.T) {
	config := CreateDefaultConfig()
	config.Sources[0].Enabled = true
	config.CacheTTL = time.Hour

	client := NewClient(config, nil)

	mockDocs := []Doc{
		{Title: "Cached Doc", Content: "Cached content", Source: "test"},
	}
	mockProtocol := &MockProtocol{
		protocolType: HTTPProtocolType,
		docs:         mockDocs,
	}
	client.protocols[HTTPProtocolType] = mockProtocol

	docs1, err := client.FetchFromSource(context.Background(), "mattermost_docs", "test", 5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !mockProtocol.fetchCalled {
		t.Error("First fetch should call protocol")
	}

	mockProtocol.fetchCalled = false

	docs2, err := client.FetchFromSource(context.Background(), "mattermost_docs", "test", 5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if mockProtocol.fetchCalled {
		t.Error("Second fetch should use cache, not call protocol")
	}

	if len(docs1) != len(docs2) {
		t.Error("Cached results should match original results")
	}
}

func TestClient_LimitEnforcement(t *testing.T) {
	config := CreateDefaultConfig()
	config.Sources[0].Enabled = true
	config.Sources[0].MaxDocsPerCall = 2

	client := NewClient(config, nil)

	mockDocs := []Doc{
		{Title: "Doc 1", Source: "test"},
		{Title: "Doc 2", Source: "test"},
		{Title: "Doc 3", Source: "test"},
	}
	mockProtocol := &MockProtocol{
		protocolType: HTTPProtocolType,
		docs:         mockDocs,
	}
	client.protocols[HTTPProtocolType] = mockProtocol

	docs, err := client.FetchFromSource(context.Background(), "mattermost_docs", "test", 10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(docs) > 2 {
		t.Errorf("Expected at most 2 docs due to source limit, got %d", len(docs))
	}
}
