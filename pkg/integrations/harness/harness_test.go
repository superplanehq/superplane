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
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"name":"Test User"}}`)),
			}},
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
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "token-123", httpContext.Requests[0].Header.Get("x-api-key"))
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ng/api/user/currentUser")
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
}
