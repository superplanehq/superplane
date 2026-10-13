package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type gcpErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type GCPAPIError struct {
	StatusCode int
	Message    string
}

func (e *GCPAPIError) Error() string {
	return fmt.Sprintf("GCP request failed (%d): %s", e.StatusCode, e.Message)
}

func ParseGCPError(statusCode int, body []byte) error {
	var apiErr gcpErrorResponse
	message := strings.TrimSpace(string(body))
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		message = apiErr.Error.Message
	}
	return &GCPAPIError{StatusCode: statusCode, Message: message}
}

func IsAlreadyExistsError(err error) bool {
	var apiErr *GCPAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}

func IsNotFoundError(err error) bool {
	var apiErr *GCPAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// APIErrorMessage formats an API error for execution state output. On a 403
// it appends the IAM role hint, since a missing role is the most common cause
// of failed GCP API calls from the integration.
func APIErrorMessage(err error, action, roleHint string) string {
	var apiErr *GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHint)
	}
	return fmt.Sprintf("%s: %v", action, err)
}
