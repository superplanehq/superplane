package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
				{
					"id":   "prod_custom",
					"name": "Hosted credit custom",
					"metadata": map[string]string{
						"superplane_credit_pack": "true",
					},
					"prices": []map[string]any{
						{"amount_type": "custom", "minimum_amount": 100, "preset_amount": 2500},
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
	require.Len(t, packs, 4)
	assert.Equal(t, []string{"prod_25", "prod_100", "prod_500", "prod_custom"}, []string{packs[0].ID, packs[1].ID, packs[2].ID, packs[3].ID})
	assert.Equal(t, []int64{2500, 10000, 50000, 0}, []int64{packs[0].AmountCents, packs[1].AmountCents, packs[2].AmountCents, packs[3].AmountCents})
	assert.False(t, packs[0].CustomPrice)
	assert.True(t, packs[3].CustomPrice)
}

func Test__ListCreditPacksKeepsFixedPriceOnMixedProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":   "prod_mixed",
					"name": "Hosted credit mixed",
					"metadata": map[string]string{
						"superplane_credit_pack": "true",
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 5000},
						{"amount_type": "custom", "minimum_amount": 100},
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
	require.Len(t, packs, 1)
	assert.Equal(t, "prod_mixed", packs[0].ID)
	assert.Equal(t, int64(5000), packs[0].AmountCents)
	assert.False(t, packs[0].CustomPrice)
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
	assert.False(t, pack.CustomPrice)
	assert.Equal(t, "usd", pack.Currency)
}

func Test__GetCreditPackReturnsCustomPack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/products/prod_custom", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":   "prod_custom",
			"name": "Hosted credit custom",
			"metadata": map[string]string{
				"superplane_credit_pack": "true",
			},
			"prices": []map[string]any{
				{"amount_type": "custom", "minimum_amount": 100, "preset_amount": 2500},
			},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	pack, err := client.GetCreditPack(context.Background(), "prod_custom")
	require.NoError(t, err)
	assert.Equal(t, "prod_custom", pack.ID)
	assert.Equal(t, int64(0), pack.AmountCents)
	assert.True(t, pack.CustomPrice)
}

func Test__GetCreditPackKeepsFixedPriceOnMixedProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/products/prod_mixed", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":   "prod_mixed",
			"name": "Hosted credit mixed",
			"metadata": map[string]string{
				"superplane_credit_pack": "true",
			},
			"prices": []map[string]any{
				{"amount_type": "custom", "minimum_amount": 100},
				{"amount_type": "fixed", "price_amount": 5000},
			},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	pack, err := client.GetCreditPack(context.Background(), "prod_mixed")
	require.NoError(t, err)
	assert.Equal(t, "prod_mixed", pack.ID)
	assert.Equal(t, int64(5000), pack.AmountCents)
	assert.False(t, pack.CustomPrice)
}

func Test__CreateCheckoutForwardsSubscriptionOptions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		seats    *int
		metadata map[string]string
	}{
		{name: "Business"},
		{name: "Team", seats: new(5), metadata: map[string]string{CheckoutIntentMetadataKey: "intent-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expiresAt := time.Now().Add(time.Hour).UTC()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/checkouts/", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, "203.0.113.10", body["customer_ip_address"])
				assert.Equal(t, "org-1", body["external_customer_id"])
				assert.NotContains(t, body, "prices")
				assert.NotContains(t, body, "customer_email")
				if tc.metadata == nil {
					assert.NotContains(t, body, "metadata")
				} else {
					assert.Equal(t, map[string]any{CheckoutIntentMetadataKey: "intent-1"}, body["metadata"])
				}
				assert.NotContains(t, body, "allow_discount_codes", "Polar default allows discount codes")
				if tc.seats == nil {
					assert.NotContains(t, body, "seats")
					assert.NotContains(t, body, "min_seats")
					assert.NotContains(t, body, "max_seats")
				}
				if tc.seats != nil {
					for _, field := range []string{"seats", "min_seats", "max_seats"} {
						assert.Equal(t, float64(*tc.seats), body[field], field)
					}
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":          "chk_1",
					"url":         "https://buy.polar.sh/polar_c_test",
					"customer_id": "cust_1",
					"status":      "open",
					"expires_at":  expiresAt.Format(time.RFC3339Nano),
					"seats":       tc.seats,
					"metadata":    map[string]any{CheckoutIntentMetadataKey: "intent-1", "attempt": 2},
				}))
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL, "oat_test", server.Client())
			session, err := client.CreateCheckout(context.Background(), CheckoutInput{ProductID: "prod_1", ExternalCustomerID: "org-1", SuccessURL: "http://localhost:8000/return", CustomerIP: "203.0.113.10", Seats: tc.seats, Metadata: tc.metadata})
			require.NoError(t, err)
			assert.Equal(t, "chk_1", session.ID)
			assert.Equal(t, "https://buy.polar.sh/polar_c_test", session.URL)
			assert.Equal(t, "cust_1", session.CustomerID)
			assert.Equal(t, CheckoutStatusOpen, session.Status)
			require.NotNil(t, session.ExpiresAt)
			assert.True(t, session.ExpiresAt.Equal(expiresAt))
			assert.Equal(t, tc.seats, session.Seats)
			assert.Equal(t, "intent-1", session.IntentID())
			assert.Equal(t, "2", session.Metadata["attempt"], "numeric metadata is read as text")
		})
	}
}

