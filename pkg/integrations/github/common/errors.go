package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
)

// ExplainedGitHubError wraps an underlying GitHub error with an unpacked,
// diagnostic message including status, message, hints, and response body.
type ExplainedGitHubError struct {
	msg   string
	cause error
}

func (e *ExplainedGitHubError) Error() string {
	return e.msg
}

func (e *ExplainedGitHubError) Unwrap() error {
	return e.cause
}

// ExplainGitHubError unwraps GitHub errors into user-friendly diagnostic errors.
// It handles:
// 1. *ghinstallation.HTTPError: surfaces status code, response message, response body,
//    and contextual hints for common token mint failures (suspended installation,
//    IP allow list restrictions, uninstalled app, permission issues) while logging the full body.
// 2. *github.ErrorResponse: surfaces validation errors (e.g. 422 messages).
func ExplainGitHubError(err error) error {
	if err == nil {
		return nil
	}

	var installErr *ghinstallation.HTTPError
	if errors.As(err, &installErr) {
		return explainInstallationError(err, installErr)
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		return explainErrorResponse(err, ghErr)
	}

	return err
}

func explainInstallationError(err error, installErr *ghinstallation.HTTPError) error {
	if installErr.Response == nil {
		msg := installErr.Message
		if msg == "" && installErr.RootCause != nil {
			msg = installErr.RootCause.Error()
		}
		if msg == "" {
			msg = "failed to mint GitHub installation token"
		} else {
			msg = fmt.Sprintf("failed to mint GitHub installation token: %s", msg)
		}
		log.WithFields(log.Fields{
			"installation_id": installErr.InstallationID,
		}).Warn(msg)
		return &ExplainedGitHubError{msg: msg, cause: err}
	}

	var bodyBytes []byte
	if installErr.Response.Body != nil {
		bodyBytes, _ = io.ReadAll(installErr.Response.Body)
		_ = installErr.Response.Body.Close()
		installErr.Response.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	status := installErr.Response.Status
	if status == "" {
		status = fmt.Sprintf("%d %s", installErr.Response.StatusCode, http.StatusText(installErr.Response.StatusCode))
	}

	fields := log.Fields{
		"status":      status,
		"status_code": installErr.Response.StatusCode,
	}
	if installErr.InstallationID != 0 {
		fields["installation_id"] = installErr.InstallationID
	}
	if len(bodyBytes) > 0 {
		fields["response_body"] = string(bodyBytes)
	}
	log.WithFields(fields).Warn("GitHub installation token mint failed")

	var parsedMsg string
	if len(bodyBytes) > 0 {
		var ghResp struct {
			Message          string `json:"message"`
			DocumentationURL string `json:"documentation_url"`
		}
		if jsonErr := json.Unmarshal(bodyBytes, &ghResp); jsonErr == nil && ghResp.Message != "" {
			parsedMsg = strings.TrimSpace(ghResp.Message)
		}
	}
	rawBody := strings.TrimSpace(string(bodyBytes))

	textForHint := parsedMsg
	if textForHint == "" {
		textForHint = rawBody
	}
	if textForHint == "" {
		textForHint = installErr.Message
	}
	hint := installationErrorHint(installErr.Response.StatusCode, textForHint)

	var msg string
	switch {
	case parsedMsg != "" && rawBody != "" && rawBody != parsedMsg:
		if hint != "" {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s (hint: %s; response body: %s)", status, parsedMsg, hint, rawBody)
		} else {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s (response body: %s)", status, parsedMsg, rawBody)
		}
	case parsedMsg != "":
		if hint != "" {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s (hint: %s)", status, parsedMsg, hint)
		} else {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s", status, parsedMsg)
		}
	case rawBody != "":
		if hint != "" {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s (hint: %s)", status, rawBody, hint)
		} else {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s", status, rawBody)
		}
	default:
		fallback := installErr.Message
		if fallback == "" {
			fallback = "unknown token refresh failure"
		}
		if hint != "" {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s (hint: %s)", status, fallback, hint)
		} else {
			msg = fmt.Sprintf("GitHub installation token mint failed (status %s): %s", status, fallback)
		}
	}

	return &ExplainedGitHubError{msg: msg, cause: err}
}

func installationErrorHint(statusCode int, text string) string {
	lower := strings.ToLower(text)
	switch {
	case statusCode == http.StatusForbidden && (strings.Contains(lower, "suspended") || strings.Contains(lower, "suspension")):
		return "the GitHub App installation is suspended"
	case statusCode == http.StatusForbidden && (strings.Contains(lower, "ip allow list") || strings.Contains(lower, "ip allowlist") || strings.Contains(lower, "ip list")):
		return "organization IP allow list policy blocked token request"
	case statusCode == http.StatusNotFound || strings.Contains(lower, "not found"):
		return "the GitHub App installation was not found or has been uninstalled"
	case statusCode == http.StatusForbidden && strings.Contains(lower, "resource not accessible"):
		return "the GitHub App lacks required permissions for this repository"
	default:
		return ""
	}
}

func explainErrorResponse(err error, ghErr *github.ErrorResponse) error {
	msg := ghErr.Message
	for _, inner := range ghErr.Errors {
		if inner.Message == "" {
			continue
		}
		if msg == "" {
			msg = inner.Message
			continue
		}
		msg = fmt.Sprintf("%s: %s", msg, inner.Message)
	}

	if msg == "" {
		return err
	}
	return &ExplainedGitHubError{msg: msg, cause: err}
}
