// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/datasources"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
)

// MetadataProvider interface for getting tool metadata
type MetadataProvider interface {
	GetSupportedDataSources(toolName string) []string
	BuildCompositeQuery(toolName, topic, dataSource string) string
	BuildSearchQueries(toolName, topic string) map[string]string
}

// SearchMultipleSources searches external documentation using multiple specialized queries
// This is the unified implementation used by both PM and Dev services
func SearchMultipleSources(
	ctx context.Context,
	toolName string,
	topic string,
	limit int,
	metadataProvider MetadataProvider,
	commonDataSourcesClient *datasources.Client,
	pluginAPI mmapi.Client,
	logPrefix string,
) (map[string][]datasources.Doc, error) {
	if commonDataSourcesClient == nil {
		if pluginAPI != nil {
			pluginAPI.LogWarn(logPrefix + " common data sources client not available")
		}
		return nil, fmt.Errorf("common data sources client not available")
	}

	if metadataProvider == nil {
		if pluginAPI != nil {
			pluginAPI.LogWarn(logPrefix + " metadata provider not available")
		}
		return nil, fmt.Errorf("metadata provider not available")
	}

	supportedSources := metadataProvider.GetSupportedDataSources(toolName)
	if len(supportedSources) == 0 {
		if pluginAPI != nil {
			pluginAPI.LogWarn(logPrefix+" no supported sources", "tool", toolName)
		}
		return nil, fmt.Errorf("no external data sources configured for tool %s", toolName)
	}

	// Filter to only enabled sources
	enabledSources := make([]string, 0, len(supportedSources))
	disabledSources := make([]string, 0)
	for _, source := range supportedSources {
		isEnabled := commonDataSourcesClient.IsSourceEnabled(source)
		if isEnabled {
			enabledSources = append(enabledSources, source)
		} else {
			disabledSources = append(disabledSources, source)
		}
	}

	if len(enabledSources) == 0 {
		if pluginAPI != nil {
			pluginAPI.LogWarn(logPrefix+" no enabled sources",
				"supported", len(supportedSources),
				"disabled_count", len(disabledSources))
		}
		return nil, nil
	}

	if pluginAPI != nil {
		pluginAPI.LogDebug(logPrefix+" search starting",
			"tool", toolName,
			"sources", len(enabledSources),
			"limit", limit)
	}

	queries := metadataProvider.BuildSearchQueries(toolName, topic)
	if len(queries) == 0 {
		if pluginAPI != nil {
			pluginAPI.LogWarn(logPrefix + " no queries generated")
		}
		return nil, nil
	}

	if pluginAPI != nil {
		pluginAPI.LogDebug(logPrefix+" queries generated",
			"count", len(queries))
	}

	allResults := make(map[string][]datasources.Doc)
	queriesExecuted := 0
	queriesFailed := 0

	for queryType, query := range queries {
		if strings.TrimSpace(query) == "" {
			continue
		}

		queriesExecuted++
		if pluginAPI != nil {
			pluginAPI.LogDebug(logPrefix+" executing query",
				"queryType", queryType,
				"sources", len(enabledSources))
		}

		results, err := commonDataSourcesClient.FetchFromMultipleSources(ctx, enabledSources, query, limit)
		if err != nil {
			queriesFailed++
			if pluginAPI != nil {
				pluginAPI.LogWarn(logPrefix+" search failed",
					"tool", toolName,
					"queryType", queryType,
					"error", err.Error())
			}
			continue
		}

		docsCount := 0
		for _, docs := range results {
			docsCount += len(docs)
		}

		if pluginAPI != nil {
			pluginAPI.LogDebug(logPrefix+" query complete",
				"queryType", queryType,
				"sources_returned", len(results),
				"total_docs", docsCount)
		}

		// Aggregate results with queryType prefix to distinguish different aspects
		for source, docs := range results {
			key := queryType + "_" + source
			allResults[key] = docs
			if pluginAPI != nil && len(docs) > 0 {
				pluginAPI.LogDebug(logPrefix+" source results",
					"source", source,
					"queryType", queryType,
					"docs", len(docs))
			}
		}
	}

	if pluginAPI != nil {
		totalDocs := 0
		for _, docs := range allResults {
			totalDocs += len(docs)
		}
		pluginAPI.LogDebug(logPrefix+" multi-source search complete",
			"tool", toolName,
			"queries_executed", queriesExecuted,
			"queries_failed", queriesFailed,
			"result_sources", len(allResults),
			"total_docs", totalDocs)
	}

	if len(allResults) == 0 {
		if pluginAPI != nil {
			pluginAPI.LogWarn(logPrefix+" all queries returned no results",
				"tool", toolName,
				"queries_executed", queriesExecuted,
				"queries_failed", queriesFailed)
		}
		return nil, fmt.Errorf("all queries failed for tool %s", toolName)
	}

	return allResults, nil
}
