// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"os"
	"testing"
)

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateDefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil default config")
	}

	enabledByDefault := map[string]bool{
		SourceCommunityForum:       true,
		SourceFeatureRequests:      false, // DISABLED - no API access
		SourceProductBoardFeatures: true,
		SourceZendeskTickets:       true,
	}

	for _, source := range config.Sources {
		expectedEnabled := enabledByDefault[source.Name]
		if source.Enabled != expectedEnabled {
			t.Errorf("Source %s: expected enabled=%v, got enabled=%v", source.Name, expectedEnabled, source.Enabled)
		}
	}

	expectedSources := []string{"mattermost_docs", "mattermost_handbook", "mattermost_forum", "mattermost_blog", "mattermost_newsroom", "github_repos", "community_forum", "mattermost_hub", "confluence_docs", "plugin_marketplace", "feature_requests", "jira_docs", "productboard_features", "zendesk_tickets"}
	if len(config.Sources) != len(expectedSources) {
		t.Errorf("Expected %d sources, got %d", len(expectedSources), len(config.Sources))
	}

	for i, expectedName := range expectedSources {
		if config.Sources[i].Name != expectedName {
			t.Errorf("Expected source %s at index %d, got %s", expectedName, i, config.Sources[i].Name)
		}
	}

	if config.CacheTTL != DefaultCacheTTL {
		t.Errorf("Expected default cache TTL %v, got %v", DefaultCacheTTL, config.CacheTTL)
	}

	// Check allowed domains (mattermost.uservoice.com temporarily removed due to feature_requests being disabled)
	expectedDomains := []string{"docs.mattermost.com", "handbook.mattermost.com", "forum.mattermost.com", "mattermost.com", "api.github.com", "community.mattermost.com", "hub.mattermost.com", "mattermost.atlassian.net", "integrations.mattermost.com"}
	if len(config.AllowedDomains) != len(expectedDomains) {
		t.Errorf("Expected %d allowed domains, got %d", len(expectedDomains), len(config.AllowedDomains))
	}
}

func TestConfig_IsEnabled(t *testing.T) {
	config := CreateDefaultConfig()
	if !config.IsEnabled() {
		t.Error("Config should be enabled with default settings since some sources are enabled")
	}

	for i := range config.Sources {
		config.Sources[i].Enabled = false
	}
	if config.IsEnabled() {
		t.Error("Config should not be enabled when all sources are disabled")
	}

	config.Sources[0].Enabled = true
	if !config.IsEnabled() {
		t.Error("Config should be enabled when at least one source is enabled")
	}

	var nilConfig *Config
	if nilConfig.IsEnabled() {
		t.Error("Nil config should not be enabled")
	}
}

func TestConfig_GetEnabledSources(t *testing.T) {
	config := CreateDefaultConfig()

	initialEnabled := config.GetEnabledSources()
	initialCount := len(initialEnabled)

	// Track which sources are already enabled
	alreadyEnabled := make(map[string]bool)
	for _, source := range initialEnabled {
		alreadyEnabled[source.Name] = true
	}

	var indicesToEnable []int
	for i, source := range config.Sources {
		if !source.Enabled {
			indicesToEnable = append(indicesToEnable, i)
			if len(indicesToEnable) >= 2 {
				break
			}
		}
	}

	for _, idx := range indicesToEnable {
		config.Sources[idx].Enabled = true
	}

	enabled := config.GetEnabledSources()
	expectedCount := initialCount + len(indicesToEnable)
	if len(enabled) != expectedCount {
		t.Errorf("Expected %d enabled sources, got %d", expectedCount, len(enabled))
	}

	for _, source := range enabled {
		// Just verify we got source configs back
		if source.Name == "" {
			t.Error("Got source with empty name")
		}
	}

	if len(indicesToEnable) > 0 && config.Sources[indicesToEnable[0]].Name == "mattermost_docs" {
		found := false
		for _, source := range enabled {
			if source.Name == "mattermost_docs" {
				found = true
				break
			}
		}
		if !found {
			t.Error("mattermost_docs should be enabled but wasn't found")
		}
	}
}

