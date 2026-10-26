package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
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
		if req.URL.Path == "/checkouts/" {
			writeFakeJSON(t, w, map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}})
			return
		}
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
	require.Equal(t, models.BillingPlanSourceSystem, current.PlanSource)
	require.True(t, current.AllowsHostedExecution(time.Now()))
	require.True(t, current.AllowsCreditPurchase())
	require.NoError(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID))
	hasCard = false
	require.ErrorIs(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID), models.ErrPaymentMethodRequired)
	plan.RequirePaymentMethod = false
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	require.NoError(t, polar.AssertOrganizationPaymentMethod(db, r.Organization.ID))
}

func TestSelectNoneBillingPlanWhenNoPublicPlansExist(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	cleanupDB := database.DB(context.WithoutCancel(t.Context()))
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	plans, err := models.ListBillingPlans(db)
	require.NoError(t, err)
	for _, plan := range plans {
		if plan.Category != "paid" || plan.Visibility != "public" {
			continue
		}
		original := plan
		plan.Visibility = "private"
		require.NoError(t, models.SaveBillingPlan(db, &plan))
		t.Cleanup(func() { require.NoError(t, models.SaveBillingPlan(cleanupDB, &original)) })
	}
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodGet, req.Method)
		require.Equal(t, "/checkouts/", req.URL.Path)
		writeFakeJSON(t, w, map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}})
	})
	usePolarTestServer(t, server)
	plan := models.BillingPlan{Key: "private_" + uuid.NewString()[:8], Name: "Private", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	t.Cleanup(func() { require.NoError(t, models.DeleteBillingPlan(cleanupDB, plan.ID)) })
	req := &pb.SelectBillingPlanRequest{PlanKey: models.BillingPlanNone}
	response, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), req, r.Account.ID.String(), "http://localhost")
	require.NoError(t, err)
	require.Empty(t, response.CheckoutUrl)
	current, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.BillingPlanNone, current.Plan)
	require.Equal(t, models.BillingPlanNone, current.Catalog.Key)
	require.Equal(t, models.BillingPlanSourceSystem, current.PlanSource)
	require.False(t, current.AllowsHostedExecution(time.Now()))
	require.False(t, current.AllowsCreditPurchase())

	plan.Visibility = "public"
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), req, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
}

func TestNewBillingIsOptInAndKeepsMigratedTrial(t *testing.T) {
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")
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
	require.True(t, response.BillingEnabled)
	require.Empty(t, response.AvailablePlans)
	require.False(t, response.RequirePaymentMethod)
	require.True(t, response.HostedExecutionAllowed)
	require.Equal(t, models.BillingPlanTrial, response.CatalogKey)
	require.Equal(t, trial.TrialCreditCents, response.WelcomeRemainingCents)

	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	_, err = CreatePaymentMethodSession(t.Context(), r.Organization.ID.String(), &pb.CreatePaymentMethodSessionRequest{}, r.Account.ID.String())
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Zero(t, requests, "legacy onboarding must not contact Polar")

	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	plan.RequirePaymentMethod = true
	require.NoError(t, models.SaveBillingPlan(db, &plan))
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
			if req.Method == http.MethodGet {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			checkoutCalls++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, "http://localhost/org/setup?step=billing&subscribed=1&checkout_id={CHECKOUT_ID}", body["success_url"])
			require.Equal(t, []any{product}, body["products"])
			for _, field := range []string{"seats", "min_seats", "max_seats"} {
				require.NotContains(t, body, field)
			}
			require.Equal(t, r.Organization.ID.String(), body["external_customer_id"])
			require.Contains(t, body["metadata"], polar.CheckoutIntentMetadataKey)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "chk_selected", "status": "open", "url": "https://polar.sh/checkout/selected"}))
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

