package common

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// RetryableFunc defines a function that can be retried.
// It should return an error if the operation failed and needs to be retried.
type RetryableFunc func() error

// Config holds the configuration for retry behavior.
type Config struct {
	maxRetries   int
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
}

// Option is a functional option for configuring retry behavior.
type Option func(*Config)

// WithMaxRetries sets the maximum number of retry attempts.
// Default is 3 retries.
func WithMaxRetries(n int) Option {
	return func(c *Config) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithInitialDelay sets the initial delay before the first retry.
// Default is 1 second.
func WithInitialDelay(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.initialDelay = d
		}
	}
}

// WithMaxDelay sets the maximum delay between retries.
// Default is 30 seconds.
func WithMaxDelay(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.maxDelay = d
		}
	}
}

// WithMultiplier sets the exponential backoff multiplier.
// Default is 2.0 (doubles each retry).
func WithMultiplier(m float64) Option {
	return func(c *Config) {
		if m > 0 {
			c.multiplier = m
		}
	}
}

// defaultConfig returns the default retry configuration.
func defaultConfig() *Config {
	return &Config{
		maxRetries:   3,
		initialDelay: 1 * time.Second,
		maxDelay:     30 * time.Second,
		multiplier:   2.0,
	}
}

// Do executes the provided function with exponential backoff retry logic.
// It respects context cancellation and will stop retrying if the context is cancelled.
//
// The function will:
// - Execute immediately on the first attempt
// - Retry on failure with exponential backoff
// - Return nil if any attempt succeeds
// - Return the last error if all attempts fail
// - Return context.Canceled or context.DeadlineExceeded if context is cancelled
//
// Example usage:
//
//	err := retry.Do(ctx, func() error {
//	    return someAPICall()
//	})
//
//	err := retry.Do(ctx, fn,
//	    retry.WithMaxRetries(5),
//	    retry.WithInitialDelay(time.Second),
//	    retry.WithMaxDelay(30*time.Second),
//	)
func Do(ctx context.Context, fn RetryableFunc, opts ...Option) error {
	if fn == nil {
		return errors.New("retry: function cannot be nil")
	}

	// Apply default config
	cfg := defaultConfig()

	// Apply custom options
	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error

	// First attempt (attempt 0)
	if err := fn(); err == nil {
		return nil
	} else {
		lastErr = err
	}

	// Retry attempts
	for attempt := 1; attempt <= cfg.maxRetries; attempt++ {
		// Check context before sleeping
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry aborted after %d attempts: %w", attempt, ctx.Err())
		default:
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, cfg.initialDelay, cfg.maxDelay, cfg.multiplier)

		// Sleep with context cancellation support
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("retry aborted during backoff (attempt %d/%d): %w", attempt, cfg.maxRetries, ctx.Err())
		case <-timer.C:
			// Continue to next attempt
		}

		// Execute the function
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	// All retries exhausted
	return fmt.Errorf("retry failed after %d attempts: %w", cfg.maxRetries+1, lastErr)
}

// calculateDelay computes the delay for the current attempt using exponential backoff.
// The delay is capped at maxDelay.
func calculateDelay(attempt int, initialDelay, maxDelay time.Duration, multiplier float64) time.Duration {
	// Calculate: initialDelay * (multiplier ^ (attempt - 1))
	// For attempt 1: initialDelay * 1
	// For attempt 2: initialDelay * multiplier
	// For attempt 3: initialDelay * multiplier^2
	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt-1))

	// Cap at maxDelay
	if time.Duration(delay) > maxDelay {
		return maxDelay
	}

	return time.Duration(delay)
}
