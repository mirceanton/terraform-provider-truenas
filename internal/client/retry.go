package client

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
