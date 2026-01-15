package resources

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// App state constants matching TrueNAS API values.
const (
	AppStateRunning   = "RUNNING"
	AppStateStopped   = "STOPPED"
	AppStateDeploying = "DEPLOYING"
	AppStateStarting  = "STARTING"
	AppStateStopping  = "STOPPING"
	AppStateCrashed   = "CRASHED"
)

// normalizeDesiredState converts user input to uppercase for comparison.
func normalizeDesiredState(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// isStableState returns true if the state is stable (not transitional).
func isStableState(state string) bool {
	switch state {
	case AppStateRunning, AppStateStopped, AppStateCrashed:
		return true
	default:
		return false
	}
}

// isValidDesiredState returns true if the state is valid for desired_state.
func isValidDesiredState(state string) bool {
	normalized := normalizeDesiredState(state)
	return normalized == AppStateRunning || normalized == AppStateStopped
}

// stateQueryFunc is a function type for querying app state.
type stateQueryFunc func(ctx context.Context, name string) (string, error)

// waitForStableState polls until the app reaches a stable state or timeout.
// Returns the final state or an error if timeout is reached.
func waitForStableState(ctx context.Context, name string, timeout time.Duration, queryState stateQueryFunc) (string, error) {
	const pollInterval = 5 * time.Second

	deadline := time.Now().Add(timeout)

	for {
		state, err := queryState(ctx, name)
		if err != nil {
			return "", fmt.Errorf("failed to query app state: %w", err)
		}

		if isStableState(state) {
			return state, nil
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for app state: app %q is stuck in %s state after %v", name, state, timeout)
		}

		// For testing, use shorter interval if timeout is very short
		sleepDuration := pollInterval
		if timeout < pollInterval {
			sleepDuration = timeout / 10
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(sleepDuration):
			// Continue polling
		}
	}
}
