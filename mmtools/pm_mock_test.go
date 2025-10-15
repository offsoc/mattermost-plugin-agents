// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/config"
	pmconfig "github.com/mattermost/mattermost-plugin-ai/config/pm"
	sharedconfig "github.com/mattermost/mattermost-plugin-ai/config/shared"
	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/pm"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPMToolsWithHubChannelMocks(t *testing.T) {
	// Use the shared ResolveAssetPath function to find the fallback-data directory
	fallbackPath := datasources.ResolveAssetPath("fallback-data", "")

	// Check if Hub fallback files exist and have content
	hubContactSalesPath := filepath.Join(fallbackPath, "hub-contact-sales.txt")
	hubCustomerFeedbackPath := filepath.Join(fallbackPath, "hub-customer-feedback.txt")

	contactSalesInfo, err1 := os.Stat(hubContactSalesPath)
	customerFeedbackInfo, err2 := os.Stat(hubCustomerFeedbackPath)

	hasContactSalesData := err1 == nil && contactSalesInfo.Size() > 0
	hasCustomerFeedbackData := err2 == nil && customerFeedbackInfo.Size() > 0

	if !hasContactSalesData && !hasCustomerFeedbackData {
		t.Skip("Hub fallback data files are empty or missing (placeholder files)")
	}

	// Create external docs config with only Hub enabled for this test
	commonDataSourcesConfig := datasources.CreateDefaultConfig()
	commonDataSourcesConfig.FallbackDirectory = fallbackPath
	commonDataSourcesConfig.EnableSource(datasources.SourceMattermostHub)

	// Create PM config data
	pmConfigData := pmconfig.Config{
		Enabled: true, // Enable PM tools for this test
		ChannelTargets: map[string][]string{
			pm.ToolNameCompileMarketResearch: {"contact-sales"},
			pm.ToolNameAnalyzeFeatureGaps:    {"customer-feedback"},
		},
		MockMode: &pmconfig.MockModeConfig{
			Enabled:   true,
			Directory: fallbackPath,
		},
	}

	// Marshal PM config to JSON
	pmConfigJSON, _ := json.Marshal(pmConfigData)

	// Marshal shared config with data sources
	sharedCfg := sharedconfig.Config{
		DataSources: commonDataSourcesConfig,
	}
	sharedConfigJSON, _ := json.Marshal(sharedCfg)

	// Create test config container with PM tools configuration
	testConfig := &config.Config{
		RoleConfigs: map[string]json.RawMessage{
			"pm":     pmConfigJSON,
			"shared": sharedConfigJSON,
		},
	}
	configContainer := &config.Container{}
	configContainer.Update(testConfig)

	// Create tool provider using NewMMToolProvider
	provider := NewMMToolProvider(nil, nil, &http.Client{}, configContainer, nil)

	// Register PM provider manually (mimics server/main.go logic)
	RegisterRoleProviderForTest(provider, configContainer, "pm")

	// Verify PM role provider was registered
	require.NotEmpty(t, provider.roleProviders, "PM role provider should be registered when PMTools.Enabled=true and mockLoader is provided")

	pmProvider := getPMProvider(t, provider)

	t.Run("CompileMarketResearch with Hub channel mock", func(t *testing.T) {
		// Create mock LLM context
		llmContext := &llm.Context{
			RequestingUser: &model.User{Id: "user123"},
		}

		// Create argument getter
		argsGetter := func(args interface{}) error {
			if mrArgs, ok := args.(*pm.CompileMarketResearchArgs); ok {
				mrArgs.PrimaryFeatures = []string{"enterprise"}
				mrArgs.ResearchIntent = "competitive_analysis"
				mrArgs.Context = "pricing"
				mrArgs.TimeRange = "month"
				return nil
			}
			return nil
		}

		// Execute the tool via PM provider - should return mock Hub channel content
		result, err := pmProvider.ToolCompileMarketResearch(llmContext, argsGetter)

		require.NoError(t, err)
		require.Contains(t, result, "Market Research: enterprise pricing")

		// Verify it contains actual Hub channel content from fallback data
		require.Contains(t, result, "search_mattermost_hub")
		require.Contains(t, result, "Segments:")
		// Check for actual company/contact names from the fallback data
		require.True(t, strings.Contains(result, "Lockheed Martin") || strings.Contains(result, "TurnOut") || strings.Contains(result, "Extend"), "Should contain at least one company from Hub data")

		t.Logf("Market research with Hub mock data: %s", result)
	})

	t.Run("AnalyzeFeatureGaps with Hub channel mock", func(t *testing.T) {
		// Create mock LLM context
		llmContext := &llm.Context{
			RequestingUser: &model.User{Id: "user123"},
		}

		// Create argument getter
		argsGetter := func(args interface{}) error {
			if gapArgs, ok := args.(*pm.AnalyzeFeatureGapsArgs); ok {
				gapArgs.PrimaryFeatures = []string{"search"}
				gapArgs.GapAnalysisType = "user_request_gaps"
				gapArgs.Context = "functionality"
				gapArgs.TimeRange = "month"
				return nil
			}
			return nil
		}

		// Execute the tool via PM provider - should return mock Hub channel content
		result, err := pmProvider.ToolAnalyzeFeatureGaps(llmContext, argsGetter)

		require.NoError(t, err)
		require.Contains(t, result, "Feature Gap Analysis: search functionality")

		// Verify it contains actual Hub channel content from fallback data
		require.Contains(t, result, "search_mattermost_hub")
		require.Contains(t, result, "Segments:")
		// Check for actual CSM names or companies from the Hub customer feedback data
		require.True(t, strings.Contains(result, "Biophore") || strings.Contains(result, "ENTES") || strings.Contains(result, "Marek Health"), "Should contain at least one customer/company from Hub feedback data")

		t.Logf("Feature gap analysis with Hub mock data: %s", result)
	})
}

func TestPMToolsGracefulDegradation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test graceful degradation with invalid/non-existent sources
	invalidConfig := &datasources.Config{
		Sources: []datasources.SourceConfig{
			{
				Name:     "invalid_source",
				Enabled:  true,
				Protocol: datasources.HTTPProtocolType,
				Endpoints: map[string]string{
					datasources.EndpointBaseURL: "https://invalid-domain-that-does-not-exist.com",
				},
				Auth:           datasources.AuthConfig{Type: datasources.AuthTypeNone},
				Sections:       []string{"test"},
				MaxDocsPerCall: 1,
				RateLimit: datasources.RateLimitConfig{
					RequestsPerMinute: 1,
					BurstSize:         1,
					Enabled:           true,
				},
			},
		},
		AllowedDomains: []string{"invalid-domain-that-does-not-exist.com"},
		CacheTTL:       datasources.DefaultCacheTTL,
	}

	invalidClient := datasources.NewClient(invalidConfig, nil)
	defer invalidClient.Close()

	// Test that invalid sources fail gracefully
	docs, err := invalidClient.FetchFromSource(context.Background(), "invalid_source", "test", 1)

	// The HTTP client might succeed but return empty docs, or might fail with network error
	// Both behaviors are acceptable for graceful degradation
	if err != nil {
		t.Logf("Expected error for invalid source: %v", err)
		assert.Nil(t, docs)
	} else {
		t.Logf("Invalid source returned %d docs (graceful degradation)", len(docs))
		// Even if no error, should not crash the application
	}

	// Test with non-existent source name
	docs2, err2 := invalidClient.FetchFromSource(context.Background(), "non_existent_source", "test", 1)
	assert.Error(t, err2)
	assert.Nil(t, docs2)
	t.Logf("Expected error for non-existent source: %v", err2)
}
