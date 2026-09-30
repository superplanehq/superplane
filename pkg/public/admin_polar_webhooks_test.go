package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminPolarWebhooks(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-polar-webhooks@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns configured false when Polar is not set", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		t.Setenv("POLAR_API_BASE_URL", "")

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPolarWebhooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.False(t, body.Configured)
		assert.Empty(t, body.Items)
		assert.Equal(t, 1, body.Page)
		assert.Equal(t, 50, body.Limit)
	})

	t.Run("lists Polar deliveries", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/webhooks/deliveries", r.URL.Path)
			assert.Equal(t, "false", r.URL.Query().Get("succeeded"))
			assert.Equal(t, "order.paid", r.URL.Query().Get("event_type"))
			assert.Equal(t, "1", r.URL.Query().Get("page"))
			assert.Equal(t, "50", r.URL.Query().Get("limit"))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id":         "del_1",
						"created_at": "2026-09-14T12:00:00Z",
						"succeeded":  false,
						"http_code":  500,
						"response":   "unable to apply order",
						"webhook_event": map[string]any{
							"id":        "evt_1",
							"type":      "order.paid",
							"succeeded": true,
							"payload":   `{"type":"order.paid"}`,
						},
					},
				},
				"pagination": map[string]any{"total_count": 1, "max_page": 1},
			}))
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks?succeeded=false&event_type=order.paid",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPolarWebhooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.True(t, body.Configured)
		require.Len(t, body.Items, 1)
		assert.Equal(t, "del_1", body.Items[0].ID)
		assert.Equal(t, "evt_1", body.Items[0].EventID)
		assert.Equal(t, "order.paid", body.Items[0].EventType)
		require.NotNil(t, body.Items[0].HTTPCode)
		assert.Equal(t, 500, *body.Items[0].HTTPCode)
		require.NotNil(t, body.Items[0].EventSucceeded)
		assert.True(t, *body.Items[0].EventSucceeded)
		assert.Equal(t, 1, body.Total)
	})

	t.Run("rejects invalid succeeded filter", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", "http://polar.example")

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks?succeeded=maybe",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("returns 502 when Polar rejects the token", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "missing webhooks:read", http.StatusForbidden)
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadGateway, response.Code)
		assert.Contains(t, response.Body.String(), "webhooks:read")
	})

	t.Run("returns 502 when Polar is unavailable", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadGateway, response.Code)
		assert.Contains(t, response.Body.String(), "Failed to load Polar webhook deliveries")
	})

	t.Run("redelivers a Polar webhook event", func(t *testing.T) {
		var posted string
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			posted = r.URL.Path
			w.WriteHeader(http.StatusAccepted)
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/polar/webhooks/evt_1/redeliver",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "/webhooks/events/evt_1/redeliver", posted)

		var body map[string]string
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, "accepted", body["status"])
	})

	t.Run("redeliver returns 400 when Polar is not set", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		t.Setenv("POLAR_API_BASE_URL", "")

		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/polar/webhooks/evt_1/redeliver",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("redeliver returns 404 when Polar event is missing", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Polar-Version", r.Header.Get("Polar-Version"))
			http.Error(w, "missing", http.StatusNotFound)
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/polar/webhooks/evt_missing/redeliver",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns 502 when Polar rejects the pinned API version", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/polar/webhooks/evt_1/redeliver",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadGateway, response.Code)
		assert.Contains(t, response.Body.String(), "API version "+polar.APIVersion())
	})

	t.Run("lists deliveries with payload API version", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "del_1",
						"webhook_event": map[string]any{
							"id":      "evt_1",
							"type":    "order.paid",
							"payload": `{"type":"order.paid","api_version":"2026-04"}`,
						},
					},
				},
				"pagination": map[string]any{"total_count": 1, "max_page": 1},
			}))
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPolarWebhooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Items, 1)
		assert.Equal(t, "2026-04", body.Items[0].APIVersion)
	})
}

func TestAdminPolarWebhookEndpoints(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-polar-endpoints@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks/endpoints",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns configured false when Polar is not set", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		t.Setenv("POLAR_API_VERSION", "")

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks/endpoints",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPolarWebhookEndpointsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.False(t, body.Configured)
		assert.Equal(t, polar.APIVersion(), body.APIVersion)
		assert.Empty(t, body.Endpoints)
	})

	t.Run("lists endpoints with the pinned version", func(t *testing.T) {
		server.BaseURL = "https://app.superplane.com"
		server.WebhooksBaseURL = "https://hooks.superplane.com/"
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/webhooks/endpoints", r.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id":          "end_1",
						"url":         "https://app.superplane.com/api/v1/polar/webhooks",
						"api_version": "2026-04",
						"format":      "raw",
					},
					{"id": "end_hooks", "url": "https://HOOKS.superplane.com/api/v1/polar/webhooks/"},
					{"id": "end_other_install", "url": "https://staging.superplane.com/api/v1/polar/webhooks"},
					{"id": "end_other_path", "url": "https://app.superplane.com/hooks/polar"},
				},
				"pagination": map[string]any{"total_count": 4, "max_page": 1},
			}))
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)
		t.Setenv("POLAR_API_VERSION", "")

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks/endpoints",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPolarWebhookEndpointsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.True(t, body.Configured)
		assert.Equal(t, polar.APIVersion(), body.APIVersion)
		require.Len(t, body.Endpoints, 4)
		assert.Equal(t, "end_1", body.Endpoints[0].ID)
		assert.Equal(t, "2026-04", body.Endpoints[0].APIVersion)
		assert.Equal(t, "raw", body.Endpoints[0].Format)
		current := map[string]bool{}
		for _, endpoint := range body.Endpoints {
			current[endpoint.ID] = endpoint.Current
		}
		assert.Equal(t, map[string]bool{
			"end_1":             true,
			"end_hooks":         true,
			"end_other_install": false,
			"end_other_path":    false,
		}, current)
	})

	t.Run("returns 502 when Polar rejects the token", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "missing webhooks:read", http.StatusForbidden)
		}))
		t.Cleanup(upstream.Close)

		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_API_BASE_URL", upstream.URL)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/polar/webhooks/endpoints",
			authCookie: token,
		})
		assert.Equal(t, http.StatusBadGateway, response.Code)
		assert.Contains(t, response.Body.String(), "webhooks:read")
	})
}
