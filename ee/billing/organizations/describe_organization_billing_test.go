package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

func Test__DescribeOrganizationBillingHostedCreditAvailableUsesExactBalance(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	// A sub-cent remainder rounds down to zero cents, but the run gate stays open.
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   5000,
		ExpectedMicros: before.WelcomeRemainingMicros,
		Note:           "support",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Zero(t, resp.RemainingCreditCents)
	assert.True(t, resp.HostedCreditAvailable)
	require.NoError(t, billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID))

	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   0,
		ExpectedMicros: 5000,
		Note:           "support",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)

	resp, err = DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Zero(t, resp.RemainingCreditCents)
	assert.False(t, resp.HostedCreditAvailable)
	assert.ErrorIs(t, billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID), billingmodels.ErrHostedCreditEmpty)
}

func Test__DescribeOrganizationBillingCatalogPlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	catalog := billingmodels.BillingPlan{
		Key: "billing_response_custom", Name: "Team", Category: "paid", Visibility: "private",
		PriceCents: 2900, PerSeat: true, SeatLimit: new(50), PolarProductID: new("prod_billing_response_custom"),
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		require.NoError(t, database.Conn().Model(&catalog).Update("polar_product_id", nil).Error)
		require.NoError(t, billingmodels.DeleteBillingPlan(database.Conn(), catalog.ID))
	})
	periodStart := time.Now().UTC()
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, err := billingmodels.ApplyPolarSubscription(database.DB(t.Context()), r.Organization.ID, billingmodels.PolarSubscriptionApply{
		ID: "sub_billing_response_custom", ProductID: *catalog.PolarProductID, Status: billingmodels.PolarSubscriptionStatusActive,
		PurchasedSeats: new(10), PeriodStart: &periodStart, PeriodEnd: &periodEnd,
	}, nil)
	require.NoError(t, err)

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Equal(t, catalog.Key, resp.Plan)
	assert.Equal(t, "Team", resp.PlanName)
	assert.Equal(t, "paid", resp.PlanCategory)
	assert.Equal(t, int64(2900), resp.PlanPriceCents)
	assert.True(t, resp.PlanPerSeat)

	db := database.Conn()
	for range 8 {
		support.CreateUser(t, r, r.Organization.ID)
	}
	expected := map[string]any{
		"purchasedSeats": float64(10), "occupiedSeats": float64(9), "availableSeats": float64(1),
		"pendingInvitationSeats": float64(0), "invitableSeats": float64(1),
		"planMaximumSeats": float64(50), "purchasableSeats": float64(40), "expectedRenewalSeats": float64(9),
		"currentMonthlyTotalCents": "29000", "expectedRenewalCents": "26100",
	}
	check := func() {
		t.Helper()
		resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
		require.NoError(t, err)
		assertBillingSeatJSON(t, resp, expected)
	}
	check()

	now := time.Now()
	invitation := models.OrganizationInvitation{
		OrganizationID: r.Organization.ID, Email: "pending@example.com", Role: models.RoleOrgOperator,
		InvitedBy: r.User, State: models.InvitationStatePending, SentAt: now, ExpiresAt: now.Add(time.Hour),
	}
	require.NoError(t, invitation.Save(db))
	expected["pendingInvitationSeats"], expected["invitableSeats"] = float64(1), float64(0)
	check()

	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, new(80)))
	expected["planMaximumSeats"], expected["purchasableSeats"] = float64(80), float64(70)
	check()

	require.NoError(t, db.Delete(&models.User{}, "id = ?", r.User).Error)
	expected["occupiedSeats"], expected["availableSeats"], expected["expectedRenewalSeats"] = float64(8), float64(2), float64(8)
	expected["invitableSeats"] = float64(1)
	expected["expectedRenewalCents"] = "23200"
	check()

	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", nil).Error)
	expected["purchasedSeats"], expected["availableSeats"], expected["purchasableSeats"] = float64(0), float64(0), float64(80)
	expected["invitableSeats"] = float64(0)
	expected["currentMonthlyTotalCents"] = "0"
	// A plan without paid seats needs repair, so it has no renewal estimate.
	delete(expected, "expectedRenewalSeats")
	delete(expected, "expectedRenewalCents")
	check()

	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, nil))
	delete(expected, "planMaximumSeats")
	delete(expected, "purchasableSeats")
	check()

	require.NoError(t, db.Delete(&models.User{}, "organization_id = ?", r.Organization.ID).Error)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 10).Error)
	expected["occupiedSeats"], expected["purchasedSeats"], expected["availableSeats"], expected["invitableSeats"] = float64(0), float64(10), float64(10), float64(9)
	expected["currentMonthlyTotalCents"], expected["expectedRenewalSeats"], expected["expectedRenewalCents"] = "29000", float64(1), "2900"
	check()
}

