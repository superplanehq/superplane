package notion

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

func integration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}}
}

type failingHTTP struct {
	requests int
}

func (h *failingHTTP) Do(*http.Request) (*http.Response, error) {
	h.requests++
	return nil, errors.New("connection reset")
}

func TestNotionSync(t *testing.T) {
	t.Run("verifies the token before marking the connection ready", func(t *testing.T) {
		h := &contexts.HTTPContext{Responses: []*http.Response{response(200, `{"object":"user","id":"bot","type":"bot"}`)}}
		ctx := integration()
		err := (&Notion{}).Sync(core.SyncContext{Configuration: ctx.Configuration, Integration: ctx, HTTP: h})
		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.State)
		require.Len(t, h.Requests, 1)
		assert.Equal(t, "https://api.notion.com/v1/users/me", h.Requests[0].URL.String())
		assert.Equal(t, http.MethodGet, h.Requests[0].Method)
		assert.Equal(t, "Bearer test-token", h.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, "2026-03-11", h.Requests[0].Header.Get("Notion-Version"))
		assert.Nil(t, ctx.Metadata)
	})

	for _, status := range []int{401, 403, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			h := &contexts.HTTPContext{Responses: []*http.Response{response(status, `{"code":"unauthorized","message":"test-token"}`)}}
			ctx := integration()
			err := (&Notion{}).Sync(core.SyncContext{Configuration: ctx.Configuration, Integration: ctx, HTTP: h})
			require.Error(t, err)
			assert.NotEqual(t, "ready", ctx.State)
			assert.NotContains(t, err.Error(), "test-token")
			var apiError *APIError
			require.ErrorAs(t, err, &apiError)
			assert.Equal(t, status, apiError.StatusCode)
		})
	}

	t.Run("empty token does not make a request", func(t *testing.T) {
		h := &contexts.HTTPContext{}
		ctx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "  "}}
		err := (&Notion{}).Sync(core.SyncContext{Configuration: ctx.Configuration, Integration: ctx, HTTP: h})
		require.Error(t, err)
		assert.Empty(t, h.Requests)
		assert.NotEqual(t, "ready", ctx.State)
	})

	t.Run("transport failure is not labeled invalid credentials", func(t *testing.T) {
		ctx := integration()
		err := (&Notion{}).Sync(core.SyncContext{Configuration: ctx.Configuration, Integration: ctx, HTTP: &failingHTTP{}})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", ctx.State)
	})
}
