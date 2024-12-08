package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// RateLimit implements token bucket algorithm for rate limiting
type RateLimit struct {
	ticker  *time.Ticker
	tokens  chan struct{}
	timeout time.Duration
}

// NewRateLimit creates a new RateLimit instance
func NewRateLimit(rate int, timeout time.Duration) *RateLimit {
	rl := &RateLimit{
		ticker:  time.NewTicker(time.Minute / time.Duration(rate)),
		tokens:  make(chan struct{}, rate),
		timeout: timeout,
	}

	// Initialize token bucket
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	go rl.refillTokens()
	return rl
}

// refillTokens periodically adds new tokens to the bucket
func (rl *RateLimit) refillTokens() {
	for range rl.ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}

// Acquire attempts to acquire a token with timeout
func (rl *RateLimit) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	case <-time.After(rl.timeout):
		return fmt.Errorf("rate limit timeout")
	}
}

// Stop stops the rate limiter
func (rl *RateLimit) Stop() {
	rl.ticker.Stop()
}
