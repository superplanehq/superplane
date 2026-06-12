package coolify

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

func Test__Coolify_DeployApplication__Setup(t *testing.T) {
	component := &DeployApplication{}

	t.Run("missing application -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "application is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"application": "abc123"}})
		require.NoError(t, err)
	})
}

func Test__Coolify_DeployApplication__Execute(t *testing.T) {
	component := &DeployApplication{}

	t.Run("queues deploy and emits deployment metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"message":"Deployments queued.","deployments":[{"resource_uuid":"abc123","deployment_uuid":"deploy-xyz","message":"Deployment queued."}]}`,
					)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"application": "abc123",
				"force":       true,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, DeployApplicationPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "abc123", data["applicationUuid"])
		assert.Equal(t, true, data["force"])
		assert.Equal(t, "deploy-xyz", data["deploymentUuid"])
		assert.Equal(t, "Deployments queued.", data["message"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)

		request := httpCtx.Requests[0]
		assert.Equal(t, "/api/v1/deploy", request.URL.Path)
		query := request.URL.Query()
		assert.Equal(t, "abc123", query.Get("uuid"))
		assert.Equal(t, "true", query.Get("force"))
	})

	t.Run("force=false omits the force query param", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Deployments queued.","deployments":[]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"application": "abc123",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)

		query := httpCtx.Requests[0].URL.Query()
		assert.Equal(t, "abc123", query.Get("uuid"))
		assert.Empty(t, query.Get("force"))
	})

	t.Run("API error -> wrapped", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Application is already deploying."}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"application": "abc123"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deploy application")
		assert.Contains(t, err.Error(), "already deploying")
	})
}