func TestSelectBillingPlanBlocksPausedSubscription(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	free := models.BillingPlan{Key: "free_" + uuid.NewString()[:8], Name: "Free", Category: "paid", Visibility: "public", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &free))
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		t.Errorf("a paused subscription must not contact Polar: %s", req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	usePolarTestServer(t, server)

	product := "prod_paused_" + uuid.NewString()
	paid := models.BillingPlan{Key: "paid_" + uuid.NewString()[:8], Name: "Paid", Category: "paid", Visibility: "public", PriceCents: 25000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &paid))
	start := time.Now().UTC().Truncate(time.Second)
	end := start.AddDate(0, 1, 0)
	payload, err := json.Marshal(map[string]any{"id": "sub_" + uuid.NewString(), "product_id": product, "status": "paused", "external_customer_id": r.Organization.ID.String(), "current_period_start": start, "current_period_end": end, "modified_at": start})
	require.NoError(t, err)
	var snapshot polar.SubscriptionData
	require.NoError(t, json.Unmarshal(payload, &snapshot))
	require.NoError(t, polar.ApplySubscription(t.Context(), db, snapshot))

	// A local plan change would clear the stored subscription state, so a
	// paused subscription must block it the same way a paid one does.
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: free.Key}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Contains(t, err.Error(), "Manage your current subscription in the billing portal before selecting another plan.")

	// The blocked selection preserves the subscription state, so the deletion
	// guard still sees a live subscription.
	current, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.BillingPlanSourcePolar, current.PlanSource)
	require.NotNil(t, current.PolarSubscriptionID)
	require.Equal(t, models.PolarSubscriptionStatusPaused, current.PolarSubscriptionStatus)
	require.True(t, current.HasLivePolarSubscription(time.Now()))
	require.True(t, current.MayBeBillable(time.Now()))
}

func TestSelectBillingPlanBlocksElapsedScheduledCancellation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	usePolarTestServer(t, polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		t.Errorf("an elapsed scheduled cancellation must not start checkout: %s", req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	product := "prod_elapsed_" + uuid.NewString()
	paid := models.BillingPlan{Key: "paid_" + uuid.NewString()[:8], Name: "Paid", Category: "paid", Visibility: "public", PriceCents: 25000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &paid))
	end := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	start := end.AddDate(0, -1, 0)
	payload, err := json.Marshal(map[string]any{
		"id": "sub_" + uuid.NewString(), "product_id": product, "status": "active", "cancel_at_period_end": true,
		"external_customer_id": r.Organization.ID.String(), "current_period_start": start, "current_period_end": end, "modified_at": start,
	})
	require.NoError(t, err)
	var snapshot polar.SubscriptionData
	require.NoError(t, json.Unmarshal(payload, &snapshot))
	require.NoError(t, polar.ApplySubscription(t.Context(), db, snapshot))

	// The period elapsed, but Polar can resume the subscription. Only a
	// terminal status proves that a replacement checkout cannot double bill.
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: paid.Key}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Contains(t, err.Error(), "Manage your current subscription in the billing portal before selecting another plan.")
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

func TestSelectPerSeatPlanValidatesBeforePolar(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	member := support.CreateUser(t, r, r.Organization.ID)
	product := "prod_team_" + uuid.NewString()
	team := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public", PerSeat: true, SeatLimit: new(5), PriceCents: 2000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &team))
	unlinked := models.BillingPlan{Key: "unlinked_" + uuid.NewString()[:8], Name: "Unlinked", Category: "paid", Visibility: "public", PerSeat: true, SeatLimit: new(5), CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &unlinked))
	flat := models.BillingPlan{Key: "flat_" + uuid.NewString()[:8], Name: "Flat", Category: "paid", Visibility: "public", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &flat))
	before, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	calls := 0
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		calls++
		t.Errorf("invalid checkout contacted Polar: %s", req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	usePolarTestServer(t, server)
	for _, tc := range []struct {
		name, key string
		seats     *int32
		code      codes.Code
		message   string
	}{
		{"missing", team.Key, nil, codes.InvalidArgument, "Select a seat count for this plan."},
		{"zero", team.Key, new(int32(0)), codes.InvalidArgument, "Select at least 2 seats to cover current members."},
		{"negative", team.Key, new(int32(-1)), codes.InvalidArgument, "Select at least 2 seats to cover current members."},
		{"insufficient", team.Key, new(int32(1)), codes.InvalidArgument, "Select at least 2 seats to cover current members."},
		{"excessive", team.Key, new(int32(6)), codes.InvalidArgument, "The Team plan allows up to 5 seats."},
		{"flat zero", flat.Key, new(int32(0)), codes.InvalidArgument, "Seats are only available for per-seat plans."},
		{"flat positive", flat.Key, new(int32(3)), codes.InvalidArgument, "Seats are only available for per-seat plans."},
		{"none zero", models.BillingPlanNone, new(int32(0)), codes.InvalidArgument, "Seats are only available for per-seat plans."},
		{"unlinked zero price", unlinked.Key, new(int32(3)), codes.FailedPrecondition, "Checkout is not available for this plan."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: tc.key, Seats: tc.seats}, r.Account.ID.String(), "http://localhost")
			require.Equal(t, tc.code, grpcerrors.Code(err))
			require.Contains(t, err.Error(), tc.message)
			require.Zero(t, calls)
			after, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before.PlanID, after.PlanID)
			require.Equal(t, before.PurchasedSeats, after.PurchasedSeats)
		})
	}
	// A lower override replaces the target catalog maximum, even below occupancy.
	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, new(1)))
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: team.Key, Seats: new(int32(2))}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Contains(t, err.Error(), "The Team plan allows up to 1 seats.")
	require.Zero(t, calls)
	require.NoError(t, db.Delete(member).Error)
	require.NoError(t, models.DeleteOrganizationSeatLimitOverride(db, r.Organization.ID))
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: team.Key, Seats: new(int32(0))}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Contains(t, err.Error(), "Select at least 1 seats to cover current members.")
	require.Zero(t, calls)
	empty, err := models.CreateOrganization("empty-checkout-"+uuid.NewString()[:8], "")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, empty.ID, features.FeatureNewBilling))
	_, err = SelectBillingPlan(t.Context(), empty.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: team.Key, Seats: new(int32(0))}, r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Contains(t, err.Error(), "Select at least 1 seats to cover current members.")
	require.Zero(t, calls)
}

