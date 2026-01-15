package resources

import "testing"

func TestNormalizeDesiredState(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"running", "RUNNING"},
		{"RUNNING", "RUNNING"},
		{"Running", "RUNNING"},
		{"stopped", "STOPPED"},
		{"STOPPED", "STOPPED"},
		{"  running  ", "RUNNING"},
	}

	for _, tc := range tests {
		got := normalizeDesiredState(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeDesiredState(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsStableState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{AppStateRunning, true},
		{AppStateStopped, true},
		{AppStateCrashed, true},
		{AppStateDeploying, false},
		{AppStateStarting, false},
		{AppStateStopping, false},
		{"UNKNOWN", false},
	}

	for _, tc := range tests {
		got := isStableState(tc.state)
		if got != tc.expected {
			t.Errorf("isStableState(%q) = %v, want %v", tc.state, got, tc.expected)
		}
	}
}

func TestIsValidDesiredState(t *testing.T) {
	tests := []struct {
		state    string
		expected bool
	}{
		{"running", true},
		{"RUNNING", true},
		{"stopped", true},
		{"STOPPED", true},
		{"deploying", false},
		{"crashed", false},
		{"paused", false},
	}

	for _, tc := range tests {
		got := isValidDesiredState(tc.state)
		if got != tc.expected {
			t.Errorf("isValidDesiredState(%q) = %v, want %v", tc.state, got, tc.expected)
		}
	}
}
