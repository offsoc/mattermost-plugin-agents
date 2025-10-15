// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"net/url"
	"sync"
	"time"
)

// HTTPCircuitBreaker tracks failures per domain to prevent cascading failures
type HTTPCircuitBreaker struct {
	mu                   sync.RWMutex
	domainFailures       map[string][]time.Time
	maxFailuresPerDomain int
	failureWindow        time.Duration
}

// newHTTPCircuitBreaker creates a new circuit breaker with default settings
func newHTTPCircuitBreaker() *HTTPCircuitBreaker {
	return &HTTPCircuitBreaker{
		domainFailures:       make(map[string][]time.Time),
		maxFailuresPerDomain: 50,              // Increased from 10 to avoid blocking during tests with multiple queries
		failureWindow:        5 * time.Minute, // Only count failures within last 5 minutes
	}
}

// recordFailure records a failure for a domain with timestamp
func (cb *HTTPCircuitBreaker) recordFailure(urlStr string) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return
	}
	cb.mu.Lock()
	now := time.Now()
	cb.domainFailures[parsed.Host] = append(cb.domainFailures[parsed.Host], now)
	cb.mu.Unlock()
}

// isOpen checks if the circuit breaker is open for a domain (too many failures)
func (cb *HTTPCircuitBreaker) isOpen(urlStr string) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	cb.mu.RLock()
	failureTimestamps := cb.domainFailures[parsed.Host]
	cb.mu.RUnlock()

	// Count failures within the time window
	now := time.Now()
	recentFailures := 0
	for _, timestamp := range failureTimestamps {
		if now.Sub(timestamp) <= cb.failureWindow {
			recentFailures++
		}
	}

	return recentFailures >= cb.maxFailuresPerDomain
}
