// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration

package mmtools

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/config"
	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createRealCommonDataSourcesClient creates a real common data sources client for integration testing
func createRealCommonDataSourcesClient() *datasources.Client {
	testConfig := &datasources.Config{
		Sources: []datasources.SourceConfig{
			{
				Name:     datasources.SourceMattermostDocs,
				Enabled:  true,
				Protocol: datasources.HTTPProtocolType,
				Endpoints: map[string]string{
					datasources.EndpointBaseURL:   datasources.MattermostDocsURL,
					datasources.EndpointAdmin:     datasources.DocsAdminPath,
					datasources.EndpointDeveloper: datasources.DocsDeveloperPath,
					datasources.EndpointAPI:       datasources.DocsAPIPath,
					datasources.EndpointMobile:    datasources.DocsMobilePath,
				},
				Auth:           datasources.AuthConfig{Type: datasources.AuthTypeNone},
				Sections:       []string{datasources.SectionAdmin, datasources.SectionDeveloper, datasources.SectionAPI, datasources.SectionMobile},
				MaxDocsPerCall: 3, // Limit for testing
				RateLimit: datasources.RateLimitConfig{
					RequestsPerMinute: datasources.DefaultRequestsPerMinuteHTTP,
					BurstSize:         datasources.DefaultBurstSizeHTTP,
					Enabled:           true,
				},
			},
			{
				Name:     datasources.SourceGitHubRepos,
				Enabled:  true,
				Protocol: datasources.GitHubAPIProtocolType,
				Endpoints: map[string]string{
					datasources.EndpointOwner: datasources.GitHubOwnerMattermost,
					datasources.EndpointRepos: "mattermost-server", // Single repo for testing
				},
				Auth:           datasources.AuthConfig{Type: datasources.AuthTypeToken, Key: ""}, // No token for public repos
				Sections:       []string{datasources.SectionIssues, datasources.SectionReleases},
				MaxDocsPerCall: 5, // Limit for testing
				RateLimit: datasources.RateLimitConfig{
					RequestsPerMinute: datasources.DefaultRequestsPerMinuteGitHub,
					BurstSize:         datasources.DefaultBurstSizeGitHub,
					Enabled:           true,
				},
			},
			{
				Name:     datasources.SourceConfluenceDocs,
				Enabled:  false, // Disabled by default for integration tests (requires auth)
				Protocol: datasources.ConfluenceProtocolType,
				Endpoints: map[string]string{
					datasources.EndpointBaseURL: datasources.ConfluenceURL,
					datasources.EndpointSpaces:  datasources.ConfluenceSpaces,
					datasources.EndpointEmail:   "",
				},
				Auth:           datasources.AuthConfig{Type: datasources.AuthTypeToken, Key: ""}, // Would need real email:token
				Sections:       []string{datasources.SectionProductRequirements, datasources.SectionMarketResearch, datasources.SectionFeatureSpecs},
				MaxDocsPerCall: 3, // Limit for testing
				RateLimit: datasources.RateLimitConfig{
					RequestsPerMinute: datasources.DefaultRequestsPerMinuteConfluence,
					BurstSize:         datasources.DefaultBurstSizeConfluence,
					Enabled:           true,
				},
			},
		},
		AllowedDomains: []string{"docs.mattermost.com", "api.github.com", "mattermost.atlassian.net"},
		GitHubToken:    "", // No token needed for public repos
		CacheTTL:       datasources.DefaultCacheTTL,
	}

	return datasources.NewClient(testConfig, nil)
}

func TestConfluenceProtocolIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Confluence integration test in short mode")
	}

	confluenceToken := os.Getenv("MM_AI_CONFLUENCE_DOCS_TOKEN")
	if confluenceToken == "" {
		t.Skip("Skipping Confluence integration test - no MM_AI_CONFLUENCE_DOCS_TOKEN environment variable")
	}

	confluenceConfig := &datasources.Config{
		Sources: []datasources.SourceConfig{
			{
				Name:     datasources.SourceConfluenceDocs,
				Enabled:  true,
				Protocol: datasources.ConfluenceProtocolType,
				Endpoints: map[string]string{
					datasources.EndpointBaseURL: datasources.ConfluenceURL,
					datasources.EndpointSpaces:  datasources.ConfluenceSpaces,
				},
				Auth:           datasources.AuthConfig{Type: datasources.AuthTypeToken, Key: confluenceToken},
				Sections:       []string{datasources.SectionProductRequirements, datasources.SectionMarketResearch},
				MaxDocsPerCall: 2, // Limit for testing
				RateLimit: datasources.RateLimitConfig{
					RequestsPerMinute: 5, // Lower rate for testing
					BurstSize:         2,
					Enabled:           true,
				},
			},
		},
		AllowedDomains: []string{"mattermost.atlassian.net"},
		CacheTTL:       datasources.DefaultCacheTTL,
	}

	confluenceClient := datasources.NewClient(confluenceConfig, nil)
	defer confluenceClient.Close()

	t.Run("Fetch Confluence Product Requirements", func(t *testing.T) {
		docs, err := confluenceClient.FetchFromSource(context.Background(),
			datasources.SourceConfluenceDocs, "requirements", 2)

		require.NoError(t, err)
		require.NotNil(t, docs)

		if len(docs) > 0 {
			assert.NotEmpty(t, docs[0].Title)
			assert.NotEmpty(t, docs[0].Content)
			assert.NotEmpty(t, docs[0].URL)
			assert.Contains(t, docs[0].URL, "mattermost.atlassian.net")
			assert.Equal(t, datasources.SourceConfluenceDocs, docs[0].Source)

			t.Logf("Fetched Confluence doc: %s - %s", docs[0].Title, docs[0].URL)
			t.Logf("Content preview: %.200s...", docs[0].Content)
		}
	})

	t.Run("Fetch Confluence Market Research", func(t *testing.T) {
		docs, err := confluenceClient.FetchFromSource(context.Background(),
			datasources.SourceConfluenceDocs, "market research", 2)

		require.NoError(t, err)
		require.NotNil(t, docs)

		if len(docs) > 0 {
			assert.NotEmpty(t, docs[0].Title)
			assert.NotEmpty(t, docs[0].Content)
			assert.NotEmpty(t, docs[0].URL)
			assert.Contains(t, docs[0].URL, "mattermost.atlassian.net")
			assert.Equal(t, datasources.SourceConfluenceDocs, docs[0].Source)

			t.Logf("Fetched market research doc: %s - %s", docs[0].Title, docs[0].URL)
			t.Logf("Content preview: %.200s...", docs[0].Content)
		}
	})

	t.Run("Test Confluence Rate Limiting", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			docs, err := confluenceClient.FetchFromSource(context.Background(),
				datasources.SourceConfluenceDocs, "test", 1)

			require.NoError(t, err)
			require.NotNil(t, docs)

			t.Logf("Rate limit test %d: fetched %d docs", i+1, len(docs))
		}
	})
}

// MockPluginAPI for testing
type MockPluginAPI struct{}

func (m *MockPluginAPI) GetPluginStatus(pluginID string) (*model.PluginStatus, error) {
	return &model.PluginStatus{State: model.PluginStateRunning}, nil
}

// MockSearch for testing
type MockSearch struct{}

func (m *MockSearch) Enabled() bool {
	return true
}

func TestCompileMarketResearchWithCommonDataSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	commonDataSourcesClient := createRealCommonDataSourcesClient()
	defer commonDataSourcesClient.Close()

	t.Run("Fetch from Mattermost Docs", func(t *testing.T) {
		docs, err := commonDataSourcesClient.FetchFromSource(context.Background(),
			datasources.SourceMattermostDocs, "mobile", 2)

		require.NoError(t, err)
		require.NotNil(t, docs)

		if len(docs) > 0 {
			assert.NotEmpty(t, docs[0].Title)
			assert.NotEmpty(t, docs[0].Content)
			assert.NotEmpty(t, docs[0].URL)
			assert.Contains(t, docs[0].URL, "docs.mattermost.com")
			assert.Equal(t, datasources.SourceMattermostDocs, docs[0].Source)

			t.Logf("Fetched doc: %s - %s", docs[0].Title, docs[0].URL)
			t.Logf("Content preview: %.200s...", docs[0].Content)
		}
	})

	t.Run("Fetch from GitHub", func(t *testing.T) {
		docs, err := commonDataSourcesClient.FetchFromSource(context.Background(),
			datasources.SourceGitHubRepos, "mobile", 2)

		require.NoError(t, err)
		require.NotNil(t, docs)

		if len(docs) > 0 {
			assert.NotEmpty(t, docs[0].Title)
			assert.NotEmpty(t, docs[0].Content)
			assert.NotEmpty(t, docs[0].URL)
			assert.Contains(t, docs[0].URL, "github.com")
			assert.Contains(t, docs[0].Source, "github")

			t.Logf("Fetched GitHub doc: %s - %s", docs[0].Title, docs[0].URL)
			t.Logf("Content preview: %.200s...", docs[0].Content)
		}
	})

	t.Run("Fetch from Multiple Sources", func(t *testing.T) {
		multiDocs, err := commonDataSourcesClient.FetchFromMultipleSources(context.Background(),
			[]string{datasources.SourceMattermostDocs, datasources.SourceGitHubRepos}, "API", 2)

		require.NoError(t, err)
		require.NotNil(t, multiDocs)

		t.Logf("Fetched from %d sources", len(multiDocs))
		for sourceName, docs := range multiDocs {
			t.Logf("Source %s: %d documents", sourceName, len(docs))
			for i, doc := range docs {
				t.Logf("  Doc %d: %s - %s", i+1, doc.Title, doc.URL)
			}
		}
	})
}

