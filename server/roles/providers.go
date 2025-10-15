// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package roles

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-ai/config"
	devconfig "github.com/mattermost/mattermost-plugin-ai/config/dev"
	pmconfig "github.com/mattermost/mattermost-plugin-ai/config/pm"
	sharedconfig "github.com/mattermost/mattermost-plugin-ai/config/shared"
	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/mmtools"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/dev"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/pm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/shared"
	"github.com/mattermost/mattermost-plugin-ai/search"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// RegisterToolProviders registers role-specific tool providers (PM, Dev)
// This keeps role-specific logic isolated from core mmtools package
func RegisterToolProviders(
	pluginAPI *pluginapi.Client,
	toolProvider *mmtools.MMToolProvider,
	searchService *search.Search,
	httpClient *http.Client,
	mmClient mmapi.Client,
	configContainer *config.Container,
) {
	// Load shared datasources configuration from RoleConfigs
	var sharedCfg sharedconfig.Config
	var dataSourcesClient *datasources.Client
	if err := configContainer.GetRoleConfig("shared", &sharedCfg); err == nil && sharedCfg.DataSources != nil {
		if sharedCfg.DataSources.IsEnabled() {
			// Apply environment variable fallbacks
			datasources.ApplyEnvironmentFallbacks(sharedCfg.DataSources)
			dataSourcesClient = datasources.NewClient(sharedCfg.DataSources, mmClient)
		}
	}

	// Register PM role provider if enabled
	var pmCfg pmconfig.Config
	if err := configContainer.GetRoleConfig("pm", &pmCfg); err == nil && pmCfg.Enabled {
		// Create mock loader if configured
		var mockLoader shared.MockLoader
		if pmCfg.MockMode != nil && pmCfg.MockMode.Enabled {
			mockLoader = shared.NewMockLoader(pmCfg.MockMode.Enabled, pmCfg.MockMode.Directory)
		} else {
			mockLoader = shared.NewMockLoader(false, "")
		}

		// Check if we have any data sources available
		if (mockLoader != nil && mockLoader.IsEnabled()) || dataSourcesClient != nil {
			pmService := pm.NewService(
				mmClient,
				searchService,
				mockLoader,
				dataSourcesClient,
				toolProvider.GetVectorCache(),
				toolProvider,
			)
			pmProvider := pm.NewProvider(httpClient, pmService, toolProvider)
			toolProvider.RegisterRole(pmProvider)

			pluginAPI.Log.Info("PM role tools registered")
		} else {
			pluginAPI.Log.Warn("PM role enabled but no data sources available")
		}
	}

	// Register Dev role provider if enabled
	var devCfg devconfig.Config
	if err := configContainer.GetRoleConfig("dev", &devCfg); err == nil && devCfg.Enabled {
		// Dev role requires external documentation sources
		if dataSourcesClient != nil {
			// Create mock loader (shared with PM if configured)
			var mockLoader shared.MockLoader
			if devCfg.MockMode != nil && devCfg.MockMode.Enabled {
				mockLoader = shared.NewMockLoader(devCfg.MockMode.Enabled, devCfg.MockMode.Directory)
			} else {
				mockLoader = shared.NewMockLoader(false, "")
			}

			devService := dev.NewService(
				mmClient,
				searchService,
				mockLoader,
				dataSourcesClient,
				toolProvider.GetVectorCache(),
				toolProvider,
			)
			devProvider := dev.NewProvider(httpClient, devService, toolProvider)
			toolProvider.RegisterRole(devProvider)

			pluginAPI.Log.Info("Dev role tools registered")
		} else {
			pluginAPI.Log.Warn("Dev role enabled but no data sources available")
		}
	}
}
