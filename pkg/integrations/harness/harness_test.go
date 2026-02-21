package harness

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Harness__Sync(t *testing.T) {
	integration := &Harness{}

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"defaultAccountIdentifier":"acc-123"}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"content":[]}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "token-123",
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "token-123", httpContext.Requests[0].Header.Get("x-api-key"))
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ng/api/user/currentUser")
		assert.Equal(t, "token-123", httpContext.Requests[1].Header.Get("x-api-key"))
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/ng/api/organizations")
		assert.Contains(t, httpContext.Requests[1].URL.RawQuery, "accountIdentifier=acc-123")
	})

	t.Run("nil baseURL -> defaults and still verifies", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"defaultAccountIdentifier":"acc-123"}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"content":[]}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "token-123",
			"baseURL":  nil,
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), DefaultBaseURL)
	})

	t.Run("missing api token -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL": DefaultBaseURL,
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "apiToken")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("invalid account scope -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid account"}`)),
				},
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid account"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "pat.wrong-account.test",
			"baseURL":  DefaultBaseURL,
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "failed to verify account scope")
		require.ErrorContains(t, err, "invalid account")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("service-account key currentUser 400 still verifies", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(
						`{"message":"Invalid request: Current user can be accessed only by 'USER' principal type"}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"content":[]}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "service-account-token-without-pat-prefix",
			"baseURL":  DefaultBaseURL,
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ng/api/user/currentUser")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/ng/api/organizations")
	})

	t.Run("service-account key currentUser 500 still verifies", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"Internal Server Error"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"content":[]}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "service-account-token-without-pat-prefix",
			"baseURL":  DefaultBaseURL,
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ng/api/user/currentUser")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/ng/api/organizations")
	})
}
