package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListHostedCreditProducts(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := ListHostedCreditProducts(context.Background(), "not-a-uuid", &pb.ListHostedCreditProductsRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("billing not configured", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		resp, err := ListHostedCreditProducts(context.Background(), r.Organization.ID.String(), &pb.ListHostedCreditProductsRequest{})
		require.NoError(t, err)
		assert.False(t, resp.BillingEnabled)
		assert.Empty(t, resp.Products)
	})

	t.Run("lists credit packs", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "/products/", req.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id":   "prod_25",
						"name": "Hosted credit 25",
						"metadata": map[string]any{
							"superplane_credit_pack": true,
						},
						"prices": []map[string]any{
							{"amount_type": "fixed", "price_amount": 2500},
						},
					},
				},
				"pagination": map[string]any{"max_page": 1},
			}))
		})
		usePolarTestServer(t, server)

		resp, err := ListHostedCreditProducts(context.Background(), r.Organization.ID.String(), &pb.ListHostedCreditProductsRequest{})
		require.NoError(t, err)
		assert.True(t, resp.BillingEnabled)
		require.Len(t, resp.Products, 1)
		assert.Equal(t, "prod_25", resp.Products[0].Id)
		assert.Equal(t, int64(2500), resp.Products[0].AmountCents)
	})
}

func Test__CreateHostedCreditCheckout(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := CreateHostedCreditCheckout(context.Background(), "bad", &pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"}, "", "")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("billing not configured", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		_, err := CreateHostedCreditCheckout(context.Background(), r.Organization.ID.String(), &pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"}, "", "")
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("product id is required", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		_, err := CreateHostedCreditCheckout(context.Background(), r.Organization.ID.String(), &pb.CreateHostedCreditCheckoutRequest{}, "", "")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("rejects non credit packs", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":   "prod_other",
				"name": "Support",
				"metadata": map[string]any{
					"superplane_credit_pack": false,
				},
				"prices": []map[string]any{
					{"amount_type": "fixed", "price_amount": 1000},
				},
			}))
		})
		usePolarTestServer(t, server)

		_, err := CreateHostedCreditCheckout(
			context.Background(),
			r.Organization.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_other"},
			r.Account.ID.String(),
			"http://localhost:8000",
		)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("creates checkout and stores customer id", func(t *testing.T) {
		createdOwners := []string{}
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/products/"):
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":   "prod_25",
					"name": "Hosted credit 25",
					"metadata": map[string]any{
						"superplane_credit_pack": true,
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 2500},
					},
				}))
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/customers/external/"):
				http.Error(w, "missing", http.StatusNotFound)
			case req.Method == http.MethodPost && req.URL.Path == "/customers/":
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				_, hasEmail := body["email"]
				assert.False(t, hasEmail)
				assert.Equal(t, "team", body["type"])
				assert.Equal(t, r.Organization.ID.String(), body["external_id"])
				owner, _ := body["owner"].(map[string]any)
				ownerEmail, _ := owner["email"].(string)
				createdOwners = append(createdOwners, ownerEmail)
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":          "cust_1",
					"external_id": r.Organization.ID.String(),
					"email":       nil,
				}))
			case req.Method == http.MethodPost && req.URL.Path == "/checkouts/":
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				assert.Equal(t, "203.0.113.10", body["customer_ip_address"])
				assert.Nil(t, body["customer_email"])
				assert.Equal(t, "http://localhost:8000/"+r.Organization.ID.String()+"/organization/llm-spend?credit=added&checkout_id={CHECKOUT_ID}", body["success_url"])
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"url":         "https://buy.example/checkout",
					"customer_id": "cust_1",
				}))
			default:
				http.NotFound(w, req)
			}
		})
		usePolarTestServer(t, server)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "203.0.113.10"))
		spoofedIP := "198.51.100.1"
		resp, err := CreateHostedCreditCheckout(
			ctx,
			r.Organization.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25", CustomerIpAddress: spoofedIP},
			r.Account.ID.String(),
			"http://localhost:8000",
		)
		require.NoError(t, err)
		assert.Equal(t, "https://buy.example/checkout", resp.CheckoutUrl)
		require.Len(t, createdOwners, 1)
		assert.Equal(t, r.Account.Email, createdOwners[0])

		settings, err := models.FindOrganizationLLMSettings(database.Conn(), r.Organization.ID)
		require.NoError(t, err)
		require.NotNil(t, settings)
		require.NotNil(t, settings.PolarCustomerID)
		assert.Equal(t, "cust_1", *settings.PolarCustomerID)
	})

	t.Run("requires account email", func(t *testing.T) {
		calledPolar := false
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			calledPolar = true
			http.NotFound(w, req)
		})
		usePolarTestServer(t, server)

		_, err := CreateHostedCreditCheckout(
			context.Background(),
			r.Organization.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"},
			"",
			"http://localhost:8000",
		)
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.False(t, calledPolar)
	})

	t.Run("uses org user email when account id is missing", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/products/"):
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":   "prod_25",
					"name": "Hosted credit 25",
					"metadata": map[string]any{
						"superplane_credit_pack": true,
					},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 2500},
					},
				}))
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/customers/external/"):
				http.Error(w, "missing", http.StatusNotFound)
			case req.Method == http.MethodPost && req.URL.Path == "/customers/":
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":          "cust_user_email",
					"external_id": r.Organization.ID.String(),
					"email":       nil,
				}))
			case req.Method == http.MethodPost && req.URL.Path == "/checkouts/":
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"url":         "https://buy.example/checkout",
					"customer_id": "cust_user_email",
				}))
			default:
				http.NotFound(w, req)
			}
		})
		usePolarTestServer(t, server)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", r.User.String()))
		resp, err := CreateHostedCreditCheckout(
			ctx,
			r.Organization.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"},
			"",
			"http://localhost:8000",
		)
		require.NoError(t, err)
		assert.Equal(t, "https://buy.example/checkout", resp.CheckoutUrl)
	})

	t.Run("reports polar rate limit", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Retry-After", "30")
			http.Error(w, "slow down", http.StatusTooManyRequests)
		})
		usePolarTestServer(t, server)

		_, err := CreateHostedCreditCheckout(
			context.Background(),
			r.Organization.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"},
			r.Account.ID.String(),
			"http://localhost:8000",
		)
		assert.Equal(t, codes.ResourceExhausted, grpcerrors.Code(err))
	})

	t.Run("creates a polar team customer per organization with the same owner", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("billing-org"), "")
		require.NoError(t, err)
		created := map[string]string{}
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/products/"):
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":       "prod_25",
					"name":     "Hosted credit 25",
					"metadata": map[string]any{"superplane_credit_pack": true},
					"prices": []map[string]any{
						{"amount_type": "fixed", "price_amount": 2500},
					},
				}))
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/customers/external/"):
				http.Error(w, "missing", http.StatusNotFound)
			case req.Method == http.MethodPost && req.URL.Path == "/customers/":
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				externalID, _ := body["external_id"].(string)
				_, hasEmail := body["email"]
				assert.False(t, hasEmail)
				owner, _ := body["owner"].(map[string]any)
				ownerEmail, _ := owner["email"].(string)
				created[externalID] = ownerEmail
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":          "cust_" + externalID[:8],
					"external_id": externalID,
					"email":       nil,
				}))
			case req.Method == http.MethodPost && req.URL.Path == "/checkouts/":
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"url":         "https://buy.example/checkout",
					"customer_id": "cust_ok",
				}))
			default:
				http.NotFound(w, req)
			}
		})
		usePolarTestServer(t, server)

		_, err = CreateHostedCreditCheckout(
			context.Background(),
			r.Organization.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"},
			r.Account.ID.String(),
			"http://localhost:8000",
		)
		require.NoError(t, err)
		_, err = CreateHostedCreditCheckout(
			context.Background(),
			other.ID.String(),
			&pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_25"},
			r.Account.ID.String(),
			"http://localhost:8000",
		)
		require.NoError(t, err)
		require.Len(t, created, 2)
		assert.Equal(t, r.Account.Email, created[r.Organization.ID.String()])
		assert.Equal(t, r.Account.Email, created[other.ID.String()])
	})
}