func TestTeamCheckoutCapacityLifecycle(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	support.CreateUser(t, r, r.Organization.ID)
	product := "prod_team_" + uuid.NewString()
	team := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public", PerSeat: true, SeatLimit: new(3), PriceCents: 2000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &team))
	before, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.BillingPlanTrial, before.Catalog.Key)
	selected := 3
	polarCheckouts := newFakePolarCheckouts(t, r.Organization.ID)
	polarCheckouts.onCreate = func(body map[string]any) {
		for _, field := range []string{"seats", "min_seats", "max_seats"} {
			require.Equal(t, float64(selected), body[field], field)
		}
		require.Equal(t, []any{product}, body["products"])
	}
	polarCheckouts.serve(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/customers/external/"+r.Organization.ID.String()+"/payment-methods" {
			t.Errorf("unexpected Polar request %s", req.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
	})
	for _, tc := range []struct {
		name     string
		override bool
		maximum  *int
		selected int
	}{
		{"target catalog instead of Trial limit", false, new(3), 3},
		{"higher override", true, new(8), 8},
		{"lower override", true, new(2), 2},
		{"unlimited override", true, nil, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.override {
				require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, tc.maximum))
			}
			billing, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
			require.NoError(t, err)
			require.NotNil(t, billing.OccupiedSeats)
			require.EqualValues(t, 2, *billing.OccupiedSeats)
			found := false
			for _, option := range billing.AvailablePlans {
				if option.Key != team.Key {
					continue
				}
				found = true
				if tc.maximum == nil {
					require.Nil(t, option.SeatMaximum)
				} else {
					require.EqualValues(t, *tc.maximum, *option.SeatMaximum)
				}
			}
			require.True(t, found, "the target Team plan must be available")
			selected = tc.selected
			response, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: team.Key, Seats: new(int32(selected))}, r.Account.ID.String(), "http://localhost")
			require.NoError(t, err)
			created := polarCheckouts.checkouts()
			require.Equal(t, "https://polar.sh/checkout/"+created[len(created)-1].ID, response.CheckoutUrl)
			after, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before.PlanID, after.PlanID, "abandoned checkout must preserve Trial")
			require.Equal(t, before.PurchasedSeats, after.PurchasedSeats)
			// Polar expires the abandoned session before the next selection.
			polarCheckouts.setStatus(created[len(created)-1], "expired")
		})
	}
	require.Equal(t, 4, polarCheckouts.createCalls())
	polarCheckouts.loseResponse = "502"
	_, err = SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: team.Key, Seats: new(int32(selected))}, r.Account.ID.String(), "http://localhost")
	require.Error(t, err)
	unchanged, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, before.PlanID, unchanged.PlanID)
	require.Equal(t, before.PurchasedSeats, unchanged.PurchasedSeats)
	start := time.Now().UTC().Truncate(time.Second)
	end := start.AddDate(0, 1, 0)
	payload, err := json.Marshal(map[string]any{"id": "sub_" + uuid.NewString(), "product_id": product, "status": "active", "seats": 3, "external_customer_id": r.Organization.ID.String(), "current_period_start": start, "current_period_end": end, "modified_at": start})
	require.NoError(t, err)
	var snapshot polar.SubscriptionData
	require.NoError(t, json.Unmarshal(payload, &snapshot))
	require.NoError(t, polar.ApplySubscription(t.Context(), db, snapshot))
	// A webhook replay must preserve the same purchased capacity.
	require.NoError(t, polar.ApplySubscription(t.Context(), db, snapshot))
	active, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, team.ID, active.PlanID)
	require.True(t, active.CurrentPeriodStart.Equal(start))
	require.True(t, active.CurrentPeriodEnd.Equal(end))
	seats, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, new(3), seats.Purchased)
	require.Equal(t, 2, seats.Occupied)
	require.Equal(t, new(1), seats.Available)
}

