package launchdarkly

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

func Test__LaunchDarkly__Sync(t *testing.T) {
	integration := &LaunchDarkly{}

	t.Run("missing token -> error", func(t *testing.T) {
		ctx := core.SyncContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
		}

		err := integration.Sync(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiAccessToken is required")
	})

	t.Run("valid token -> ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiAccessToken": "token-123",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})
}