func TestCreateCreditCheckoutLocksChargeAndCreditMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/checkouts/", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []any{"prod_credits"}, body["products"])
		assert.Equal(t, "org-1", body["external_customer_id"])
		assert.Equal(t, "https://app.example/return", body["success_url"])
		assert.Equal(t, "203.0.113.10", body["customer_ip_address"])
		assert.NotContains(t, body, "allow_discount_codes", "Polar default allows discount codes")
		assert.Equal(t, map[string]any{
			"prod_credits": []any{map[string]any{
				"amount_type": "fixed", "price_amount": float64(10579),
				"price_currency": "usd", "tax_behavior": "exclusive",
			}},
		}, body["prices"])
		assert.Equal(t, map[string]any{
			"superplane_credit_purchase_version": "1",
			"superplane_credit_amount_cents":     "10000",
			"superplane_credit_subtotal_cents":   "10579",
		}, body["metadata"])
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"url": "https://buy.example/credits", "customer_id": "cust_1",
		}))
	}))
	t.Cleanup(server.Close)

	quote, err := CalculateCreditPurchase(10000, 500, 50)
	require.NoError(t, err)
	client := NewClient(server.URL, "oat_test", server.Client())
	session, err := client.CreateCreditCheckout(context.Background(), "prod_credits", "org-1", "https://app.example/return", "203.0.113.10", quote)
	require.NoError(t, err)
	assert.Equal(t, "https://buy.example/credits", session.URL)
	assert.Equal(t, "cust_1", session.CustomerID)
}

func TestGetCreditPackReturnsSelectedCurrencyAndCustomLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id": "prod_custom", "name": "Custom credit",
			"metadata": map[string]any{"superplane_credit_pack": true},
			"prices": []map[string]any{
				{"amount_type": "fixed", "price_amount": 2500, "is_archived": true},
				{"amount_type": "custom", "price_currency": "eur", "minimum_amount": 100, "maximum_amount": 50000},
			},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	pack, err := client.GetCreditPack(context.Background(), "prod_custom")
	require.NoError(t, err)
	assert.True(t, pack.CustomPrice)
	assert.Equal(t, "eur", pack.Currency)
	assert.Equal(t, int64(100), pack.MinimumAmountCents)
	assert.Equal(t, int64(50000), pack.MaximumAmountCents)
}

func TestGetCreditPackRejectsRecurringProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id": "prod_recurring", "is_recurring": true,
			"metadata": map[string]any{"superplane_credit_pack": true},
			"prices":   []map[string]any{{"amount_type": "fixed", "price_amount": 2500}},
		}))
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCreditPack(t.Context(), "prod_recurring")
	require.Error(t, err)
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
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
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

func Test__CreateCheckoutClassifiesUncertainOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		unknown bool
	}{
		{"server error", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", http.StatusBadGateway) }, true},
		{"dropped connection", func(w http.ResponseWriter, _ *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			require.NoError(t, err)
			require.NoError(t, conn.Close())
		}, true},
		{"created without id", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"url": "https://buy.polar.sh/c"})
		}, true},
		{"created without url", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "chk_1"})
		}, true},
		{"validation rejection", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "invalid", http.StatusUnprocessableEntity)
		}, false},
		{"bad request", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "bad", http.StatusBadRequest) }, false},
		{"unauthorized", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusUnauthorized) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			t.Cleanup(server.Close)

			client := NewClient(server.URL, "oat_test", server.Client())
			_, err := client.CreateCheckout(context.Background(), CheckoutInput{ProductID: "prod_1", ExternalCustomerID: "org-1", SuccessURL: "http://localhost/return"})
			require.Error(t, err)
			assert.Equal(t, tc.unknown, IsOutcomeUnknown(err))
		})
	}
}

