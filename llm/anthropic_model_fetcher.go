// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"context"
	"net/http"

	anthropicSDK "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicModelFetcher struct {
	client anthropicSDK.Client
}

func NewAnthropicModelFetcher(apiKey string, httpClient *http.Client) *AnthropicModelFetcher {
	client := anthropicSDK.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(httpClient),
	)

	return &AnthropicModelFetcher{
		client: client,
	}
}

func (a *AnthropicModelFetcher) FetchModels(ctx context.Context) ([]ModelInfo, error) {
	// Use AutoPaging to automatically handle pagination
	autoPager := a.client.Models.ListAutoPaging(ctx, anthropicSDK.ModelListParams{})

	var models []ModelInfo

	// Iterate through all pages
	for autoPager.Next() {
		model := autoPager.Current()
		models = append(models, ModelInfo{
			ID:          model.ID,
			DisplayName: model.DisplayName,
		})
	}

	// Check if there was an error during iteration
	if err := autoPager.Err(); err != nil {
		return nil, err
	}

	return models, nil
}
