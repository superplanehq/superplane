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

	t.Run("request URI query is used when the URL query is empty", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "https://app.example/api/v1/webhooks/id", nil)
		request.URL.RawQuery = ""
		request.RequestURI = "/api/v1/webhooks/id?event=task.updated"

		assert.Equal(t, "task.updated", webhookDeliveryQuery(request).Get("event"))
	})
}
