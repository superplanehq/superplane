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
	h := &Harness{}

	t.Run("success validating credentials -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"SUCCESS","data":{}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{},
			Configuration: map[string]any{
				"accountId": "account-123",
				"apiToken":  "token-xyz",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ng/api/user/currentUser")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "accountIdentifier=account-123")
	})

	t.Run("failure validating credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"status":"ERROR","message":"Invalid API key"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{},
			Configuration: map[string]any{
				"accountId": "account-123",
				"apiToken":  "invalid-token",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
	})
}
