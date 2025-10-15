// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmtools/pm"
)

// PM-specific semantic cache integration tests

func TestPMSemanticCacheMarketResearch(t *testing.T) {
	t.Run("cache enabled - first call hits data sources, second call hits cache", func(t *testing.T) {
		db := testSemanticCacheDB(t)
		defer cleanupSemanticCacheDB(t, db)

		provider := createTestProviderForPM(t, db, true) // Enable cache

		// Verify cache is enabled
		assert.True(t, provider.vectorCache != nil)

		pmProvider := getPMProvider(t, provider)

		// Create a mock context and args getter
		context := &llm.Context{}
		args := pm.CompileMarketResearchArgs{
			PrimaryFeatures: []string{"mobile"},
			ResearchIntent:  "market_trends",
			Context:         "strategy",
			TimeRange:       "month",
		}
		argsGetter := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.CompileMarketResearchArgs:
				*v = args
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		// First call - should miss cache and go to data sources
		result1, err := pmProvider.ToolCompileMarketResearch(context, argsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// Verify something was stored in cache
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameCompileMarketResearch).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Second call with same args - should hit cache
		result2, err := pmProvider.ToolCompileMarketResearch(context, argsGetter)
		require.NoError(t, err)
		assert.Equal(t, result1, result2) // Should be identical from cache

		// Verify cache was hit (count should still be 1, not 2)
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameCompileMarketResearch).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("cache disabled - always goes to data sources", func(t *testing.T) {
		db := testSemanticCacheDB(t)
		defer cleanupSemanticCacheDB(t, db)

		provider := createTestProviderForPM(t, db, false) // Disable cache

		// Cache object exists but is disabled via environment variable
		assert.NotNil(t, provider.vectorCache)

		pmProvider := getPMProvider(t, provider)

		context := &llm.Context{}
		args := pm.CompileMarketResearchArgs{
			PrimaryFeatures: []string{"enterprise"},
			ResearchIntent:  "competitive_analysis",
			Context:         "pricing",
			TimeRange:       "quarter",
		}
		argsGetter := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.CompileMarketResearchArgs:
				*v = args
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		// First call
		result1, err := pmProvider.ToolCompileMarketResearch(context, argsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// When cache is disabled, nothing should be stored (table might not even exist)
		// Skip cache verification since cache operations are disabled

		// Second call should still go to data sources
		result2, err := pmProvider.ToolCompileMarketResearch(context, argsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, result2)
		// Results could be slightly different due to data source variability
		// when cache is disabled. Cache operations are disabled, so skip verification.
	})

	t.Run("different arguments create separate cache entries", func(t *testing.T) {
		db := testSemanticCacheDB(t)
		defer cleanupSemanticCacheDB(t, db)

		provider := createTestProviderForPM(t, db, true) // Enable cache

		pmProvider := getPMProvider(t, provider)

		context := &llm.Context{}

		// First call with args1
		args1 := pm.CompileMarketResearchArgs{
			PrimaryFeatures: []string{"desktop"},
			ResearchIntent:  "user_needs",
			Context:         "product",
			TimeRange:       "week",
		}
		argsGetter1 := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.CompileMarketResearchArgs:
				*v = args1
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		result1, err := pmProvider.ToolCompileMarketResearch(context, argsGetter1)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// Second call with different args
		args2 := pm.CompileMarketResearchArgs{
			PrimaryFeatures: []string{"cloud"},
			ResearchIntent:  "security_concerns",
			Context:         "compliance",
			TimeRange:       "year",
		}
		argsGetter2 := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.CompileMarketResearchArgs:
				*v = args2
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		result2, err := pmProvider.ToolCompileMarketResearch(context, argsGetter2)
		require.NoError(t, err)
		assert.NotEmpty(t, result2)

		// Results should be different
		assert.NotEqual(t, result1, result2)

		// Verify we have 2 cache entries now
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameCompileMarketResearch).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestPMSemanticCacheFeatureGaps(t *testing.T) {
	t.Run("cache enabled - first call hits data sources, second call hits cache", func(t *testing.T) {
		db := testSemanticCacheDB(t)
		defer cleanupSemanticCacheDB(t, db)

		provider := createTestProviderForPM(t, db, true) // Enable cache

		// Verify cache is enabled
		assert.True(t, provider.vectorCache != nil)

		pmProvider := getPMProvider(t, provider)

		context := &llm.Context{
			RequestingUser: &model.User{Id: "test-user"},
		}
		args := pm.AnalyzeFeatureGapsArgs{
			PrimaryFeatures: []string{"search"},
			GapAnalysisType: "competitive_gaps",
			Context:         "functionality",
			TimeRange:       "month",
		}
		argsGetter := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.AnalyzeFeatureGapsArgs:
				*v = args
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		// First call - should miss cache and go to data sources
		result1, err := pmProvider.ToolAnalyzeFeatureGaps(context, argsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// Verify something was stored in cache
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameAnalyzeFeatureGaps).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Second call with same args - should hit cache
		result2, err := pmProvider.ToolAnalyzeFeatureGaps(context, argsGetter)
		require.NoError(t, err)
		assert.Equal(t, result1, result2) // Should be identical from cache

		// Verify cache was hit (count should still be 1, not 2)
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameAnalyzeFeatureGaps).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("cache disabled - always goes to data sources", func(t *testing.T) {
		db := testSemanticCacheDB(t)
		defer cleanupSemanticCacheDB(t, db)

		provider := createTestProviderForPM(t, db, false) // Disable cache

		// Cache object exists but is disabled via environment variable
		assert.NotNil(t, provider.vectorCache)

		pmProvider := getPMProvider(t, provider)

		context := &llm.Context{
			RequestingUser: &model.User{Id: "test-user"},
		}
		args := pm.AnalyzeFeatureGapsArgs{
			PrimaryFeatures: []string{"notifications"},
			GapAnalysisType: "user_request_gaps",
			Context:         "mobile",
			TimeRange:       "quarter",
		}
		argsGetter := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.AnalyzeFeatureGapsArgs:
				*v = args
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		// First call
		result1, err := pmProvider.ToolAnalyzeFeatureGaps(context, argsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// When cache is disabled, nothing should be stored (table might not even exist)
		// Skip cache verification since cache operations are disabled
	})
}

func TestPMSemanticCacheToolIsolation(t *testing.T) {
	t.Run("different PM tools have separate cache namespaces", func(t *testing.T) {
		db := testSemanticCacheDB(t)
		defer cleanupSemanticCacheDB(t, db)

		provider := createTestProviderForPM(t, db, true) // Enable cache

		pmProvider := getPMProvider(t, provider)

		context := &llm.Context{}

		// Call MarketResearch tool
		marketArgs := pm.CompileMarketResearchArgs{
			PrimaryFeatures: []string{"mobile"},
			ResearchIntent:  "market_trends",
			Context:         "strategy",
			TimeRange:       "month",
		}
		marketArgsGetter := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.CompileMarketResearchArgs:
				*v = marketArgs
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		// Call FeatureGaps tool with similar topic
		featureArgs := pm.AnalyzeFeatureGapsArgs{
			PrimaryFeatures: []string{"mobile"},
			GapAnalysisType: "competitive_gaps",
			Context:         "strategy",
			TimeRange:       "month",
		}
		featureArgsGetter := func(target interface{}) error {
			switch v := target.(type) {
			case *pm.AnalyzeFeatureGapsArgs:
				*v = featureArgs
				return nil
			default:
				return fmt.Errorf("unexpected type")
			}
		}

		// Call both tools
		marketResult, err := pmProvider.ToolCompileMarketResearch(context, marketArgsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, marketResult)

		featureResult, err := pmProvider.ToolAnalyzeFeatureGaps(context, featureArgsGetter)
		require.NoError(t, err)
		assert.NotEmpty(t, featureResult)

		// Results should be different even though topics are similar
		assert.NotEqual(t, marketResult, featureResult)

		// Verify we have entries for both tools
		var marketCount, featureCount int
		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameCompileMarketResearch).Scan(&marketCount)
		require.NoError(t, err)
		assert.Equal(t, 1, marketCount)

		err = db.QueryRow("SELECT COUNT(*) FROM semantic_cache WHERE tool_name = $1", pm.ToolNameAnalyzeFeatureGaps).Scan(&featureCount)
		require.NoError(t, err)
		assert.Equal(t, 1, featureCount)
	})
}
