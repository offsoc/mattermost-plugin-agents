// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"
)

func TestWebsocketFeatureDetection(t *testing.T) {
	analyzer := NewTopicAnalyzer()

	testCases := []struct {
		name     string
		topic    string
		expected []string
	}{
		{
			name:     "websocket should be detected not web",
			topic:    "How does Mattermost handle websocket connections?",
			expected: []string{"websockets"}, // Should detect websockets, NOT web
		},
		{
			name:     "websockets plural",
			topic:    "Explain websockets in Mattermost",
			expected: []string{"websockets"},
		},
		{
			name:     "web should still work",
			topic:    "How does the web interface work?",
			expected: []string{"web"},
		},
		{
			name:     "webapp should work",
			topic:    "Debug the webapp",
			expected: []string{"web"},
		},
		{
			name:     "both websocket and web mentioned",
			topic:    "How do websockets work in the web client?",
			expected: []string{"websockets", "web"}, // Both should be detected
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			features := analyzer.GetMattermostFeatures(tc.topic)

			// Check if we got the expected features
			if len(features) != len(tc.expected) {
				t.Errorf("Expected %d features, got %d: %v", len(tc.expected), len(features), features)
				return
			}

			// Build a map for easier checking
			featureMap := make(map[string]bool)
			for _, f := range features {
				featureMap[f] = true
			}

			for _, exp := range tc.expected {
				if !featureMap[exp] {
					t.Errorf("Expected feature '%s' not found in results: %v", exp, features)
				}
			}
		})
	}
}
