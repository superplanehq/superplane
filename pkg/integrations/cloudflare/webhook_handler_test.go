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

func Test__CloudflareWebhookHandler__Setup_TunnelHealth(t *testing.T) {
	handler := &CloudflareWebhookHandler{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "dest456"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "policy456"}}`)),
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
			URL:           "https://example.com/webhook-tunnel",
			Configuration: OnTunnelHealthSpec{Tunnel: "tun123", NewStatus: []string{"TUNNEL_STATUS_TYPE_DOWN"}},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, CloudflareWebhookMetadata{
		AccountID:            "account123",
		DestinationID:        "dest456",
		NotificationPolicyID: "policy456",
	}, metadata)
	require.Len(t, httpContext.Requests, 2)

	var policy map[string]any
	require.NoError(t, json.NewDecoder(httpContext.Requests[1].Body).Decode(&policy))
	assert.Equal(t, "tunnel_health_event", policy["alert_type"])
	assert.Equal(t, map[string]any{
		"tunnel_id":  []any{"tun123"},
		"new_status": []any{"TUNNEL_STATUS_TYPE_DOWN"},
	}, policy["filters"])
}

func Test__CloudflareWebhookHandler__Cleanup(t *testing.T) {
	handler := &CloudflareWebhookHandler{}
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken":  "token123",
			"accountId": "account123",
		},
	}
	metadata := map[string]any{
		"accountId":            "account123",
		"notificationPolicyId": "policy123",
		"destinationId":        "dest123",
	}

	t.Run("deletes policy then destination", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{Metadata: metadata},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/alerting/v3/policies/policy123")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/alerting/v3/destinations/webhooks/dest123")
	})

	t.Run("404 on policy delete still deletes destination", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"code":1003,"message":"not found"}]}`)),
				},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{Metadata: metadata},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
	})

	t.Run("404 on destination delete succeeds", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"code":1003}]}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{Metadata: metadata},
		})

		require.NoError(t, err)
	})

	t.Run("non-404 error on policy fails cleanup before destination", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`{"success":false}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{Metadata: metadata},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "notification policy")
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("non-404 error on destination fails cleanup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))},
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"success":false}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     &contexts.WebhookContext{Metadata: metadata},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "alerting webhook destination")
	})
}

func Test__CloudflareWebhookHandler__CompareConfig(t *testing.T) {
	handler := &CloudflareWebhookHandler{}

	t.Run("tunnel health config never matches load balancing defaults", func(t *testing.T) {
		tunnel := map[string]any{"tunnel": "", "newStatus": []string{"Down"}}
		lb := map[string]any{"pool": "", "newHealth": []string{"Unhealthy"}, "eventSource": []string{"pool", "origin"}}

		ok, err := handler.CompareConfig(tunnel, lb)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("tunnel health legacy Down normalizes equal to API status", func(t *testing.T) {
		a := map[string]any{"tunnel": "t1", "newStatus": []string{"Down"}}
		b := map[string]any{"tunnel": "t1", "newStatus": []string{"TUNNEL_STATUS_TYPE_DOWN"}}

		ok, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("same pool but different newHealth -> not equal", func(t *testing.T) {
		a := map[string]any{"pool": "pool123", "newHealth": []string{"Unhealthy"}, "eventSource": []string{"pool"}}
		b := map[string]any{"pool": "pool123", "newHealth": []string{"Healthy"}, "eventSource": []string{"pool"}}

		ok, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("same pool but different eventSource -> not equal", func(t *testing.T) {
		a := map[string]any{"pool": "pool123", "newHealth": []string{"Unhealthy"}, "eventSource": []string{"pool"}}
		b := map[string]any{"pool": "pool123", "newHealth": []string{"Unhealthy"}, "eventSource": []string{"origin"}}

		ok, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("matching filters including defaults -> equal", func(t *testing.T) {
		a := map[string]any{"pool": "", "newHealth": []string{"Unhealthy"}, "eventSource": []string{"pool", "origin"}}
		b := map[string]any{}

		ok, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("identical normalized configs -> equal", func(t *testing.T) {
		spec := OnLoadBalancingHealthAlertSpec{
			Pool:        "pool123",
			NewHealth:   []string{"Unhealthy"},
			EventSource: []string{"origin"},
		}

		ok, err := handler.CompareConfig(spec, spec)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}
