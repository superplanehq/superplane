package argocd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ArgoCD_SyncApplication__Setup(t *testing.T) {
	component := &SyncApplication{}

	t.Run("missing project returns an error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"application": "payments"}})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing application returns an error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"project": "platform"}})

		require.ErrorContains(t, err, "application is required")
	})

	t.Run("revision and per-source revisions cannot both be set", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"project":     "platform",
			"application": "payments",
			"revision":    "main",
			"revisions":   []any{"main", "production"},
		}})

		require.ErrorContains(t, err, "revision and revisions cannot both be set")
	})

	t.Run("valid configuration succeeds", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"project":              "platform",
			"application":          "payments",
			"applicationNamespace": "argocd",
			"revision":             "main",
			"prune":                true,
			"dryRun":               true,
			"force":                true,
			"strategy":             "hook",
		}})

		require.NoError(t, err)
	})
}

func Test__ArgoCD_SyncApplication__Execute(t *testing.T) {
	component := &SyncApplication{}

	t.Run("syncs an application and emits returned state", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"metadata": {"name": "payments", "namespace": "argocd", "uid": "app-1"},
					"spec": {"project": "platform", "source": {"repoURL": "https://github.com/example/platform.git", "path": "apps/payments", "targetRevision": "main"}, "destination": {"namespace": "payments"}},
					"status": {"sync": {"status": "Synced", "revision": "abc123"}, "health": {"status": "Healthy"}}
				}`)),
			}},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"serverUrl": "https://argocd.example.com",
				"authToken": "token",
			}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"project":              "platform",
				"application":          "payments",
				"applicationNamespace": "argocd",
				"revision":             "main",
				"prune":                true,
				"dryRun":               true,
				"force":                true,
				"strategy":             "hook",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, SyncApplicationPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)["data"].(ApplicationOutput)
		assert.Equal(t, "payments", payload.Application.Name)
		assert.Equal(t, "Synced", payload.Sync.Status)
		assert.Equal(t, "Healthy", payload.Health.Status)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/api/v1/applications/payments/sync", request.URL.Path)
		assert.Equal(t, "Bearer token", request.Header.Get("Authorization"))
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		assert.Equal(t, "payments", body["name"])
		assert.Equal(t, "platform", body["project"])
		assert.Equal(t, "argocd", body["appNamespace"])
		assert.Equal(t, "main", body["revision"])
		assert.Equal(t, true, body["prune"])
		assert.Equal(t, true, body["dryRun"])
		assert.Equal(t, map[string]any{
			"hook": map[string]any{
				"syncStrategyApply": map[string]any{"force": true},
			},
		}, body["strategy"])
	})

	t.Run("syncs a multi-source application with per-source revisions", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"metadata":{"name":"payments"}}`)),
			}},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"serverUrl": "https://argocd.example.com",
				"authToken": "token",
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"project":     "platform",
				"application": "payments",
				"revisions":   []any{"main", "production"},
			},
		})

		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.NewDecoder(httpCtx.Requests[0].Body).Decode(&body))
		assert.Equal(t, []any{"main", "production"}, body["revisions"])
		assert.NotContains(t, body, "revision")
	})

	t.Run("Argo CD error is returned", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusConflict,
				Body:       io.NopCloser(strings.NewReader(`{"error":"operation already running"}`)),
			}},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"serverUrl": "https://argocd.example.com",
				"authToken": "token",
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"project": "platform", "application": "payments"},
		})

		require.ErrorContains(t, err, "operation already running")
	})
}