func Test__CreateBillingPortalSession(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := CreateBillingPortalSession(context.Background(), "bad", &pb.CreateBillingPortalSessionRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("billing not configured", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		_, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("missing customer", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "missing", http.StatusNotFound)
		})
		usePolarTestServer(t, server)

		_, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("opens portal by organization external id", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "/customer-sessions/", req.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			assert.Equal(t, r.Organization.ID.String(), body["external_customer_id"])
			assert.Equal(t, r.Organization.ID.String(), body["external_member_id"])
			_, hasCustomerID := body["customer_id"]
			assert.False(t, hasCustomerID)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"customer_portal_url": "https://polar.example/portal",
				"customer_id":         "cust_ext",
			}))
		})
		usePolarTestServer(t, server)

		resp, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		require.NoError(t, err)
		assert.Equal(t, "https://polar.example/portal", resp.PortalUrl)
	})

	t.Run("falls back to stored customer id", func(t *testing.T) {
		require.NoError(t, models.SetOrganizationPolarCustomerID(database.Conn(), r.Organization.ID, "cust_stored"))
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "/customer-sessions/", req.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			if _, ok := body["external_customer_id"]; ok {
				http.Error(w, "missing", http.StatusNotFound)
				return
			}
			assert.Equal(t, "cust_stored", body["customer_id"])
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"customer_portal_url": "https://polar.example/portal-stored",
				"customer_id":         "cust_stored",
			}))
		})
		usePolarTestServer(t, server)

		resp, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		require.NoError(t, err)
		assert.Equal(t, "https://polar.example/portal-stored", resp.PortalUrl)
	})

	t.Run("refreshes stale stored customer", func(t *testing.T) {
		require.NoError(t, models.SetOrganizationPolarCustomerID(database.Conn(), r.Organization.ID, "cust_stale"))
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.Method == http.MethodPost && req.URL.Path == "/customer-sessions/":
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				if _, ok := body["external_customer_id"]; ok {
					http.Error(w, "missing", http.StatusNotFound)
					return
				}
				if body["customer_id"] == "cust_stale" {
					http.Error(w, "missing", http.StatusNotFound)
					return
				}
				assert.Equal(t, "cust_fresh", body["customer_id"])
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"customer_portal_url": "https://polar.example/portal-fresh",
					"customer_id":         "cust_fresh",
				}))
			case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/customers/external/"):
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":          "cust_fresh",
					"external_id": r.Organization.ID.String(),
				}))
			default:
				http.NotFound(w, req)
			}
		})
		usePolarTestServer(t, server)

		resp, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		require.NoError(t, err)
		assert.Equal(t, "https://polar.example/portal-fresh", resp.PortalUrl)

		settings, err := models.FindOrganizationLLMSettings(database.Conn(), r.Organization.ID)
		require.NoError(t, err)
		require.NotNil(t, settings.PolarCustomerID)
		assert.Equal(t, "cust_fresh", *settings.PolarCustomerID)
	})

	t.Run("opens portal with owner member for team customers", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.Method == http.MethodPost && req.URL.Path == "/customer-sessions/":
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				if _, ok := body["member_id"]; !ok {
					http.Error(w, `{"detail":[{"loc":["body","member_id"],"msg":"member_id is required for team customers."}]}`, http.StatusUnprocessableEntity)
					return
				}
				assert.Equal(t, r.Organization.ID.String(), body["external_customer_id"])
				assert.Equal(t, "mem_owner", body["member_id"])
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"customer_portal_url": "https://polar.example/portal-member",
					"customer_id":         "cust_ext",
				}))
			case req.Method == http.MethodGet && req.URL.Path == "/members/":
				assert.Equal(t, r.Organization.ID.String(), req.URL.Query().Get("external_customer_id"))
				assert.Equal(t, "owner", req.URL.Query().Get("role"))
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{{"id": "mem_owner", "role": "owner"}},
				}))
			default:
				http.NotFound(w, req)
			}
		})
		usePolarTestServer(t, server)

		resp, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		require.NoError(t, err)
		assert.Equal(t, "https://polar.example/portal-member", resp.PortalUrl)
	})

	t.Run("reports missing polar scope", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		})
		usePolarTestServer(t, server)

		_, err := CreateBillingPortalSession(context.Background(), r.Organization.ID.String(), &pb.CreateBillingPortalSessionRequest{})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Contains(t, err.Error(), "Hosted billing token is missing a required Polar scope.")
	})
}

