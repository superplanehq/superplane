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

func Test__Render_CancelDeploy__Setup(t *testing.T) {
	component := &CancelDeploy{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"deployId": "dep-123"}})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing deployId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"service": "srv-123"}})
		require.ErrorContains(t, err, "deployId is required")
	})

	t.Run("valid configuration -> success and requests webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "srv-123", "deployId": "dep-123"},
			Integration:   integrationCtx,
		})
		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})
}

func Test__Render_CancelDeploy__Execute(t *testing.T) {
	component := &CancelDeploy{}

	t.Run("cancel in progress -> stores metadata and schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-123","status":"deactivating","trigger":"api","createdAt":"2026-02-05T16:10:00.000000Z"}`,
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
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration:  map[string]any{"service": "srv-123", "deployId": "dep-123"},
		})

		require.NoError(t, err)
		// Component waits for deploy_ended; no emit yet
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, "dep-123", executionState.KVs["deploy_id"])
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, CancelDeployPollInterval, requestCtx.Duration)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/deploys/dep-123/cancel")
	})

	t.Run("cancel already finished -> emits immediately", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-123","status":"canceled","trigger":"api","createdAt":"2026-02-05T16:10:00.000000Z","finishedAt":"2026-02-05T16:12:00.000000Z"}`,
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
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			Configuration:  map[string]any{"service": "srv-123", "deployId": "dep-123"},
		})

		require.NoError(t, err)
		assert.Equal(t, CancelDeploySuccessOutputChannel, executionState.Channel)
		assert.Equal(t, CancelDeployPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})
}

func Test__Render_CancelDeploy__HandleWebhook(t *testing.T) {
	component := &CancelDeploy{}

	payload := map[string]any{
		"id":        "evt-cancel123",
		"type":      "deploy_ended",
		"timestamp": "2026-02-08T21:08:59.718Z",
		"serviceId": "srv-123",
		"data": map[string]any{
			"id":        "evt-cancel123",
			"serviceId": "srv-123",
			"status":    "canceled",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)

	t.Run("deploy_ended webhook resolves and emits cancel result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-cancel123","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-123","type":"deploy_ended","details":{"deployId":"dep-123","status":"canceled"}}`,
					)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-123",
					Status:    "deactivating",
					ServiceID: "srv-123",
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:        body,
			Headers:     headers,
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Webhook:     &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == "deploy_id" && value == "dep-123" {
					return &core.ExecutionContext{
						Metadata:       metadataCtx,
						ExecutionState: executionState,
					}, nil
				}

				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)

		updatedMetadata, ok := metadataCtx.Metadata.(DeployExecutionMetadata)
		require.True(t, ok)
		require.NotNil(t, updatedMetadata.Deploy)
		assert.Equal(t, "canceled", updatedMetadata.Deploy.Status)

		assert.Equal(t, CancelDeploySuccessOutputChannel, executionState.Channel)
		assert.Equal(t, CancelDeployPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "dep-123", data["deployId"])
		assert.Equal(t, "canceled", data["status"])
	})
}
