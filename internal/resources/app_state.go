package resources

import "strings"

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
