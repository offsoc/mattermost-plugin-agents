// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/semanticcache"
)

type CacheHelper struct {
	vectorCache *semanticcache.SimpleCache
	pluginAPI   mmapi.Client
}

func NewCacheHelper(vectorCache *semanticcache.SimpleCache, pluginAPI mmapi.Client) *CacheHelper {
	return &CacheHelper{
		vectorCache: vectorCache,
		pluginAPI:   pluginAPI,
	}
}

func (h *CacheHelper) ExecuteWithCache(
	toolName string,
	args interface{},
	fn func() (string, error),
) (string, error) {
	cacheKeyJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cache key: %w", err)
	}
	cacheKey := string(cacheKeyJSON)

	if h.vectorCache != nil {
		if cachedResponse, hit := h.vectorCache.Lookup(toolName, cacheKey); hit {
			if h.pluginAPI != nil {
				h.pluginAPI.LogDebug("semantic cache HIT - returning cached response",
					"tool", toolName,
					"cache_key", cacheKey)
			}
			return cachedResponse, nil
		}
		if h.pluginAPI != nil {
			h.pluginAPI.LogDebug("semantic cache MISS - executing full search",
				"tool", toolName,
				"cache_key", cacheKey)
		}
	}

	result, err := fn()
	if err != nil {
		return "", err
	}

	if h.vectorCache != nil {
		h.vectorCache.Store(toolName, cacheKey, result)
		if h.pluginAPI != nil {
			h.pluginAPI.LogDebug("semantic cache STORE - caching new response",
				"tool", toolName,
				"cache_key", cacheKey,
				"result_length", len(result))
		}
	}

	return result, nil
}
