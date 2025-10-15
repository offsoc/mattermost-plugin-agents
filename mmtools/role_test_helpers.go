// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/config"
	devconfig "github.com/mattermost/mattermost-plugin-ai/config/dev"
	pmconfig "github.com/mattermost/mattermost-plugin-ai/config/pm"
	sharedconfig "github.com/mattermost/mattermost-plugin-ai/config/shared"
	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/dev"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/pm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
	"github.com/stretchr/testify/require"
)

// EnableRoleBotForTest enables a specific role bot for testing if not already enabled.
// It automatically registers a cleanup function with t.Cleanup() to restore
// the original environment variable state.
//
// Example usage:
//
//	EnableRoleBotForTest(t, "ENABLE_PM_BOT")
//	EnableRoleBotForTest(t, "ENABLE_DEVBOT")
func EnableRoleBotForTest(t *testing.T, envVar string) {
	t.Helper()

	// Check if already set
	originalValue, wasSet := os.LookupEnv(envVar)

	// Set it to true for the test
	os.Setenv(envVar, "true")

	// Register cleanup function with t.Cleanup()
	t.Cleanup(func() {
		if wasSet {
			os.Setenv(envVar, originalValue)
		} else {
			os.Unsetenv(envVar)
		}
	})
}

// EnableRoleBotForTestWithCleanup enables a specific role bot for testing if not already enabled.
// It returns a cleanup function that should be called with defer to restore
// the original environment variable state. Use this when you need explicit control
// over when cleanup happens.
//
// Example usage:
//
//	defer EnableRoleBotForTestWithCleanup(t, "ENABLE_PM_BOT")()
func EnableRoleBotForTestWithCleanup(t *testing.T, envVar string) func() {
	t.Helper()

	// Check if already set
	originalValue, wasSet := os.LookupEnv(envVar)

	// Set it to true for the test
	os.Setenv(envVar, "true")

	// Return cleanup function
	return func() {
		if wasSet {
			os.Setenv(envVar, originalValue)
		} else {
			os.Unsetenv(envVar)
		}
	}
}

// EnableDevBotForTest is a convenience wrapper for enabling the dev bot.
// Ready for future implementation when DevBot is added.
func EnableDevBotForTest(t *testing.T) {
	EnableRoleBotForTest(t, "ENABLE_DEVBOT")
}

// RegisterRoleProviderForTest registers a role provider (PM or Dev) on the given MMToolProvider
// This mimics the registration logic in server/main.go for testing purposes
//
// IMPORTANT: This MUST be called after creating an MMToolProvider in tests to ensure
// the role-specific tools and datasources are properly initialized.
//
// Example usage:
//
//	toolProvider := mmtools.NewMMToolProvider(mmClient, nil, &http.Client{}, configContainer, nil)
//	mmtools.RegisterRoleProviderForTest(toolProvider, configContainer, "pm")   // For PM bot tests
//	mmtools.RegisterRoleProviderForTest(toolProvider, configContainer, "dev")  // For Dev bot tests
func RegisterRoleProviderForTest(provider *MMToolProvider, configContainer *config.Container, roleName string) {
	switch roleName {
	case "pm":
		var pmCfg pmconfig.Config
		if err := configContainer.GetRoleConfig("pm", &pmCfg); err != nil || !pmCfg.Enabled {
			return
		}

		var sharedCfg sharedconfig.Config
		_ = configContainer.GetRoleConfig("shared", &sharedCfg)

		// Create mock loader if configured
		var mockLoader shared.MockLoader
		switch {
		case pmCfg.MockMode != nil && pmCfg.MockMode.Enabled:
			mockLoader = shared.NewMockLoader(pmCfg.MockMode.Enabled, pmCfg.MockMode.Directory)
		case sharedCfg.DataSources != nil && sharedCfg.DataSources.FallbackDirectory != "":
			mockLoader = shared.NewMockLoader(true, sharedCfg.DataSources.FallbackDirectory)
		default:
			mockLoader = shared.NewMockLoader(false, "")
		}

		// Create datasources client if configured
		var dataSourcesClient *datasources.Client
		if sharedCfg.DataSources != nil && sharedCfg.DataSources.IsEnabled() {
			dataSourcesClient = datasources.NewClient(sharedCfg.DataSources, nil)
		}

		// Check if we have any data sources available (mimics server/main.go logic)
		if (mockLoader != nil && mockLoader.IsEnabled()) || dataSourcesClient != nil {
			// Create PM service
			pmService := pm.NewService(
				nil, // mmClient
				nil, // searchService
				mockLoader,
				dataSourcesClient,
				provider.GetVectorCache(),
				provider,
			)

			// Create and register PM provider
			pmProvider := pm.NewProvider(nil, pmService, provider)
			provider.RegisterRole(pmProvider)
		}

	case "dev":
		var devCfg devconfig.Config
		if err := configContainer.GetRoleConfig("dev", &devCfg); err != nil || !devCfg.Enabled {
			return
		}

		var sharedCfg sharedconfig.Config
		_ = configContainer.GetRoleConfig("shared", &sharedCfg)

		// Create mock loader if configured
		var mockLoader shared.MockLoader
		switch {
		case devCfg.MockMode != nil && devCfg.MockMode.Enabled:
			mockLoader = shared.NewMockLoader(devCfg.MockMode.Enabled, devCfg.MockMode.Directory)
		case sharedCfg.DataSources != nil && sharedCfg.DataSources.FallbackDirectory != "":
			mockLoader = shared.NewMockLoader(true, sharedCfg.DataSources.FallbackDirectory)
		default:
			mockLoader = shared.NewMockLoader(false, "")
		}

		// Create datasources client if configured
		var dataSourcesClient *datasources.Client
		if sharedCfg.DataSources != nil && sharedCfg.DataSources.IsEnabled() {
			dataSourcesClient = datasources.NewClient(sharedCfg.DataSources, nil)
		}

		// Check if we have any data sources available (mimics server/main.go logic)
		if (mockLoader != nil && mockLoader.IsEnabled()) || dataSourcesClient != nil {
			// Create Dev service
			devService := dev.NewService(
				nil, // mmClient
				nil, // searchService
				mockLoader,
				dataSourcesClient,
				provider.GetVectorCache(),
				provider,
			)

			// Create and register Dev provider
			devProvider := dev.NewProvider(nil, devService, provider)
			provider.RegisterRole(devProvider)
		}
	}
}

