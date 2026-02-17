package honeycomb

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withDefaultTransport(t *testing.T, rt roundTripFunc) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func Test__Honeycomb__Sync(t *testing.T) {
	h := &Honeycomb{}

	t.Run("missing api key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site": BaseURLUS,
			},
		}

		err := h.Sync(core.SyncContext{
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})

		require.Error(t, err)
	})

	t.Run("valid key -> ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   BaseURLUS,
				"apiKey": "test-api-key",
			},
		}

		err := h.Sync(core.SyncContext{
			Integration: integrationCtx,
			HTTP:        httpCtx,
		})

		require.NoError(t, err)
		require.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 1)
		require.Contains(t, httpCtx.Requests[0].URL.String(), BaseURLUS+"/1/auth")
	})

	t.Run("invalid key -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"Unknown API key"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   BaseURLUS,
				"apiKey": "bad-key",
			},
		}

		err := h.Sync(core.SyncContext{
			Integration: integrationCtx,
			HTTP:        httpCtx,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid api key")
		require.NotEqual(t, "ready", integrationCtx.State)
	})
}
