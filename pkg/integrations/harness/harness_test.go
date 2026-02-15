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
					Body:       io.NopCloser(strings.NewReader(`{"data":{"name":"Test User"}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"content":[]}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token-123",
			"accountId": "acc-123",
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
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/pipeline/api/pipelines/list")
		assert.Contains(t, httpContext.Requests[1].URL.RawQuery, "accountIdentifier=acc-123")
	})

	t.Run("missing accountId -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken": "token-123",
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "accountId is required")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("invalid account scope -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"name":"Test User"}}`)),
				},
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid account"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token-123",
			"accountId": "wrong-account",
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "failed to verify account scope")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("project scope without orgId -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"apiToken":  "token-123",
			"accountId": "acc-123",
			"projectId": "proj-123",
		}}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "orgId is required when projectId is set")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})
}
