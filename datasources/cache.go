// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"sync"
	"time"
)

// TTLCache implements a simple time-to-live cache with background cleanup.
// Callers must call Close() when done to stop the background cleanup goroutine.
type TTLCache struct {
	items     map[string]*cacheItem
	ttl       time.Duration
	mu        sync.RWMutex
	cleanup   chan struct{}
	closeOnce sync.Once
}

// cacheItem represents a cached item with expiration time
type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewTTLCache creates a new TTL cache with the specified time-to-live duration
func NewTTLCache(ttl time.Duration) *TTLCache {
	cache := &TTLCache{
		items:   make(map[string]*cacheItem),
		ttl:     ttl,
		cleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go cache.startCleanup()

	return cache
}

// Get retrieves a value from the cache by key
// Returns nil if the key doesn't exist or has expired
func (c *TTLCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil
	}

	if time.Now().After(item.expiresAt) {
		// Don't delete here to avoid write lock during read
		return nil
	}

	return item.value
}

// Set stores a value in the cache with the configured TTL
func (c *TTLCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a specific key from the cache
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *TTLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
}

// Size returns the current number of items in the cache
func (c *TTLCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Keys returns a snapshot of all keys currently stored in the cache.
// Expired keys may be included until the background cleanup removes them.
func (c *TTLCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

// GetStats returns cache statistics for monitoring
func (c *TTLCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	expired := 0

	itemsCopy := make(map[string]*cacheItem, len(c.items))
	for key, item := range c.items {
		itemsCopy[key] = item
	}

	for _, item := range itemsCopy {
		if now.After(item.expiresAt) {
			expired++
		}
	}

	return map[string]interface{}{
		"total_items":   len(c.items),
		"expired_items": expired,
		"ttl_hours":     c.ttl.Hours(),
	}
}

// Close stops the background cleanup goroutine.
// Safe to call multiple times - only the first call has effect.
func (c *TTLCache) Close() {
	c.closeOnce.Do(func() {
		close(c.cleanup)
	})
}

// startCleanup runs the background cleanup goroutine
func (c *TTLCache) startCleanup() {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpiredItems()
		case <-c.cleanup:
			return
		}
	}
}

// removeExpiredItems removes all expired items from the cache
func (c *TTLCache) removeExpiredItems() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}
