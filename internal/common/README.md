# Common Utilities Package

This package provides shared utility functions used across the GitHub Gold Miner application.

## Modules

### Error Handling (`errors.go`)

Custom error types and error codes for consistent error handling across the application.

### Retry Mechanism (`retry.go`)

A robust, context-aware retry mechanism with exponential backoff for handling transient failures in external API calls.

## Retry Mechanism

### Features

- **Exponential Backoff**: Configurable backoff with customizable multiplier
- **Context Support**: Full respect for context cancellation and deadlines
- **Functional Options**: Clean, idiomatic Go configuration pattern
- **Zero External Dependencies**: Uses only the Go standard library
- **High Performance**: Minimal allocations (32 B/op for success case)
- **Well Tested**: 84.1% test coverage with comprehensive edge cases

### Quick Start

#### Basic Usage

```go
import "github-gold-miner/internal/common"

ctx := context.Background()
err := common.Do(ctx, func() error {
    return someAPICall()
})
```

#### With Custom Configuration

```go
err := common.Do(ctx, apiCall,
    common.WithMaxRetries(5),
    common.WithInitialDelay(time.Second),
    common.WithMaxDelay(30*time.Second),
)
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithMaxRetries(n)` | Maximum number of retry attempts | 3 |
| `WithInitialDelay(d)` | Initial delay before first retry | 1s |
| `WithMaxDelay(d)` | Maximum delay between retries (cap) | 30s |
| `WithMultiplier(m)` | Exponential backoff multiplier | 2.0 |

### Backoff Strategy

With default settings (multiplier=2.0, initialDelay=1s):

| Attempt | Delay | Cumulative Time |
|---------|-------|----------------|
| 1 | 0s (immediate) | 0s |
| 2 | 1s | 1s |
| 3 | 2s | 3s |
| 4 | 4s | 7s |

### Use Cases

This retry mechanism is designed for the three main external API integrations:

#### 1. GitHub API
```go
// Recommended: 3 retries, 1s initial delay
err := common.Do(ctx, githubCall,
    common.WithMaxRetries(3),
    common.WithInitialDelay(time.Second),
    common.WithMaxDelay(10*time.Second),
)
```

**Rationale**: GitHub has strict rate limits. Short retries handle network issues without burning quota.

#### 2. Gemini API
```go
// Recommended: 5 retries, 2s initial delay
err := common.Do(ctx, geminiCall,
    common.WithMaxRetries(5),
    common.WithInitialDelay(2*time.Second),
    common.WithMaxDelay(30*time.Second),
)
```

**Rationale**: LLM APIs can timeout. More retries with longer delays accommodate inference time.

#### 3. Feishu Webhook
```go
// Recommended: 3 retries, 500ms initial delay
err := common.Do(ctx, webhookCall,
    common.WithMaxRetries(3),
    common.WithInitialDelay(500*time.Millisecond),
    common.WithMaxDelay(5*time.Second),
)
```

**Rationale**: Fast webhooks need quick retries without blocking the notification pipeline.

### Context Handling

The retry mechanism fully respects context cancellation:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := common.Do(ctx, fn)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    time.Sleep(5*time.Second)
    cancel()
}()

err := common.Do(ctx, fn)
if errors.Is(err, context.Canceled) {
    // Handle cancellation
}
```

### Error Handling

Errors are wrapped with context:

```go
err := common.Do(ctx, fn)
if err != nil {
    // Original error is preserved via error wrapping
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        // Handle specific error type
    }

    // Unwrap to get original error
    originalErr := errors.Unwrap(err)
}
```

Error messages include retry context:
```
retry failed after 4 attempts: original error message
retry aborted during backoff (attempt 2/3): context canceled
```

### Testing

When writing tests, use minimal delays for fast execution:

```go
func TestWithRetry(t *testing.T) {
    err := common.Do(ctx, fn,
        common.WithMaxRetries(2),
        common.WithInitialDelay(1*time.Millisecond), // Fast!
    )
    // assertions...
}
```

### Performance

Benchmark results (Apple M3 Pro):

```
BenchmarkDo_Success-12        97072490     11.78 ns/op     32 B/op   1 allocs/op
BenchmarkDo_WithRetries-12     2651139    463.3 ns/op    560 B/op   9 allocs/op
```

The retry mechanism is highly efficient:
- **Success path**: Only 32 bytes allocated
- **Retry path**: 560 bytes total for multiple attempts
- **Zero overhead** for successful operations

### Design Decisions

1. **Functional Options Pattern**: Provides clean, extensible API without breaking changes
2. **Exponential Backoff**: Prevents overwhelming failed services while being aggressive enough
3. **Context-First**: Context is required parameter, enforcing proper timeout/cancellation
4. **Error Wrapping**: Preserves original errors via `%w` for proper error chain handling
5. **No Logging**: Keeps package focused; consumers can add logging at call sites
6. **Timer Cleanup**: Properly stops timers to prevent goroutine/memory leaks

### Integration Guide

See [INTEGRATION.md](./INTEGRATION.md) for detailed examples of integrating retry into the GitHub, Gemini, and Feishu adapters.

### API Reference

#### `Do(ctx context.Context, fn RetryableFunc, opts ...Option) error`

Executes the provided function with exponential backoff retry logic.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `fn`: Function to execute (must return error)
- `opts`: Optional configuration options

**Returns:**
- `nil` if any attempt succeeds
- Last error if all attempts fail
- Context error if context is cancelled

**Example:**
```go
err := common.Do(ctx, func() error {
    return apiCall()
}, common.WithMaxRetries(3))
```

#### Type `RetryableFunc`

```go
type RetryableFunc func() error
```

A function that can be retried. Should return an error if the operation failed.

### Thread Safety

The retry mechanism is thread-safe. Multiple goroutines can call `Do()` concurrently without coordination.

### Limitations

1. **No jitter**: Backoff delays are deterministic (could be added via `WithMultiplier` variance)
2. **No conditional retry**: All errors trigger retry (filter at call site if needed)
3. **No retry budgets**: Each call is independent (application-level circuit breaker needed for that)

### Future Enhancements

Potential additions if needed:
- Jitter support for reducing thundering herd
- Predicate function for selective retry
- Retry budget/circuit breaker integration
- Detailed retry metrics/observability hooks
