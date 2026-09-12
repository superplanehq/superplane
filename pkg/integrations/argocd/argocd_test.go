package argocd

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

func Test__ArgoCD__Sync(t *testing.T) {
	integration := &ArgoCD{}

	t.Run("missing server URL returns an error", func(t *testing.T) {
		err := integration.Sync(syncContext(map[string]any{"authToken": "token"}, &contexts.HTTPContext{}))

		require.ErrorContains(t, err, "serverUrl is required")
	})

	t.Run("missing authentication token returns an error", func(t *testing.T) {
		err := integration.Sync(syncContext(map[string]any{"serverUrl": "https://argocd.example.com"}, &contexts.HTTPContext{}))

		require.ErrorContains(t, err, "authToken is required")
	})

	t.Run("invalid server URL returns an error", func(t *testing.T) {
		err := integration.Sync(syncContext(map[string]any{
			"serverUrl": "argocd.example.com",
			"authToken": "token",
		}, &contexts.HTTPContext{}))

		require.ErrorContains(t, err, "invalid serverUrl")
	})

	t.Run("valid credentials mark the integration ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"Version":"v2.14.0"}`)),
			}},
		}

		ctx := syncContext(map[string]any{
			"serverUrl": "https://argocd.example.com/",
			"authToken": "token",
		}, httpCtx)

		err := integration.Sync(ctx)

		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.Integration.(*contexts.IntegrationContext).State)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://argocd.example.com/api/version", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "Bearer token", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("API credential rejection returns an error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid token"}`)),
			}},
		}

		err := integration.Sync(syncContext(map[string]any{
			"serverUrl": "https://argocd.example.com",
			"authToken": "token",
		}, httpCtx))

		require.ErrorContains(t, err, "failed to verify Argo CD credentials")
		require.ErrorContains(t, err, "invalid token")
	})
}

func syncContext(configuration map[string]any, httpCtx core.HTTPContext) core.SyncContext {
	integration := &contexts.IntegrationContext{Configuration: configuration}

	return core.SyncContext{
		Configuration: configuration,
		HTTP:          httpCtx,
		Integration:   integration,
	}
}
