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

func Test__Render_OnBuild__Setup(t *testing.T) {
	trigger := &OnBuild{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "rnd_test"},
		Metadata: Metadata{
			Workspace: &WorkspaceMetadata{
				ID:   "usr-123",
				Plan: "professional",
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"cursor":"x","service":{"id":"srv-cukouhrtq21c73e9scng","name":"backend-api"}}]`,
				)),
			},
		},
	}

	err := trigger.Setup(core.TriggerContext{
		HTTP:        httpCtx,
		Metadata:    metadataCtx,
		Integration: integrationCtx,
		Configuration: map[string]any{
			"service":    "srv-cukouhrtq21c73e9scng",
			"eventTypes": []string{"build_ended"},
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:   webhookStrategyIntegration,
		EventTypes: []string{"build_ended"},
	}, webhookConfiguration)
}

func Test__Render_OnBuild__Setup__OrganizationWorkspace(t *testing.T) {
	trigger := &OnBuild{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "rnd_test"},
		Metadata: Metadata{
			Workspace: &WorkspaceMetadata{
				ID:   "tea-123",
				Plan: "organization",
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"cursor":"x","service":{"id":"srv-cukouhrtq21c73e9scng","name":"backend-api"}}]`,
				)),
			},
		},
	}

	err := trigger.Setup(core.TriggerContext{
		HTTP:        httpCtx,
		Metadata:    metadataCtx,
		Integration: integrationCtx,
		Configuration: map[string]any{
			"service": "srv-cukouhrtq21c73e9scng",
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:     webhookStrategyResourceType,
		ResourceType: webhookResourceTypeBuild,
		EventTypes:   []string{"build_ended"},
	}, webhookConfiguration)
}

func Test__Render_OnBuild__HandleWebhook(t *testing.T) {
	trigger := &OnBuild{}

	payload := map[string]any{
		"type":      "build_ended",
		"timestamp": "2026-02-05T16:00:01.000000Z",
		"data": map[string]any{
			"id":          "evj-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "failed",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)
	eventCtx := &contexts.EventContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"id":"evj-cukouhrtq21c73e9scng","timestamp":"2026-02-05T16:00:01.000000Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"build_ended","details":{"buildId":"bld-cukouhrtq21c73e9scng","status":"failed"}}`,
				)),
			},
		},
	}

	status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		HTTP:          httpCtx,
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
		Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        eventCtx,
	})

	assert.Equal(t, http.StatusOK, status)
	require.NoError(t, webhookErr)
	require.Equal(t, 1, eventCtx.Count())
	assert.Equal(t, "render.build.ended", eventCtx.Payloads[0].Type)
	assert.Equal(t, map[string]any{
		"eventId":     "evj-cukouhrtq21c73e9scng",
		"buildId":     "bld-cukouhrtq21c73e9scng",
		"serviceId":   "srv-cukouhrtq21c73e9scng",
		"serviceName": "backend-api",
		"status":      "failed",
	}, eventCtx.Payloads[0].Data)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Contains(t, httpCtx.Requests[0].URL.Path, "/v1/events/evj-cukouhrtq21c73e9scng")
}

func Test__Render_OnBuild__HandleWebhook__WithoutEventResolution(t *testing.T) {
	trigger := &OnBuild{}

	payload := map[string]any{
		"type":      "build_ended",
		"timestamp": "2026-02-05T16:00:01.000000Z",
		"data": map[string]any{
			"id":          "evj-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "failed",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)
	eventCtx := &contexts.EventContext{}

	status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        eventCtx,
	})

	assert.Equal(t, http.StatusOK, status)
	require.NoError(t, webhookErr)
	require.Equal(t, 1, eventCtx.Count())
	assert.Equal(t, "render.build.ended", eventCtx.Payloads[0].Type)
	assert.Equal(t, map[string]any{
		"eventId":     "evj-cukouhrtq21c73e9scng",
		"serviceId":   "srv-cukouhrtq21c73e9scng",
		"serviceName": "backend-api",
		"status":      "failed",
	}, eventCtx.Payloads[0].Data)
}