type teamCheckoutFixture struct {
	t        *testing.T
	r        *support.ResourceRegistry
	team     models.BillingPlan
	product  string
	provider *fakePolarCheckouts
}

func newTeamCheckoutFixture(t *testing.T) *teamCheckoutFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	product := "prod_team_" + uuid.NewString()
	team := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public", PerSeat: true, SeatLimit: new(10), PriceCents: 2000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &team))
	f := &teamCheckoutFixture{t: t, r: r, team: team, product: product, provider: newFakePolarCheckouts(t, r.Organization.ID)}
	f.provider.serve(nil)
	return f
}

func (f *teamCheckoutFixture) selectSeats(seats int) (*pb.CreateBusinessCheckoutResponse, error) {
	f.t.Helper()
	return SelectBillingPlan(f.t.Context(), f.r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: f.team.Key, Seats: new(int32(seats))}, f.r.Account.ID.String(), "http://localhost")
}

func (f *teamCheckoutFixture) intents() []models.OrganizationBillingCheckout {
	f.t.Helper()
	var intents []models.OrganizationBillingCheckout
	require.NoError(f.t, database.DB(f.t.Context()).Where("organization_id = ?", f.r.Organization.ID).Order("created_at ASC").Find(&intents).Error)
	return intents
}

func (f *teamCheckoutFixture) updateIntent(id uuid.UUID, columns map[string]any) {
	f.t.Helper()
	require.NoError(f.t, database.DB(f.t.Context()).Model(&models.OrganizationBillingCheckout{}).Where("id = ?", id).Updates(columns).Error)
}

func (f *teamCheckoutFixture) checkoutURL(checkout *fakePolarCheckout) string {
	return "https://polar.sh/checkout/" + checkout.ID
}

func TestConcurrentTeamCheckoutRequestsCreateOnePayableCheckout(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	f.provider.createDelay = 200 * time.Millisecond

	const requests = 5
	var wg sync.WaitGroup
	start := make(chan struct{})
	urls := make([]string, requests)
	errs := make([]error, requests)
	for i := range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			response, err := f.selectSeats(3)
			errs[i] = err
			if err == nil {
				urls[i] = response.CheckoutUrl
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Equal(t, 1, f.provider.createCalls(), "Polar must receive one create call")
	created := f.provider.checkouts()
	require.Len(t, created, 1)
	succeeded := 0
	for i := range requests {
		if errs[i] != nil {
			require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(errs[i]))
			continue
		}
		succeeded++
		require.Equal(t, f.checkoutURL(created[0]), urls[i])
	}
	require.GreaterOrEqual(t, succeeded, 1)
	intents := f.intents()
	require.Len(t, intents, 1)
	require.Equal(t, models.BillingCheckoutStateOpen, intents[0].State)
	require.Equal(t, created[0].ID, intents[0].PolarCheckoutIDValue())
	require.Equal(t, new(3), intents[0].Seats)
}

func TestTeamCheckoutCallsPolarWithoutHoldingTheOrganizationLock(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	f.provider.requireUnlockedOrganization = true

	f.provider.loseResponse = "502"
	_, err := f.selectSeats(3) // create call
	require.Error(t, err)
	f.provider.loseResponse = ""
	_, err = f.selectSeats(3) // list call, adopts the checkout
	require.NoError(t, err)
	_, err = f.selectSeats(3) // get call
	require.NoError(t, err)
	require.Equal(t, 1, f.provider.createCalls())
}

