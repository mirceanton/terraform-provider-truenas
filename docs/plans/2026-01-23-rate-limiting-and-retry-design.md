# Rate Limiting and Retry Design

**Date:** 2026-01-23
**Issues:** #10 (connection handshake failures), #11 (rate limiting)
**Branch:** `feature/rate-limiting-and-retry`

## Problem

TrueNAS enforces rate limits: "20 Auth attempts AND/OR unauthenticated API requests in a 60 second period. Exceeding this limit results in a 10-minute rate limit cooldown."

The current SSH+midclt approach creates a new connection (auth attempt) per API call. With many resources, this quickly hits the rate limit, manifesting as "Failed connection handshake" errors.

## Solution Overview

Add a `RateLimitedClient` wrapper that:

1. **Proactively rate limits** calls using a token bucket (18/min default)
2. **Retries transient errors** with exponential backoff (3 retries default)
3. **Classifies errors** to distinguish retriable (connection) from terminal (validation)

```
┌─────────────────────┐
│  Terraform Resources │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  RateLimitedClient  │  ← New wrapper
│  - Token bucket     │
│  - Retry with backoff│
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  SSHClient          │  ← Existing (unchanged)
└─────────────────────┘
```

## Provider Configuration

```hcl
provider "truenas" {
  host     = "truenas.local"
  # ... existing config ...

  rate_limit  = 18  # calls per minute (default: 18)
  max_retries = 3   # retry attempts (default: 3, 0 = disabled)
}
```

## Rate Limiter

Uses `golang.org/x/time/rate` token bucket:

- 18 calls/minute = ~3.3 seconds between calls
- Burst size of 1 (strict pacing, no bursting)
- Respects context cancellation

```go
interval := time.Minute / time.Duration(callsPerMinute)
limiter := rate.NewLimiter(rate.Every(interval), 1)
```

## Retry Logic

Exponential backoff with jitter on transient errors:

| Attempt | Base Delay | With Jitter |
|---------|------------|-------------|
| 1       | 2s         | 1.5s - 2.5s |
| 2       | 4s         | 3s - 5s     |
| 3       | 8s         | 6s - 10s    |

Maximum delay capped at 30 seconds. Jitter prevents thundering herd.

```go
func calculateBackoff(attempt int) time.Duration {
    base := 2 * time.Second
    max := 30 * time.Second

    delay := base * time.Duration(1<<attempt)
    if delay > max {
        delay = max
    }

    // Add jitter: ±25%
    jitter := time.Duration(rand.Int63n(int64(delay/2))) - delay/4
    return delay + jitter
}
```

## Error Classification

Pluggable interface to support future WebSocket client:

```go
type RetryClassifier interface {
    IsRetriable(err error) bool
}
```

**SSHRetryClassifier** - matches connection error strings:

- `"Failed connection handshake"`
- `"Unexpected closure of remote connection"`
- `"connection refused"`
- `"connection reset"`
- `"i/o timeout"`

**Non-retriable errors** (fail fast):

- `[EINVAL]`, `[ENOENT]`, `[EEXIST]` - structured API errors
- Any error that reached the TrueNAS API successfully

## Call Wrapper

```go
func (c *RateLimitedClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limiter: %w", err)
    }

    var lastErr error
    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        result, err := c.client.Call(ctx, method, params)
        if err == nil {
            return result, nil
        }

        lastErr = err
        if !c.classifier.IsRetriable(err) {
            return nil, err
        }

        if attempt == c.maxRetries {
            break
        }

        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(calculateBackoff(attempt)):
        }
    }

    return nil, fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr)
}
```

## File Structure

```
internal/client/
├── client.go              # Existing Client interface
├── ssh.go                 # Existing SSHClient (unchanged)
├── ratelimit.go           # NEW: RateLimitedClient wrapper
├── ratelimit_test.go      # NEW: Unit tests
├── retry.go               # NEW: RetryClassifier, backoff logic
├── retry_test.go          # NEW: Retry/classifier tests
└── errors.go              # Existing (unchanged)

internal/provider/
└── provider.go            # Add rate_limit, max_retries to schema
```

## Implementation Order

1. Add `retry.go` - RetryClassifier interface, SSHRetryClassifier, backoff
2. Add `retry_test.go` - Test error classification and backoff
3. Add `ratelimit.go` - RateLimitedClient wrapper
4. Add `ratelimit_test.go` - Test rate limiting and retry
5. Update `provider.go` - Add schema fields, wrap client
6. Manual testing against TrueNAS
7. Update docs with new provider options

## Testing

**Unit tests (automated):**

- Retry logic with mock client
- Error classification patterns
- Rate limiter timing
- Context cancellation
- Backoff calculation

**Manual integration tests:**

- [ ] Apply 20+ resources without rate limit errors
- [ ] Confirm calls spaced ~3.3s apart (debug logging)
- [ ] Simulate network blip, verify retry recovery
- [ ] Verify validation errors fail immediately
- [ ] Test custom rate_limit and max_retries values

## Future Considerations

**WebSocket/JSON-RPC migration:**

A single persistent WebSocket connection would largely eliminate the rate limit problem (1 auth vs 1 per call). The wrapper design supports this:

- `WebSocketClient` implements same `Client` interface
- `WebSocketRetryClassifier` uses structured error codes (-32000)
- Rate limiting may become optional for WebSocket

**Structured error codes:**

WebSocket gives access to JSON-RPC error codes:
- `-32000`: TOO_MANY_CONCURRENT_CALLS
- `-32001`: TRUENAS_CALL_ERROR

This enables precise retry classification vs current string matching.

## Dependencies

```
go get golang.org/x/time/rate
```
