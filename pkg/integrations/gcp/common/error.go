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
