package dokploy

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

func Test__Dokploy_DeployApplication__Setup(t *testing.T) {
	component := &DeployApplication{}

	t.Run("missing application -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "application is required")
	})

	t.Run("blank application -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"application": "   "}})
		require.ErrorContains(t, err, "application is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"application": "app-1"}})
		require.NoError(t, err)
	})
}

func Test__Dokploy_DeployApplication__Execute(t *testing.T) {
	component := &DeployApplication{}

	// Dokploy returns `true` on Cloud and an empty body on self-hosted, so
	// success is determined by the status code rather than the payload.
	t.Run("queues deploy and emits metadata", func(t *testing.T) {
		httpCtx := okResponse(`true`)
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"application": "app-1",
				"title":       "Release v1.4.0",
				"description": "Queued by SuperPlane",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, DeployApplicationPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "app-1", data["applicationId"])
		assert.Equal(t, "Release v1.4.0", data["title"])
		assert.Equal(t, "Queued by SuperPlane", data["description"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/api/application.deploy", request.URL.Path)
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
		assert.Equal(t, "dokploy_test_key", request.Header.Get("x-api-key"))

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sent := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &sent))
		assert.Equal(t, "app-1", sent["applicationId"])
		assert.Equal(t, "Release v1.4.0", sent["title"])
		assert.Equal(t, "Queued by SuperPlane", sent["description"])
	})

	t.Run("empty response body is still a success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           okResponse(``),
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration:  map[string]any{"application": "app-1"},
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
	})

	t.Run("optional fields are omitted from request and payload", func(t *testing.T) {
		httpCtx := okResponse(`true`)
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration:  map[string]any{"application": "app-1"},
		})

		require.NoError(t, err)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "app-1", data["applicationId"])
		assert.NotContains(t, data, "title")
		assert.NotContains(t, data, "description")

		body, err := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, err)

		sent := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &sent))
		assert.NotContains(t, sent, "title")
		assert.NotContains(t, sent, "description")
	})

	t.Run("API error -> wrapped", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Application not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"application": "app-1"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deploy application")
		assert.Contains(t, err.Error(), "Application not found")
	})
}
