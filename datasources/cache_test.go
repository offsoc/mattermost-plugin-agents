// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"testing"
	"time"
)

func TestTTLCache_SetAndGet(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	defer cache.Close()

	cache.Set("key1", "value1")
	result := cache.Get("key1")
	if result != "value1" {
		t.Errorf("Expected 'value1', got %v", result)
	}

	result = cache.Get("nonexistent")
	if result != nil {
		t.Errorf("Expected nil for non-existent key, got %v", result)
	}
}

func TestTTLCache_Expiration(t *testing.T) {
	cache := NewTTLCache(100 * time.Millisecond)
	defer cache.Close()

	cache.Set("key1", "value1")

	result := cache.Get("key1")
	if result != "value1" {
		t.Errorf("Expected 'value1', got %v", result)
	}

	time.Sleep(150 * time.Millisecond)

	result = cache.Get("key1")
	if result != nil {
		t.Errorf("Expected nil for expired key, got %v", result)
	}
}

func TestTTLCache_Delete(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Delete("key1")

	result := cache.Get("key1")
	if result != nil {
		t.Errorf("Expected nil after delete, got %v", result)
	}
}

func TestTTLCache_Clear(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}

	result1 := cache.Get("key1")
	result2 := cache.Get("key2")
	if result1 != nil || result2 != nil {
		t.Errorf("Expected nil values after clear, got %v, %v", result1, result2)
	}
}

func TestTTLCache_Size(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	defer cache.Close()

	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

func TestTTLCache_GetStats(t *testing.T) {
	cache := NewTTLCache(time.Hour)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	stats := cache.GetStats()

	totalItems, ok := stats["total_items"].(int)
	if !ok || totalItems != 2 {
		t.Errorf("Expected total_items 2, got %v", stats["total_items"])
	}

	ttlHours, ok := stats["ttl_hours"].(float64)
	if !ok || ttlHours != 1.0 {
		t.Errorf("Expected ttl_hours 1.0, got %v", stats["ttl_hours"])
	}
}

func TestTTLCache_ConcurrentAccess(t *testing.T) {
	cache := NewTTLCache(time.Minute)
	defer cache.Close()

	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			cache.Set(string(rune('a'+i%26)), i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Get(string(rune('a' + i%26)))
		}
		done <- true
	}()

	<-done
	<-done

	cache.Set("test", "value")
	result := cache.Get("test")
	if result != "value" {
		t.Errorf("Expected 'value' after concurrent access, got %v", result)
	}
}
