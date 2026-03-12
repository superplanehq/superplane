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

func Test__Render_Deploy__Setup(t *testing.T) {
	component := &Deploy{}

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

func Test__Render_Deploy__Execute(t *testing.T) {
	component := &Deploy{}

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

func Test__Render_Deploy__HandleWebhook(t *testing.T) {
	component := &Deploy{}

	payload := map[string]any{
		"id":        "evt-cph1rs3idesc73a2b2mg",
		"type":      "deploy_ended",
		"timestamp": "2026-02-08T21:08:59.718Z",
		"serviceId": "srv-cukouhrtq21c73e9scng",
		"data": map[string]any{
			"id":        "evt-cph1rs3idesc73a2b2mg",
			"serviceId": "srv-cukouhrtq21c73e9scng",
			"status":    "live",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)

	t.Run("uses event details to resolve deploy and emit result", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-cph1rs3idesc73a2b2mg","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"deployId":"dep-cukouhrtq21c73e9scng","status":"live"}}`,
					)),
				},
			},
		}

		lookupOrder := []string{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-cukouhrtq21c73e9scng",
					Status:    "build_in_progress",
					ServiceID: "srv-cukouhrtq21c73e9scng",
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
				lookupOrder = append(lookupOrder, key+":"+value)
				if key == "deploy_id" && value == "dep-cukouhrtq21c73e9scng" {
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
		assert.Equal(t, []string{
			"deploy_id:dep-cukouhrtq21c73e9scng",
		}, lookupOrder)

		updatedMetadata, ok := metadataCtx.Metadata.(DeployExecutionMetadata)
		require.True(t, ok)
		require.NotNil(t, updatedMetadata.Deploy)
		assert.Equal(t, "live", updatedMetadata.Deploy.Status)
		assert.Equal(t, "2026-02-08T21:08:59.718Z", updatedMetadata.Deploy.FinishedAt)

		assert.Equal(t, DeploySuccessOutputChannel, executionState.Channel)
		assert.Equal(t, DeployPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "dep-cukouhrtq21c73e9scng", data["deployId"])
		assert.Equal(t, "live", data["status"])
		assert.Equal(t, "evt-cph1rs3idesc73a2b2mg", data["eventId"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/v1/events/evt-cph1rs3idesc73a2b2mg")
	})

	t.Run("event without deploy details is ignored", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-cph1rs3idesc73a2b2mg","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"autoscaling_config_changed","details":null}`,
					)),
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
				if key == "deploy_id" {
					return &core.ExecutionContext{
						ExecutionState: executionState,
					}, nil
				}
				return nil, assert.AnError
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Empty(t, executionState.Channel)
	})

	t.Run("event id from data.id resolves deploy", func(t *testing.T) {
		payload := map[string]any{
			"type": "deploy_ended",
			"data": map[string]any{
				"id":        "evt-cph1rs3idesc73a2b2mg",
				"serviceId": "srv-cukouhrtq21c73e9scng",
				"status":    "live",
			},
		}
		body, marshalErr := json.Marshal(payload)
		require.NoError(t, marshalErr)

		headers := buildSignedHeaders(secret, body)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-cph1rs3idesc73a2b2mg","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"deployId":"dep-cukouhrtq21c73e9scng","status":"live"}}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-cukouhrtq21c73e9scng",
					Status:    "build_in_progress",
					ServiceID: "srv-cukouhrtq21c73e9scng",
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:        body,
			Headers:     headers,
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Webhook:     &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == "deploy_id" && value == "dep-cukouhrtq21c73e9scng" {
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
		assert.Equal(t, DeploySuccessOutputChannel, executionState.Channel)
	})

	t.Run("event details with generic id resolve deploy", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-cph1rs3idesc73a2b2mg","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"id":"dep-cukouhrtq21c73e9scng","status":"live"}}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-cukouhrtq21c73e9scng",
					Status:    "build_in_progress",
					ServiceID: "srv-cukouhrtq21c73e9scng",
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:        body,
			Headers:     headers,
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Webhook:     &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == "deploy_id" && value == "dep-cukouhrtq21c73e9scng" {
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
		assert.Equal(t, DeploySuccessOutputChannel, executionState.Channel)
	})

	t.Run("event details with nested deploy resolve deploy", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evt-cph1rs3idesc73a2b2mg","timestamp":"2026-02-08T21:08:59.718Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"deploy":{"id":"dep-cukouhrtq21c73e9scng","status":"live"}}}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadataCtx := &contexts.MetadataContext{
			Metadata: DeployExecutionMetadata{
				Deploy: &DeployMetadata{
					ID:        "dep-cukouhrtq21c73e9scng",
					Status:    "build_in_progress",
					ServiceID: "srv-cukouhrtq21c73e9scng",
				},
			},
		}

		status, webhookErr := component.HandleWebhook(core.WebhookRequestContext{
			Body:        body,
			Headers:     headers,
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Webhook:     &contexts.NodeWebhookContext{Secret: secret},
			FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
				if key == "deploy_id" && value == "dep-cukouhrtq21c73e9scng" {
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
		assert.Equal(t, DeploySuccessOutputChannel, executionState.Channel)
	})
}
