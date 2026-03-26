package digitalocean

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

func Test__UpdateApp__HandleAction(t *testing.T) {
	component := &UpdateApp{}

	t.Run("deployment active -> emits app output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetDeployment response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-001",
							"phase": "ACTIVE"
						}
					}`)),
				},
				// GetApp response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "app-001",
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"},
							"live_url": "https://my-app.ondigitalocean.app",
							"default_ingress": "https://my-app.ondigitalocean.app"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-001",
				"deploymentID": "dep-001",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.app.updated", executionState.Type)
	})

	t.Run("deployment error -> fails execution with details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-002",
							"phase": "ERROR",
							"cause": "build failed",
							"progress": {
								"error_steps": 1,
								"total_steps": 3,
								"steps": [
									{"name": "build", "status": "ERROR"},
									{"name": "deploy", "status": "PENDING"}
								]
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-002",
				"deploymentID": "dep-002",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "deployment_failed", executionState.FailureReason)
		assert.Contains(t, executionState.FailureMessage, "build failed")
		assert.Contains(t, executionState.FailureMessage, "build")
	})

	t.Run("deployment building -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-003",
							"phase": "BUILDING"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-003",
				"deploymentID": "dep-003",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("already finished -> no-op", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			Finished: true,
			KVs:      map[string]string{},
		}

		requestCtx := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, requestCtx.Action)
	})

	t.Run("unknown action -> returns error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action: unknown")
	})
}