func Test__CreateCheckoutReportsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.CreateCheckout(context.Background(), CheckoutInput{ProductID: "prod_1", ExternalCustomerID: "org-1", SuccessURL: "http://localhost/return"})
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

func Test__ListSubscriptionsFiltersByCustomerAndProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subscriptions/", r.URL.Path)
		assert.Equal(t, "org-1", r.URL.Query().Get("external_customer_id"))
		assert.Equal(t, "prod_business", r.URL.Query().Get("product_id"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":                   "sub_canceled",
					"status":               "canceled",
					"current_period_start": "2026-08-01T12:00:00Z",
					"current_period_end":   "2026-09-01T12:00:00Z",
					"customer_id":          "cust_1",
					"external_customer_id": "org-1",
					"customer": map[string]any{
						"id":          "cust_1",
						"external_id": "org-1",
					},
				},
				{
					"id":                   "sub_active",
					"status":               "active",
					"current_period_start": "2026-09-01T12:00:00Z",
					"current_period_end":   "2026-10-01T12:00:00Z",
					"customer_id":          "cust_1",
					"external_customer_id": "org-1",
					"customer": map[string]any{
						"id":          "cust_1",
						"external_id": "org-1",
					},
				},
			},
			"pagination": map[string]any{"max_page": 1},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	subs, err := client.ListSubscriptions(context.Background(), "org-1", "prod_business")
	require.NoError(t, err)
	require.Len(t, subs, 2)
	assert.Equal(t, "sub_canceled", subs[0].ID)
	assert.Equal(t, "canceled", subs[0].Status)
	assert.Equal(t, "sub_active", subs[1].ID)
	assert.Equal(t, "active", subs[1].Status)
	assert.Equal(t, "org-1", subs[1].organizationExternalID())
	assert.False(t, subs[1].CurrentPeriodStart.Time.IsZero())
	assert.False(t, subs[1].CurrentPeriodEnd.Time.IsZero())
}

func Test__ListSubscriptionsRequiresExternalCustomerID(t *testing.T) {
	client := NewClient("http://polar.example", "oat_test", nil)
	_, err := client.ListSubscriptions(context.Background(), "  ", "prod_business")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "external customer id is required")
}

func Test__CancelSubscriptionAtPeriodEndPatchesPolar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/subscriptions/sub_active", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, true, body["cancel_at_period_end"])
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_active",
			"status":               "active",
			"cancel_at_period_end": true,
			"current_period_start": "2026-09-01T12:00:00Z",
			"current_period_end":   "2026-10-01T12:00:00Z",
			"external_customer_id": "org-1",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	sub, err := client.CancelSubscriptionAtPeriodEnd(context.Background(), "sub_active")
	require.NoError(t, err)
	assert.Equal(t, "sub_active", sub.ID)
	assert.Equal(t, "active", sub.Status)
	assert.True(t, sub.CancelAtPeriodEnd)
}

func Test__ResumeSubscriptionPatchesPolar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/subscriptions/sub_active", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, false, body["cancel_at_period_end"])
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_active",
			"status":               "active",
			"cancel_at_period_end": false,
			"current_period_start": "2026-09-01T12:00:00Z",
			"current_period_end":   "2026-10-01T12:00:00Z",
			"external_customer_id": "org-1",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	sub, err := client.ResumeSubscription(context.Background(), "sub_active")
	require.NoError(t, err)
	assert.Equal(t, "sub_active", sub.ID)
	assert.False(t, sub.CancelAtPeriodEnd)
}

func Test__CancelSubscriptionAtPeriodEndRequiresID(t *testing.T) {
	client := NewClient("http://polar.example", "oat_test", nil)
	_, err := client.CancelSubscriptionAtPeriodEnd(context.Background(), "  ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscription id is required")
}

func Test__UpdateSubscriptionSeatsInvoicesTheProratedChange(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/subscriptions/sub_team", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "sub_team", "status": "active", "seats": 5}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	updated, err := client.UpdateSubscriptionSeats(context.Background(), " sub_team ", 5)
	require.NoError(t, err)
	assert.Equal(t, 5, *updated.Seats)
	assert.Equal(t, map[string]any{"seats": float64(5), "proration_behavior": "invoice"}, body)
}

