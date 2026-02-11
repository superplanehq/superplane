package render

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

func Test__Render_GetService__Execute(t *testing.T) {
	component := &GetService{}

	t.Run("valid input -> gets service and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"srv-abc123","name":"my-api","type":"web_service","suspended":"not_suspended","autoDeploy":"yes","repo":"https://github.com/org/repo","branch":"main","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z"}`,
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
				"service": "srv-abc123",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "render.service", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emitted := readMap(executionState.Payloads[0])
		payload := readMap(emitted["data"])
		assert.Equal(t, "srv-abc123", payload["id"])
		assert.Equal(t, "my-api", payload["name"])
		assert.Equal(t, "not_suspended", payload["suspended"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	})

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{},
		})

		require.ErrorContains(t, err, "serviceID is required")
	})
}

func Test__Render_UpdateEnvVars__Execute(t *testing.T) {
	component := &UpdateEnvVars{}

	t.Run("valid input -> updates env vars and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"key":"K1","value":"V1"},{"key":"K2","value":"V2"}]`,
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
				"service": "srv-abc123",
				"envVars": []any{
					map[string]any{"key": "K1", "value": "V1"},
					map[string]any{"key": "K2", "value": "V2"},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "render.envVars.updated", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emitted := readMap(executionState.Payloads[0])
		payload := readMap(emitted["data"])
		assert.Equal(t, "srv-abc123", payload["serviceId"])
		
		envVars := payload["envVars"].([]any)
		require.Len(t, envVars, 2)
		assert.Equal(t, "K1", readMap(envVars[0])["key"])
	})
}

func Test__Render_Rollback__Execute(t *testing.T) {
	component := &Rollback{}

	t.Run("valid input -> triggers rollback and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-new789","status":"created","createdAt":"2026-02-05T16:20:00Z","finishedAt":""}`,
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
				"service":  "srv-abc123",
				"deployId": "dep-old456",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "render.deploy.rollback", executionState.Type)
		
		emitted := readMap(executionState.Payloads[0])
		payload := readMap(emitted["data"])
		assert.Equal(t, "dep-new789", payload["deployId"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	})
}

func Test__Render_PurgeCache__Execute(t *testing.T) {
	component := &PurgeCache{}

	t.Run("valid input -> purges cache and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"service": "srv-abc123",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "render.cache.purged", executionState.Type)

		emitted := readMap(executionState.Payloads[0])
		payload := readMap(emitted["data"])
		assert.Equal(t, "srv-abc123", payload["serviceId"])
		
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
	})
}

func Test__Render_CancelDeploy__Execute(t *testing.T) {
	component := &CancelDeploy{}

	t.Run("valid input -> cancels deploy and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-xyz789","status":"canceled","createdAt":"2026-02-05T16:10:00Z","finishedAt":"2026-02-05T16:12:00Z"}`,
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
				"service":  "srv-abc123",
				"deployId": "dep-xyz789",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "render.deploy.canceled", executionState.Type)
		
		emitted := readMap(executionState.Payloads[0])
		payload := readMap(emitted["data"])
		assert.Equal(t, "dep-xyz789", payload["deployId"])
		assert.Equal(t, "canceled", payload["status"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	})
}

func Test__Render_GetDeploy__Execute(t *testing.T) {
	component := &GetDeploy{}

	t.Run("valid input -> gets deploy and emits result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-xyz789","status":"live","createdAt":"2026-02-05T16:10:00Z","finishedAt":"2026-02-05T16:15:00Z"}`,
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
				"service":  "srv-abc123",
				"deployId": "dep-xyz789",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "render.deploy", executionState.Type)
		
		emitted := readMap(executionState.Payloads[0])
		payload := readMap(emitted["data"])
		assert.Equal(t, "dep-xyz789", payload["deployId"])
		assert.Equal(t, "live", payload["status"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	})
}
