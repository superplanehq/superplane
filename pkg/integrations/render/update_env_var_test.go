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

func Test__Render_UpdateEnvVar__Setup(t *testing.T) {
	component := &UpdateEnvVar{}

	t.Run("missing serviceId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"key": "FOO", "value": "bar"},
		})
		require.ErrorContains(t, err, "serviceId is required")
	})

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"serviceId": "srv-abc123", "value": "bar"},
		})
		require.ErrorContains(t, err, "key is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"serviceId": "srv-abc123", "key": "FOO", "value": "bar"},
		})
		require.NoError(t, err)
	})
}

func Test__Render_UpdateEnvVar__Execute(t *testing.T) {
	component := &UpdateEnvVar{}

	t.Run("valid input -> updates env var and emits", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"key":"FOO","value":"bar"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"serviceId": "srv-abc123", "key": "FOO", "value": "bar"},
		})

		require.NoError(t, err)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, UpdateEnvVarPayloadType, executionState.Type)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/services/srv-abc123/env-vars/FOO")

		body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, readErr)
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "bar", payload["value"])
	})
}