func Test__UpdateSubscriptionSeatsRequiresIDInRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "active"}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.UpdateSubscriptionSeats(context.Background(), " ", 5)
	require.ErrorContains(t, err, "subscription id is required")
	_, err = client.UpdateSubscriptionSeats(context.Background(), "sub_team", 5)
	require.ErrorContains(t, err, "did not return an id")
	assert.True(t, IsOutcomeUnknown(err), "a 200 without a snapshot may still have charged the customer")
}

func Test__SubscriptionUpdateErrorsAreTyped(t *testing.T) {
	tests := []struct {
		name   string
		status int
		code   string
		match  func(error) bool
		also   func(error) bool
		not    []func(error) bool
	}{
		{name: "payment failed", status: http.StatusPaymentRequired, code: "PaymentFailed", match: IsPaymentFailed,
			not: []func(error) bool{IsOutcomeUnknown}},
		{name: "payment action required", status: http.StatusPaymentRequired, code: "PaymentActionRequired", match: IsPaymentActionRequired,
			not: []func(error) bool{IsPaymentFailed}},
		{name: "locked", status: http.StatusConflict, code: "SubscriptionLocked", match: IsSubscriptionLocked, also: IsConflict},
		{name: "already canceled", status: http.StatusForbidden, code: "AlreadyCanceledSubscription", match: IsSubscriptionInactive,
			not: []func(error) bool{IsUnauthorized}},
		{name: "inactive", status: http.StatusForbidden, code: "InactiveSubscription", match: IsSubscriptionInactive,
			not: []func(error) bool{IsUnauthorized}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"error": tt.code, "detail": "details"}))
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL, "oat_test", server.Client())
			_, err := client.UpdateSubscriptionSeats(context.Background(), "sub_team", 5)
			require.Error(t, err)
			assert.True(t, tt.match(err))
			if tt.also != nil {
				assert.True(t, tt.also(err))
			}
			for _, other := range tt.not {
				assert.False(t, other(err))
			}
		})
	}
}

func Test__UntypedSubscriptionErrorsKeepExistingMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		match  func(error) bool
	}{
		{name: "forbidden without a code", status: http.StatusForbidden, body: `{"error":"NotPermitted"}`, match: IsUnauthorized},
		{name: "forbidden with plain text", status: http.StatusForbidden, body: "forbidden", match: IsUnauthorized},
		{name: "conflict without a code", status: http.StatusConflict, body: `{"error":"ResourceAlreadyExists"}`, match: IsConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL, "oat_test", server.Client())
			_, err := client.UpdateSubscriptionSeats(context.Background(), "sub_team", 5)
			assert.True(t, tt.match(err))
			assert.False(t, IsSubscriptionInactive(err) || IsSubscriptionLocked(err) || IsOutcomeUnknown(err))
		})
	}
}

func Test__UpdateSubscriptionSeatsReportsUnknownOutcome(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusBadGateway)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "oat_test", server.Client())
		_, err := client.UpdateSubscriptionSeats(context.Background(), "sub_team", 5)
		assert.True(t, IsOutcomeUnknown(err))
		assert.Contains(t, err.Error(), "502")
	})

	t.Run("timeout", func(t *testing.T) {
		release := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			<-release
		}))
		t.Cleanup(server.Close)
		t.Cleanup(func() { close(release) })

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		client := NewClient(server.URL, "oat_test", server.Client())
		_, err := client.UpdateSubscriptionSeats(ctx, "sub_team", 5)
		assert.True(t, IsOutcomeUnknown(err))
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("validation rejection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "invalid seats", http.StatusUnprocessableEntity)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "oat_test", server.Client())
		_, err := client.UpdateSubscriptionSeats(context.Background(), "sub_team", 5)
		require.Error(t, err)
		assert.True(t, IsInvalidRequest(err))
		assert.False(t, IsOutcomeUnknown(err), "a 422 means Polar did not apply the update")
	})
}

