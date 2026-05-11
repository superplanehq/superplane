package cloudflare

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

func Test__CloudflareWebhookHandler__Setup(t *testing.T) {
	handler := &CloudflareWebhookHandler{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "dest123"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "policy123"}}`)),
			},
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":  "token123",
				"accountId": "account123",
			},
		},
		Webhook: &contexts.WebhookContext{
			URL:           "https://example.com/webhook",
			Configuration: OnLoadBalancingHealthAlertSpec{Pool: "pool123", NewHealth: []string{"Unhealthy"}, EventSource: []string{"origin"}},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, CloudflareWebhookMetadata{
		AccountID:            "account123",
		DestinationID:        "dest123",
		NotificationPolicyID: "policy123",
	}, metadata)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/alerting/v3/destinations/webhooks", httpContext.Requests[0].URL.String())
	assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/alerting/v3/policies", httpContext.Requests[1].URL.String())

	var policy map[string]any
	require.NoError(t, json.NewDecoder(httpContext.Requests[1].Body).Decode(&policy))
	assert.Equal(t, "load_balancing_health_alert", policy["alert_type"])
	assert.Equal(t, true, policy["enabled"])
	assert.Equal(t, map[string]any{
		"pool_id":      []any{"pool123"},
		"new_health":   []any{"Unhealthy"},
		"event_source": []any{"origin"},
	}, policy["filters"])
}
