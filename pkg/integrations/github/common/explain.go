package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
)

const (
	githubErrorBodyLimit = 8 * 1024
	githubErrorTextLimit = 512
)

type explainingTransport struct {
	base http.RoundTripper
}

type githubAPIError struct {
	Message string               `json:"message"`
	Errors  []githubAPIErrorItem `json:"errors"`
}

type githubAPIErrorItem struct {
	Message string `json:"message"`
}

func WrapInstallationTransport(base http.RoundTripper) http.RoundTripper {
	return &explainingTransport{base: base}
}

func (t *explainingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err == nil {
		return resp, nil
	}
	return resp, ExplainError(err)
}

func ExplainError(err error) error {
	if err == nil {
		return nil
	}

	var response *github.ErrorResponse
	if errors.As(err, &response) {
		return explainGitHubResponse(err, response)
	}

	var installation *ghinstallation.HTTPError
	if errors.As(err, &installation) {
		return explainInstallationError(err, installation)
	}

	return err
}

func explainGitHubResponse(err error, response *github.ErrorResponse) error {
	if response == nil {
		return err
	}

	msg := response.Message
	for _, inner := range response.Errors {
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
	return errors.New(msg)
}

func explainInstallationError(err error, installation *ghinstallation.HTTPError) error {
	if installation == nil || installation.Response == nil || installation.Response.Body == nil {
		return err
	}

	body := installation.Response.Body
	installation.Response.Body = nil
	raw, ok := readGitHubBody(body)
	if !ok {
		return err
	}

	logInstallationRefusal(err, installation.Response, raw)
	reason := githubRefusalReason(raw)
	if reason == "" {
		return err
	}
	return fmt.Errorf("%s: %s: %w", httpStatusText(installation.Response), reason, err)
}

func readGitHubBody(body io.ReadCloser) (string, bool) {
	raw, err := io.ReadAll(io.LimitReader(body, githubErrorBodyLimit))
	_ = body.Close()
	if err != nil && len(raw) == 0 {
		return "", false
	}
	return string(raw), true
}

func logInstallationRefusal(err error, resp *http.Response, body string) {
	if body == "" {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"status": httpStatusText(resp),
		"body":   truncateBytes(body, githubErrorBodyLimit),
	}).Error("GitHub installation token request failed")
}

func githubRefusalReason(raw string) string {
	var payload githubAPIError
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		if reason := joinGitHubMessages(payload.Message, payload.Errors); reason != "" {
			return reason
		}
	}
	return truncateRunes(strings.TrimSpace(raw), githubErrorTextLimit)
}

func joinGitHubMessages(message string, details []githubAPIErrorItem) string {
	parts := make([]string, 0, 1+len(details))
	if text := strings.TrimSpace(message); text != "" {
		parts = append(parts, text)
	}
	for _, detail := range details {
		text := strings.TrimSpace(detail.Message)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, ": ")
}

func httpStatusText(resp *http.Response) string {
	if resp == nil {
		return "0"
	}
	if status := strings.TrimSpace(resp.Status); status != "" {
		return status
	}
	if resp.StatusCode != 0 {
		return strconv.Itoa(resp.StatusCode)
	}
	return "0"
}

func truncateBytes(value string, limit int) string {
	if limit < 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}

func truncateRunes(value string, limit int) string {
	if limit < 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