func TestAnalyzeFeatureGapsWithCommonDataSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	commonDataSourcesClient := createRealCommonDataSourcesClient()
	defer commonDataSourcesClient.Close()

	t.Run("Search Integration Documentation", func(t *testing.T) {
		docs, err := commonDataSourcesClient.FetchFromSource(context.Background(),
			datasources.SourceMattermostDocs, "integration", 3)

		require.NoError(t, err)
		require.NotNil(t, docs)

		if len(docs) > 0 {
			assert.NotEmpty(t, docs[0].Title)
			assert.NotEmpty(t, docs[0].Content)
			assert.NotEmpty(t, docs[0].URL)

			t.Logf("Integration doc: %s - %s", docs[0].Title, docs[0].URL)
			t.Logf("Content preview: %.300s...", docs[0].Content)
		}
	})

	t.Run("Search GitHub for Features", func(t *testing.T) {
		docs, err := commonDataSourcesClient.FetchFromSource(context.Background(),
			datasources.SourceGitHubRepos, "webhook", 3)

		require.NoError(t, err)
		require.NotNil(t, docs)

		if len(docs) > 0 {
			assert.NotEmpty(t, docs[0].Title)
			assert.NotEmpty(t, docs[0].Content)
			assert.NotEmpty(t, docs[0].URL)

			t.Logf("GitHub feature doc: %s - %s", docs[0].Title, docs[0].URL)
			t.Logf("Content preview: %.300s...", docs[0].Content)
		}
	})

	t.Run("Cross-Source Feature Analysis", func(t *testing.T) {
		multiDocs, err := commonDataSourcesClient.FetchFromMultipleSources(context.Background(),
			[]string{datasources.SourceMattermostDocs, datasources.SourceGitHubRepos}, "plugin", 2)

		require.NoError(t, err)
		require.NotNil(t, multiDocs)

		for sourceName, docs := range multiDocs {
			t.Logf("Analyzing %s for plugin-related features:", sourceName)
			for i, doc := range docs {
				t.Logf("  Feature doc %d: %s", i+1, doc.Title)

				if strings.Contains(strings.ToLower(doc.Content), "api") ||
					strings.Contains(strings.ToLower(doc.Content), "webhook") ||
					strings.Contains(strings.ToLower(doc.Content), "integration") {
					t.Logf("    Found integration-related content in: %s", doc.URL)
				}
			}
		}
	})
}

func TestToolProviderWithCommonDataSourcesDisabled(t *testing.T) {
	provider := &MMToolProvider{
		pluginAPI:           nil,
		search:              nil,
		httpClient:          &http.Client{},
		commonDataSrcClient: nil, // Common data sources disabled
		configContainer:     &config.Container{},
	}

	require.NotNil(t, provider)
	assert.Nil(t, provider.commonDataSrcClient)
}

func TestCommonDataSourcesRealClientFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	realClient := createRealCommonDataSourcesClient()
	defer realClient.Close()

	t.Run("Client Creation and Sources", func(t *testing.T) {
		require.NotNil(t, realClient)

		sources := realClient.GetAvailableSources()
		require.NotNil(t, sources)
		assert.Contains(t, sources, datasources.SourceMattermostDocs)
		assert.Contains(t, sources, datasources.SourceGitHubRepos)

		t.Logf("Available sources: %v", sources)
	})

	t.Run("Source Enablement", func(t *testing.T) {
		assert.True(t, realClient.IsSourceEnabled(datasources.SourceMattermostDocs))
		assert.True(t, realClient.IsSourceEnabled(datasources.SourceGitHubRepos))

		assert.False(t, realClient.IsSourceEnabled("non_existent_source"))
	})

	t.Run("Source Configuration", func(t *testing.T) {
		config, exists := realClient.GetSourceConfig(datasources.SourceMattermostDocs)
		assert.True(t, exists)
		assert.Equal(t, datasources.SourceMattermostDocs, config.Name)
		assert.Equal(t, datasources.HTTPProtocolType, config.Protocol)
		assert.True(t, config.Enabled)

		t.Logf("Mattermost docs config: %+v", config)
	})

	t.Run("Caching Behavior", func(t *testing.T) {
		docs1, err1 := realClient.FetchFromSource(context.Background(),
			datasources.SourceMattermostDocs, "cache_test", 1)
		require.NoError(t, err1)

		docs2, err2 := realClient.FetchFromSource(context.Background(),
			datasources.SourceMattermostDocs, "cache_test", 1)
		require.NoError(t, err2)

		assert.NotNil(t, docs1)
		assert.NotNil(t, docs2)

		t.Logf("First fetch returned %d docs, second fetch returned %d docs", len(docs1), len(docs2))
	})
}