func TestTeamCheckoutRecoversFromLostCreateResponse(t *testing.T) {
	for _, loss := range []string{"502", "drop"} {
		t.Run(loss, func(t *testing.T) {
			f := newTeamCheckoutFixture(t)
			f.provider.loseResponse = loss

			_, err := f.selectSeats(3)
			require.Error(t, err)
			intents := f.intents()
			require.Len(t, intents, 1)
			require.Equal(t, models.BillingCheckoutStateAmbiguous, intents[0].State)
			require.Nil(t, intents[0].PolarCheckoutID, "no Polar identifier is known yet")

			f.provider.loseResponse = ""
			response, err := f.selectSeats(3)
			require.NoError(t, err)
			created := f.provider.checkouts()
			require.Len(t, created, 1)
			require.Equal(t, 1, f.provider.createCalls(), "recovery must not create a second checkout")
			require.Equal(t, f.checkoutURL(created[0]), response.CheckoutUrl)
			intents = f.intents()
			require.Len(t, intents, 1)
			require.Equal(t, models.BillingCheckoutStateOpen, intents[0].State)
			require.Equal(t, created[0].ID, intents[0].PolarCheckoutIDValue())
		})
	}
}

func TestAmbiguousTeamCheckoutWithoutMatchWaitsForGraceWindow(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	f.provider.loseResponse = "502"
	f.provider.recordLost = false
	_, err := f.selectSeats(3)
	require.Error(t, err)
	f.provider.loseResponse = ""

	_, err = f.selectSeats(3)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Equal(t, 1, f.provider.createCalls(), "a lost response must not allow another checkout at once")

	intents := f.intents()
	require.Len(t, intents, 1)
	f.updateIntent(intents[0].ID, map[string]any{"created_at": time.Now().Add(-11 * time.Minute)})
	response, err := f.selectSeats(3)
	require.NoError(t, err)
	require.Equal(t, 2, f.provider.createCalls())
	created := f.provider.checkouts()
	require.Len(t, created, 1)
	require.Equal(t, f.checkoutURL(created[0]), response.CheckoutUrl)
	intents = f.intents()
	require.Len(t, intents, 2)
	require.Equal(t, models.BillingCheckoutStateReleased, intents[0].State)
	require.Equal(t, models.BillingCheckoutStateOpen, intents[1].State)
}

func TestStaleCreatingTeamCheckoutIsReconciledLikeAnAmbiguousOne(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	f.provider.loseResponse = "502"
	_, err := f.selectSeats(3)
	require.Error(t, err)
	f.provider.loseResponse = ""
	// The server stopped after the claim and before it recorded the outcome.
	intents := f.intents()
	require.Len(t, intents, 1)
	f.updateIntent(intents[0].ID, map[string]any{"state": models.BillingCheckoutStateCreating, "created_at": time.Now().Add(-11 * time.Minute)})

	response, err := f.selectSeats(3)
	require.NoError(t, err)
	require.Equal(t, 1, f.provider.createCalls())
	require.Equal(t, f.checkoutURL(f.provider.checkouts()[0]), response.CheckoutUrl)
}

func TestTeamCheckoutInFlightCreateBlocksOtherRequests(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	claimed, created, err := models.ClaimOrganizationBillingCheckout(database.DB(t.Context()), f.r.Organization.ID, f.team.Key, f.product, new(3))
	require.NoError(t, err)
	require.True(t, created)

	_, err = f.selectSeats(3)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Zero(t, f.provider.createCalls())
	require.Equal(t, claimed.ID, f.intents()[0].ID)
}

