package client

import (
	"testing"
)

func TestParseError_EINVAL(t *testing.T) {
	raw := `[EINVAL] storage.config.host_path_config.path: Field was not expected`

	err := ParseTrueNASError(raw)

	if err.Code != "EINVAL" {
		t.Errorf("expected code EINVAL, got %s", err.Code)
	}
	if err.Field != "storage.config.host_path_config.path" {
		t.Errorf("expected field storage.config.host_path_config.path, got %s", err.Field)
	}
}

func TestParseError_ENOENT(t *testing.T) {
	raw := `[ENOENT] Unable to locate app "nonexistent"`

	err := ParseTrueNASError(raw)

	if err.Code != "ENOENT" {
		t.Errorf("expected code ENOENT, got %s", err.Code)
	}
	if err.Suggestion == "" {
		t.Error("expected suggestion for ENOENT error")
	}
}

func TestParseError_EFAULT(t *testing.T) {
	raw := `[EFAULT] Failed 'up' action for app caddy: image not found`

	err := ParseTrueNASError(raw)

	if err.Code != "EFAULT" {
		t.Errorf("expected code EFAULT, got %s", err.Code)
	}
}

func TestParseError_NoCode(t *testing.T) {
	raw := `Some random error message`

	err := ParseTrueNASError(raw)

	if err.Code != "UNKNOWN" {
		t.Errorf("expected code UNKNOWN, got %s", err.Code)
	}
	if err.Message != raw {
		t.Errorf("expected message to be preserved")
	}
}

func TestTrueNASError_Error(t *testing.T) {
	err := &TrueNASError{
		Code:       "EINVAL",
		Message:    "test message",
		Suggestion: "try this",
	}

	errStr := err.Error()

	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}
}
