package organizations

import (
	"context"
	"encoding/json"
	"net/http"
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
		t.Errorf("selecting None must not contact Polar: %s", req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
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
			checkoutCalls++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, "http://localhost/org/setup?step=billing&subscribed=1&checkout_id={CHECKOUT_ID}", body["success_url"])
			require.Equal(t, []any{product}, body["products"])
			for _, field := range []string{"seats", "min_seats", "max_seats"} {
				require.NotContains(t, body, field)
			}
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
	failCheckout := false
	calls := 0
	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/customers/external/" + r.Organization.ID.String():
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_team", "external_id": r.Organization.ID.String()}))
		case "/customers/external/" + r.Organization.ID.String() + "/payment-methods":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
		case "/checkouts/":
			calls++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			for _, field := range []string{"seats", "min_seats", "max_seats"} {
				require.Equal(t, float64(selected), body[field], field)
			}
			require.Equal(t, []any{product}, body["products"])
			if failCheckout {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"url": "https://polar.sh/checkout/team"}))
		default:
			t.Errorf("unexpected Polar request %s", req.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	usePolarTestServer(t, server)
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
			require.Equal(t, "https://polar.sh/checkout/team", response.CheckoutUrl)
			after, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before.PlanID, after.PlanID, "abandoned checkout must preserve Trial")
			require.Equal(t, before.PurchasedSeats, after.PurchasedSeats)
		})
	}
	require.Equal(t, 4, calls)
	failCheckout = true
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
