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
