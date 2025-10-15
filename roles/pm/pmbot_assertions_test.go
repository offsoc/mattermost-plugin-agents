// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pm_test

import (
	"fmt"
	"sync"
)

// Helper functions for extracting typed fields from log field maps
func getStringField(fields map[string]interface{}, key string) string {
	if val, exists := fields[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntField(fields map[string]interface{}, key string) int {
	if val, exists := fields[key]; exists {
		if i, ok := val.(int); ok {
			return i
		}
		if i64, ok := val.(int64); ok {
			return int(i64)
		}
	}
	return 0
}

// Topic tracking for cleaner logging
var (
	topicRegistry = make(map[string]string) // topic -> short ID
	topicLogged   = make(map[string]bool)   // track which topics have been logged
	topicCounter  = 0
	topicMutex    sync.Mutex // protect concurrent access to maps
)

func getTopicID(topic string) string {
	topicMutex.Lock()
	defer topicMutex.Unlock()

	if id, exists := topicRegistry[topic]; exists {
		return id
	}
	topicCounter++
	id := fmt.Sprintf("T%d", topicCounter)
	topicRegistry[topic] = id
	return id
}

func getTopicDisplayWithDescription(topic string, t interface{ Logf(string, ...interface{}) }) string {
	topicID := getTopicID(topic)

	topicMutex.Lock()
	alreadyLogged := topicLogged[topicID]
	if !alreadyLogged {
		topicLogged[topicID] = true
	}
	topicMutex.Unlock()

	if !alreadyLogged {
		t.Logf("TOPIC MAPPING: %s = %s", topicID, topic)
	}

	return topicID
}
