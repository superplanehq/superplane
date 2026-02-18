package gcp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_ExchangeToken(t *testing.T) {
	ctx := context.Background()
	audience := "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/oidc"
	oidcToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.sig"

	t.Run("success returns access token and expires_in", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"ya29.abc","expires_in":3600,"token_type":"Bearer"}`)),
				},
			},
		}
		token, expiresIn, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.NoError(t, err)
		assert.Equal(t, "ya29.abc", token)
		assert.Equal(t, 3600*time.Second, expiresIn)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, stsTokenURL, req.URL.String())
		assert.Empty(t, req.Header.Get("Authorization"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Contains(t, string(body), audience)
		assert.Contains(t, string(body), oidcToken)
	})

	t.Run("non-200 status returns error with API message", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"code":400,"message":"Invalid audience","status":"INVALID_ARGUMENT"}}`)),
				},
			},
		}
		_, _, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "Invalid audience")
	})

	t.Run("non-200 with non-JSON body uses raw body", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("internal error")),
				},
			},
		}
		_, _, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "internal error")
	})

	t.Run("invalid JSON response returns parse error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{not json`)),
				},
			},
		}
		_, _, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse STS response")
	})

	t.Run("empty access_token returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"","expires_in":3600,"token_type":"Bearer"}`)),
				},
			},
		}
		_, _, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing access_token")
	})

	t.Run("zero or negative expires_in defaults to one hour", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"tok","expires_in":0,"token_type":"Bearer"}`)),
				},
			},
		}
		_, expiresIn, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.NoError(t, err)
		assert.Equal(t, time.Hour, expiresIn)
	})

	t.Run("HTTP Do error returns wrapped error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{}}
		_, _, err := ExchangeToken(ctx, httpCtx, oidcToken, audience)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "STS request failed")
	})
}
