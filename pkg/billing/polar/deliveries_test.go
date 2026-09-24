package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ListWebhookDeliveriesMapsDeliveriesAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/webhooks/deliveries", r.URL.Path)
		assert.Equal(t, "Bearer oat_test", r.Header.Get("Authorization"))
		assert.Equal(t, "false", r.URL.Query().Get("succeeded"))
		assert.Equal(t, "order.paid", r.URL.Query().Get("event_type"))
		assert.Equal(t, "2", r.URL.Query().Get("page"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))

		code := 500
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":         "del_failed",
					"created_at": "2026-09-14T12:00:00Z",
					"succeeded":  false,
					"http_code":  code,
					"response":   "unable to apply order",
					"webhook_event": map[string]any{
						"id":        "evt_1",
						"type":      "order.paid",
						"succeeded": true,
						"payload":   `{"type":"order.paid","data":{"id":"ord_1"}}`,
					},
				},
				{
					"id":         "del_object_payload",
					"created_at": "2026-09-14T12:01:00Z",
					"succeeded":  false,
					"http_code":  nil,
					"response":   "",
					"webhook_event": map[string]any{
						"id":   "evt_2",
						"type": "subscription.updated",
						"payload": map[string]any{
							"type": "subscription.updated",
						},
					},
				},
			},
			"pagination": map[string]any{"total_count": 12, "max_page": 1},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	succeeded := false
	page, err := client.ListWebhookDeliveries(context.Background(), WebhookDeliveryFilter{
		Succeeded: &succeeded,
		EventType: "order.paid",
		Page:      2,
		Limit:     25,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.Equal(t, 12, page.Total)
	assert.Equal(t, 2, page.Page)
	assert.Equal(t, 25, page.Limit)

	assert.Equal(t, "del_failed", page.Items[0].ID)
	assert.Equal(t, "evt_1", page.Items[0].EventID)
	assert.Equal(t, "order.paid", page.Items[0].EventType)
	require.NotNil(t, page.Items[0].HTTPCode)
	assert.Equal(t, 500, *page.Items[0].HTTPCode)
	assert.Equal(t, "unable to apply order", page.Items[0].Response)
	assert.Equal(t, `{"type":"order.paid","data":{"id":"ord_1"}}`, page.Items[0].Payload)
	require.NotNil(t, page.Items[0].EventSucceeded)
	assert.True(t, *page.Items[0].EventSucceeded)

	assert.Equal(t, "evt_2", page.Items[1].EventID)
	assert.Nil(t, page.Items[1].HTTPCode)
	assert.Nil(t, page.Items[1].EventSucceeded)
	assert.JSONEq(t, `{"type":"subscription.updated"}`, page.Items[1].Payload)
}

func Test__ListWebhookDeliveriesOmitsSucceededWhenUnset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.URL.Query().Get("succeeded"))
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items":      []map[string]any{},
			"pagination": map[string]any{"total_count": 0, "max_page": 0},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	page, err := client.ListWebhookDeliveries(context.Background(), WebhookDeliveryFilter{})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 50, page.Limit)
}

func Test__ListWebhookDeliveriesCapsLimitAndReportsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "100", r.URL.Query().Get("limit"))
		http.Error(w, "missing webhooks:read", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.ListWebhookDeliveries(context.Background(), WebhookDeliveryFilter{Limit: 500})
	require.ErrorIs(t, err, ErrUnauthorized)
}

func Test__ListWebhookDeliveriesReportsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "8")
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.ListWebhookDeliveries(context.Background(), WebhookDeliveryFilter{})
	require.ErrorIs(t, err, ErrRateLimited)
	assert.Contains(t, err.Error(), "retry after 8")
}

func Test__RedeliverWebhookEventPostsEventPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/webhooks/events/evt_1/redeliver", r.URL.Path)
		assert.Equal(t, "Bearer oat_test", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	require.NoError(t, client.RedeliverWebhookEvent(context.Background(), " evt_1 "))
}

func Test__RedeliverWebhookEventRequiresEventID(t *testing.T) {
	client := NewClient("http://polar.example", "oat_test", nil)
	err := client.RedeliverWebhookEvent(context.Background(), "  ")
	require.EqualError(t, err, "webhook event id is required")
}

func Test__RedeliverWebhookEventReportsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	err := client.RedeliverWebhookEvent(context.Background(), "evt_missing")
	require.True(t, IsNotFound(err))
}
