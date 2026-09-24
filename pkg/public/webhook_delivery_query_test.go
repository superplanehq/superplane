package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestWebhookDeliveryQuery(t *testing.T) {
	t.Run("query event is kept", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "https://app.example/api/v1/webhooks/id?event=task.created", nil)
		request = mux.SetURLVars(request, map[string]string{"event": "task.updated"})

		assert.Equal(t, "task.created", webhookDeliveryQuery(request).Get("event"))
	})

	t.Run("path event is used when the query is empty", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "https://app.example/api/v1/webhooks/id/task.created", nil)
		request = mux.SetURLVars(request, map[string]string{"event": "task.created"})

		assert.Equal(t, "task.created", webhookDeliveryQuery(request).Get("event"))
	})

	t.Run("path event is used when mux vars are missing", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodPost,
			"https://app.example/api/v1/webhooks/5645309a-9ead-4a23-bd15-4a66f9d6090f/task.created",
			nil,
		)

		assert.Equal(t, "task.created", webhookDeliveryQuery(request).Get("event"))
	})

	t.Run("request URI path is used when the URL path was stripped", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodPost,
			"https://app.example/api/v1/webhooks/5645309a-9ead-4a23-bd15-4a66f9d6090f",
			nil,
		)
		request.RequestURI = "/api/v1/webhooks/5645309a-9ead-4a23-bd15-4a66f9d6090f/task.created"

		assert.Equal(t, "task.created", webhookDeliveryQuery(request).Get("event"))
	})

	t.Run("webhook id path is not treated as an event", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodPost,
			"https://app.example/api/v1/webhooks/5645309a-9ead-4a23-bd15-4a66f9d6090f",
			nil,
		)

		assert.Empty(t, webhookDeliveryQuery(request).Get("event"))
	})

	t.Run("request URI query is used when the URL query is empty", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "https://app.example/api/v1/webhooks/id", nil)
		request.URL.RawQuery = ""
		request.RequestURI = "/api/v1/webhooks/id?event=task.updated"

		assert.Equal(t, "task.updated", webhookDeliveryQuery(request).Get("event"))
	})
}