// Use the gateway marshaler to distinguish optional zeros from omitted fields.
func assertBillingSeatJSON(t *testing.T, resp *pb.DescribeOrganizationBillingResponse, expected map[string]any) {
	t.Helper()
	marshaler := runtime.JSONPb{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true}}
	body, err := marshaler.Marshal(resp)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(body, &fields))
	for _, key := range []string{"purchasedSeats", "occupiedSeats", "availableSeats", "pendingInvitationSeats", "invitableSeats", "planMaximumSeats", "purchasableSeats", "expectedRenewalSeats", "currentMonthlyTotalCents", "expectedRenewalCents"} {
		value, present := expected[key]
		if !present {
			assert.NotContains(t, fields, key)
			continue
		}
		require.Contains(t, fields, key)
		assert.Equal(t, value, fields[key], key)
	}
}

func Test__DescribeOrganizationBillingSeatUsageOnNonPerSeatPlans(t *testing.T) {
	for _, key := range []string{"business_legacy", "trial", "none"} {
		t.Run(key, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			db := database.Conn()
			catalog, err := billingmodels.FindBillingPlan(db, key)
			require.NoError(t, err)
			require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
				"plan_id": catalog.ID, "plan": catalog.ResponsePlan(), "plan_source": "admin",
				"trial_ends_at": time.Now().Add(time.Hour), "purchased_seats": 99,
			}).Error)
			support.CreateUser(t, r, r.Organization.ID)
			now := time.Now()
			invitation := models.OrganizationInvitation{
				OrganizationID: r.Organization.ID, Email: "pending@example.com", Role: models.RoleOrgOperator,
				InvitedBy: r.User, State: models.InvitationStatePending, SentAt: now, ExpiresAt: now.Add(time.Hour),
			}
			require.NoError(t, invitation.Save(db))
			resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
			require.NoError(t, err)
			assert.False(t, resp.PlanPerSeat)
			assertBillingSeatJSON(t, resp, map[string]any{"occupiedSeats": float64(2), "pendingInvitationSeats": float64(1)})
		})
	}
}

func Test__DescribeOrganizationBillingNonePlanWithCreditDoesNotAllowRuns(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	_, err := billingmodels.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Equal(t, "none", resp.PlanCategory)
	assert.Greater(t, resp.WelcomeRemainingCents, int64(0))
	assert.False(t, resp.HostedExecutionAllowed)
	assert.ErrorIs(t, billingmodels.AssertHostedRunAllowed(database.Conn(), r.Organization.ID, nil), billingmodels.ErrHostedSubscriptionRequired)
}

func Test__DescribeOrganizationBillingDisabledWhenFlagOff(t *testing.T) {
	r := support.Setup(t)
	setLegacyBusinessProduct(t, new("prod_business"))
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.False(t, resp.BillingEnabled)
	assert.False(t, resp.SubscriptionCheckoutEnabled)
	assert.True(t, resp.HostedExecutionAllowed)
	assert.True(t, resp.HostedCreditAvailable)
	assertBillingSeatJSON(t, resp, nil)

	_, err = SyncOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	_, err = CreateBusinessCheckout(context.Background(), r.Organization.ID.String(), &pb.CreateBusinessCheckoutRequest{}, r.Account.ID.String(), "http://localhost")
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	_, err = CancelOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.CancelOrganizationSubscriptionRequest{})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	_, err = ResumeOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.ResumeOrganizationSubscriptionRequest{})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
}

