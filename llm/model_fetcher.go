// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"context"
	"net/http"
)

// ModelInfo represents information about an available model
type ModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// ModelFetcher is an interface for fetching available models from different LLM providers
type ModelFetcher interface {
	// FetchModels retrieves the list of available models for the service
	FetchModels(ctx context.Context) ([]ModelInfo, error)
}

// NewModelFetcher creates a ModelFetcher for a given service configuration
func NewModelFetcher(serviceType string, apiKey string, apiURL string, httpClient *http.Client) ModelFetcher {
	switch serviceType {
	case "anthropic":
		return NewAnthropicModelFetcher(apiKey, httpClient)
	default:
		return nil
	}
}
