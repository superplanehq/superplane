package billinghttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminPolarProducts(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	upstreamStatus := http.StatusOK
	pages := []string{}
	polarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/products/", r.URL.Path)
		require.Equal(t, "Bearer oat_products", r.Header.Get("Authorization"))
		w.Header().Set("Polar-Version", r.Header.Get("Polar-Version"))
		if upstreamStatus != http.StatusOK {
			http.Error(w, "private upstream error", upstreamStatus)
			return
		}
		require.Equal(t, "false", r.URL.Query().Get("is_archived"))
		if r.URL.Query().Get("is_recurring") == "true" {
			price := []map[string]any{{"amount_type": "fixed", "price_amount": 19900, "price_currency": "usd"}}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "flat", "name": "Business", "is_recurring": true, "prices": price},
					{"id": "seats", "name": "Team", "is_recurring": true, "prices": []map[string]any{{"amount_type": "seat_based", "price_currency": "usd"}}},
					{"id": "one_time", "name": "Credits", "prices": price},
					{"id": "archived", "name": "Archived", "is_recurring": true, "is_archived": true, "prices": price},
					{"id": "eur", "name": "Euro plan", "is_recurring": true, "prices": []map[string]any{{"price_currency": "eur"}}},
					{"id": "no_active_price", "name": "Old plan", "is_recurring": true, "prices": []map[string]any{{"price_currency": "usd", "is_archived": true}}},
				}, "pagination": map[string]any{"max_page": 1},
			}))
			return
		}
		require.Equal(t, "false", r.URL.Query().Get("is_recurring"))
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		items := []map[string]any{{"id": "custom", "name": "Custom credit", "prices": []map[string]any{{"amount_type": "custom", "price_currency": "usd"}}}}
		if page == "1" {
			price := []map[string]any{{"amount_type": "fixed", "price_amount": 2500, "price_currency": "usd"}}
			items = []map[string]any{
				{"id": "starter", "name": "Starter credit", "prices": price},
				{"id": "recurring", "name": "Subscription", "is_recurring": true, "prices": price},
				{"id": "archived", "name": "Archived", "is_archived": true, "prices": price},
				{"id": "eur", "name": "Euro credit", "prices": []map[string]any{{"amount_type": "fixed", "price_amount": 2500, "price_currency": "eur"}}},
			}
			for len(items) < 100 {
				items = append(items, map[string]any{"id": "unsupported", "prices": []any{}})
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]any{"max_page": 2}}))
	}))
	t.Cleanup(polarServer.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_products")
	t.Setenv("POLAR_API_BASE_URL", polarServer.URL)

	response := request("GET", "polar/products", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var products []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &products))
	require.Len(t, products, 2)
	require.Equal(t, "custom", products[0].ID)
	require.Equal(t, "Custom credit", products[0].Name)
	require.Equal(t, "starter", products[1].ID)
	require.Equal(t, "Starter credit", products[1].Name)
	require.Equal(t, []string{"1", "2"}, pages)
	require.NotContains(t, response.Body.String(), "oat_products")
	response = request("GET", "polar/products?kind=subscription", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &products))
	require.Len(t, products, 2)
	require.Equal(t, "flat", products[0].ID)
	require.Equal(t, "Business", products[0].Name)
	require.Equal(t, "seats", products[1].ID)
	require.Equal(t, "Team", products[1].Name)
	response = request("GET", "polar/products?kind=invalid", "")
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())

	for _, test := range []struct {
		name           string
		upstream, want int
	}{
		{"token lacks product access", http.StatusForbidden, http.StatusBadGateway},
		{"Polar rate limit", http.StatusTooManyRequests, http.StatusTooManyRequests},
		{"Polar unavailable", http.StatusServiceUnavailable, http.StatusBadGateway},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstreamStatus = test.upstream
			response := request("GET", "polar/products", "")
			require.Equal(t, test.want, response.Code, response.Body.String())
			require.NotContains(t, response.Body.String(), "private upstream error")
		})
	}
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	response = request("GET", "polar/products", "")
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
}