func Test__SyncOrganizationBilling(t *testing.T) {
	r := support.Setup(t)
	setLegacyBusinessProduct(t, new("prod_business"))

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := SyncOrganizationBilling(context.Background(), "not-a-uuid", &pb.SyncOrganizationBillingRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("returns local plan when billing is not configured", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")

		resp, err := SyncOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
		require.NoError(t, err)
		assert.Equal(t, billingmodels.BillingPlanNone, resp.Plan)
		assert.False(t, resp.CreditPurchaseAllowed)
	})

	t.Run("applies an active Polar subscription", func(t *testing.T) {
		periodStart := time.Now().UTC().Truncate(time.Second)
		periodEnd := periodStart.AddDate(0, 1, 0)
		server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/payment-methods") {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			subscription := map[string]any{
				"id":                   "sub_grpc_sync",
				"status":               "active",
				"product_id":           "prod_business",
				"current_period_start": periodStart.Format(time.RFC3339),
				"current_period_end":   periodEnd.Format(time.RFC3339),
				"customer_id":          "cust_polar_1",
				"external_customer_id": r.Organization.ID.String(),
				"customer": map[string]any{
					"id":          "cust_polar_1",
					"external_id": r.Organization.ID.String(),
				},
			}
			assert.Equal(t, "/subscriptions/", req.URL.Path)
			assert.Equal(t, r.Organization.ID.String(), req.URL.Query().Get("external_customer_id"))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items":      []map[string]any{subscription},
				"pagination": map[string]any{"max_page": 1},
			}))
		})
		polartest.UseServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := SyncOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
		require.NoError(t, err)
		assert.Equal(t, billingmodels.BillingPlanBusiness, resp.Plan)
		assert.True(t, resp.CreditPurchaseAllowed)
		assert.Equal(t, "active", resp.PolarSubscriptionStatus)
	})

	t.Run("returns local plan when Polar fails", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("billing-sync"), "")
		require.NoError(t, err)

		server := polartest.APIServer(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "polar down", http.StatusInternalServerError)
		})
		polartest.UseServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := SyncOrganizationBilling(context.Background(), other.ID.String(), &pb.SyncOrganizationBillingRequest{})
		require.NoError(t, err)
		assert.Equal(t, billingmodels.BillingPlanNone, resp.Plan)
		assert.False(t, resp.CreditPurchaseAllowed)
	})
}

func Test__DescribeOrganizationBillingLapsesExpiredTrial(t *testing.T) {
	r := support.Setup(t)
	setLegacyBusinessProduct(t, new("prod_business"))
	ended := time.Now().Add(-time.Hour)
	trial, err := billingmodels.FindBillingPlan(database.Conn(), billingmodels.BillingPlanTrial)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":          billingmodels.BillingPlanTrial,
			"plan_id":       trial.ID,
			"trial_ends_at": ended,
		}).Error)

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, resp.Plan)
	assert.False(t, resp.CreditPurchaseAllowed)

	stored, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, stored.Plan)
}

func Test__CancelOrganizationSubscription(t *testing.T) {
	r := support.Setup(t)
	setLegacyBusinessProduct(t, new("prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
		ID:          "sub_grpc_cancel",
		Status:      billingmodels.PolarSubscriptionStatusActive,
		PeriodStart: &periodStart,
		PeriodEnd:   &periodEnd,
	}, nil)
	require.NoError(t, err)

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := CancelOrganizationSubscription(context.Background(), "bad", &pb.CancelOrganizationSubscriptionRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("rejects trial", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("billing-cancel"), "")
		require.NoError(t, err)
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		_, err = CancelOrganizationSubscription(context.Background(), other.ID.String(), &pb.CancelOrganizationSubscriptionRequest{})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("cancels at period end", func(t *testing.T) {
		server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/payment-methods") {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/subscriptions/sub_grpc_cancel", req.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":                   "sub_grpc_cancel",
				"status":               "active",
				"product_id":           "prod_business",
				"cancel_at_period_end": true,
				"current_period_start": periodStart.Format(time.RFC3339),
				"current_period_end":   periodEnd.Format(time.RFC3339),
				"external_customer_id": r.Organization.ID.String(),
				"customer": map[string]any{
					"id":          "cust_polar_1",
					"external_id": r.Organization.ID.String(),
				},
			}))
		})
		polartest.UseServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := CancelOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.CancelOrganizationSubscriptionRequest{})
		require.NoError(t, err)
		assert.Equal(t, billingmodels.BillingPlanBusiness, resp.Plan)
		assert.True(t, resp.CreditPurchaseAllowed)
		assert.True(t, resp.CancelAtPeriodEnd)
	})
}

