package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Code
}

// ParseARMError extracts error code and message from Azure ARM REST API error body.
// See https://learn.microsoft.com/en-us/rest/api/azure/#error-response
func ParseARMError(body []byte) *Error {
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	code := strings.TrimSpace(payload.Error.Code)
	message := strings.TrimSpace(payload.Error.Message)
	if code == "" && message == "" {
		return nil
	}
	return &Error{Code: code, Message: message}
}

// ParseAzureADError extracts error from Azure AD token endpoint response.
func ParseAzureADError(body []byte) *Error {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	code := strings.TrimSpace(payload.Error)
	message := strings.TrimSpace(payload.ErrorDescription)
	if code == "" && message == "" {
		return nil
	}
	return &Error{Code: code, Message: message}
}

func IsAlreadyExistsErr(err error) bool {
	var azureErr *Error
	if errors.As(err, &azureErr) {
		return strings.Contains(azureErr.Code, "ResourceAlreadyExists") ||
			strings.Contains(azureErr.Code, "Conflict")
	}
	return false
}

func IsNotFoundErr(err error) bool {
	var azureErr *Error
	if errors.As(err, &azureErr) {
		return strings.Contains(azureErr.Code, "ResourceNotFound") ||
			strings.Contains(azureErr.Code, "NotFound")
	}
	return false
}
