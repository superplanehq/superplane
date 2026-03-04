package fluxcd

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

func Test__ReconcileSource__Setup(t *testing.T) {
	component := &ReconcileSource{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing kind -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"kind": "",
				"name": "my-app",
			},
		})

		require.ErrorContains(t, err, "kind is required")
	})

	t.Run("missing name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"kind": "Kustomization",
				"name": "",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("unsupported kind -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"kind": "UnsupportedKind",
				"name": "my-app",
			},
		})

		require.ErrorContains(t, err, "unsupported resource kind")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"kind": "Kustomization",
				"name": "my-app",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid HelmRelease configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"kind":      "HelmRelease",
				"namespace": "default",
				"name":      "nginx",
			},
		})

		require.NoError(t, err)
	})
}

func Test__ReconcileSource__Execute(t *testing.T) {
	component := &ReconcileSource{}

	t.Run("successful reconciliation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
						"kind": "Kustomization",
						"metadata": {
							"name": "my-app",
							"namespace": "flux-system",
							"annotations": {
								"reconcile.fluxcd.io/requestedAt": "2026-01-15T10:30:00Z"
							},
							"resourceVersion": "12346"
						},
						"status": {
							"lastAppliedRevision": "main@sha1:abc123",
							"lastAttemptedRevision": "main@sha1:abc123"
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"server":    "https://kubernetes.example.com:6443",
				"token":     "test-token",
				"namespace": "flux-system",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"kind":      "Kustomization",
				"namespace": "flux-system",
				"name":      "my-app",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "fluxcd.reconciliation", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Contains(t, req.URL.String(), "/apis/kustomize.toolkit.fluxcd.io/v1/namespaces/flux-system/kustomizations/my-app")
		assert.Equal(t, "application/merge-patch+json", req.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"kustomizations.kustomize.toolkit.fluxcd.io \"my-app\" not found"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"server":    "https://kubernetes.example.com:6443",
				"token":     "test-token",
				"namespace": "flux-system",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"kind":      "Kustomization",
				"namespace": "flux-system",
				"name":      "my-app",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to reconcile resource")
	})

	t.Run("uses integration default namespace when not specified", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"metadata": {
							"name": "my-app",
							"namespace": "flux-system",
							"annotations": {},
							"resourceVersion": "100"
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"server":    "https://kubernetes.example.com:6443",
				"token":     "test-token",
				"namespace": "custom-ns",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"kind": "Kustomization",
				"name": "my-app",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/namespaces/custom-ns/")
	})

	t.Run("HelmRelease reconciliation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"metadata": {
							"name": "nginx",
							"namespace": "default",
							"annotations": {},
							"resourceVersion": "200"
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"server":    "https://kubernetes.example.com:6443",
				"token":     "test-token",
				"namespace": "flux-system",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"kind":      "HelmRelease",
				"namespace": "default",
				"name":      "nginx",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/apis/helm.toolkit.fluxcd.io/v2/namespaces/default/helmreleases/nginx")
	})
}

func Test__ReconcileSource__OutputChannels(t *testing.T) {
	component := &ReconcileSource{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
