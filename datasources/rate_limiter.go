// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	tokens     chan struct{}
	refillRate time.Duration
	done       chan struct{}
}

// NewRateLimiter creates a new rate limiter with the specified requests per minute and burst size
func NewRateLimiter(requestsPerMinute, burstSize int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60 // Default to 60 requests per minute
	}
	if burstSize <= 0 {
		burstSize = 10 // Default burst size
	}

	refillRate := time.Minute / time.Duration(requestsPerMinute)

	limiter := &RateLimiter{
		tokens:     make(chan struct{}, burstSize),
		refillRate: refillRate,
		done:       make(chan struct{}),
	}

	for i := 0; i < burstSize; i++ {
		limiter.tokens <- struct{}{}
	}

	go limiter.refillTokens()

	return limiter
}

// Wait blocks until a token is available or context is canceled
func (r *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-r.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return context.Canceled
	}
}

// TryAcquire attempts to acquire a token without blocking
// Returns true if a token was acquired, false otherwise
func (r *RateLimiter) TryAcquire() bool {
	select {
	case <-r.tokens:
		return true
	default:
		return false
	}
}

// Close stops the rate limiter and releases resources
func (r *RateLimiter) Close() {
	close(r.done)
}

// refillTokens periodically adds tokens to the bucket
func (r *RateLimiter) refillTokens() {
	ticker := time.NewTicker(r.refillRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case r.tokens <- struct{}{}:
			default:
			}
		case <-r.done:
			return
		}
	}
}

// EnsureRateLimiter initializes a rate limiter if enabled and not already set
func EnsureRateLimiter(limiter **RateLimiter, config RateLimitConfig) {
	if config.Enabled && *limiter == nil {
		*limiter = NewRateLimiter(config.RequestsPerMinute, config.BurstSize)
	}
}

// WaitRateLimiter waits for rate limiter if it exists
func WaitRateLimiter(ctx context.Context, limiter *RateLimiter) error {
	if limiter != nil {
		return limiter.Wait(ctx)
	}
	return nil
}

// CloseRateLimiter closes and nils out a rate limiter
func CloseRateLimiter(limiter **RateLimiter) {
	if *limiter != nil {
		(*limiter).Close()
		*limiter = nil
	}
}
