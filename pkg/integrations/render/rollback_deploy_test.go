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

func Test__Render_RollbackDeploy__Setup(t *testing.T) {
	component := &RollbackDeploy{}

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

func Test__Render_RollbackDeploy__Execute(t *testing.T) {
	component := &RollbackDeploy{}

	t.Run("valid configuration -> triggers rollback and schedules poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-new","status":"build_in_progress","trigger":"rollback","createdAt":"2026-02-05T16:18:00.000000Z"}`,
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
			Configuration:  map[string]any{"service": "srv-123", "deployId": "dep-old"},
		})

		require.NoError(t, err)
		// Component waits for deploy_ended; no emit yet
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, "dep-new", executionState.KVs["deploy_id"])
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, RollbackDeployPollInterval, requestCtx.Duration)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/rollback")

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)

		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "dep-old", payload["deployId"])
	})
}

func Test__Render_RollbackDeploy__HandleWebhook(t *testing.T) {
	component := &RollbackDeploy{}

	payload := map[string]any{
		"id":        "evt-rollback123",
		"type":      "deploy_ended",
		"timestamp": "2026-02-08T21:08:59.718Z",
		"serviceId": "srv-123",
		"data": map[string]any{
			"id":        "evt-rollback123",
			"serviceId": "srv-123",
			"status":    "live",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)

	t.Run("deploy_ended webhook resolves and emits rollback result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-rollback123","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-123","type":"deploy_ended","details":{"deployId":"dep-new","status":"live"}}`,
					)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-new",
					Status:    "build_in_progress",
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
				if key == "deploy_id" && value == "dep-new" {
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
		assert.Equal(t, "live", updatedMetadata.Deploy.Status)

		assert.Equal(t, RollbackDeploySuccessOutputChannel, executionState.Channel)
		assert.Equal(t, RollbackDeployPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "dep-new", data["deployId"])
		assert.Equal(t, "live", data["status"])
	})

	t.Run("failed rollback -> emits to failed channel", func(t *testing.T) {
		failedPayload := map[string]any{
			"id":        "evt-rollback456",
			"type":      "deploy_ended",
			"timestamp": "2026-02-08T21:08:59.718Z",
			"serviceId": "srv-123",
			"data": map[string]any{
				"id":        "evt-rollback456",
				"serviceId": "srv-123",
				"status":    "build_failed",
			},
		}

		failedBody, marshalErr := json.Marshal(failedPayload)
		require.NoError(t, marshalErr)
		failedHeaders := buildSignedHeaders(secret, failedBody)

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-rollback456","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-123","type":"deploy_ended","details":{"deployId":"dep-new","status":"build_failed"}}`,
					)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-new",
					Status:    "build_in_progress",
					ServiceID: "srv-123",
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:        failedBody,
			Headers:     failedHeaders,
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Webhook:     &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == "deploy_id" && value == "dep-new" {
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
		assert.Equal(t, RollbackDeployFailedOutputChannel, executionState.Channel)
	})
}
