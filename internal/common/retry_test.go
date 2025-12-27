package common

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := Do(ctx, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	tests := []struct {
		name            string
		failUntilN      int
		maxRetries      int
		expectedAttempts int
		shouldSucceed   bool
	}{
		{
			name:            "success on second attempt",
			failUntilN:      2,
			maxRetries:      3,
			expectedAttempts: 2,
			shouldSucceed:   true,
		},
		{
			name:            "success on third attempt",
			failUntilN:      3,
			maxRetries:      3,
			expectedAttempts: 3,
			shouldSucceed:   true,
		},
		{
			name:            "success on last retry",
			failUntilN:      4,
			maxRetries:      3,
			expectedAttempts: 4,
			shouldSucceed:   true,
		},
		{
			name:            "fail all attempts",
			failUntilN:      10,
			maxRetries:      3,
			expectedAttempts: 4, // 1 initial + 3 retries
			shouldSucceed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			attempts := 0

			err := Do(ctx, func() error {
				attempts++
				if attempts < tt.failUntilN {
					return errors.New("temporary failure")
				}
				return nil
			}, WithMaxRetries(tt.maxRetries), WithInitialDelay(1*time.Millisecond))

			if tt.shouldSucceed && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}

			if !tt.shouldSucceed && err == nil {
				t.Error("expected error, got nil")
			}

			if attempts != tt.expectedAttempts {
				t.Errorf("expected %d attempts, got %d", tt.expectedAttempts, attempts)
			}
		})
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	tests := []struct {
		name          string
		cancelAfter   time.Duration
		initialDelay  time.Duration
		expectedError error
	}{
		{
			name:          "cancel before retry",
			cancelAfter:   5 * time.Millisecond,
			initialDelay:  100 * time.Millisecond,
			expectedError: context.Canceled,
		},
		{
			name:          "cancel during backoff",
			cancelAfter:   20 * time.Millisecond,
			initialDelay:  10 * time.Millisecond,
			expectedError: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Cancel context after a delay
			go func() {
				time.Sleep(tt.cancelAfter)
				cancel()
			}()

			attempts := 0
			err := Do(ctx, func() error {
				attempts++
				return errors.New("always fails")
			}, WithInitialDelay(tt.initialDelay), WithMaxRetries(5))

			if err == nil {
				t.Error("expected error, got nil")
			}

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error to contain %v, got: %v", tt.expectedError, err)
			}

			if attempts == 0 {
				t.Error("expected at least one attempt")
			}
		})
	}
}

func TestDo_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	err := Do(ctx, func() error {
		attempts++
		return errors.New("always fails")
	}, WithInitialDelay(30*time.Millisecond), WithMaxRetries(10))

	if err == nil {
		t.Error("expected error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded error, got: %v", err)
	}

	// Should have made at least 1 attempt
	if attempts == 0 {
		t.Error("expected at least one attempt")
	}
}

