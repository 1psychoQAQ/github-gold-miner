# Retry Mechanism Integration Guide

This guide shows how to integrate the retry mechanism into the three external API adapters: GitHub, Gemini, and Feishu.

## Overview

The retry package provides a robust, context-aware retry mechanism with exponential backoff. It's designed to handle transient failures in external API calls.

## Key Features

- **Exponential Backoff**: Delays increase exponentially (configurable multiplier)
- **Context Support**: Respects context cancellation and deadlines
- **Functional Options**: Clean, idiomatic configuration API
- **Zero Dependencies**: Uses only the standard library
- **Testable**: Easy to test with fast delays in tests

## Integration Examples

### 1. GitHub API (fetcher.go)

**Before:**
```go
func (f *Fetcher) GetTrendingRepos(ctx context.Context, language string, since string) ([]*domain.Repo, error) {
    result, _, err := f.client.Search.Repositories(ctx, query, opts)
    if err != nil {
        return nil, fmt.Errorf("GitHub API 调用失败: %w", err)
    }
    // ... process result
}
```

**After:**
```go
import "github-gold-miner/internal/common"

func (f *Fetcher) GetTrendingRepos(ctx context.Context, language string, since string) ([]*domain.Repo, error) {
    var result *github.RepositoriesSearchResult

    err := common.Do(ctx, func() error {
        var apiErr error
        result, _, apiErr = f.client.Search.Repositories(ctx, query, opts)
        return apiErr
    },
        common.WithMaxRetries(3),
        common.WithInitialDelay(time.Second),
        common.WithMaxDelay(10*time.Second),
    )

    if err != nil {
        return nil, fmt.Errorf("GitHub API 调用失败: %w", err)
    }

    // ... process result
}
```

**Recommended Configuration for GitHub:**
- Max retries: 3
- Initial delay: 1 second
- Max delay: 10 seconds
- Multiplier: 2.0 (default)

**Rationale:** GitHub API has rate limits (5000/hour authenticated, 60/hour unauthenticated). Short retries handle transient network issues without burning through rate limit quota.

---

### 2. Gemini API (appraiser.go)

**Before:**
```go
func (g *GeminiAppraiser) Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error) {
    resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
        return repo, fmt.Errorf("AI 调用失败: %w", err)
    }
    // ... process response
}
```

**After:**
```go
import "github-gold-miner/internal/common"

func (g *GeminiAppraiser) Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error) {
    var resp *genai.GenerateContentResponse

    err := common.Do(ctx, func() error {
        var apiErr error
        resp, apiErr = g.model.GenerateContent(ctx, genai.Text(prompt))
        return apiErr
    },
        common.WithMaxRetries(5),
        common.WithInitialDelay(2*time.Second),
        common.WithMaxDelay(30*time.Second),
    )

    if err != nil {
        return repo, fmt.Errorf("AI 调用失败: %w", err)
    }

    // ... process response
}
```

**Recommended Configuration for Gemini:**
- Max retries: 5
- Initial delay: 2 seconds
- Max delay: 30 seconds
- Multiplier: 2.0 (default)

**Rationale:** LLM APIs can be slow and have occasional timeouts. More retries with longer delays accommodate model inference time while respecting the 30-second timeout in analyzer.go.

---

### 3. Feishu Webhook (notifier.go)

**Before:**
```go
func (n *Notifier) Notify(ctx context.Context, repo *domain.Repo) error {
    body, _ := json.Marshal(payload)
    resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("发送请求失败: %w", err)
    }
    // ... check response
}
```

**After:**
```go
import "github-gold-miner/internal/common"

func (n *Notifier) Notify(ctx context.Context, repo *domain.Repo) error {
    body, _ := json.Marshal(payload)

    err := common.Do(ctx, func() error {
        resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(body))
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
            return fmt.Errorf("飞书 API 报错: 状态码 %d", resp.StatusCode)
        }
        return nil
    },
        common.WithMaxRetries(3),
        common.WithInitialDelay(500*time.Millisecond),
        common.WithMaxDelay(5*time.Second),
    )

    if err != nil {
        return fmt.Errorf("发送请求失败: %w", err)
    }
    return nil
}
```

**Recommended Configuration for Feishu:**
- Max retries: 3
- Initial delay: 500ms
- Max delay: 5 seconds
- Multiplier: 2.0 (default)

**Rationale:** Webhook calls should be fast. Short delays and fewer retries prevent blocking the notification pipeline while still handling temporary network issues.

---

## Testing Integration

When writing tests for code that uses retry, use minimal delays:

```go
func TestSomething(t *testing.T) {
    err := common.Do(ctx, fn,
        common.WithMaxRetries(2),
        common.WithInitialDelay(1*time.Millisecond), // Fast for testing
    )
    // ... assertions
}
```

## Error Handling

The retry mechanism preserves error context:

```go
err := common.Do(ctx, fn, opts...)
if err != nil {
    // Check for specific errors
    if errors.Is(err, context.Canceled) {
        // Handle cancellation
    } else if errors.Is(err, context.DeadlineExceeded) {
        // Handle timeout
    } else {
        // Handle other errors
        // The original error is wrapped and accessible via errors.Unwrap()
    }
}
```

## Performance Considerations

### Backoff Calculation

For the default configuration (multiplier=2.0, initial=1s):
- Attempt 1: Immediate
- Attempt 2: 1s delay (total: 1s)
- Attempt 3: 2s delay (total: 3s)
- Attempt 4: 4s delay (total: 7s)

### Memory

Benchmarks show minimal allocations:
- Success on first try: 32 B/op, 1 alloc/op
- With retries: 560 B/op, 9 allocs/op

## Best Practices

1. **Choose appropriate retry counts**: More retries for LLM calls, fewer for webhooks
2. **Set context timeouts**: Prevent infinite retries in stuck operations
3. **Use exponential backoff**: Avoid overwhelming failed services
4. **Log retry attempts**: Consider adding logging for debugging (not included to keep package focused)
5. **Handle specific errors**: Not all errors should be retried (e.g., 404, validation errors)

## Advanced: Selective Retry

For retrying only specific errors:

```go
err := common.Do(ctx, func() error {
    resp, err := apiCall()
    if err != nil {
        // Don't retry client errors
        if isClientError(err) {
            return nil // Success to stop retrying
        }
        return err // Retry server errors
    }
    return nil
})
```

## Migration Checklist

- [ ] Add `import "github-gold-miner/internal/common"`
- [ ] Wrap API call in closure function
- [ ] Extract result to outer scope variable
- [ ] Configure retry options based on API characteristics
- [ ] Update tests to use fast delays
- [ ] Verify error handling works as expected
- [ ] Test context cancellation behavior
