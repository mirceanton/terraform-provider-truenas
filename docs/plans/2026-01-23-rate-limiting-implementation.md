# Rate Limiting and Retry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add rate limiting and retry logic to prevent TrueNAS API rate limit errors.

**Architecture:** A `RateLimitedClient` wrapper that implements the `Client` interface, adding token bucket rate limiting and exponential backoff retry for transient errors. The wrapper delegates to the underlying `SSHClient` for actual API calls.

**Tech Stack:** Go, `golang.org/x/time/rate` for token bucket, Terraform Plugin Framework

**Coverage Baseline:** `internal/client` is at 81.6% - maintain or improve this.

---

## Task 1: Add golang.org/x/time/rate dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

Run:
```bash
go get golang.org/x/time/rate
```

**Step 2: Verify it was added**

Run:
```bash
grep "golang.org/x/time" go.mod
```

Expected: Line containing `golang.org/x/time`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add golang.org/x/time/rate dependency"
```

---

## Task 2: Create RetryClassifier interface and SSHRetryClassifier

**Files:**
- Create: `internal/client/retry.go`
- Create: `internal/client/retry_test.go`

**Step 1: Write the failing tests**

Create `internal/client/retry_test.go`:

```go
package client

import (
	"errors"
	"testing"
)

func TestSSHRetryClassifier_IsRetriable(t *testing.T) {
	classifier := &SSHRetryClassifier{}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "connection handshake failure",
			err:      errors.New("Failed connection handshake"),
			expected: true,
		},
		{
			name:     "unexpected closure",
			err:      errors.New("Unexpected closure of remote connection"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("read: connection reset by peer"),
			expected: true,
		},
		{
			name:     "io timeout",
			err:      errors.New("i/o timeout"),
			expected: true,
		},
		{
			name:     "validation error not retriable",
			err:      errors.New("[EINVAL] name: invalid character"),
			expected: false,
		},
		{
			name:     "not found error not retriable",
			err:      errors.New("[ENOENT] resource not found"),
			expected: false,
		},
		{
			name:     "generic error not retriable",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "case insensitive matching",
			err:      errors.New("CONNECTION REFUSED"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsRetriable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetriable(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client -run TestSSHRetryClassifier -v
```

Expected: FAIL - `SSHRetryClassifier` undefined

**Step 3: Write the implementation**

Create `internal/client/retry.go`:

```go
package client

import "strings"

// RetryClassifier determines if an error is retriable.
type RetryClassifier interface {
	IsRetriable(err error) bool
}

// SSHRetryClassifier classifies errors from SSH+midclt by string matching.
type SSHRetryClassifier struct{}

// retriablePatterns contains error substrings that indicate transient failures.
var retriablePatterns = []string{
	"failed connection handshake",
	"unexpected closure of remote connection",
	"connection refused",
	"connection reset",
	"i/o timeout",
	"no route to host",
	"network is unreachable",
}

// IsRetriable returns true if the error is a transient connection failure.
func (c *SSHRetryClassifier) IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	for _, pattern := range retriablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/client -run TestSSHRetryClassifier -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/retry.go internal/client/retry_test.go
git commit -m "feat(client): add RetryClassifier interface and SSHRetryClassifier"
```

---

## Task 3: Add backoff calculation

**Files:**
- Modify: `internal/client/retry.go`
- Modify: `internal/client/retry_test.go`

**Step 1: Write the failing tests**

Add to `internal/client/retry_test.go`:

```go
import (
	"errors"
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		minWant time.Duration
		maxWant time.Duration
	}{
		{attempt: 0, minWant: 1500 * time.Millisecond, maxWant: 2500 * time.Millisecond},
		{attempt: 1, minWant: 3 * time.Second, maxWant: 5 * time.Second},
		{attempt: 2, minWant: 6 * time.Second, maxWant: 10 * time.Second},
		{attempt: 3, minWant: 12 * time.Second, maxWant: 20 * time.Second},
		{attempt: 10, minWant: 22500 * time.Millisecond, maxWant: 37500 * time.Millisecond}, // capped at 30s base
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune('0'+tt.attempt)), func(t *testing.T) {
			// Run multiple times to account for jitter
			for i := 0; i < 10; i++ {
				result := CalculateBackoff(tt.attempt)
				if result < tt.minWant || result > tt.maxWant {
					t.Errorf("CalculateBackoff(%d) = %v, want between %v and %v",
						tt.attempt, result, tt.minWant, tt.maxWant)
				}
			}
		})
	}
}

func TestCalculateBackoff_Deterministic(t *testing.T) {
	// With same seed, should get consistent behavior within jitter range
	result1 := CalculateBackoff(0)
	result2 := CalculateBackoff(0)

	// Both should be in valid range even if different due to jitter
	minWant := 1500 * time.Millisecond
	maxWant := 2500 * time.Millisecond

	if result1 < minWant || result1 > maxWant {
		t.Errorf("first call: got %v, want between %v and %v", result1, minWant, maxWant)
	}
	if result2 < minWant || result2 > maxWant {
		t.Errorf("second call: got %v, want between %v and %v", result2, minWant, maxWant)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client -run TestCalculateBackoff -v
```

Expected: FAIL - `CalculateBackoff` undefined

**Step 3: Write the implementation**

Add to `internal/client/retry.go`:

```go
import (
	"math/rand"
	"strings"
	"time"
)

const (
	backoffBase = 2 * time.Second
	backoffMax  = 30 * time.Second
)

// CalculateBackoff returns the delay before retry attempt n (0-indexed).
// Uses exponential backoff with jitter: base * 2^attempt ± 25%.
func CalculateBackoff(attempt int) time.Duration {
	delay := backoffBase * time.Duration(1<<attempt)
	if delay > backoffMax {
		delay = backoffMax
	}

	// Add jitter: ±25% randomness
	jitter := time.Duration(rand.Int63n(int64(delay/2))) - delay/4
	return delay + jitter
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/client -run TestCalculateBackoff -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/retry.go internal/client/retry_test.go
git commit -m "feat(client): add exponential backoff calculation with jitter"
```

---

## Task 4: Create RateLimitedClient wrapper

**Files:**
- Create: `internal/client/ratelimit.go`
- Create: `internal/client/ratelimit_test.go`

**Step 1: Write the failing tests for constructor**

Create `internal/client/ratelimit_test.go`:

```go
package client

import (
	"testing"
)

func TestNewRateLimitedClient(t *testing.T) {
	mock := &MockClient{}

	client := NewRateLimitedClient(mock, 18, 3, &SSHRetryClassifier{})

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.client != mock {
		t.Error("expected client to wrap mock")
	}
	if client.maxRetries != 3 {
		t.Errorf("expected maxRetries=3, got %d", client.maxRetries)
	}
}

func TestNewRateLimitedClient_Defaults(t *testing.T) {
	mock := &MockClient{}

	// Test with zero values - should use defaults
	client := NewRateLimitedClient(mock, 0, 0, nil)

	if client.limiter == nil {
		t.Error("expected limiter to be initialized with default")
	}
	if client.classifier == nil {
		t.Error("expected classifier to default to SSHRetryClassifier")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client -run TestNewRateLimitedClient -v
```

Expected: FAIL - `NewRateLimitedClient` undefined

**Step 3: Write the implementation**

Create `internal/client/ratelimit.go`:

```go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"time"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"golang.org/x/time/rate"
)

const (
	defaultRateLimit  = 18 // calls per minute
	defaultMaxRetries = 3
)

// RateLimitedClient wraps a Client with rate limiting and retry logic.
type RateLimitedClient struct {
	client     Client
	limiter    *rate.Limiter
	maxRetries int
	classifier RetryClassifier
}

// Compile-time check that RateLimitedClient implements Client.
var _ Client = (*RateLimitedClient)(nil)

// NewRateLimitedClient creates a client wrapper with rate limiting and retry.
// If callsPerMinute is 0, defaults to 18. If maxRetries is 0, defaults to 3.
// If classifier is nil, defaults to SSHRetryClassifier.
func NewRateLimitedClient(client Client, callsPerMinute int, maxRetries int, classifier RetryClassifier) *RateLimitedClient {
	if callsPerMinute <= 0 {
		callsPerMinute = defaultRateLimit
	}
	if maxRetries < 0 {
		maxRetries = defaultMaxRetries
	}
	if classifier == nil {
		classifier = &SSHRetryClassifier{}
	}

	// Convert calls/minute to rate.Limiter parameters
	interval := time.Minute / time.Duration(callsPerMinute)
	limiter := rate.NewLimiter(rate.Every(interval), 1)

	return &RateLimitedClient{
		client:     client,
		limiter:    limiter,
		maxRetries: maxRetries,
		classifier: classifier,
	}
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/client -run TestNewRateLimitedClient -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ratelimit.go internal/client/ratelimit_test.go
git commit -m "feat(client): add RateLimitedClient struct and constructor"
```

---

## Task 5: Implement Call method with rate limiting and retry

**Files:**
- Modify: `internal/client/ratelimit.go`
- Modify: `internal/client/ratelimit_test.go`

**Step 1: Write the failing tests**

Add to `internal/client/ratelimit_test.go`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRateLimitedClient_Call_Success(t *testing.T) {
	expected := json.RawMessage(`{"result": "ok"}`)
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return expected, nil
		},
	}

	client := NewRateLimitedClient(mock, 60, 3, &SSHRetryClassifier{})
	result, err := client.Call(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_Call_RetryOnTransientError(t *testing.T) {
	callCount := 0
	expected := json.RawMessage(`{"result": "ok"}`)

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("Failed connection handshake")
			}
			return expected, nil
		},
	}

	// Use high rate limit to avoid rate limiting delays in test
	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})
	result, err := client.Call(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_Call_NoRetryOnValidationError(t *testing.T) {
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			return nil, errors.New("[EINVAL] name: invalid")
		},
	}

	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})
	_, err := client.Call(context.Background(), "test.method", nil)

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries), got %d", callCount)
	}
}

func TestRateLimitedClient_Call_ExhaustedRetries(t *testing.T) {
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			return nil, errors.New("connection refused")
		},
	}

	client := NewRateLimitedClient(mock, 6000, 2, &SSHRetryClassifier{})
	_, err := client.Call(context.Background(), "test.method", nil)

	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	// Initial call + 2 retries = 3 total
	if callCount != 3 {
		t.Errorf("expected 3 calls (1 + 2 retries), got %d", callCount)
	}
	if !errors.Is(err, context.DeadlineExceeded) && !containsString(err.Error(), "after 2 retries") {
		t.Errorf("expected retry exhaustion error, got: %v", err)
	}
}

func TestRateLimitedClient_Call_ContextCancellation(t *testing.T) {
	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return nil, errors.New("connection refused")
		},
	}

	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Call(ctx, "test.method", nil)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_Call_RetriesDisabled(t *testing.T) {
	callCount := 0

	mock := &MockClient{
		CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			return nil, errors.New("connection refused")
		},
	}

	// maxRetries = 0 means no retries
	client := NewRateLimitedClient(mock, 6000, 0, &SSHRetryClassifier{})
	client.maxRetries = 0 // Explicitly set to 0
	_, err := client.Call(context.Background(), "test.method", nil)

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (retries disabled), got %d", callCount)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client -run TestRateLimitedClient_Call -v
```

Expected: FAIL - `Call` method not implemented

**Step 3: Write the implementation**

Add to `internal/client/ratelimit.go`:

```go
// Call executes a method with rate limiting and retry logic.
func (c *RateLimitedClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Wait for rate limiter (respects context cancellation)
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

		// Don't retry non-retriable errors
		if !c.classifier.IsRetriable(err) {
			return nil, err
		}

		// Don't sleep after final attempt
		if attempt == c.maxRetries {
			break
		}

		// Backoff before retry
		delay := CalculateBackoff(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr)
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/client -run TestRateLimitedClient_Call -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ratelimit.go internal/client/ratelimit_test.go
git commit -m "feat(client): implement Call with rate limiting and retry"
```

---

## Task 6: Implement CallAndWait method

**Files:**
- Modify: `internal/client/ratelimit.go`
- Modify: `internal/client/ratelimit_test.go`

**Step 1: Write the failing tests**

Add to `internal/client/ratelimit_test.go`:

```go
func TestRateLimitedClient_CallAndWait_Success(t *testing.T) {
	expected := json.RawMessage(`{"job": "complete"}`)
	mock := &MockClient{
		CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			return expected, nil
		},
	}

	client := NewRateLimitedClient(mock, 60, 3, &SSHRetryClassifier{})
	result, err := client.CallAndWait(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestRateLimitedClient_CallAndWait_RetryOnTransientError(t *testing.T) {
	callCount := 0
	expected := json.RawMessage(`{"job": "complete"}`)

	mock := &MockClient{
		CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
			callCount++
			if callCount < 2 {
				return nil, errors.New("Unexpected closure of remote connection")
			}
			return expected, nil
		},
	}

	client := NewRateLimitedClient(mock, 6000, 3, &SSHRetryClassifier{})
	result, err := client.CallAndWait(context.Background(), "test.method", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	if string(result) != string(expected) {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client -run TestRateLimitedClient_CallAndWait -v
```

Expected: FAIL - `CallAndWait` method not implemented

**Step 3: Write the implementation**

Add to `internal/client/ratelimit.go`:

```go
// CallAndWait executes a method that returns a job, with rate limiting and retry.
func (c *RateLimitedClient) CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		result, err := c.client.CallAndWait(ctx, method, params)
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

		delay := CalculateBackoff(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", c.maxRetries, lastErr)
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/client -run TestRateLimitedClient_CallAndWait -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ratelimit.go internal/client/ratelimit_test.go
git commit -m "feat(client): implement CallAndWait with rate limiting and retry"
```

---

## Task 7: Implement delegating methods for Client interface

**Files:**
- Modify: `internal/client/ratelimit.go`
- Modify: `internal/client/ratelimit_test.go`

**Step 1: Write the failing test**

Add to `internal/client/ratelimit_test.go`:

```go
import (
	"io/fs"

	"github.com/deevus/terraform-provider-truenas/internal/api"
)

func TestRateLimitedClient_DelegatingMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("GetVersion", func(t *testing.T) {
		expected := api.Version{Edition: api.EditionScale, Major: 24, Minor: 10}
		mock := &MockClient{
			GetVersionFunc: func(ctx context.Context) (api.Version, error) {
				return expected, nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		result, err := client.GetVersion(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		called := false
		mock := &MockClient{
			WriteFileFunc: func(ctx context.Context, path string, params WriteFileParams) error {
				called = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.WriteFile(ctx, "/test", DefaultWriteFileParams([]byte("content")))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected WriteFile to be called on underlying client")
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		expected := []byte("file content")
		mock := &MockClient{
			ReadFileFunc: func(ctx context.Context, path string) ([]byte, error) {
				return expected, nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		result, err := client.ReadFile(ctx, "/test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != string(expected) {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("Close", func(t *testing.T) {
		closed := false
		mock := &MockClient{
			CloseFunc: func() error {
				closed = true
				return nil
			},
		}
		client := NewRateLimitedClient(mock, 60, 3, nil)

		err := client.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !closed {
			t.Error("expected Close to be called on underlying client")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/client -run TestRateLimitedClient_DelegatingMethods -v
```

Expected: FAIL - methods not implemented

**Step 3: Write the implementation**

Add to `internal/client/ratelimit.go`:

```go
// GetVersion delegates to the underlying client (no rate limiting needed for cached call).
func (c *RateLimitedClient) GetVersion(ctx context.Context) (api.Version, error) {
	return c.client.GetVersion(ctx)
}

// WriteFile delegates to the underlying client.
func (c *RateLimitedClient) WriteFile(ctx context.Context, path string, params WriteFileParams) error {
	return c.client.WriteFile(ctx, path, params)
}

// ReadFile delegates to the underlying client.
func (c *RateLimitedClient) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return c.client.ReadFile(ctx, path)
}

// DeleteFile delegates to the underlying client.
func (c *RateLimitedClient) DeleteFile(ctx context.Context, path string) error {
	return c.client.DeleteFile(ctx, path)
}

// RemoveDir delegates to the underlying client.
func (c *RateLimitedClient) RemoveDir(ctx context.Context, path string) error {
	return c.client.RemoveDir(ctx, path)
}

// RemoveAll delegates to the underlying client.
func (c *RateLimitedClient) RemoveAll(ctx context.Context, path string) error {
	return c.client.RemoveAll(ctx, path)
}

// FileExists delegates to the underlying client.
func (c *RateLimitedClient) FileExists(ctx context.Context, path string) (bool, error) {
	return c.client.FileExists(ctx, path)
}

// Chown delegates to the underlying client.
func (c *RateLimitedClient) Chown(ctx context.Context, path string, uid, gid int) error {
	return c.client.Chown(ctx, path, uid, gid)
}

// ChmodRecursive delegates to the underlying client.
func (c *RateLimitedClient) ChmodRecursive(ctx context.Context, path string, mode fs.FileMode) error {
	return c.client.ChmodRecursive(ctx, path, mode)
}

// MkdirAll delegates to the underlying client.
func (c *RateLimitedClient) MkdirAll(ctx context.Context, path string, mode fs.FileMode) error {
	return c.client.MkdirAll(ctx, path, mode)
}

// Close closes the underlying client.
func (c *RateLimitedClient) Close() error {
	return c.client.Close()
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/client -run TestRateLimitedClient_DelegatingMethods -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/client/ratelimit.go internal/client/ratelimit_test.go
git commit -m "feat(client): implement delegating methods for Client interface"
```

---

## Task 8: Add rate_limit and max_retries to provider schema

**Files:**
- Modify: `internal/provider/provider.go`
- Modify: `internal/provider/provider_test.go`

**Step 1: Write the failing test**

Add to `internal/provider/provider_test.go` (or create if doesn't exist):

```go
func TestProviderSchema_RateLimitAttributes(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Check rate_limit attribute exists
	rateLimitAttr, ok := resp.Schema.Attributes["rate_limit"]
	if !ok {
		t.Error("expected rate_limit attribute in schema")
	} else {
		if rateLimitAttr.IsRequired() {
			t.Error("rate_limit should be optional")
		}
	}

	// Check max_retries attribute exists
	maxRetriesAttr, ok := resp.Schema.Attributes["max_retries"]
	if !ok {
		t.Error("expected max_retries attribute in schema")
	} else {
		if maxRetriesAttr.IsRequired() {
			t.Error("max_retries should be optional")
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/provider -run TestProviderSchema_RateLimitAttributes -v
```

Expected: FAIL - attributes not found

**Step 3: Write the implementation**

Modify `internal/provider/provider.go`:

Add to `TrueNASProviderModel`:
```go
type TrueNASProviderModel struct {
	Host       types.String   `tfsdk:"host"`
	AuthMethod types.String   `tfsdk:"auth_method"`
	SSH        *SSHBlockModel `tfsdk:"ssh"`
	RateLimit  types.Int64    `tfsdk:"rate_limit"`
	MaxRetries types.Int64    `tfsdk:"max_retries"`
}
```

Add to schema attributes in `Schema()` method:
```go
"rate_limit": schema.Int64Attribute{
	Description: "Maximum API calls per minute. Default: 18. " +
		"TrueNAS allows 20 auth attempts per minute; this provides a safety margin.",
	Optional: true,
},
"max_retries": schema.Int64Attribute{
	Description: "Maximum retry attempts for transient connection errors. Default: 3. " +
		"Set to 0 to disable retries.",
	Optional: true,
},
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/provider -run TestProviderSchema_RateLimitAttributes -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/provider/provider.go internal/provider/provider_test.go
git commit -m "feat(provider): add rate_limit and max_retries schema attributes"
```

---

## Task 9: Wire RateLimitedClient in provider Configure

**Files:**
- Modify: `internal/provider/provider.go`

**Step 1: Update imports**

Add import for client package if not already present (it should be).

**Step 2: Modify Configure method**

Update `internal/provider/provider.go` Configure method after creating sshClient:

```go
func (p *TrueNASProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// ... existing code up to creating sshClient ...

	// Create SSH client (validates config and applies defaults)
	sshClient, err := client.NewSSHClient(sshConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create SSH Client",
			err.Error(),
		)
		return
	}

	// Apply rate limiting defaults
	rateLimit := 0 // 0 means use default (18)
	if !config.RateLimit.IsNull() {
		rateLimit = int(config.RateLimit.ValueInt64())
	}

	maxRetries := -1 // -1 means use default (3)
	if !config.MaxRetries.IsNull() {
		maxRetries = int(config.MaxRetries.ValueInt64())
	}

	// Wrap client with rate limiting and retry
	rateLimitedClient := client.NewRateLimitedClient(
		sshClient,
		rateLimit,
		maxRetries,
		&client.SSHRetryClassifier{},
	)

	// Set client for data sources and resources
	resp.DataSourceData = rateLimitedClient
	resp.ResourceData = rateLimitedClient
}
```

**Step 3: Run all tests**

Run:
```bash
go test ./... -v
```

Expected: All PASS

**Step 4: Commit**

```bash
git add internal/provider/provider.go
git commit -m "feat(provider): wire RateLimitedClient in Configure"
```

---

## Task 10: Verify coverage and run full test suite

**Files:** None (verification only)

**Step 1: Run full test suite**

Run:
```bash
mise run test
```

Expected: All tests pass

**Step 2: Check coverage**

Run:
```bash
mise run coverage
```

Expected: `internal/client` coverage should be >= 81.6% (baseline)

**Step 3: Run linter**

Run:
```bash
mise run lint
```

Expected: No errors

**Step 4: Final commit if any fixes needed**

If coverage or linting revealed issues, fix them and commit:
```bash
git add -A
git commit -m "fix: address coverage/lint issues"
```

---

## Manual Testing Checklist

After implementation, test against a real TrueNAS instance:

- [ ] Apply 10+ resources without rate limit errors
- [ ] Verify calls are paced (add debug logging if needed)
- [ ] Test `rate_limit = 5` to verify slower pacing
- [ ] Test `max_retries = 0` to verify retries disabled
- [ ] Simulate network blip and verify retry recovery
- [ ] Verify validation errors (`[EINVAL]`) fail immediately without retry delay
