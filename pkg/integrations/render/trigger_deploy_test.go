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

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})
}

func Test__Render_TriggerDeploy__Execute(t *testing.T) {
	component := &TriggerDeploy{}

	t.Run("valid input with clear cache -> triggers deploy and schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"deploy":{"id":"dep-cukouhrtq21c73e9scng","status":"build_in_progress","createdAt":"2026-02-05T16:10:00.000000Z","finishedAt":null}}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Metadata:       metadataCtx,
			ExecutionState: executionState,
			Requests:       requestCtx,
			Configuration: map[string]any{
				"service":    "srv-cukouhrtq21c73e9scng",
				"clearCache": true,
			},
		})

		require.NoError(t, err)
		// Component waits for deploy_ended; no emit yet
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, "dep-cukouhrtq21c73e9scng", executionState.KVs["deploy_id"])
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, DeployPollInterval, requestCtx.Duration)

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

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "rnd_test"},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{},
		})

		require.ErrorContains(t, err, "service is required")
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
				"service": "srv-missing",
			},
		})

		require.Error(t, err)
	})
}
