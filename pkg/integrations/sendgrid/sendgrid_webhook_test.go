package sendgrid

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
	url           string
	configuration any
	secret        []byte
}

func (t *testWebhookContext) GetID() string                 { return "wh_123" }
func (t *testWebhookContext) GetURL() string                { return t.url }
func (t *testWebhookContext) GetSecret() ([]byte, error)    { return t.secret, nil }
func (t *testWebhookContext) GetMetadata() any              { return nil }
func (t *testWebhookContext) GetConfiguration() any         { return t.configuration }
func (t *testWebhookContext) SetSecret(secret []byte) error { t.secret = secret; return nil }

func Test__SendGrid__SetupWebhook_UsesProvidedVerificationKey(t *testing.T) {
	integration := &SendGrid{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     http.Header{},
				Request:    &http.Request{},
			},
		},
	}

	webhookCtx := &testWebhookContext{
		url:           "https://example.com/webhook",
		configuration: WebhookConfiguration{},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey":          "sg-test",
			"verificationKey": "public-key",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	_, err := integration.SetupWebhook(core.SetupWebhookContext{
		HTTP:        httpCtx,
		Webhook:     webhookCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "public-key", string(webhookCtx.secret))
	secret, ok := integrationCtx.Secrets[webhookVerificationKeySecret]
	require.True(t, ok)
	assert.Equal(t, "public-key", string(secret.Value))
}

func Test__SendGrid__SetupWebhook_EnablesSignedWebhook(t *testing.T) {
	integration := &SendGrid{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     http.Header{},
				Request:    &http.Request{},
			},
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader(`{"public_key":"public-key-from-api","enabled":true}`)),
				Header:     http.Header{},
				Request:    &http.Request{},
			},
		},
	}

	webhookCtx := &testWebhookContext{
		url:           "https://example.com/webhook",
		configuration: WebhookConfiguration{},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "sg-test"},
		Secrets:       map[string]core.IntegrationSecret{},
	}

	_, err := integration.SetupWebhook(core.SetupWebhookContext{
		HTTP:        httpCtx,
		Webhook:     webhookCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "public-key-from-api", string(webhookCtx.secret))
	secret, ok := integrationCtx.Secrets[webhookVerificationKeySecret]
	require.True(t, ok)
	assert.Equal(t, "public-key-from-api", string(secret.Value))
}

func Test__SendGrid__CleanupWebhook_DisablesWebhook(t *testing.T) {
	integration := &SendGrid{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader(`{"enabled":true,"url":"https://example.com/webhook"}`)),
				Header:     http.Header{},
				Request:    &http.Request{},
			},
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     http.Header{},
				Request:    &http.Request{},
			},
		},
	}

	webhookCtx := &testWebhookContext{
		url:           "https://example.com/webhook",
		configuration: WebhookConfiguration{},
	}

	err := integration.CleanupWebhook(core.CleanupWebhookContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sg-test"},
			Secrets:       map[string]core.IntegrationSecret{},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 2)

	getReq := httpCtx.Requests[0]
	assert.Equal(t, http.MethodGet, getReq.Method)
	assert.Contains(t, getReq.URL.String(), "/user/webhooks/event/settings")

	req := httpCtx.Requests[1]
	assert.Equal(t, http.MethodPatch, req.Method)
	assert.Contains(t, req.URL.String(), "/user/webhooks/event/settings")

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, false, payload["enabled"])
	assert.Equal(t, "https://example.com/webhook", payload["url"])
}

func Test__SendGrid__CleanupWebhook_SkipsNonHTTPS(t *testing.T) {
	integration := &SendGrid{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Body:       io.NopCloser(strings.NewReader(`{"enabled":true,"url":"http://example.com/webhook"}`)),
				Header:     http.Header{},
				Request:    &http.Request{},
			},
		},
	}

	webhookCtx := &testWebhookContext{
		url:           "http://example.com/webhook",
		configuration: WebhookConfiguration{},
	}

	err := integration.CleanupWebhook(core.CleanupWebhookContext{
		HTTP:    httpCtx,
		Webhook: webhookCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sg-test"},
			Secrets:       map[string]core.IntegrationSecret{},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)

	getReq := httpCtx.Requests[0]
	assert.Equal(t, http.MethodGet, getReq.Method)
	assert.Contains(t, getReq.URL.String(), "/user/webhooks/event/settings")
}