func Test__GetSubscriptionReadsSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/subscriptions/sub_team", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id": "sub_team", "status": "active", "seats": 7, "modified_at": "2026-11-08T00:00:00Z",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	snapshot, err := client.GetSubscription(context.Background(), "sub_team")
	require.NoError(t, err)
	assert.Equal(t, 7, *snapshot.Seats)
	assert.False(t, snapshot.ModifiedAt.Time.IsZero())

	_, err = client.GetSubscription(context.Background(), " ")
	require.ErrorContains(t, err, "subscription id is required")
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
					"checkout_id":  "checkout-1",
					"currency":     "usd",
					"net_amount":   9000,
					"tax_amount":   1000,
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
	assert.Equal(t, "checkout-1", orders[0].CheckoutID)
	assert.Equal(t, "usd", orders[0].Currency)
	assert.Equal(t, int64(9000), orders[0].NetAmountCents)
	assert.Equal(t, int64(1000), orders[0].TaxAmountCents)
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

func Test__APIVersionUsesPinnedVersionUnlessOverridden(t *testing.T) {
	t.Setenv("POLAR_API_VERSION", "")
	assert.Equal(t, defaultAPIVersion, APIVersion())
	t.Setenv("POLAR_API_VERSION", " 2027-01 ")
	assert.Equal(t, "2027-01", APIVersion())
}

func Test__RequestsSendPinnedAPIVersion(t *testing.T) {
	t.Setenv("POLAR_API_VERSION", "")
	var received []string
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.Header.Get("Polar-Version"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":          "cust_1",
			"external_id": "org-1",
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCustomerByExternalID(context.Background(), "org-1")
	require.NoError(t, err)
	_, err = client.CreateCustomer(context.Background(), CreateCustomerInput{ExternalID: "org-1"})
	require.NoError(t, err)
	assert.Equal(t, []string{defaultAPIVersion, defaultAPIVersion}, received)
}

func Test__RequestsSendOverriddenAPIVersion(t *testing.T) {
	t.Setenv("POLAR_API_VERSION", "2027-01")
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2027-01", r.Header.Get("Polar-Version"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_1"}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCustomerByExternalID(context.Background(), "org-1")
	require.NoError(t, err)
}

func Test__NotFoundWithoutVersionHeaderReportsUnsupportedAPIVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCustomerByExternalID(context.Background(), "org-1")
	require.Error(t, err)
	assert.True(t, IsUnsupportedAPIVersion(err))
	assert.False(t, IsNotFound(err))
	assert.Contains(t, err.Error(), APIVersion())
}

func Test__NotFoundWithVersionHeaderReportsNotFound(t *testing.T) {
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCustomerByExternalID(context.Background(), "org-1")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.False(t, IsUnsupportedAPIVersion(err))
}

func Test__EnsureCustomerDoesNotCreateWhenAPIVersionIsUnsupported(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			created = true
		}
		http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.EnsureCustomer(context.Background(), CreateCustomerInput{ExternalID: "org-1"})
	require.Error(t, err)
	assert.True(t, IsUnsupportedAPIVersion(err))
	assert.False(t, created)
}

func Test__GetCreditPackReportsUnsupportedAPIVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.GetCreditPack(context.Background(), "prod_1")
	require.Error(t, err)
	assert.True(t, IsUnsupportedAPIVersion(err))
	assert.False(t, IsNotFound(err))
}

// echoPolarVersion mirrors real Polar, which echoes Polar-Version on every
// response for a supported version.
func echoPolarVersion(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Polar-Version", r.Header.Get("Polar-Version"))
		handler(w, r)
	}
}

func TestSubscriptionDashboardURL(t *testing.T) {
	for _, tc := range []struct{ name, environment, slug, id, want string }{
		{"production", "production", "superplane-inc", "sub_123", "https://polar.sh/dashboard/superplane-inc/sales/subscriptions/sub_123"},
		{"sandbox", "sandbox", "superplane-inc", "sub_123", "https://sandbox.polar.sh/dashboard/superplane-inc/sales/subscriptions/sub_123"},
		{"default production slug", "production", "", "sub_123", "https://polar.sh/dashboard/superplane-inc/sales/subscriptions/sub_123"},
		{"default sandbox slug", "sandbox", "", "sub_123", "https://sandbox.polar.sh/dashboard/superplane/sales/subscriptions/sub_123"},
		{"no id", "production", "superplane-inc", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POLAR_ENVIRONMENT", tc.environment)
			t.Setenv("POLAR_DASHBOARD_ORGANIZATION_SLUG", tc.slug)
			assert.Equal(t, tc.want, SubscriptionDashboardURL(tc.id))
		})
	}
}
