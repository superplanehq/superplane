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

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"key": "DATABASE_URL", "value": "postgres://..."},
		})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123", "value": "postgres://..."},
		})
		require.ErrorContains(t, err, "key is required")
	})

	t.Run("set strategy without value -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123", "key": "DATABASE_URL"},
		})
		require.ErrorContains(t, err, "value is required")
	})

	t.Run("generate strategy without value -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123", "key": "DATABASE_URL", "valueStrategy": "generate"},
		})
		require.NoError(t, err)
	})
}

func Test__Render_UpdateEnvVar__Execute(t *testing.T) {
	component := &UpdateEnvVar{}

	t.Run("set strategy -> updates env var and does not emit value by default", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"key":"DATABASE_URL","value":"postgres://example"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"service":       "srv-123",
				"key":           "DATABASE_URL",
				"valueStrategy": "set",
				"value":         "postgres://example",
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, UpdateEnvVarPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, "DATABASE_URL", data["key"])
		assert.Equal(t, false, data["valueGenerated"])
		_, hasValue := data["value"]
		assert.False(t, hasValue)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/env-vars/DATABASE_URL")

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)

		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "postgres://example", payload["value"])
		_, hasGenerateValue := payload["generateValue"]
		assert.False(t, hasGenerateValue)
	})

	t.Run("generate strategy with emitValue -> emits generated value", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"key":"API_TOKEN","value":"generated"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"service":       "srv-123",
				"key":           "API_TOKEN",
				"valueStrategy": "generate",
				"emitValue":     true,
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, UpdateEnvVarPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, "API_TOKEN", data["key"])
		assert.Equal(t, true, data["valueGenerated"])
		assert.Equal(t, "generated", data["value"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/env-vars/API_TOKEN")

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)

		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, true, payload["generateValue"])
		_, hasValue := payload["value"]
		assert.False(t, hasValue)
	})
}
