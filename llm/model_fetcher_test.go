// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package llm

import (
	"net/http"
	"testing"
)

func TestNewModelFetcher(t *testing.T) {
	tests := []struct {
		name        string
		serviceType string
		apiKey      string
		apiURL      string
		shouldBeNil bool
	}{
		{
			name:        "Anthropic service should return fetcher",
			serviceType: "anthropic",
			apiKey:      "test-key",
			apiURL:      "",
			shouldBeNil: false,
		},
		{
			name:        "OpenAI service should return nil (not yet implemented)",
			serviceType: "openai",
			apiKey:      "test-key",
			apiURL:      "",
			shouldBeNil: true,
		},
		{
			name:        "Unknown service should return nil",
			serviceType: "unknown",
			apiKey:      "test-key",
			apiURL:      "",
			shouldBeNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{}
			fetcher := NewModelFetcher(tt.serviceType, tt.apiKey, tt.apiURL, httpClient)

			if tt.shouldBeNil && fetcher != nil {
				t.Errorf("Expected nil fetcher for service type %s, but got non-nil", tt.serviceType)
			}

			if !tt.shouldBeNil && fetcher == nil {
				t.Errorf("Expected non-nil fetcher for service type %s, but got nil", tt.serviceType)
			}
		})
	}
}