func TestOpenTeamCheckoutStaysUnresolvedUntilPolarSaysOtherwise(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	first, err := f.selectSeats(3)
	require.NoError(t, err)
	checkout := f.provider.checkouts()[0]
	intent := f.intents()[0]
	require.NotNil(t, intent.ExpiresAt, "the Polar expiry is stored")

	// Browser closure and a past local expiry do not cancel the checkout.
	f.updateIntent(intent.ID, map[string]any{"expires_at": time.Now().Add(-time.Hour)})
	for _, status := range []string{"open", "confirmed"} {
		f.provider.setStatus(checkout, status)
		again, err := f.selectSeats(3)
		require.NoError(t, err, status)
		require.Equal(t, first.CheckoutUrl, again.CheckoutUrl, status)
		_, err = f.selectSeats(4)
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), status)
		require.Equal(t, "A checkout is already in progress. Finish it or wait until it expires.", grpcerrors.StatusMessage(err))
		require.Equal(t, 1, f.provider.createCalls(), status)
		require.Equal(t, models.BillingCheckoutStateOpen, f.intents()[0].State, status)
	}

	f.provider.setStatus(checkout, "confirmed")
	_, err = f.selectSeats(4)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), "a confirmed checkout is not a payment result")
}

func TestTeamCheckoutIsReleasedByAVerifiedTerminalStatus(t *testing.T) {
	for _, status := range []string{"expired", "failed"} {
		t.Run(status, func(t *testing.T) {
			f := newTeamCheckoutFixture(t)
			_, err := f.selectSeats(3)
			require.NoError(t, err)
			f.provider.setStatus(f.provider.checkouts()[0], status)

			response, err := f.selectSeats(4)
			require.NoError(t, err)
			require.Equal(t, 2, f.provider.createCalls())
			created := f.provider.checkouts()
			require.Equal(t, f.checkoutURL(created[1]), response.CheckoutUrl)
			intents := f.intents()
			require.Len(t, intents, 2)
			require.Equal(t, status, intents[0].State)
			require.NotNil(t, intents[0].ResolvedAt)
			require.Equal(t, models.BillingCheckoutStateOpen, intents[1].State)
			require.Equal(t, new(4), intents[1].Seats)
		})
	}
}

func TestPaidTeamCheckoutResolvesAndGuardsAgainstAnUnappliedSubscription(t *testing.T) {
	for _, tc := range []struct {
		name                string
		subscriptionVisible bool
		readFails           bool
	}{
		{"subscription not visible yet", false, false},
		{"subscription visible at Polar", true, false},
		{"subscription read fails", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newTeamCheckoutFixture(t)
			start := time.Now().UTC().Truncate(time.Second)
			f.provider.serve(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/subscriptions/" {
					t.Errorf("unexpected Polar request %s", req.URL.Path)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				items := []any{}
				if tc.readFails {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				if tc.subscriptionVisible {
					items = append(items, map[string]any{"id": "sub_" + f.r.Organization.ID.String(), "product_id": f.product, "status": "active", "seats": 3, "external_customer_id": f.r.Organization.ID.String(), "current_period_start": start, "current_period_end": start.AddDate(0, 1, 0), "modified_at": start})
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}}))
			})
			_, err := f.selectSeats(3)
			require.NoError(t, err)
			f.provider.setStatus(f.provider.checkouts()[0], "succeeded")

			_, err = f.selectSeats(3)
			intents := f.intents()
			if tc.readFails {
				require.Error(t, err)
				require.Equal(t, models.BillingCheckoutStateOpen, intents[0].State)
				_, err = f.selectSeats(4)
				require.Error(t, err)
				require.Len(t, f.intents(), 1)
				require.Equal(t, 1, f.provider.createCalls())
				return
			}
			require.Equal(t, models.BillingCheckoutStateSucceeded, intents[0].State)
			if tc.subscriptionVisible {
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
				require.Len(t, intents, 1)
				require.Equal(t, 1, f.provider.createCalls())
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2, f.provider.createCalls())
		})
	}
}

func TestRejectedTeamCheckoutReleasesTheIntent(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	f.provider.rejectStatus = http.StatusUnprocessableEntity
	_, err := f.selectSeats(3)
	require.Error(t, err)
	intents := f.intents()
	require.Len(t, intents, 1)
	require.Equal(t, models.BillingCheckoutStateReleased, intents[0].State)
	require.Contains(t, *intents[0].LastError, "422")

	f.provider.rejectStatus = 0
	_, err = f.selectSeats(3)
	require.NoError(t, err)
	require.Equal(t, 2, f.provider.createCalls())
}