func TestLoadConfig_WithEnvironmentFallbacks(t *testing.T) {
	os.Setenv("MM_AI_GITHUB_TOKEN", "test-github-token")
	os.Setenv("MM_AI_MATTERMOST_DOCS_TOKEN", "test-docs-token")
	defer func() {
		os.Unsetenv("MM_AI_GITHUB_TOKEN")
		os.Unsetenv("MM_AI_MATTERMOST_DOCS_TOKEN")
	}()

	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.GitHubToken != "test-github-token" {
		t.Errorf("Expected GitHub token 'test-github-token', got '%s'", config.GitHubToken)
	}

	// Check that source-specific token was set
	var mattermostDocsSource *SourceConfig
	for i := range config.Sources {
		if config.Sources[i].Name == "mattermost_docs" {
			mattermostDocsSource = &config.Sources[i]
			break
		}
	}

	if mattermostDocsSource == nil {
		t.Fatal("Could not find mattermost_docs source")
	}

	if mattermostDocsSource.Auth.Key != "test-docs-token" {
		t.Errorf("Expected mattermost_docs auth key 'test-docs-token', got '%s'", mattermostDocsSource.Auth.Key)
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	config, err := LoadConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("Should not error on non-existent file: %v", err)
	}

	if config == nil {
		t.Fatal("Expected default config for non-existent file")
	}

	defaultConfig := CreateDefaultConfig()
	if len(config.Sources) != len(defaultConfig.Sources) {
		t.Error("Non-existent file config should match default config")
	}
}

func TestSourceConfig_RateLimitDefaults(t *testing.T) {
	config := CreateDefaultConfig()

	mattermostDocs := config.Sources[0]
	if mattermostDocs.RateLimit.RequestsPerMinute != 30 {
		t.Errorf("Expected mattermost_docs rate limit 30, got %d", mattermostDocs.RateLimit.RequestsPerMinute)
	}
	if mattermostDocs.RateLimit.BurstSize != 5 {
		t.Errorf("Expected mattermost_docs burst size 5, got %d", mattermostDocs.RateLimit.BurstSize)
	}
	if !mattermostDocs.RateLimit.Enabled {
		t.Error("Expected mattermost_docs rate limiting to be enabled")
	}

	githubRepos := config.Sources[5]
	if githubRepos.RateLimit.RequestsPerMinute != 60 {
		t.Errorf("Expected github_repos rate limit 60, got %d", githubRepos.RateLimit.RequestsPerMinute)
	}
	if githubRepos.RateLimit.BurstSize != 10 {
		t.Errorf("Expected github_repos burst size 10, got %d", githubRepos.RateLimit.BurstSize)
	}

	communityForum := config.Sources[6]
	if communityForum.RateLimit.RequestsPerMinute != 20 {
		t.Errorf("Expected community_forum rate limit 20, got %d", communityForum.RateLimit.RequestsPerMinute)
	}
	if communityForum.RateLimit.BurstSize != 3 {
		t.Errorf("Expected community_forum burst size 3, got %d", communityForum.RateLimit.BurstSize)
	}
}

func TestSourceConfig_Endpoints(t *testing.T) {
	config := CreateDefaultConfig()

	mattermostDocs := config.Sources[0]
	expectedEndpoints := map[string]string{
		"base_url":        "https://docs.mattermost.com",
		"admin":           "/administration-guide/administration-guide-index.html",
		"developer":       "/deployment-guide/deployment-guide-index.html",
		"api":             "/administration/changelog.html",
		"mobile":          "/deployment-guide/mobile/mobile-app-deployment.html",
		"mobile_apps":     "/deployment-guide/mobile/mobile-faq.html",
		"mobile_strategy": "/product-overview/mattermost-mobile-releases.html",
	}

	for key, expectedValue := range expectedEndpoints {
		if mattermostDocs.Endpoints[key] != expectedValue {
			t.Errorf("Expected %s endpoint '%s', got '%s'", key, expectedValue, mattermostDocs.Endpoints[key])
		}
	}

	githubRepos := config.Sources[5]
	if githubRepos.Endpoints["owner"] != "mattermost" {
		t.Errorf("Expected github owner 'mattermost', got '%s'", githubRepos.Endpoints["owner"])
	}
	if githubRepos.Endpoints["repos"] != GitHubReposList {
		t.Errorf("Expected github repos list, got '%s'", githubRepos.Endpoints["repos"])
	}

	communityForum := config.Sources[6]
	if communityForum.Endpoints["base_url"] != "https://community.mattermost.com" {
		t.Errorf("Expected community forum base_url, got '%s'", communityForum.Endpoints["base_url"])
	}
}

func TestSourceConfig_ProtocolTypes(t *testing.T) {
	config := CreateDefaultConfig()

	expectedProtocols := []ProtocolType{HTTPProtocolType, HTTPProtocolType, DiscourseProtocolType, HTTPProtocolType, HTTPProtocolType, GitHubAPIProtocolType, MattermostProtocolType, MattermostProtocolType, ConfluenceProtocolType, HTTPProtocolType, FileProtocolType, JiraProtocolType, FileProtocolType, FileProtocolType}

	for i, source := range config.Sources {
		if source.Protocol != expectedProtocols[i] {
			t.Errorf("Expected source %s to have protocol %s, got %s", source.Name, expectedProtocols[i], source.Protocol)
		}
	}
}
