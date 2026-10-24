package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const installationLibraryMessage = `received non 2xx response status "403 Forbidden" when fetching https://api.github.com/app/installations/168860812/access_tokens`

func TestExplainError_InstallationSuspended(t *testing.T) {
	hook := captureErrorLogs(t)
	body := &closeTracker{Reader: strings.NewReader(
		`{"message":"This installation has been suspended","errors":[{"message":"organization policy"}],"documentation_url":"https://docs.github.com/rest/apps/installations#sentinel-body-field"}`,
	)}
	cause := installationHTTPError(http.StatusForbidden, "403 Forbidden", body)
	wrapped := fmt.Errorf("could not refresh installation id 168860812's token: %w", cause)

	explained := ExplainError(wrapped)

	require.Error(t, explained)
	assert.True(t, strings.HasPrefix(explained.Error(), "403"))
	assert.Contains(t, explained.Error(), "This installation has been suspended")
	assert.Contains(t, explained.Error(), "organization policy")
	assert.NotContains(t, explained.Error(), "sentinel-body-field")
	assert.Equal(t, http.StatusForbidden, StatusCode(explained))
	assert.ErrorIs(t, explained, cause)
	assert.True(t, body.closed)

	require.NotEmpty(t, hook.entries)
	entry := hook.entries[0]
	assert.Equal(t, log.ErrorLevel, entry.Level)
	assert.Contains(t, fmt.Sprint(entry.Data["status"]), "403")
	assert.Contains(t, fmt.Sprint(entry.Data["body"]), "sentinel-body-field")

	assert.NotPanics(t, func() {
		again := ExplainError(explained)
		assert.Contains(t, again.Error(), "This installation has been suspended")
		assert.Equal(t, http.StatusForbidden, StatusCode(again))
	})
}

func TestExplainError_PlainTextBody(t *testing.T) {
	plain := strings.Repeat("x", githubErrorTextLimit) + "TAIL"
	body := &closeTracker{Reader: strings.NewReader(plain)}
	wrapped := fmt.Errorf(
		"could not refresh installation id 1's token: %w",
		installationHTTPError(http.StatusForbidden, "403 Forbidden", body),
	)

	var explained error
	assert.NotPanics(t, func() {
		explained = ExplainError(wrapped)
	})

	require.Error(t, explained)
	assert.Contains(t, explained.Error(), strings.Repeat("x", githubErrorTextLimit))
	assert.NotContains(t, explained.Error(), "TAIL")
	assert.True(t, strings.HasPrefix(explained.Error(), "403"))
	assert.True(t, body.closed)
}

func TestExplainError_NilBody(t *testing.T) {
	cause := &ghinstallation.HTTPError{
		Message:  installationLibraryMessage,
		Response: &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden"},
	}
	wrapped := fmt.Errorf("could not refresh installation id 168860812's token: %w", cause)

	var explained error
	assert.NotPanics(t, func() {
		explained = ExplainError(wrapped)
	})

	require.Error(t, explained)
	assert.Equal(t, wrapped.Error(), explained.Error())
	assert.Contains(t, explained.Error(), installationLibraryMessage)
	assert.Equal(t, http.StatusForbidden, StatusCode(explained))
}

func TestExplainError_NilResponse(t *testing.T) {
	cause := &ghinstallation.HTTPError{Message: installationLibraryMessage}
	wrapped := fmt.Errorf("token refresh failed: %w", cause)

	assert.NotPanics(t, func() {
		assert.Equal(t, wrapped.Error(), ExplainError(wrapped).Error())
	})
}

func TestExplainError_ClosedBody(t *testing.T) {
	cause := installationHTTPError(http.StatusForbidden, "403 Forbidden", closedBody{})
	wrapped := fmt.Errorf("could not refresh installation id 168860812's token: %w", cause)

	var explained error
	assert.NotPanics(t, func() {
		explained = ExplainError(wrapped)
	})

	require.Error(t, explained)
	assert.Equal(t, wrapped.Error(), explained.Error())
	assert.Contains(t, explained.Error(), installationLibraryMessage)
}

func TestExplainError_GitHubErrorResponse(t *testing.T) {
	err := &github.ErrorResponse{
		Message: "Validation Failed",
		Errors: []github.Error{{
			Message: "No commits between base and head",
		}},
	}

	explained := ExplainError(err)

	require.Error(t, explained)
	assert.Equal(t, "Validation Failed: No commits between base and head", explained.Error())
}

func TestExplainError_InstallationTransportRefresh(t *testing.T) {
	body := &closeTracker{Reader: strings.NewReader(`{"message":"This installation has been suspended"}`)}
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Status:     "403 Forbidden",
			Body:       body,
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	installation, err := ghinstallation.New(base, 1, 168860812, testAppPrivateKey(t))
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/acme/web", nil)
	require.NoError(t, err)

	_, err = WrapInstallationTransport(installation).RoundTrip(req)

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "403"))
	assert.Contains(t, err.Error(), "This installation has been suspended")
	assert.Equal(t, http.StatusForbidden, StatusCode(err))
	assert.True(t, body.closed)
}

type closeTracker struct {
	io.Reader
	closed bool
}

func (c *closeTracker) Read(p []byte) (int, error) {
	if c.closed {
		return 0, errors.New("http: read on closed response body")
	}
	return c.Reader.Read(p)
}

func (c *closeTracker) Close() error {
	c.closed = true
	return nil
}

type closedBody struct{}

func (closedBody) Read([]byte) (int, error) {
	return 0, errors.New("http: read on closed response body")
}

func (closedBody) Close() error {
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func installationHTTPError(status int, statusText string, body io.ReadCloser) *ghinstallation.HTTPError {
	return &ghinstallation.HTTPError{
		Message: installationLibraryMessage,
		Response: &http.Response{
			StatusCode: status,
			Status:     statusText,
			Body:       body,
			Header:     make(http.Header),
		},
	}
}

func testAppPrivateKey(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

type logCapture struct {
	mu      sync.Mutex
	entries []*log.Entry
}

func (l *logCapture) Levels() []log.Level {
	return []log.Level{log.ErrorLevel}
}

func (l *logCapture) Fire(entry *log.Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	return nil
}

func captureErrorLogs(t *testing.T) *logCapture {
	t.Helper()
	hook := &logCapture{}
	logger := log.StandardLogger()
	previous := logger.ReplaceHooks(make(log.LevelHooks))
	for _, hooks := range previous {
		for _, existing := range hooks {
			logger.AddHook(existing)
		}
	}
	logger.AddHook(hook)
	t.Cleanup(func() {
		logger.ReplaceHooks(previous)
	})
	return hook
}