// createTestProviderForPM creates a minimal MMToolProvider configured for PM tools testing
func createTestProviderForPM(t *testing.T, db *sql.DB, enableCache bool) *MMToolProvider {
	if enableCache {
		t.Setenv("VECTOR_CACHE_ENABLED", "1")
		t.Setenv("VECTOR_CACHE_THRESHOLD", "0.85")
	} else {
		t.Setenv("VECTOR_CACHE_ENABLED", "0")
	}

	// Create PM config
	pmConfigData := pmconfig.Config{
		Enabled: true, // Enable PM tools for testing
		ChannelTargets: map[string][]string{
			pm.ToolNameCompileMarketResearch: {"contact-sales"},
			pm.ToolNameAnalyzeFeatureGaps:    {"customer-feedback"},
		},
	}

	// Marshal PM config to JSON
	pmConfigJSON, _ := json.Marshal(pmConfigData)

	// Create shared config with datasources
	sharedCfg := sharedconfig.Config{
		DataSources: &datasources.Config{
			Sources: []datasources.SourceConfig{
				{
					Name:           datasources.SourceMattermostHub,
					Enabled:        true,
					Protocol:       datasources.MattermostProtocolType,
					Endpoints:      map[string]string{datasources.EndpointBaseURL: datasources.MattermostHubURL},
					Auth:           datasources.AuthConfig{Type: datasources.AuthTypeToken},
					Sections:       []string{datasources.SectionContactSales, datasources.SectionCustomerFeedback},
					MaxDocsPerCall: 5,
					RateLimit: datasources.RateLimitConfig{
						RequestsPerMinute: 60,
						BurstSize:         10,
						Enabled:           true,
					},
				},
			},
			FallbackDirectory: datasources.FallbackDataDirectory,
		},
	}
	sharedConfigJSON, _ := json.Marshal(sharedCfg)

	testConfig := &config.Config{
		RoleConfigs: map[string]json.RawMessage{
			"pm":     pmConfigJSON,
			"shared": sharedConfigJSON,
		},
	}

	configContainer := &config.Container{}
	configContainer.Update(testConfig)

	// Use NewMMToolProvider - now returns provider without PM/Dev pre-registered
	provider := NewMMToolProvider(nil, nil, nil, configContainer, db)

	// Register PM provider using shared helper
	RegisterRoleProviderForTest(provider, configContainer, "pm")

	return provider
}

// getPMProvider extracts the PM provider from roleProviders
func getPMProvider(t *testing.T, provider *MMToolProvider) *pm.Provider {
	t.Helper()
	for _, rp := range provider.roleProviders {
		if p, ok := rp.(*pm.Provider); ok {
			return p
		}
	}
	require.FailNow(t, "PM provider should be registered")
	return nil
}
