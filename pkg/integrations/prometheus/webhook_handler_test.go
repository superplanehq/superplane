package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type testWebhookContext struct {
	secret        []byte
	configuration any
}

func (t *testWebhookContext) GetID() string                 { return "wh_123" }
func (t *testWebhookContext) GetURL() string                { return "https://example.com/webhook" }
func (t *testWebhookContext) GetSecret() ([]byte, error)    { return t.secret, nil }
func (t *testWebhookContext) GetMetadata() any              { return nil }
func (t *testWebhookContext) GetConfiguration() any         { return t.configuration }
func (t *testWebhookContext) SetSecret(secret []byte) error { t.secret = secret; return nil }

func Test__PrometheusWebhookHandler__CompareConfig(t *testing.T) {
	handler := &PrometheusWebhookHandler{}

	equal, err := handler.CompareConfig(struct{}{}, struct{}{})
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = handler.CompareConfig(map[string]any{"a": "b"}, map[string]any{"x": "y"})
	require.NoError(t, err)
	assert.True(t, equal)
}

func Test__PrometheusWebhookHandler__Setup(t *testing.T) {
	handler := &PrometheusWebhookHandler{}
	webhookCtx := &testWebhookContext{configuration: struct{}{}}
	integrationCtx := &contexts.IntegrationContext{}

	_, err := handler.Setup(core.WebhookHandlerContext{
		Webhook:     webhookCtx,
		Integration: integrationCtx,
	})
	require.NoError(t, err)
	assert.Empty(t, webhookCtx.secret)
}

func Test__PrometheusWebhookHandler__Cleanup(t *testing.T) {
	handler := &PrometheusWebhookHandler{}
	err := handler.Cleanup(core.WebhookHandlerContext{})
	require.NoError(t, err)
}