func Test__ResumeOrganizationSubscription(t *testing.T) {
	r := support.Setup(t)
	setLegacyBusinessProduct(t, new("prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
		ID:                "sub_grpc_resume",
		Status:            billingmodels.PolarSubscriptionStatusActive,
		PeriodStart:       &periodStart,
		PeriodEnd:         &periodEnd,
		CancelAtPeriodEnd: true,
	}, nil)
	require.NoError(t, err)

	t.Run("keeps Business", func(t *testing.T) {
		var patches atomic.Int32
		server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/payment-methods") {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			assert.Equal(t, "/subscriptions/sub_grpc_resume", req.URL.Path)
			if req.Method == http.MethodPatch {
				patches.Add(1)
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":                   "sub_grpc_resume",
				"status":               "active",
				"product_id":           "prod_business",
				"cancel_at_period_end": false,
				"current_period_start": periodStart.Format(time.RFC3339),
				"current_period_end":   periodEnd.Format(time.RFC3339),
				"external_customer_id": r.Organization.ID.String(),
				"customer": map[string]any{
					"id":          "cust_polar_1",
					"external_id": r.Organization.ID.String(),
				},
			}))
		})
		polartest.UseServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := ResumeOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.ResumeOrganizationSubscriptionRequest{})
		require.NoError(t, err)
		assert.Equal(t, billingmodels.BillingPlanBusiness, resp.Plan)
		assert.True(t, resp.CreditPurchaseAllowed)
		assert.False(t, resp.CancelAtPeriodEnd)
		assert.Equal(t, int32(1), patches.Load(), "only the resume updates the subscription")
	})
}

func TestBusinessCheckoutSuccessURLIncludesCheckoutID(t *testing.T) {
	id := uuid.New()
	assert.Equal(t, "https://app.superplane.com/"+id.String()+"/organization/billing?subscribed=1&checkout_id={CHECKOUT_ID}", businessCheckoutSuccessURL("https://app.superplane.com/", id))
}

func setLegacyBusinessProduct(t *testing.T, productID *string) *billingmodels.BillingPlan {
	t.Helper()
	db := database.DB(t.Context())
	catalog, err := billingmodels.FindBillingPlan(db, billingmodels.BillingPlanBusiness)
	require.NoError(t, err)
	originalProduct := catalog.PolarProductID
	if originalProduct != nil {
		originalProduct = new(*originalProduct)
	}
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.WithoutCancel(t.Context())).Model(catalog).Update("polar_product_id", originalProduct).Error)
	})
	require.NoError(t, db.Model(catalog).Update("polar_product_id", productID).Error)
	return catalog
}

func TestCreateBusinessCheckoutAsksForPlanSelection(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	setLegacyBusinessProduct(t, new("prod_business"))
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	requests := 0
	server := polartest.APIServer(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	})
	polartest.UseServer(t, server)

	response, err := CreateBusinessCheckout(t.Context(), r.Organization.ID.String(), &pb.CreateBusinessCheckoutRequest{}, r.Account.ID.String(), "http://localhost")

	require.Nil(t, response)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.Equal(t, "Select a plan in Billing.", grpcerrors.StatusMessage(err))
	require.Zero(t, requests)
}