func TestDo_NilFunction(t *testing.T) {
	ctx := context.Background()
	err := Do(ctx, nil)

	if err == nil {
		t.Error("expected error for nil function, got nil")
	}

	expectedMsg := "retry: function cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestDo_WithOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		failUntilN  int
		shouldCheck func(t *testing.T, attempts int, duration time.Duration, err error)
	}{
		{
			name: "custom max retries",
			opts: []Option{
				WithMaxRetries(5),
				WithInitialDelay(1 * time.Millisecond),
			},
			failUntilN: 10, // Never succeeds
			shouldCheck: func(t *testing.T, attempts int, duration time.Duration, err error) {
				if attempts != 6 { // 1 initial + 5 retries
					t.Errorf("expected 6 attempts, got %d", attempts)
				}
				if err == nil {
					t.Error("expected error, got nil")
				}
			},
		},
		{
			name: "zero retries",
			opts: []Option{
				WithMaxRetries(0),
			},
			failUntilN: 10,
			shouldCheck: func(t *testing.T, attempts int, duration time.Duration, err error) {
				if attempts != 1 { // Only initial attempt
					t.Errorf("expected 1 attempt, got %d", attempts)
				}
				if err == nil {
					t.Error("expected error, got nil")
				}
			},
		},
		{
			name: "exponential backoff timing",
			opts: []Option{
				WithMaxRetries(3),
				WithInitialDelay(10 * time.Millisecond),
				WithMaxDelay(100 * time.Millisecond),
				WithMultiplier(2.0),
			},
			failUntilN: 10,
			shouldCheck: func(t *testing.T, attempts int, duration time.Duration, err error) {
				// Expected delays: 10ms, 20ms, 40ms = 70ms total
				// Allow some tolerance for execution time
				if duration < 70*time.Millisecond || duration > 150*time.Millisecond {
					t.Errorf("expected duration ~70ms, got %v", duration)
				}
			},
		},
		{
			name: "max delay cap",
			opts: []Option{
				WithMaxRetries(5),
				WithInitialDelay(10 * time.Millisecond),
				WithMaxDelay(20 * time.Millisecond),
				WithMultiplier(2.0),
			},
			failUntilN: 10,
			shouldCheck: func(t *testing.T, attempts int, duration time.Duration, err error) {
				// Expected delays: 10ms, 20ms, 20ms, 20ms, 20ms = 90ms total (capped)
				// Without cap: 10, 20, 40, 80, 160
				if duration < 90*time.Millisecond || duration > 180*time.Millisecond {
					t.Errorf("expected duration ~90ms (with cap), got %v", duration)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			attempts := 0
			start := time.Now()

			err := Do(ctx, func() error {
				attempts++
				if attempts < tt.failUntilN {
					return errors.New("temporary failure")
				}
				return nil
			}, tt.opts...)

			duration := time.Since(start)
			tt.shouldCheck(t, attempts, duration, err)
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	tests := []struct {
		name         string
		attempt      int
		initialDelay time.Duration
		maxDelay     time.Duration
		multiplier   float64
		expected     time.Duration
	}{
		{
			name:         "first retry",
			attempt:      1,
			initialDelay: 100 * time.Millisecond,
			maxDelay:     1 * time.Second,
			multiplier:   2.0,
			expected:     100 * time.Millisecond,
		},
		{
			name:         "second retry",
			attempt:      2,
			initialDelay: 100 * time.Millisecond,
			maxDelay:     1 * time.Second,
			multiplier:   2.0,
			expected:     200 * time.Millisecond,
		},
		{
			name:         "third retry",
			attempt:      3,
			initialDelay: 100 * time.Millisecond,
			maxDelay:     1 * time.Second,
			multiplier:   2.0,
			expected:     400 * time.Millisecond,
		},
		{
			name:         "capped at max delay",
			attempt:      5,
			initialDelay: 100 * time.Millisecond,
			maxDelay:     500 * time.Millisecond,
			multiplier:   2.0,
			expected:     500 * time.Millisecond, // Would be 1600ms without cap
		},
		{
			name:         "multiplier of 1.5",
			attempt:      2,
			initialDelay: 100 * time.Millisecond,
			maxDelay:     1 * time.Second,
			multiplier:   1.5,
			expected:     150 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDelay(tt.attempt, tt.initialDelay, tt.maxDelay, tt.multiplier)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDo_ErrorWrapping(t *testing.T) {
	ctx := context.Background()
	originalErr := errors.New("original error")

	err := Do(ctx, func() error {
		return originalErr
	}, WithMaxRetries(2), WithInitialDelay(1*time.Millisecond))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check that the original error is wrapped
	if !errors.Is(err, originalErr) {
		t.Errorf("expected error to wrap original error, got: %v", err)
	}

	// Check error message format
	expectedContains := "retry failed after 3 attempts"
	if !contains(err.Error(), expectedContains) {
		t.Errorf("expected error message to contain %q, got: %q", expectedContains, err.Error())
	}
}

func TestDo_InvalidOptions(t *testing.T) {
	ctx := context.Background()

	// Invalid options should be ignored and use defaults
	err := Do(ctx, func() error {
		return nil
	},
		WithMaxRetries(-1),      // Invalid
		WithInitialDelay(-1),    // Invalid
		WithMaxDelay(-1),        // Invalid
		WithMultiplier(-1),      // Invalid
	)

	if err != nil {
		t.Errorf("expected success with invalid options (should use defaults), got: %v", err)
	}
}

// Benchmark tests
func BenchmarkDo_Success(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_ = Do(ctx, func() error {
			return nil
		})
	}
}

func BenchmarkDo_WithRetries(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		attempts := 0
		_ = Do(ctx, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("fail")
			}
			return nil
		}, WithInitialDelay(1*time.Nanosecond))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
