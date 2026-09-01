package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		_, hasCustomerEmail := body["customer_email"]
		assert.False(t, hasCustomerEmail)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"url":         "https://buy.polar.sh/polar_c_test",
			"customer_id": "cust_1",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	session, err := client.CreateCheckout(context.Background(), "prod_1", "org-1", "http://localhost:8000/return", "203.0.113.10")
	require.NoError(t, err)
	assert.Equal(t, "https://buy.polar.sh/polar_c_test", session.URL)
	assert.Equal(t, "cust_1", session.CustomerID)
}

func Test__CreateCustomerPostsTeamOwnerWithoutEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/customers/", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "team", body["type"])
		assert.Equal(t, "Acme", body["name"])
		assert.Equal(t, "org-1", body["external_id"])
		_, hasEmail := body["email"]
		assert.False(t, hasEmail)
		owner, _ := body["owner"].(map[string]any)
		assert.Equal(t, "buyer@example.com", owner["email"])
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":          "cust_1",
			"external_id": "org-1",
			"email":       nil,
			"name":        "Acme",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	customer, err := client.CreateCustomer(context.Background(), CreateCustomerInput{
		ExternalID: "org-1",
		Name:       "Acme",
		OwnerEmail: "buyer@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "cust_1", customer.ID)
}

func Test__EnsureCustomerUsesExistingAfterConflict(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/customers/external/"):
			if !created {
				http.Error(w, "missing", http.StatusNotFound)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":          "cust_existing",
				"external_id": "org-1",
				"email":       "billing+org-1@billing.superplane.com",
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/customers/":
			created = true
			http.Error(w, "conflict", http.StatusConflict)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	customer, err := client.EnsureCustomer(context.Background(), CreateCustomerInput{
		ExternalID: "org-1",
		Name:       "Acme",
		OwnerEmail: "buyer@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "cust_existing", customer.ID)
}

func Test__EnsureCustomerReusesCustomerAfterUniqueness422(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/customers/external/"):
			if !created {
				http.Error(w, "missing", http.StatusNotFound)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":          "cust_existing",
				"external_id": "org-1",
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/customers/":
			created = true
			http.Error(w, `{"detail":[{"loc":["body","external_id"],"msg":"A customer with this external ID already exists."}]}`, http.StatusUnprocessableEntity)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	customer, err := client.EnsureCustomer(context.Background(), CreateCustomerInput{
		ExternalID: "org-1",
		OwnerEmail: "buyer@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "cust_existing", customer.ID)
}

func Test__EnsureCustomerDoesNotTreatValidationAsConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/customers/external/"):
			http.Error(w, "missing", http.StatusNotFound)
		case r.Method == http.MethodPost && r.URL.Path == "/customers/":
			http.Error(w, `{"detail":[{"loc":["body","owner","email"],"msg":"value is not a valid email address"}]}`, http.StatusUnprocessableEntity)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.EnsureCustomer(context.Background(), CreateCustomerInput{
		ExternalID: "org-1",
		OwnerEmail: "buyer@example.com",
	})
	require.Error(t, err)
	assert.False(t, IsConflict(err))
	assert.Contains(t, err.Error(), "not a valid email address")
}

func Test__CreateCheckoutReportsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.CreateCheckout(context.Background(), "prod_1", "org-1", "http://localhost/return", "")
	require.ErrorIs(t, err, ErrRateLimited)
}

func Test__CreateCustomerSessionPrefersExternalID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/customer-sessions/", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "org-1", body["external_customer_id"])
		assert.Equal(t, "org-1", body["external_member_id"])
		_, hasCustomerID := body["customer_id"]
		assert.False(t, hasCustomerID)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"customer_portal_url": "https://polar.example/portal",
			"customer_id":         "cust_1",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	session, err := client.CreateCustomerSession(context.Background(), CustomerSessionRequest{
		CustomerID:         "cust_ignored",
		ExternalCustomerID: "org-1",
		ExternalMemberID:   "org-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://polar.example/portal", session.PortalURL)
	assert.Equal(t, "cust_1", session.CustomerID)
}

func Test__CreateCustomerSessionReportsTeamMemberRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":[{"loc":["body","member_id"],"msg":"member_id is required for team customers."}]}`, http.StatusUnprocessableEntity)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.CreateCustomerSession(context.Background(), CustomerSessionRequest{ExternalCustomerID: "org-1"})
	require.ErrorIs(t, err, ErrTeamMemberRequired)
}

func Test__GetOwnerMemberFiltersByExternalCustomerID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/members/", r.URL.Path)
		assert.Equal(t, "org-1", r.URL.Query().Get("external_customer_id"))
		assert.Equal(t, "owner", r.URL.Query().Get("role"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "mem_owner", "role": "owner"},
			},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	memberID, err := client.GetOwnerMember(context.Background(), "org-1", "")
	require.NoError(t, err)
	assert.Equal(t, "mem_owner", memberID)
}

func Test__ListOrdersFiltersByExternalCustomerID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/orders/", r.URL.Path)
		assert.Equal(t, "org-1", r.URL.Query().Get("external_customer_id"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":           "ord_1",
					"created_at":   "2026-08-27T12:00:00Z",
					"status":       "paid",
					"total_amount": 10000,
					"description":  "Hosted credit",
					"product":      map[string]any{"name": "$100 pack"},
				},
			},
			"pagination": map[string]any{"max_page": 1},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	orders, err := client.ListOrders(context.Background(), "org-1")
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, "ord_1", orders[0].ID)
	assert.Equal(t, int64(10000), orders[0].AmountCents)
	assert.Equal(t, "paid", orders[0].Status)
	assert.Equal(t, "$100 pack", orders[0].ProductName)
}

func Test__CreateCustomerSessionReportsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.CreateCustomerSession(context.Background(), CustomerSessionRequest{ExternalCustomerID: "org-1"})
	require.ErrorIs(t, err, ErrUnauthorized)
}

func Test__APIBaseURLUsesSandboxByDefault(t *testing.T) {
	t.Setenv("POLAR_ENVIRONMENT", "")
	t.Setenv("POLAR_API_BASE_URL", "")
	assert.Equal(t, sandboxAPIBaseURL, APIBaseURL())
	t.Setenv("POLAR_ENVIRONMENT", "production")
	assert.Equal(t, productionAPIBaseURL, APIBaseURL())
	t.Setenv("POLAR_API_BASE_URL", "http://polar.example/v1/")
	assert.Equal(t, "http://polar.example/v1", APIBaseURL())
}
