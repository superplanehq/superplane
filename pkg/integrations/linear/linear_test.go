package linear

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

func Test__Linear__Sync(t *testing.T) {
	integration := &Linear{}

	t.Run("missing apiToken -> error", func(t *testing.T) {
		ctx := core.SyncContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
		}
		err := integration.Sync(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiToken")
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		body := `{"errors":[{"message":"Unauthorized"}]}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
				},
			},
		}
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "bad-token"},
		}
		ctx := core.SyncContext{
			Configuration: map[string]any{"apiToken": "bad-token"},
			HTTP:          httpCtx,
			Integration:   appCtx,
		}
		err := integration.Sync(ctx)
		require.Error(t, err)
	})

	t.Run("successful sync", func(t *testing.T) {
		viewerResp := `{"data":{"viewer":{"id":"u1","name":"User","email":"u@x.com"}}}`
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"t1","name":"Team 1","key":"T1"}]}}}`
		labelsResp := `{"data":{"organization":{"labels":{"nodes":[{"id":"l1","name":"Bug"}]}}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(viewerResp))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(labelsResp))},
			},
		}
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}
		ctx := core.SyncContext{
			Configuration: map[string]any{"apiToken": "test-token"},
			HTTP:          httpCtx,
			Integration:   appCtx,
		}
		err := integration.Sync(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})
}
