package organizations

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestSelectBillingPlanRequiresCardWithoutCharging(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	plan := models.BillingPlan{Key: "payg_" + uuid.NewString()[:8], Name: "Pay as you go", Category: "paid", Visibility: "public", RequirePaymentMethod: true, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	hasCard := false
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodGet, req.Method, "selecting this plan must not create a charge")
		require.Equal(t, "/customers/external/"+r.Organization.ID.String()+"/payment-methods", req.URL.Path)
		items := []map[string]any{}
		if hasCard {
			items = append(items, map[string]any{"id": "pm_saved", "type": "card", "method_metadata": map[string]int{"exp_month": 12, "exp_year": time.Now().Year() + 1}})
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}}))
	})
	usePolarTestServer(t, server)
	req := &pb.SelectBillingPlanRequest{PlanKey: plan.Key}
	_, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), req, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	current, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotEqual(t, plan.ID, current.PlanID)
	hasCard = true
	response, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), req, r.Account.ID.String(), "http://localhost")
	require.NoError(t, err)
	require.Empty(t, response.CheckoutUrl)
	current, err = models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, plan.ID, current.PlanID)
	require.NoError(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID))
	hasCard = false
	require.ErrorIs(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID), models.ErrPaymentMethodRequired)
	plan.RequirePaymentMethod = false
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	require.NoError(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID))
}

func TestNewBillingIsOptInAndKeepsMigratedTrial(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	requests := 0
	server := polarAPIServer(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
	})
	usePolarTestServer(t, server)
	require.NoError(t, models.SetOrganizationPolarCustomerID(db, r.Organization.ID, "cust_legacy"))
	trial, err := models.FindBillingPlan(db, models.BillingPlanTrial)
	require.NoError(t, err)
	plan := models.BillingPlan{Key: "opt_in_" + uuid.NewString()[:8], Name: "Opt in", Category: "paid", Visibility: "public", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))

	response, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	require.False(t, response.BillingEnabled)
	require.Empty(t, response.AvailablePlans)
	require.False(t, response.SubscriptionCheckoutEnabled)
	require.False(t, response.RequirePaymentMethod)
	require.True(t, response.HostedExecutionAllowed)
	require.Equal(t, trial.ID.String(), response.CatalogId)
	require.Equal(t, models.BillingPlanTrial, response.CatalogKey)
	require.Equal(t, trial.TrialCreditCents, response.WelcomeRemainingCents)

	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	_, err = CreatePaymentMethodSession(t.Context(), r.Organization.ID.String(), &pb.CreatePaymentMethodSessionRequest{}, r.Account.ID.String())
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Zero(t, requests, "legacy onboarding must not contact Polar")

	plan.RequirePaymentMethod = true
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	require.NoError(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID))
	require.Zero(t, requests, "card requirements must remain off without the flag")
}

func TestSelectPaidPlanStartsCheckoutWithoutChangingPlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	product := "prod_selected_" + uuid.NewString()
	plan := models.BillingPlan{Key: "paid_" + uuid.NewString()[:8], Name: "Paid", Category: "paid", Visibility: "public", PriceCents: 25000, PolarProductID: &product, RequirePaymentMethod: true, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	checkoutCalls := 0
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/customers/external/" + r.Organization.ID.String():
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_existing", "external_id": r.Organization.ID.String()}))
		case "/checkouts/":
			checkoutCalls++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, "http://localhost/org/setup?step=billing&subscribed=1&checkout_id={CHECKOUT_ID}", body["success_url"])
			require.Equal(t, []any{product}, body["products"])
			require.Equal(t, r.Organization.ID.String(), body["external_customer_id"])
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"url": "https://polar.sh/checkout/selected"}))
		default:
			t.Errorf("unexpected Polar request %s", req.URL.Path)
			w.WriteHeader(500)
		}
	})
	usePolarTestServer(t, server)
	response, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key, ReturnPath: "/org/setup?step=billing"}, r.Account.ID.String(), "http://localhost")
	require.NoError(t, err)
	require.Equal(t, "https://polar.sh/checkout/selected", response.CheckoutUrl)
	require.Equal(t, 1, checkoutCalls)
	current, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotEqual(t, plan.ID, current.PlanID, "checkout redirect must not activate the plan")
	for _, returnPath := range []string{"https://evil.example/setup", "//evil.example/setup", `/\evil.example/setup`, "/setup#fragment"} {
		_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key, ReturnPath: returnPath}, r.Account.ID.String(), "http://localhost")
		require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err), returnPath)
	}
	plan.Visibility = "private"
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, 1, checkoutCalls)
}

func TestPaymentMethodSetupCreatesCustomerWithoutCheckout(t *testing.T) {
	sessionToken := "polar_cst_setup"
	r := support.Setup(t)
	defer r.Close()
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(database.DB(t.Context()), r.Organization.ID, features.FeatureNewBilling))
	created := false
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/customers/external/" + r.Organization.ID.String():
			w.WriteHeader(http.StatusNotFound)
		case "/customers/":
			require.Equal(t, http.MethodPost, req.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, r.Organization.ID.String(), body["external_id"])
			created = true
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_new", "external_id": r.Organization.ID.String()}))
		case "/customer-sessions/":
			require.True(t, created)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"customer_id": "cust_new", "customer_portal_url": "https://polar.sh/portal/payment-settings", "token": sessionToken}))
		default:
			t.Errorf("card setup must not create a checkout or subscription: %s", req.URL.Path)
			w.WriteHeader(500)
		}
	})
	usePolarTestServer(t, server)
	response, err := CreatePaymentMethodSession(t.Context(), r.Organization.ID.String(), &pb.CreatePaymentMethodSessionRequest{}, r.Account.ID.String())
	require.NoError(t, err)
	require.Equal(t, "polar_cst_setup", response.SessionToken)
	settings, err := models.FindOrganizationLLMSettings(database.DB(t.Context()), r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, "cust_new", *settings.PolarCustomerID)
	sessionToken = ""
	_, err = CreatePaymentMethodSession(t.Context(), r.Organization.ID.String(), &pb.CreatePaymentMethodSessionRequest{}, r.Account.ID.String())
	require.Equal(t, codes.Internal, grpcerrors.Code(err))
}
