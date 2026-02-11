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
	msg := strings.TrimSpace(e.Message)
	// Avoid leading ": " when code is empty or message already starts with punctuation
	if e.Code == "" {
		return strings.TrimPrefix(msg, ": ")
	}
	if msg != "" {
		return fmt.Sprintf("%s: %s", e.Code, msg)
	}
	return e.Code
}

func ParseError(body []byte) *Error {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := extractString(payload["__type"])
	if code == "" {
		code = extractString(payload["code"])
	}
	if code == "" {
		code = extractString(payload["Code"])
	}

	message := extractString(payload["message"])
	if message == "" {
		message = extractString(payload["Message"])
	}

	if errPayload, ok := payload["Error"].(map[string]any); ok {
		if code == "" {
			code = extractString(errPayload["code"])
		}
		if code == "" {
			code = extractString(errPayload["Code"])
		}
		if code == "" {
			code = extractString(errPayload["type"])
		}
		if message == "" {
			message = extractString(errPayload["message"])
		}
		if message == "" {
			message = extractString(errPayload["Message"])
		}
	}

	code = normalizeCode(code)
	if code == "" && message == "" {
		return nil
	}

	return &Error{Code: code, Message: message}
}

func extractString(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func normalizeCode(code string) string {
	if code == "" {
		return ""
	}

	parts := strings.Split(code, "#")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return code
}

func IsAlreadyExistsErr(err error) bool {
	var awsErr *Error
	if errors.As(err, &awsErr) {
		return strings.Contains(awsErr.Code, "ResourceAlreadyExists")
	}

	return false
}

func IsNotFoundErr(err error) bool {
	var awsErr *Error
	if errors.As(err, &awsErr) {
		return strings.Contains(awsErr.Code, "ResourceNotFound")
	}

	return false
}