func TestLegacyBusinessCheckoutRejectsUnsafeBillingStateBeforePolar(t *testing.T) {
	const renewalRecovery = "The last renewal payment failed. Update the payment method to recover the current subscription."
	const selectPlan = "Select a plan in Billing."
	for _, tc := range []struct {
		name, subscriptionStatus, configuration, wantMessage string
		scheduledCancellation                                bool
	}{
		{name: "past due", subscriptionStatus: billingmodels.PolarSubscriptionStatusPastDue, wantMessage: renewalRecovery},
		{name: "no subscription", wantMessage: selectPlan},
		{name: "active subscription", subscriptionStatus: billingmodels.PolarSubscriptionStatusActive, wantMessage: selectPlan},
		{name: "trialing subscription", subscriptionStatus: billingmodels.PolarSubscriptionStatusTrialing, wantMessage: selectPlan},
		{name: "scheduled cancellation", subscriptionStatus: billingmodels.PolarSubscriptionStatusCanceled, scheduledCancellation: true, wantMessage: selectPlan},
		{name: "paused subscription", subscriptionStatus: billingmodels.PolarSubscriptionStatusPaused, wantMessage: selectPlan},
		{name: "paused with scheduled cancellation", subscriptionStatus: billingmodels.PolarSubscriptionStatusPaused, scheduledCancellation: true, wantMessage: selectPlan},
		{name: "billing disabled before billing guards", subscriptionStatus: billingmodels.PolarSubscriptionStatusPastDue, configuration: "disabled", wantMessage: "Business checkout is not configured"},
		{name: "missing token before billing guards", subscriptionStatus: billingmodels.PolarSubscriptionStatusPastDue, configuration: "missing token", wantMessage: "Business checkout is not configured"},
		{name: "missing product before billing guards", subscriptionStatus: billingmodels.PolarSubscriptionStatusPastDue, configuration: "missing product", wantMessage: "Business checkout is not configured"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
			catalog := setLegacyBusinessProduct(t, new("prod_business"))
			if tc.subscriptionStatus != "" {
				require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
					"plan_id": catalog.ID, "plan": catalog.ResponsePlan(), "plan_source": billingmodels.BillingPlanSourcePolar,
					"polar_subscription_id": "sub_current", "polar_subscription_status": tc.subscriptionStatus,
					"cancel_at_period_end": tc.scheduledCancellation, "current_period_end": time.Now().Add(time.Hour),
				}).Error)
			}
			requests := 0
			server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
				requests++
				if req.URL.Path == "/checkouts/" {
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"url": "https://polar.sh/checkout/unexpected"}))
					return
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_unexpected", "external_id": r.Organization.ID.String()}))
			})
			polartest.UseServer(t, server)
			switch tc.configuration {
			case "disabled":
				t.Setenv(billingmodels.BillingEnabledEnv, "no")
			case "missing token":
				t.Setenv("POLAR_ACCESS_TOKEN", "")
			case "missing product":
				t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")
				require.NoError(t, db.Model(catalog).Update("polar_product_id", nil).Error)
			}
			before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)

			response, err := CreateBusinessCheckout(t.Context(), r.Organization.ID.String(), &pb.CreateBusinessCheckoutRequest{}, r.Account.ID.String(), "http://localhost")

			require.Nil(t, response)
			require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			require.Equal(t, tc.wantMessage, grpcerrors.StatusMessage(err))
			require.Zero(t, requests)
			after, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}

func TestDescribeBillingListsPublicPlansWithoutPolarProducts(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	plans := []billingmodels.BillingPlan{
		{Name: "Flat price", Visibility: "public", PriceCents: 2000, ParallelTaskLimit: new(2)},
		{Name: "Seat price", Visibility: "public", PriceCents: 3000, PerSeat: true, SeatLimit: new(50)},
		{Name: "Zero-price seats", Visibility: "public", PerSeat: true, SeatLimit: new(50)},
		{Name: "Private", Visibility: "private", PriceCents: 1000},
	}
	for i := range plans {
		plans[i].Key = "catalog_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		plans[i].Category = "paid"
		plans[i].CreditFrequency = "none"
		plans[i].CreditExpiration = "never"
		require.NoError(t, billingmodels.SaveBillingPlan(db, &plans[i]))
	}
	response, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	for _, plan := range plans {
		var option *pb.BillingPlanOption
		for _, available := range response.AvailablePlans {
			if available.Key == plan.Key {
				option = available
				break
			}
		}
		if plan.Visibility == "private" {
			require.Nil(t, option)
			continue
		}
		require.NotNil(t, option, "public plans must remain visible before Polar setup")
		require.Equal(t, plan.PriceCents, option.PriceCents)
		require.Equal(t, plan.PerSeat, option.PerSeat)
		if plan.ParallelTaskLimit == nil {
			require.Nil(t, option.ParallelTaskLimit, "unlimited capacity must remain unset")
		} else {
			require.NotNil(t, option.ParallelTaskLimit)
			require.EqualValues(t, *plan.ParallelTaskLimit, *option.ParallelTaskLimit)
		}
		require.True(t, option.SubscriptionRequired, "priced and per-seat plans must never use card-only activation")
		request := &pb.SelectBillingPlanRequest{PlanKey: plan.Key}
		if plan.PerSeat {
			request.Seats = new(int32(1))
		}
		_, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), request, r.Account.ID.String(), "http://localhost")
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		current, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
		require.NoError(t, err)
		require.NotEqual(t, plan.ID, current.PlanID)
	}
}