func TestPaidCheckoutBlocksClaimsDuringSubscriptionVerification(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var first sync.Once
	f.provider.serve(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/subscriptions/", req.URL.Path)
		first.Do(func() {
			close(entered)
			<-release
		})
		w.WriteHeader(http.StatusBadGateway)
	})
	_, err := f.selectSeats(3)
	require.NoError(t, err)
	f.provider.setStatus(f.provider.checkouts()[0], "succeeded")
	finished := make(chan error, 1)
	go func() {
		_, err := f.selectSeats(3)
		finished <- err
	}()
	<-entered
	// Inspect the guard while the external verification remains in flight.
	intent, claimed, claimErr := models.ClaimOrganizationBillingCheckout(database.DB(t.Context()), f.r.Organization.ID, f.team.Key, f.product, new(4))
	close(release)
	require.Error(t, <-finished)
	require.NoError(t, claimErr)
	require.False(t, claimed)
	require.Equal(t, f.intents()[0].ID, intent.ID)
	require.Equal(t, 1, f.provider.createCalls())
}

func TestLiveSubscriptionResolvesThePendingTeamCheckout(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	db := database.DB(t.Context())
	_, err := f.selectSeats(3)
	require.NoError(t, err)
	subscriptionID := "sub_" + uuid.NewString()
	checkoutID := f.provider.checkouts()[0].ID
	apply := func(status string, modifiedAt time.Time) {
		start := modifiedAt.UTC().Truncate(time.Second)
		payload, err := json.Marshal(map[string]any{"id": subscriptionID, "product_id": f.product, "checkout_id": checkoutID, "status": status, "seats": 3, "external_customer_id": f.r.Organization.ID.String(), "current_period_start": start, "current_period_end": start.AddDate(0, 1, 0), "modified_at": start})
		require.NoError(t, err)
		var snapshot polar.SubscriptionData
		require.NoError(t, json.Unmarshal(payload, &snapshot))
		require.NoError(t, polar.ApplySubscription(t.Context(), db, snapshot))
	}

	apply("incomplete", time.Now().Add(-time.Minute))
	require.Equal(t, models.BillingCheckoutStateOpen, f.intents()[0].State, "an incomplete subscription is not live")
	apply("active", time.Now())
	intents := f.intents()
	require.Equal(t, models.BillingCheckoutStateSucceeded, intents[0].State)
	require.NotNil(t, intents[0].ResolvedAt)
	// The claim itself must reject a stale request that passed handler checks
	// before the webhook activated the subscription and resolved its intent.
	_, claimed, err := models.ClaimOrganizationBillingCheckout(db, f.r.Organization.ID, f.team.Key, f.product, new(4))
	require.ErrorIs(t, err, models.ErrBillingCheckoutLiveSubscription)
	require.False(t, claimed)
	require.Len(t, f.intents(), 1)
}