func Test__HostedCreditCheckoutSuccessURL(t *testing.T) {
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	assert.Equal(
		t,
		"http://localhost:8000/11111111-1111-1111-1111-111111111111/organization/llm-spend?credit=added&checkout_id={CHECKOUT_ID}",
		hostedCreditCheckoutSuccessURL("http://localhost:8000/", orgID),
	)

	t.Setenv("BASE_URL", "https://app.example")
	assert.Equal(
		t,
		"https://app.example/11111111-1111-1111-1111-111111111111/organization/llm-spend?credit=added&checkout_id={CHECKOUT_ID}",
		hostedCreditCheckoutSuccessURL("  ", orgID),
	)
}

func Test__ClientIPFromContext(t *testing.T) {
	assert.Equal(t, "", clientIPFromContext(context.Background()))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "198.51.100.7"))
	assert.Equal(t, "198.51.100.7", clientIPFromContext(ctx))

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"cf-connecting-ip", "203.0.113.50",
		"x-forwarded-for", "198.51.100.1, 10.0.0.1",
	))
	assert.Equal(t, "203.0.113.50", clientIPFromContext(ctx))

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"grpcgateway-cf-connecting-ip", "203.0.113.50",
		"x-forwarded-for", "198.51.100.1, 10.0.0.1",
	))
	assert.Equal(t, "203.0.113.50", clientIPFromContext(ctx))

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-forwarded-for", "203.0.113.10, 10.0.0.1"))
	assert.Equal(t, "10.0.0.1", clientIPFromContext(ctx))
}

func Test__BillingState(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	enabled, hasCustomer := billingState(context.Background(), r.Organization.ID)
	assert.False(t, enabled)
	assert.False(t, hasCustomer)

	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	enabled, hasCustomer = billingState(context.Background(), r.Organization.ID)
	assert.True(t, enabled)
	assert.False(t, hasCustomer)

	require.NoError(t, models.SetOrganizationPolarCustomerID(database.Conn(), r.Organization.ID, "cust_1"))
	enabled, hasCustomer = billingState(context.Background(), r.Organization.ID)
	assert.True(t, enabled)
	assert.True(t, hasCustomer)
}

func polarAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func usePolarTestServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
}
