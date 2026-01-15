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
