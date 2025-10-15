// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"net/http"
	"os"
	"testing"
)

func TestMattermostProtocol_GetType(t *testing.T) {
	protocol := NewMattermostProtocol(&http.Client{}, nil, "")

	if protocol.GetType() != MattermostProtocolType {
		t.Errorf("Expected protocol type %s, got %s", MattermostProtocolType, protocol.GetType())
	}
}

func TestMattermostProtocol_SetAuth(t *testing.T) {
	protocol := NewMattermostProtocol(&http.Client{}, nil, "")

	auth := AuthConfig{Type: AuthTypeNone}
	protocol.SetAuth(auth)
	if protocol.apiClient.auth.Type != AuthTypeNone {
		t.Errorf("Expected auth type %s, got %s", AuthTypeNone, protocol.apiClient.auth.Type)
	}

	auth = AuthConfig{Type: AuthTypeToken, Key: "test-token"}
	protocol.SetAuth(auth)
	if protocol.apiClient.auth.Type != AuthTypeToken {
		t.Errorf("Expected auth type %s, got %s", AuthTypeToken, protocol.apiClient.auth.Type)
	}
	if protocol.apiClient.auth.Key != "test-token" {
		t.Errorf("Expected auth key 'test-token', got '%s'", protocol.apiClient.auth.Key)
	}
}

func TestMattermostProtocol_MapSectionToChannel(t *testing.T) {
	protocol := NewMattermostProtocol(&http.Client{}, nil, "")

	testCases := []struct {
		section  string
		expected string
	}{
		{SectionFeatureRequests, "ai-exchange"},
		{SectionTroubleshooting, "bugs"},
		{SectionGeneral, "ask-anything"},
		{SectionAPI, "developers"},
		{SectionMobile, "mobile"},
		{"unknown", "unknown"},
	}

	for _, tc := range testCases {
		result := protocol.mapSectionToChannel(tc.section)
		if result != tc.expected {
			t.Errorf("Expected section %s to map to channel %s, got %s", tc.section, tc.expected, result)
		}
	}
}

func TestMattermostProtocol_FormatTimestamp(t *testing.T) {
	transformer := NewMattermostTransformer()

	// Test zero timestamp
	result := transformer.FormatTimestamp(0)
	if result != "Unknown" {
		t.Errorf("Expected 'Unknown' for zero timestamp, got '%s'", result)
	}

	// Test valid timestamp (Mattermost uses milliseconds)
	timestamp := int64(1640995200000) // 2022-01-01 00:00:00 UTC
	result = transformer.FormatTimestamp(timestamp)
	// The exact time may vary due to timezone, just check format
	if len(result) != 16 || result[4] != '-' || result[7] != '-' || result[10] != ' ' || result[13] != ':' {
		t.Errorf("Expected timestamp format 'YYYY-MM-DD HH:MM', got '%s'", result)
	}
}

func TestMattermostProtocol_GeneratePostTitle(t *testing.T) {
	transformer := NewMattermostTransformer()

	testCases := []struct {
		message  string
		expected string
	}{
		{"", "Mattermost Post"},
		{"Short message", "Short message"},
		{"This is a very long message that exceeds fifty characters and should be truncated", "This is a very long message that exceeds fifty cha..."},
		{"Multi-line\nmessage\nwith\nbreaks", "Multi-line"},
	}

	for _, tc := range testCases {
		post := MattermostPost{Message: tc.message}
		result := transformer.GeneratePostTitle(post)
		if result != tc.expected {
			t.Errorf("Expected title '%s', got '%s'", tc.expected, result)
		}
	}
}

// TestMattermostProtocol_HubFallbackBooleanQueries tests Hub fallback data with complex boolean search queries
func TestMattermostProtocol_HubFallbackBooleanQueries(t *testing.T) {
	// Check if fallback files have data
	contactSalesFile := FallbackDataDirectory + "/" + HubContactSalesChannelData
	if info, err := os.Stat(contactSalesFile); err != nil || info.Size() == 0 {
		t.Skip("Hub fallback data files not available or empty (data source disabled)")
	}

	protocol := NewMattermostProtocol(&http.Client{}, nil, FallbackDataDirectory)

	source := SourceConfig{
		Name:     SourceMattermostHub,
		Protocol: MattermostProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: MattermostHubURL,
		},
		Auth:     AuthConfig{Type: AuthTypeNone}, // Use fallback files
		Sections: []string{SectionContactSales, SectionCustomerFeedback},
	}

	setupFunc := func() error {
		t.Log("Testing Mattermost Hub with fallback files - no authentication required")
		return nil
	}

	VerifyProtocolHubFallbackBooleanQuery(t, protocol, source, setupFunc)
}

// TestMattermostProtocol_HubFallbackOnAuthFailure tests that Hub falls back to mock data when auth is configured but fails
func TestMattermostProtocol_HubFallbackOnAuthFailure(t *testing.T) {
	// Check if fallback files have data
	contactSalesFile := FallbackDataDirectory + "/" + HubContactSalesChannelData
	if info, err := os.Stat(contactSalesFile); err != nil || info.Size() == 0 {
		t.Skip("Hub fallback data files not available or empty (data source disabled)")
	}

	protocol := NewMattermostProtocol(&http.Client{}, nil, FallbackDataDirectory)

	source := SourceConfig{
		Name:     SourceMattermostHub,
		Protocol: MattermostProtocolType,
		Endpoints: map[string]string{
			EndpointBaseURL: MattermostHubURL,
		},
		Auth:     AuthConfig{Type: AuthTypeToken, Key: "invalid-token"}, // Invalid auth should trigger fallback
		Sections: []string{SectionContactSales, SectionCustomerFeedback},
	}

	setupFunc := func() error {
		t.Log("Testing Mattermost Hub with invalid auth - should fallback to mock data")
		return nil
	}

	VerifyProtocolHubFallbackBooleanQuery(t, protocol, source, setupFunc)
}
