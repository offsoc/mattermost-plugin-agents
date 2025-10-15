// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_NewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(60, 10)
	defer limiter.Close()

	if limiter == nil {
		t.Error("Expected non-nil rate limiter")
	}

	limiter2 := NewRateLimiter(0, 0)
	defer limiter2.Close()

	if limiter2 == nil {
		t.Error("Expected non-nil rate limiter even with invalid values")
	}
}

func TestRateLimiter_TryAcquire(t *testing.T) {
	limiter := NewRateLimiter(60, 5)
	defer limiter.Close()

	for i := 0; i < 5; i++ {
		if !limiter.TryAcquire() {
			t.Errorf("Failed to acquire token %d from initial burst", i+1)
		}
	}

	if limiter.TryAcquire() {
		t.Error("Should not be able to acquire token after burst exhausted")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := NewRateLimiter(60, 2)
	defer limiter.Close()

	ctx := context.Background()

	for i := 0; i < 2; i++ {
		err := limiter.Wait(ctx)
		if err != nil {
			t.Errorf("Failed to wait for token %d: %v", i+1, err)
		}
	}

	start := time.Now()
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Failed to wait for refilled token: %v", err)
	}

	if duration < 100*time.Millisecond {
		t.Errorf("Wait duration too short: %v", duration)
	}
}

func TestRateLimiter_WaitWithCancelledContext(t *testing.T) {
	limiter := NewRateLimiter(60, 1)
	defer limiter.Close()

	limiter.TryAcquire()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := limiter.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestRateLimiter_WaitWithTimeout(t *testing.T) {
	limiter := NewRateLimiter(1, 1) // Very slow refill rate
	defer limiter.Close()

	limiter.TryAcquire()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx)
	duration := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	if duration < 90*time.Millisecond || duration > 200*time.Millisecond {
		t.Errorf("Timeout duration unexpected: %v", duration)
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	limiter := NewRateLimiter(120, 2) // 2 tokens per second
	defer limiter.Close()

	limiter.TryAcquire()
	limiter.TryAcquire()

	if limiter.TryAcquire() {
		t.Error("Should not be able to acquire token immediately after burst")
	}

	time.Sleep(600 * time.Millisecond)

	if !limiter.TryAcquire() {
		t.Error("Should be able to acquire token after refill period")
	}
}

func TestRateLimiter_Close(t *testing.T) {
	limiter := NewRateLimiter(60, 5)

	for i := 0; i < 5; i++ {
		limiter.TryAcquire()
	}

	limiter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled after close, got %v", err)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(60, 10)
	defer limiter.Close()

	ctx := context.Background()
	errors := make(chan error, 20)

	for i := 0; i < 20; i++ {
		go func() {
			err := limiter.Wait(ctx)
			errors <- err
		}()
	}

	successCount := 0
	for i := 0; i < 20; i++ {
		err := <-errors
		if err == nil {
			successCount++
		}
	}

	if successCount < 10 {
		t.Errorf("Expected at least 10 successful acquisitions, got %d", successCount)
	}
}
