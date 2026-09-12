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

func Test__ArgoCD_GetApplication__Setup(t *testing.T) {
	component := &GetApplication{}

	t.Run("missing project returns an error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"application": "payments"}})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing application returns an error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"project": "platform"}})

		require.ErrorContains(t, err, "application is required")
	})

	t.Run("valid configuration succeeds", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"project":              "platform",
			"application":          "payments",
			"applicationNamespace": "argocd",
		}})

		require.NoError(t, err)
	})
}

func Test__ArgoCD_GetApplication__Execute(t *testing.T) {
	component := &GetApplication{}

	t.Run("application is emitted with delivery state", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"metadata": {"name": "payments", "namespace": "argocd", "uid": "app-1"},
					"spec": {
						"project": "platform",
						"source": {"repoURL": "https://github.com/example/platform.git", "path": "apps/payments", "targetRevision": "main"},
						"destination": {"server": "https://kubernetes.default.svc", "namespace": "payments"}
					},
					"status": {
						"sync": {"status": "Synced", "revision": "abc123"},
						"health": {"status": "Healthy", "message": "All resources are healthy"},
						"operationState": {"phase": "Succeeded", "message": "successfully synced", "startedAt": "2026-08-29T10:00:00Z", "finishedAt": "2026-08-29T10:01:00Z", "operation": {"initiatedBy": {"username": "automation"}}},
						"conditions": [{"type": "ComparisonError", "message": "none"}]
					}
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
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, GetApplicationPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		payload := executionState.Payloads[0].(map[string]any)["data"].(ApplicationOutput)
		assert.Equal(t, ApplicationReference{Name: "payments", Namespace: "argocd", UID: "app-1", Project: "platform"}, payload.Application)
		assert.Equal(t, "Synced", payload.Sync.Status)
		assert.Equal(t, "Healthy", payload.Health.Status)
		require.NotNil(t, payload.Operation)
		assert.Equal(t, "Succeeded", payload.Operation.Phase)
		assert.Equal(t, "automation", payload.Operation.InitiatedBy)
		require.Len(t, payload.Sources, 1)
		assert.Equal(t, "https://github.com/example/platform.git", payload.Sources[0].RepoURL)
		assert.Equal(t, "payments", payload.Destination.Namespace)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/api/v1/applications/payments", request.URL.Path)
		assert.Equal(t, "platform", request.URL.Query().Get("project"))
		assert.Equal(t, "argocd", request.URL.Query().Get("appNamespace"))
		assert.Equal(t, "Bearer token", request.Header.Get("Authorization"))
	})

	t.Run("Argo CD error is returned", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"message":"application is outside the selected project"}`)),
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

		require.ErrorContains(t, err, "application is outside the selected project")
	})
}

func Test__ApplicationOutputFrom__UsesAllSources(t *testing.T) {
	output := applicationOutputFrom(Application{
		Spec: ApplicationSpec{
			Source: &ApplicationSource{RepoURL: "https://github.com/example/legacy.git"},
			Sources: []ApplicationSource{
				{RepoURL: "https://github.com/example/application.git", TargetRevision: "main"},
				{RepoURL: "https://github.com/example/values.git", TargetRevision: "production"},
			},
		},
	}, "platform")

	require.Len(t, output.Sources, 2)
	assert.Equal(t, "https://github.com/example/application.git", output.Sources[0].RepoURL)
	assert.Equal(t, "https://github.com/example/values.git", output.Sources[1].RepoURL)
}
