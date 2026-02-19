package grafana

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

type testWebhookContext struct {
	id            string
	url           string
	configuration any
	metadata      any
	secret        []byte
}

func (t *testWebhookContext) GetID() string                 { return t.id }
func (t *testWebhookContext) GetURL() string                { return t.url }
func (t *testWebhookContext) GetSecret() ([]byte, error)    { return t.secret, nil }
func (t *testWebhookContext) GetMetadata() any              { return t.metadata }
func (t *testWebhookContext) GetConfiguration() any         { return t.configuration }
func (t *testWebhookContext) SetSecret(secret []byte) error { t.secret = secret; return nil }

func Test__GrafanaWebhookHandler__Setup__ProvisionContactPoint(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			},
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"cp_123","name":"ignored"}`)),
			},
		},
	}

	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{"sharedSecret": "top-secret"},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, metadata)

	result, ok := metadata.(GrafanaWebhookMetadata)
	require.True(t, ok)
	assert.Equal(t, "cp_123", result.ContactPointUID)
	assert.Equal(t, []byte("top-secret"), webhookCtx.secret)

	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.String(), "/api/v1/provisioning/contact-points"))
	assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[1].URL.String(), "/api/v1/provisioning/contact-points"))

	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &payload))
	settings, ok := payload["settings"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://example.com/webhook", settings["url"])
	assert.Equal(t, "Bearer", settings["authorization_scheme"])
	assert.Equal(t, "top-secret", settings["authorization_credentials"])
}

func Test__GrafanaWebhookHandler__Setup__ManualFallbackWhenClientUnavailable(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{"sharedSecret": "top-secret"},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, metadata)
	assert.Equal(t, []byte("top-secret"), webhookCtx.secret)
}

func Test__GrafanaWebhookHandler__Setup__ManualFallbackOnNonRetriableProvisioningError(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`forbidden`)),
			},
		},
	}
	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{"sharedSecret": "top-secret"},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, metadata)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
}

func Test__GrafanaWebhookHandler__Setup__RetriesOnRetriableProvisioningError(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`internal error`)),
			},
		},
	}
	webhookCtx := &testWebhookContext{
		id:            "wh_123",
		url:           "https://example.com/webhook",
		configuration: map[string]any{"sharedSecret": "top-secret"},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.ErrorContains(t, err, "will be retried")
	assert.Nil(t, metadata)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
}

func Test__GrafanaWebhookHandler__Cleanup(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader(``)),
			},
		},
	}
	webhookCtx := &testWebhookContext{
		metadata: map[string]any{"contactPointUid": "cp_123"},
	}

	err := handler.Cleanup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.String(), "/api/v1/provisioning/contact-points/cp_123"))
}

func Test__GrafanaWebhookHandler__Cleanup__NoContactPointUIDWithoutTokenIsNoOp(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{}
	webhookCtx := &testWebhookContext{
		metadata: map[string]any{},
	}

	err := handler.Cleanup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__GrafanaWebhookHandler__Cleanup__NilMetadataIsNoOp(t *testing.T) {
	handler := &GrafanaWebhookHandler{}
	httpCtx := &contexts.HTTPContext{}
	webhookCtx := &testWebhookContext{
		metadata: nil,
	}

	err := handler.Cleanup(core.WebhookHandlerContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__GrafanaWebhookHandler__CompareConfig(t *testing.T) {
	handler := &GrafanaWebhookHandler{}

	equal, err := handler.CompareConfig(
		map[string]any{"webhookBindingKey": "node-1"},
		map[string]any{"webhookBindingKey": " node-1 "},
	)
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = handler.CompareConfig(
		map[string]any{"webhookBindingKey": "node-1", "sharedSecret": "secret-a"},
		map[string]any{"webhookBindingKey": "node-2", "sharedSecret": "secret-a"},
	)
	require.NoError(t, err)
	assert.False(t, equal)

	equal, err = handler.CompareConfig(
		map[string]any{"sharedSecret": "secret"},
		map[string]any{"sharedSecret": " secret "},
	)
	require.NoError(t, err)
	assert.True(t, equal)
}

func Test__GrafanaWebhookHandler__Merge(t *testing.T) {
	handler := &GrafanaWebhookHandler{}

	merged, changed, err := handler.Merge(
		map[string]any{"sharedSecret": "old", "webhookBindingKey": "node-1"},
		map[string]any{"sharedSecret": " new ", "webhookBindingKey": "node-1"},
	)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, OnAlertFiringConfig{SharedSecret: "new", WebhookBindingKey: "node-1"}, merged)

	merged, changed, err = handler.Merge(
		map[string]any{"sharedSecret": " same ", "webhookBindingKey": "node-1"},
		map[string]any{"sharedSecret": "same", "webhookBindingKey": "node-1"},
	)
	require.NoError(t, err)
	require.False(t, changed)
	assert.Equal(t, OnAlertFiringConfig{SharedSecret: "same", WebhookBindingKey: "node-1"}, merged)

	merged, changed, err = handler.Merge(
		map[string]any{"sharedSecret": "keep-existing", "webhookBindingKey": "node-1"},
		map[string]any{},
	)
	require.NoError(t, err)
	require.False(t, changed)
	assert.Equal(t, OnAlertFiringConfig{SharedSecret: "keep-existing", WebhookBindingKey: "node-1"}, merged)
}
