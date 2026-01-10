package client

import (
	"fmt"
	"regexp"
	"strings"
)

// TrueNASError represents a parsed error from the TrueNAS middleware.
type TrueNASError struct {
	Code       string // e.g., "EINVAL", "ENOENT", "EFAULT"
	Message    string // Raw error from middleware
	Field      string // Which field caused error (if applicable)
	JobID      int64  // For job-related errors
	Suggestion string // Actionable guidance
}

func (e *TrueNASError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)
	if e.Suggestion != "" {
		sb.WriteString("\n\nSuggestion: ")
		sb.WriteString(e.Suggestion)
	}
	return sb.String()
}

var (
	// Matches [CODE] at start of error message
	errorCodeRegex = regexp.MustCompile(`^\[([A-Z]+)\]\s*(.*)`)
	// Matches field path before colon
	fieldRegex = regexp.MustCompile(`^([\w.]+):\s*(.*)`)
)

// errorSuggestions maps error codes to helpful suggestions.
var errorSuggestions = map[string]string{
	"EINVAL": "Check the configuration schema. A field may be invalid or unexpected.",
	"ENOENT": "Resource not found. It may have been deleted outside Terraform.",
	"EFAULT": "Container failed to start. Check compose_config and image availability.",
	"EEXIST": "Resource already exists. Import it or choose a different name.",
}

// ParseTrueNASError parses a raw error string from midclt into a structured error.
func ParseTrueNASError(raw string) *TrueNASError {
	err := &TrueNASError{
		Code:    "UNKNOWN",
		Message: raw,
	}

	// Try to extract error code
	if matches := errorCodeRegex.FindStringSubmatch(raw); len(matches) == 3 {
		err.Code = matches[1]
		remainder := matches[2]

		// Try to extract field from remainder
		if fieldMatches := fieldRegex.FindStringSubmatch(remainder); len(fieldMatches) == 3 {
			err.Field = fieldMatches[1]
		}
	}

	// Add suggestion based on code
	if suggestion, ok := errorSuggestions[err.Code]; ok {
		err.Suggestion = suggestion
	}

	return err
}

// NewConnectionError creates an error for SSH connection failures.
func NewConnectionError(host string, port int, cause error) *TrueNASError {
	return &TrueNASError{
		Code:       "ECONNREFUSED",
		Message:    fmt.Sprintf("Cannot connect to %s:%d: %v", host, port, cause),
		Suggestion: "Verify SSH credentials, network connectivity, and that the TrueNAS server is running.",
	}
}

// NewTimeoutError creates an error for job timeouts.
func NewTimeoutError(jobID int64, duration string) *TrueNASError {
	return &TrueNASError{
		Code:       "ETIMEDOUT",
		Message:    fmt.Sprintf("Operation timed out after %s", duration),
		JobID:      jobID,
		Suggestion: "Increase the timeout or check the TrueNAS server for issues.",
	}
}
