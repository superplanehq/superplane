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

func Test__ListCreditPacksFiltersMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/products/", r.URL.Path)
		assert.Equal(t, "Bearer oat_test", r.Header.Get("Authorization"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":   "prod_500",
					"name": "Hosted credit 500",
					"metadata": map[string]string{
						"superplane_credit_pack": "true",
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 50000},
					},
				},
				{
					"id":   "prod_25",
					"name": "Hosted credit 25",
					"metadata": map[string]string{
						"superplane_credit_pack": "true",
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 2500},
					},
				},
				{
					"id":   "prod_other",
					"name": "Support",
					"metadata": map[string]string{
						"superplane_credit_pack": "false",
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 1000},
					},
				},
				{
					"id":   "prod_100",
					"name": "Hosted credit 100",
					"metadata": map[string]string{
						"superplane_credit_pack": "true",
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 10000},
					},
				},
			},
			"pagination": map[string]any{"max_page": 1},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	packs, err := client.ListCreditPacks(context.Background())
	require.NoError(t, err)
	require.Len(t, packs, 3)
	assert.Equal(t, []string{"prod_25", "prod_100", "prod_500"}, []string{packs[0].ID, packs[1].ID, packs[2].ID})
	assert.Equal(t, []int64{2500, 10000, 50000}, []int64{packs[0].AmountCents, packs[1].AmountCents, packs[2].AmountCents})
}

func Test__GetCreditPackRejectsNonPackProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/products/prod_other", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":   "prod_other",
			"name": "Support",
			"metadata": map[string]string{
				"superplane_credit_pack": "false",
			},
			"prices": []map[string]any{
				{"amount_type": "fixed", "price_amount": 1000},
			},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCreditPack(context.Background(), "prod_other")
	require.ErrorIs(t, err, ErrNotCreditPack)
}

func Test__GetCreditPackReturnsPack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/products/prod_25", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":   "prod_25",
			"name": "Hosted credit 25",
			"metadata": map[string]string{
				"superplane_credit_pack": "true",
			},
			"prices": []map[string]any{
				{"amount_type": "fixed", "price_amount": 2500},
			},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	pack, err := client.GetCreditPack(context.Background(), "prod_25")
	require.NoError(t, err)
	assert.Equal(t, "prod_25", pack.ID)
	assert.Equal(t, int64(2500), pack.AmountCents)
}

func Test__CreateCheckoutForwardsCustomerIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/checkouts/", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "203.0.113.10", body["customer_ip_address"])
		assert.Equal(t, "org-1", body["external_customer_id"])
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"url":         "https://buy.polar.sh/polar_c_test",
			"customer_id": "cust_1",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	session, err := client.CreateCheckout(context.Background(), "prod_1", "org-1", "owner@example.com", "http://localhost:8000/return", "203.0.113.10")
	require.NoError(t, err)
	assert.Equal(t, "https://buy.polar.sh/polar_c_test", session.URL)
	assert.Equal(t, "cust_1", session.CustomerID)
}

func Test__APIBaseURLUsesSandboxByDefault(t *testing.T) {
	t.Setenv("POLAR_ENVIRONMENT", "")
	assert.Equal(t, sandboxAPIBaseURL, APIBaseURL())
	t.Setenv("POLAR_ENVIRONMENT", "production")
	assert.Equal(t, productionAPIBaseURL, APIBaseURL())
}