func TestPreexistingPolarCheckoutIsAdoptedWithoutAnotherCreate(t *testing.T) {
	t.Run("matching session", func(t *testing.T) {
		f := newTeamCheckoutFixture(t)
		f.provider.seed(f.product, new(2), map[string]any{"campaign": "old"})
		legacy := f.provider.seed(f.product, new(3), nil)

		response, err := f.selectSeats(3)
		require.NoError(t, err)
		require.Zero(t, f.provider.createCalls())
		require.Equal(t, f.checkoutURL(legacy), response.CheckoutUrl)
		intents := f.intents()
		require.Len(t, intents, 1)
		require.Equal(t, models.BillingCheckoutStateOpen, intents[0].State)
		require.Equal(t, legacy.ID, intents[0].PolarCheckoutIDValue(), "the real Polar identifier is stored")
	})
	t.Run("different seats", func(t *testing.T) {
		f := newTeamCheckoutFixture(t)
		f.provider.seed(f.product, new(5), nil)

		_, err := f.selectSeats(3)
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		require.Zero(t, f.provider.createCalls())
	})
	t.Run("different subscription product", func(t *testing.T) {
		f := newTeamCheckoutFixture(t)
		product := "prod_old_" + uuid.NewString()
		plan := models.BillingPlan{Key: "old_" + uuid.NewString()[:8], Name: "Old", Category: "paid", Visibility: "private", PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
		require.NoError(t, models.SaveBillingPlan(database.DB(t.Context()), &plan))
		legacy := f.provider.seed(product, nil, nil)

		_, err := f.selectSeats(3)
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		require.Zero(t, f.provider.createCalls())
		intents := f.intents()
		require.Len(t, intents, 1)
		require.Equal(t, legacy.ID, intents[0].PolarCheckoutIDValue())
		require.Equal(t, plan.Key, intents[0].PlanKey)
		require.Equal(t, product, intents[0].PolarProductID)
	})
	t.Run("no payable session", func(t *testing.T) {
		f := newTeamCheckoutFixture(t)
		f.provider.setStatus(f.provider.seed(f.product, new(3), nil), "expired")

		_, err := f.selectSeats(3)
		require.NoError(t, err)
		require.Equal(t, 1, f.provider.createCalls())
		require.Len(t, f.intents(), 1)
	})
}

func TestPendingCheckoutBlocksLocalPlanChanges(t *testing.T) {
	for _, planKey := range []string{models.BillingPlanNone, "local"} {
		t.Run(planKey, func(t *testing.T) {
			f := newTeamCheckoutFixture(t)
			db := database.DB(t.Context())
			_, err := f.selectSeats(3)
			require.NoError(t, err)
			before, err := models.FindOrganizationBillingPlan(db, f.r.Organization.ID)
			require.NoError(t, err)
			if planKey == "local" {
				plan := models.BillingPlan{Key: "local_" + uuid.NewString()[:8], Name: "Local", Category: "paid", Visibility: "public", CreditFrequency: "none", CreditExpiration: "never"}
				require.NoError(t, models.SaveBillingPlan(db, &plan))
				planKey = plan.Key
			} else {
				plans, err := models.ListBillingPlans(db)
				require.NoError(t, err)
				for _, plan := range plans {
					if plan.Category != "paid" || plan.Visibility != "public" {
						continue
					}
					original := plan
					plan.Visibility = "private"
					require.NoError(t, models.SaveBillingPlan(db, &plan))
					t.Cleanup(func() {
						require.NoError(t, models.SaveBillingPlan(database.DB(context.WithoutCancel(t.Context())), &original))
					})
				}
			}
			_, err = SelectBillingPlan(t.Context(), f.r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: planKey}, f.r.Account.ID.String(), "http://localhost")
			require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			require.Contains(t, err.Error(), "checkout is already in progress")
			after, err := models.FindOrganizationBillingPlan(db, f.r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before.PlanID, after.PlanID)
			require.Equal(t, models.BillingCheckoutStateOpen, f.intents()[0].State)
			require.Equal(t, 1, f.provider.createCalls())
			f.provider.setStatus(f.provider.checkouts()[0], "expired")
			response, err := SelectBillingPlan(t.Context(), f.r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: planKey}, f.r.Account.ID.String(), "http://localhost")
			require.NoError(t, err)
			require.Empty(t, response.CheckoutUrl)
			after, err = models.FindOrganizationBillingPlan(db, f.r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, planKey, after.Catalog.Key)
			require.Equal(t, models.BillingCheckoutStateExpired, f.intents()[0].State)
		})
	}
}

func TestLocalSelectionRechecksCheckoutClaimedDuringCardVerification(t *testing.T) {
	f := newTeamCheckoutFixture(t)
	db := database.DB(t.Context())
	plan := models.BillingPlan{Key: "local_" + uuid.NewString()[:8], Name: "Local", Category: "paid", Visibility: "public", RequirePaymentMethod: true, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	before, err := models.FindOrganizationBillingPlan(db, f.r.Organization.ID)
	require.NoError(t, err)
	f.provider.serve(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/customers/external/"+f.r.Organization.ID.String()+"/payment-methods", req.URL.Path)
		_, claimed, err := models.ClaimOrganizationBillingCheckout(db, f.r.Organization.ID, f.team.Key, f.product, new(3))
		require.NoError(t, err)
		require.True(t, claimed)
		writeFakeJSON(t, w, map[string]any{"items": []any{map[string]any{"id": "pm_saved", "type": "card", "method_metadata": map[string]int{"exp_month": 12, "exp_year": time.Now().Year() + 1}}}, "pagination": map[string]int{"max_page": 1}})
	})
	_, err = SelectBillingPlan(t.Context(), f.r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key}, f.r.Account.ID.String(), "http://localhost")
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	after, err := models.FindOrganizationBillingPlan(db, f.r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, before.PlanID, after.PlanID)
	require.Equal(t, models.BillingCheckoutStateCreating, f.intents()[0].State)
}
