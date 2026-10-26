package billinghttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/organizations"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
)

func TestAdminCreditPackCatalog(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	response := request("POST", "credit-packs", `{"name":"Starter credits","amount_cents":2500,"enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var pack struct {
		ID             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		AmountCents    int64     `json:"amount_cents"`
		Enabled        bool      `json:"enabled"`
		PolarProductID *string   `json:"polar_product_id"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &pack))
	require.NotEqual(t, uuid.Nil, pack.ID)
	require.Nil(t, pack.PolarProductID)
	path := "credit-packs/" + pack.ID.String()
	t.Cleanup(func() { request("DELETE", path, "") })
	response = request("PUT", path, `{"name":"Team credits","amount_cents":5000,"enabled":false,"polar_product_id":" prod_credits "}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = request("GET", "credit-packs", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	server.InitRouter()
	response = request("GET", "credit-packs", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var packs []json.RawMessage
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &packs))
	require.Len(t, packs, 1)
	require.NoError(t, json.Unmarshal(packs[0], &pack))
	require.Equal(t, "Team credits", pack.Name)
	require.Equal(t, int64(5000), pack.AmountCents)
	require.False(t, pack.Enabled)
	require.NotNil(t, pack.PolarProductID)
	require.Equal(t, "prod_credits", *pack.PolarProductID)
	response = request("POST", "credit-packs", `{"name":"Duplicate","amount_cents":1000,"enabled":true,"polar_product_id":"prod_credits"}`)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	for _, body := range []string{
		`{"name":" ","amount_cents":1000}`, `{"name":"Invalid","amount_cents":-1}`,
		`{"name":"Invalid","amount_cents":1}`, `{"name":"Invalid","amount_cents":100000000}`,
	} {
		response = request("PUT", path, body)
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	}
	response = request("PUT", "credit-packs/"+uuid.NewString(), `{"name":"Missing","amount_cents":1000}`)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	response = request("DELETE", path, "")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	response = request("GET", "credit-packs", "")
	require.Equal(t, "[]\n", response.Body.String())
}

func TestAdminCreditPacksControlCheckoutAndGrant(t *testing.T) {
	server, registry, token := setupAdminTestServer(t)
	clearHostedProviderDefaults(t)
	request := newAdminCatalogRequest(server, token)
	ctx := t.Context()
	db := database.DB(ctx)
	_, err := billingmodels.SetAdminOrganizationPlan(db, registry.Organization.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	settings, err := billingmodels.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	original := *settings
	t.Cleanup(func() {
		_, restoreErr := billingmodels.UpdateInstallationLLMSettings(db.WithContext(context.WithoutCancel(ctx)), original)
		require.NoError(t, restoreErr)
	})
	response := request("PATCH", "installation/llm-settings", `{"credit_purchase_fee_bps":500,"credit_purchase_fixed_fee_cents":50}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var checkout struct {
		Products []string          `json:"products"`
		Metadata map[string]string `json:"metadata"`
		Prices   map[string][]struct {
			PriceAmount int64 `json:"price_amount"`
		} `json:"prices"`
	}
	productState := "active"
	checkoutCount := 0
	polarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Polar-Version", r.Header.Get("Polar-Version"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/products/prod_local":
			currency := "usd"
			if productState == "eur" {
				currency = "eur"
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id": "prod_local", "is_recurring": productState == "recurring", "is_archived": productState == "archived",
				"prices": []map[string]any{{"amount_type": "fixed", "price_amount": 1000, "price_currency": currency}},
			}))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/customers/external/"):
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_local"}))
		case r.Method == http.MethodPost && r.URL.Path == "/checkouts/":
			checkoutCount++
			require.NoError(t, json.NewDecoder(r.Body).Decode(&checkout))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"url": "https://buy.example/local", "customer_id": "cust_local"}))
		default:
			t.Errorf("Unexpected Polar request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(polarServer.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", polarServer.URL)
	t.Setenv("POLAR_WEBHOOK_SECRET", "local-pack-secret")

	response = request("POST", "credit-packs", `{"name":"Local credits","amount_cents":2500,"enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var pack billingmodels.CreditPack
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &pack))
	path := "credit-packs/" + pack.ID.String()
	orgID := registry.Organization.ID.String()
	list := func() *pb.ListHostedCreditProductsResponse {
		products, listErr := organizations.ListHostedCreditProducts(ctx, orgID, &pb.ListHostedCreditProductsRequest{})
		require.NoError(t, listErr)
		return products
	}
	require.Empty(t, list().Products, "unlinked packs must not be offered")
	response = request("PUT", path, `{"name":"Local credits","amount_cents":2500,"enabled":false,"polar_product_id":"prod_local"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Empty(t, list().Products, "disabled packs must not be offered")
	_, err = organizations.QuoteHostedCreditCheckout(ctx, orgID, &pb.QuoteHostedCreditCheckoutRequest{ProductId: "prod_local"})
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	response = request("PUT", path, `{"name":"Local credits","amount_cents":2500,"enabled":true,"polar_product_id":"prod_local"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	products := list().Products
	require.Len(t, products, 1)
	require.Equal(t, "prod_local", products[0].Id)
	require.Equal(t, "Local credits", products[0].Name)
	require.Equal(t, int64(2500), products[0].AmountCents)
	quote, err := organizations.QuoteHostedCreditCheckout(ctx, orgID, &pb.QuoteHostedCreditCheckoutRequest{ProductId: "prod_local"})
	require.NoError(t, err)
	require.Equal(t, int64(2500), quote.CreditAmountCents)
	require.Equal(t, int64(2685), quote.SubtotalCents)
	response = request("PUT", path, `{"name":"Custom credits","amount_cents":0,"enabled":true,"polar_product_id":"prod_local"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Zero(t, list().Products[0].AmountCents)
	customQuote, err := organizations.QuoteHostedCreditCheckout(ctx, orgID, &pb.QuoteHostedCreditCheckoutRequest{
		ProductId: "prod_local", CreditAmountCents: 3000,
	})
	require.NoError(t, err)
	require.Equal(t, int64(3000), customQuote.CreditAmountCents, "custom packs must not use the fixed Polar catalog amount")
	require.Equal(t, int64(3211), customQuote.SubtotalCents)
	response = request("PUT", path, `{"name":"Local credits","amount_cents":2500,"enabled":true,"polar_product_id":"prod_local"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	for _, state := range []string{"recurring", "archived", "eur"} {
		productState = state
		_, err = organizations.QuoteHostedCreditCheckout(ctx, orgID, &pb.QuoteHostedCreditCheckoutRequest{ProductId: "prod_local"})
		require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err), state)
	}
	productState = "active"
	checkoutRequest := &pb.CreateHostedCreditCheckoutRequest{ProductId: "prod_local", CreditAmountCents: 2500, ExpectedSubtotalCents: 2685}
	session, err := organizations.CreateHostedCreditCheckout(ctx, orgID, checkoutRequest, registry.Account.ID.String(), "http://localhost:8000")
	require.NoError(t, err)
	require.Equal(t, "https://buy.example/local", session.CheckoutUrl)
	require.Equal(t, []string{"prod_local"}, checkout.Products)
	require.Equal(t, int64(2685), checkout.Prices["prod_local"][0].PriceAmount)
	require.Equal(t, "2500", checkout.Metadata["superplane_credit_amount_cents"])
	response = request("PUT", path, `{"name":"Changed credits","amount_cents":5000,"enabled":true,"polar_product_id":"prod_local"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	_, err = organizations.CreateHostedCreditCheckout(ctx, orgID, checkoutRequest, registry.Account.ID.String(), "http://localhost:8000")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Equal(t, 1, checkoutCount)
	response = request("DELETE", path, "")
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Empty(t, list().Products, "deleting the final pack must not restore legacy products")
	_, err = organizations.CreateHostedCreditCheckout(ctx, orgID, checkoutRequest, registry.Account.ID.String(), "http://localhost:8000")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

	orderID := uuid.NewString()
	metadata, err := json.Marshal(checkout.Metadata)
	require.NoError(t, err)
	eventBody := func(eventType, status string) []byte {
		return []byte(fmt.Sprintf(`{"type":%q,"data":{"id":%q,"status":%q,"billing_reason":"purchase","currency":"usd","net_amount":2685,"metadata":%s,"customer":{"external_id":%q},"product":{"id":"prod_local"}}}`, eventType, orderID, status, metadata, orgID))
	}
	refund := signedPolarWebhook(t, "local-pack-secret", eventBody("order.refunded", "refunded"))
	server.billing.handlePolarWebhook(refund.recorder, refund.request)
	require.Equal(t, http.StatusInternalServerError, refund.recorder.Code, "refund must wait for the paid order")
	for range 2 {
		paid := signedPolarWebhook(t, "local-pack-secret", eventBody("order.paid", "paid"))
		server.billing.handlePolarWebhook(paid.recorder, paid.request)
		require.Equal(t, http.StatusAccepted, paid.recorder.Code)
	}
	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	require.Equal(t, billingmodels.CentsToMicros(2500), grant.AmountMicros)
	refund = signedPolarWebhook(t, "local-pack-secret", eventBody("order.refunded", "refunded"))
	server.billing.handlePolarWebhook(refund.recorder, refund.request)
	require.Equal(t, http.StatusAccepted, refund.recorder.Code)
	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, orderID)
	require.NoError(t, err)
	require.Equal(t, billingmodels.CentsToMicros(2500), reversed)
}
