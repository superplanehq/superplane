package render

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

func Test__Render_TriggerDeploy__Setup(t *testing.T) {
	component := &TriggerDeploy{}

	t.Run("missing serviceId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "serviceId is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
		})

		require.NoError(t, err)
	})
}

func Test__Render_TriggerDeploy__Execute(t *testing.T) {
	component := &TriggerDeploy{}

	t.Run("valid input with clear cache -> emits deploy payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-cukouhrtq21c73e9scng","status":"build_in_progress","createdAt":"2026-02-05T16:10:00.000000Z","finishedAt":null}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "rnd_test"},
			},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"serviceId":  "srv-cukouhrtq21c73e9scng",
				"clearCache": true,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, TriggerDeployPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.String(), "/v1/services/srv-cukouhrtq21c73e9scng/deploys")

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)

		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "clear", payload["clearCache"])
	})

	t.Run("missing serviceId -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "rnd_test"},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{},
		})

		require.ErrorContains(t, err, "serviceId is required")
	})

	t.Run("render API error -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"service not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "rnd_test"},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"serviceId": "srv-missing",
			},
		})

		require.Error(t, err)
	})
}
