package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPRequestKind(t *testing.T) {
	t.Run("classifies planning wait as long poll", func(t *testing.T) {
		assert.Equal(t, HTTPRequestKindLongPoll, HTTPRequestKind("/api/v1/runner/planning-sessions/wait"))
		assert.True(t, IsLongPollHTTPRoute("/api/v1/runner/planning-sessions/wait"))
	})

	t.Run("classifies inbound webhooks separately from the API", func(t *testing.T) {
		assert.Equal(t, HTTPRequestKindWebhook, HTTPRequestKind("/api/v1/webhooks/{webhookID}"))
		assert.Equal(t, HTTPRequestKindWebhook, HTTPRequestKind("/webhooks/{webhookID}"))
		assert.Equal(t, HTTPRequestKindWebhook, HTTPRequestKind("/api/v1/github/app/webhook"))
		assert.Equal(t, HTTPRequestKindWebhook, HTTPRequestKind("/api/v1/sentry/app/webhook"))
		assert.Equal(t, HTTPRequestKindWebhook, HTTPRequestKind("/api/v1/polar/webhooks"))
		assert.True(t, IsWebhookHTTPRoute("/api/v1/webhooks/{webhookID}"))
	})

	t.Run("classifies interactive API routes as api", func(t *testing.T) {
		assert.Equal(t, HTTPRequestKindAPI, HTTPRequestKind("/api/v1/integrations"))
		assert.Equal(t, HTTPRequestKindAPI, HTTPRequestKind("/api/v1/organizations/{id}"))
		assert.Equal(t, HTTPRequestKindAPI, HTTPRequestKind("/api/v1/agents"))
		assert.Equal(t, HTTPRequestKindAPI, HTTPRequestKind("/api/v1/canvases/{canvas_id}/runs"))
		assert.Equal(t, HTTPRequestKindAPI, HTTPRequestKind("/admin/api/polar/webhooks"))
		assert.Equal(t, HTTPRequestKindAPI, HTTPRequestKind(""))
	})
}
