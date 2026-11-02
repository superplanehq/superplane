package runner_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

// These cases exercise provider responses through the exchange boundary. They
// protect reconnect decisions without bypassing OAuth error parsing.
func TestRefreshCodexTokensClassifiesProviderResponses(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		body      string
		permanent bool
		code      string
	}{
		{name: "expired top-level code", status: 400, body: `{"code":"refresh_token_expired"}`, permanent: true, code: "refresh_token_expired"},
		{name: "reused string error", status: 403, body: `{"error":"refresh_token_reused"}`, permanent: true, code: "refresh_token_reused"},
		{name: "invalidated nested code", status: 400, body: `{"error":{"code":"refresh_token_invalidated"}}`, permanent: true, code: "refresh_token_invalidated"},
		{name: "invalid grant string error", status: 400, body: `{"error":"invalid_grant"}`, permanent: true, code: "invalid_grant"},
		{name: "invalid refresh token top-level code", status: 403, body: `{"code":"invalid_refresh_token"}`, permanent: true, code: "invalid_refresh_token"},
		{name: "expired token nested code", status: 403, body: `{"error":{"code":"token_expired"}}`, permanent: true, code: "token_expired"},
		{name: "case-insensitive terminal code", status: 400, body: `{"error":"INVALID_GRANT"}`, permanent: true, code: "invalid_grant"},
		{name: "rate limit takes precedence", status: 429, body: `{"code":"refresh_token_expired"}`, code: "refresh_token_expired"},
		{name: "server error takes precedence", status: 500, body: `{"error":"refresh_token_reused"}`, code: "refresh_token_reused"},
		{name: "service unavailable takes precedence", status: 503, body: `{"error":{"code":"invalid_grant"}}`, code: "invalid_grant"},
		{name: "unauthorized without token code", status: 401, body: `{"error":"access_denied"}`, permanent: true},
		{name: "unauthorized malformed response", status: 401, body: `not JSON`, permanent: true},
		{name: "unknown bad request", status: 400, body: `{"code":"unrecognized_error"}`},
		{name: "unknown forbidden response", status: 403, body: `{"error":{"code":"unrecognized_error"}}`},
		{name: "quota exhaustion", status: 403, body: `{"error":{"code":"insufficient_quota"}}`},
		{name: "usage limit", status: 429, body: `{"error":"usage_limit_reached"}`},
		{name: "terminal code on unrecognized status", status: 404, body: `{"error":"invalid_grant"}`, code: "invalid_grant"},
		{name: "malformed error response", status: 400, body: `{"error":`},
		{name: "non-string error response", status: 400, body: `{"error":{"code":42}}`},
		{name: "malformed successful response", status: 200, body: `not JSON`},
		{name: "missing access token", status: 200, body: `{"refresh_token":"replacement"}`},
		{name: "free-text error cannot disclose tokens", status: 401, body: `{"error":"rejected refresh token: fabricated-secret-value"}`, permanent: true},
		{name: "opaque code cannot disclose tokens", status: 401, body: `{"code":"fabricated_secret_value"}`, permanent: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: codexRefreshTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})}

			tokens, err := runner.RefreshCodexTokens(context.Background(), client, "stored-refresh-token")

			var refreshError *runner.CodexTokenRefreshError
			require.ErrorAs(t, err, &refreshError)
			assert.Equal(t, test.permanent, refreshError.Permanent)
			assert.Equal(t, test.code, refreshError.Code)
			assert.Empty(t, tokens)
			assert.NotContains(t, err.Error(), "fabricated")
		})
	}
}

func TestRefreshCodexTokensRetriesTransportFailures(t *testing.T) {
	for name, failure := range map[string]error{
		"network failure": errors.New("network unavailable with fabricated-secret-value"),
		"timeout":         context.DeadlineExceeded,
		"canceled":        context.Canceled,
	} {
		t.Run(name, func(t *testing.T) {
			client := &http.Client{Transport: codexRefreshTransport(func(*http.Request) (*http.Response, error) {
				return nil, failure
			})}

			_, err := runner.RefreshCodexTokens(context.Background(), client, "stored-refresh-token")

			var refreshError *runner.CodexTokenRefreshError
			require.ErrorAs(t, err, &refreshError)
			assert.False(t, refreshError.Permanent)
			assert.Empty(t, refreshError.Code)
			assert.NotContains(t, err.Error(), "fabricated")
		})
	}
}

type codexRefreshTransport func(*http.Request) (*http.Response, error)

func (f codexRefreshTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
