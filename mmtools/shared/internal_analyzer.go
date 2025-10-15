// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/embeddings"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mmapi"
	"github.com/mattermost/mattermost-plugin-ai/search"
)

const DefaultSearchLimit = 5

// InternalAnalyzer provides RAG search for internal Mattermost conversations
// Shared across all role implementations (PM, Dev, SRE, etc.)
type InternalAnalyzer struct {
	search     *search.Search
	mockLoader MockLoader
	pluginAPI  mmapi.Client
}

// NewInternalAnalyzer creates a new internal analyzer
func NewInternalAnalyzer(searchService *search.Search, mockLoader MockLoader, pluginAPI mmapi.Client) *InternalAnalyzer {
	return &InternalAnalyzer{
		search:     searchService,
		mockLoader: mockLoader,
		pluginAPI:  pluginAPI,
	}
}

// Search performs RAG search with automatic mock fallback
// Returns formatted results as "From {channel}: {content}" or nil if no results
func (a *InternalAnalyzer) Search(
	llmContext *llm.Context,
	query string,
	mockKey string,
) []string {
	return a.SearchWithOptions(llmContext, query, mockKey, DefaultSearchLimit, false)
}

// SearchWithLogging performs RAG search with error logging
func (a *InternalAnalyzer) SearchWithLogging(
	llmContext *llm.Context,
	query string,
	mockKey string,
	logMessage string,
) []string {
	results := a.SearchWithOptions(llmContext, query, mockKey, DefaultSearchLimit, true)
	if results == nil && a.pluginAPI != nil {
		a.pluginAPI.LogError(logMessage, "query", query)
	}
	return results
}

// SearchWithOptions performs RAG search with custom options
func (a *InternalAnalyzer) SearchWithOptions(
	llmContext *llm.Context,
	query string,
	mockKey string,
	limit int,
	logErrors bool,
) []string {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}

	if a.search != nil && a.search.Enabled() && query != "" {
		ragResults, err := a.search.SearchWithMetadata(context.Background(), query, embeddings.SearchOptions{
			Limit:  limit,
			UserID: llmContext.RequestingUser.Id,
		})

		if err != nil {
			if logErrors && a.pluginAPI != nil {
				a.pluginAPI.LogError("internal search failed", "query", query, "error", err.Error())
			}
		} else if len(ragResults) > 0 {
			insights := make([]string, 0, len(ragResults))
			for _, result := range ragResults {
				insights = append(insights, fmt.Sprintf("From %s: %s", result.ChannelName, result.Content))
			}
			return insights
		}
	}

	if a.mockLoader != nil && a.mockLoader.IsEnabled() {
		if mockResponse, found := a.mockLoader.LoadMockResponse(mockKey); found {
			return []string{mockResponse}
		}
	}

	return nil
}

// FormatInternalInsights formats a list of insights into markdown
func FormatInternalInsights(insights []string) string {
	if len(insights) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString("## Internal Team Discussions\n\n")
	for _, insight := range insights {
		result.WriteString(fmt.Sprintf("- %s\n", insight))
	}
	result.WriteString("\n")

	return result.String()
}
