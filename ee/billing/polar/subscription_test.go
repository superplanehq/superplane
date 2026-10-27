package polar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func TestVerifyOrganizationSubscriptionDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	t.Setenv("POLAR_ACCESS_TOKEN", "test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	verification, err := VerifyOrganizationSubscription(ctx, uuid.New())
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, "failed", verification)
}

func TestVerifyOrganizationSubscriptionProtectsLiveIdentity(t *testing.T) {
	orgID := uuid.New()
	periodEnd := time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	for _, tc := range []struct {
		name         string
		status       string
		cancelAtEnd  bool
		verification string
	}{
		{name: "active", status: "active", verification: "active"},
		{name: "active set to cancel", status: "active", cancelAtEnd: true, verification: "active"},
		{name: "past due", status: "past_due", verification: "active"},
		{name: "past due set to cancel", status: "past_due", cancelAtEnd: true, verification: "active"},
		{name: "paused", status: "paused", verification: "active"},
		{name: "paused set to cancel", status: "paused", cancelAtEnd: true, verification: "active"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usePolarSubscriptionServer(t, orgID.String(), []map[string]any{{
				"id": "sub_1", "status": tc.status, "cancel_at_period_end": tc.cancelAtEnd, "current_period_end": periodEnd,
				"external_customer_id": orgID.String(),
			}})

			verification, err := VerifyOrganizationSubscription(context.Background(), orgID)

			require.NoError(t, err)
			assert.Equal(t, tc.verification, verification)
		})
	}
}

func TestApplySubscriptionStoresPastDueAtFromPolar(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	failure := time.Now().Add(-2 * 24 * time.Hour).UTC().Truncate(time.Second)
	body := fmt.Sprintf(`{"type":"subscription.updated","data":{"id":"sub_late","product_id":"prod_business","status":"past_due",
		"past_due_at":%q,"current_period_start":%q,"current_period_end":%q,"external_customer_id":%q}}`,
		failure.Format(time.RFC3339), time.Now().Add(-30*24*time.Hour).UTC().Format(time.RFC3339),
		time.Now().Add(24*time.Hour).UTC().Format(time.RFC3339), org.ID.String())
	var event SubscriptionWebhookEvent
	require.NoError(t, json.Unmarshal([]byte(body), &event))

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, &event))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, plan.PastDueSince)
	assert.True(t, plan.PastDueSince.Equal(failure))
}

func Test__ApplySubscriptionActivatesBusinessAndGrantsIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_biz", "active", periodStart, periodEnd)

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)
	requireWelcomeGrantExpiresAfterCreate(t, db, org.ID)
}

func Test__ApplySubscriptionCancelRestoresTrialAndExpiresIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_biz", "active", periodStart, periodEnd)))

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_biz", "canceled", periodStart, periodEnd)))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.RemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)
}

func Test__ApplySubscriptionCancelEndsPlanAfterTrialAndExpiresIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_lapsed", "active", periodStart, periodEnd)))
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", org.ID).
		Update("trial_ends_at", time.Now().Add(-time.Hour)).Error)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", org.ID, billingmodels.LLMCreditGrantKindWelcome).
		Update("expires_at", time.Now().Add(-time.Hour)).Error)

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_lapsed", "canceled", periodStart, periodEnd)))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.IsOpenTrial(time.Now()))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, int64(0), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)
}

func Test__ApplySubscriptionResubscribeSamePeriodGrantsIncludedAgain(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_renew", "active", periodStart, periodEnd)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_renew", "canceled", periodStart, periodEnd)))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)

	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, org.ID)
	require.NoError(t, err)
	liveIncluded := 0
	canceledIncluded := 0
	for _, grant := range grants {
		if grant.Kind != billingmodels.LLMCreditGrantKindIncluded {
			continue
		}
		if grant.IsExpired(time.Now()) {
			canceledIncluded++
			require.NotNil(t, grant.PolarOrderID)
			assert.True(t, strings.HasPrefix(*grant.PolarOrderID, "canceled:"))
			continue
		}
		liveIncluded++
		require.NotNil(t, grant.PolarOrderID)
		assert.Equal(t, billingmodels.IncludedGrantKey("sub_renew", periodEnd), *grant.PolarOrderID)
	}
	assert.Equal(t, 1, liveIncluded)
	assert.Equal(t, 1, canceledIncluded)
}

func Test__ApplySubscriptionIncompleteKeepsTrial(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)

	require.NoError(t, ApplySubscriptionEvent(
		context.Background(),
		db,
		subscriptionEvent(org.ID, "sub_incomplete", "incomplete", periodStart, periodEnd),
	))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())
}

func Test__ApplySubscriptionCancelAtPeriodEndKeepsBusiness(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_ending", "active", periodStart, periodEnd)
	event.Data.CancelAtPeriodEnd = true

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.CancelAtPeriodEnd)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__CancelOrganizationSubscriptionSchedulesPeriodEnd(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_cancel", "active", periodStart, periodEnd)))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Equal(t, "/subscriptions/sub_cancel", req.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_cancel",
			"status":               "active",
			"product_id":           "prod_business",
			"cancel_at_period_end": true,
			"current_period_start": periodStart.Format(time.RFC3339),
			"current_period_end":   periodEnd.Format(time.RFC3339),
			"external_customer_id": org.ID.String(),
			"customer": map[string]any{
				"id":          "cust_polar_1",
				"external_id": org.ID.String(),
			},
		}))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, CancelOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.CancelAtPeriodEnd)
	assert.True(t, plan.IsActiveBusiness())
}

func Test__ResumeOrganizationSubscriptionClearsPeriodEndCancel(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_resume", "active", periodStart, periodEnd)
	event.Data.CancelAtPeriodEnd = true
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Equal(t, "/subscriptions/sub_resume", req.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_resume",
			"status":               "active",
			"product_id":           "prod_business",
			"cancel_at_period_end": false,
			"current_period_start": periodStart.Format(time.RFC3339),
			"current_period_end":   periodEnd.Format(time.RFC3339),
			"external_customer_id": org.ID.String(),
			"customer": map[string]any{
				"id":          "cust_polar_1",
				"external_id": org.ID.String(),
			},
		}))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, ResumeOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.False(t, plan.CancelAtPeriodEnd)
}

func Test__ResumeWithoutModifiedAtDoesNotBlockLaterWebhook(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	provider := periodStart.Add(-30 * time.Second)
	laterWebhook := periodStart.Add(-10 * time.Second)

	event := subscriptionEvent(org.ID, "sub_resume", "active", periodStart, periodEnd)
	event.Data.CancelAtPeriodEnd = true
	event.Data.ModifiedAt = polarTime{Time: provider}
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Equal(t, "/subscriptions/sub_resume", req.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_resume",
			"status":               "active",
			"product_id":           "prod_business",
			"cancel_at_period_end": false,
			"current_period_start": periodStart.Format(time.RFC3339),
			"current_period_end":   periodEnd.Format(time.RFC3339),
			"external_customer_id": org.ID.String(),
			"customer": map[string]any{
				"id":          "cust_polar_1",
				"external_id": org.ID.String(),
			},
		}))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, ResumeOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.False(t, plan.CancelAtPeriodEnd)
	require.NotNil(t, plan.PolarModifiedAt)
	assert.True(t, plan.PolarModifiedAt.Equal(provider))

	webhook := subscriptionEvent(org.ID, "sub_resume", "active", periodStart, periodEnd)
	webhook.Data.ModifiedAt = polarTime{Time: laterWebhook}
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, webhook))

	plan, err = billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.False(t, plan.CancelAtPeriodEnd)
	require.NotNil(t, plan.PolarModifiedAt)
	assert.True(t, plan.PolarModifiedAt.Equal(laterWebhook))
}

func Test__ApplySubscriptionIgnoresStaleCancelAfterResume(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	older := periodStart.Add(-time.Minute)
	newer := periodStart

	resumed := subscriptionEvent(org.ID, "sub_order", "active", periodStart, periodEnd)
	resumed.Data.ModifiedAt = polarTime{Time: newer}
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, resumed))

	staleCancel := subscriptionEvent(org.ID, "sub_order", "active", periodStart, periodEnd)
	staleCancel.Data.CancelAtPeriodEnd = true
	staleCancel.Data.ModifiedAt = polarTime{Time: older}
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, staleCancel))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.False(t, plan.CancelAtPeriodEnd)
	require.NotNil(t, plan.PolarModifiedAt)
	assert.True(t, plan.PolarModifiedAt.Equal(newer))
}

func Test__CancelOrganizationSubscriptionRejectsTrial(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

	err := CancelOrganizationSubscription(context.Background(), db, org.ID)
	require.ErrorIs(t, err, ErrSubscriptionNotCancelable)
}

func Test__CancelOrganizationSubscriptionRejectsAdminPlan(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	_, err := billingmodels.SetAdminOrganizationPlan(db, org.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

	err = CancelOrganizationSubscription(context.Background(), db, org.ID)
	require.ErrorIs(t, err, ErrAdminPlanCannotCancel)
}

func Test__ApplySubscriptionIncludedGrantIsIdempotentWithoutPeriodStart(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodEnd := time.Now().UTC().Truncate(time.Second).AddDate(0, 1, 0)
	event := &SubscriptionWebhookEvent{
		Type: subscriptionUpdatedType,
		Data: SubscriptionData{ProductID: "prod_business",
			ID:               "sub_no_start",
			Status:           "active",
			CurrentPeriodEnd: polarTime{Time: periodEnd},
			Customer: OrderCustomer{
				ID:         "cust_polar_1",
				ExternalID: org.ID.String(),
			},
		},
	}

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, org.ID)
	require.NoError(t, err)
	included := 0
	for _, grant := range grants {
		if grant.Kind != billingmodels.LLMCreditGrantKindIncluded {
			continue
		}
		included++
		require.NotNil(t, grant.PolarOrderID)
		assert.Equal(t, billingmodels.IncludedGrantKey("sub_no_start", periodEnd), *grant.PolarOrderID)
	}
	assert.Equal(t, 1, included)
}

func Test__SyncOrganizationSubscriptionActivatesBusinessAndGrantsIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_canceled", "canceled", org.ID.String(), periodStart.AddDate(0, -1, 0), periodStart),
		polarSubscriptionJSON("sub_sync_biz", "active", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))
	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())
	require.NotNil(t, plan.PolarSubscriptionID)
	assert.Equal(t, "sub_sync_biz", *plan.PolarSubscriptionID)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__SyncOrganizationSubscriptionAppliesCanceledWhenNoPaidExists(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_sync_cancel", "active", periodStart, periodEnd)))

	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_cancel", "canceled", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())
}

func Test__SyncOrganizationSubscriptionLeavesTrialWhenPolarHasNone(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	usePolarSubscriptionServer(t, org.ID.String(), nil)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceSystem, plan.PlanSource)
}

func Test__SyncOrganizationSubscriptionLeavesTrialWhenPolarIsIncomplete(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_incomplete", "incomplete", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
}

func Test__SyncOrganizationSubscriptionReplacesAdminTrialWithPolarPaid(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	_, err := billingmodels.SetAdminOrganizationPlan(db, org.ID, billingmodels.BillingPlanTrial, nil)
	require.NoError(t, err)
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_admin", "active", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourcePolar, plan.PlanSource)
	assert.True(t, plan.IsActiveBusiness())
}

func Test__SyncOrganizationSubscriptionLeavesAdminBusinessWhenPolarHasNone(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	_, err := billingmodels.SetAdminOrganizationPlan(db, org.ID, billingmodels.BillingPlanBusiness, nil)
	require.NoError(t, err)
	usePolarSubscriptionServer(t, org.ID.String(), nil)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__SyncOrganizationSubscriptionKeepsLocalPlanWhenPolarFails(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "polar down", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
}

func Test__SyncOrganizationSubscriptionNoopsWhenCheckoutDisabled(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
}

func assertNoTrialConversionGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) {
	t.Helper()
	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	for _, grant := range grants {
		if grant.PolarOrderID == nil {
			continue
		}
		assert.False(t, strings.HasPrefix(*grant.PolarOrderID, "trial-conversion:"))
	}
}

func requireWelcomeGrantExpiresAfterCreate(t *testing.T, db *gorm.DB, orgID uuid.UUID) {
	t.Helper()
	var grant billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", orgID, billingmodels.LLMCreditGrantKindWelcome).
		First(&grant).Error)
	require.NotNil(t, grant.ExpiresAt)
	assert.WithinDuration(t, grant.CreatedAt.AddDate(0, 0, billingmodels.SignupCreditDays), *grant.ExpiresAt, time.Second)
}

func subscriptionEvent(orgID uuid.UUID, subscriptionID, status string, start, end time.Time) *SubscriptionWebhookEvent {
	return &SubscriptionWebhookEvent{
		Type: subscriptionUpdatedType,
		Data: SubscriptionData{ProductID: "prod_business",
			ID:                 subscriptionID,
			Status:             status,
			CurrentPeriodStart: polarTime{Time: start},
			CurrentPeriodEnd:   polarTime{Time: end},
			Customer: OrderCustomer{
				ID:         "cust_polar_1",
				ExternalID: orgID.String(),
			},
		},
	}
}

func usePolarSubscriptionServer(t *testing.T, orgID string, items []map[string]any) {
	t.Helper()
	if items == nil {
		items = []map[string]any{}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/subscriptions/", req.URL.Path)
		assert.Equal(t, orgID, req.URL.Query().Get("external_customer_id"))
		assert.Empty(t, req.URL.Query().Get("product_id"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items":      items,
			"pagination": map[string]any{"max_page": 1},
		}))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
}

func polarSubscriptionJSON(id, status, orgID string, start, end time.Time) map[string]any {
	return map[string]any{
		"id":                   id,
		"status":               status,
		"product_id":           "prod_business",
		"current_period_start": start.Format(time.RFC3339),
		"current_period_end":   end.Format(time.RFC3339),
		"customer_id":          "cust_polar_1",
		"external_customer_id": orgID,
		"customer": map[string]any{
			"id":          "cust_polar_1",
			"external_id": orgID,
		},
	}
}

func TestUnknownProductCannotActivateSubscription(t *testing.T) {
	org := setupBillingOrganization(t)
	now := time.Now()
	event := subscriptionEvent(org.ID, "sub_unknown", "active", now, now.AddDate(0, 1, 0))
	event.Data.ProductID = "unknown-product"
	require.Error(t, ApplySubscriptionEvent(t.Context(), database.Conn(), event))
	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), org.ID)
	require.NoError(t, err)
	require.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	require.Nil(t, plan.PolarSubscriptionID)
}

func TestCatalogSubscriptionDoesNotDeliverNewPolicies(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	plan := billingmodels.BillingPlan{Key: "paid_" + uuid.NewString()[:8], Name: "Paid", Category: "paid", Visibility: "private", PolarProductID: &product, CreditAmountCents: 7500, CreditFrequency: "billing_period", CreditExpiration: "period_end"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	now := time.Now()
	event := subscriptionEvent(org.ID, "sub_custom", "active", now, now.AddDate(0, 1, 0))
	event.Data.ProductID = product
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, event))
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, event))
	order := &OrderWebhookEvent{Type: orderPaidType, Data: OrderData{ID: uuid.NewString(), ProductID: product, BillingReason: billingReasonSubscriptionCycle, Customer: OrderCustomer{ExternalID: org.ID.String()}}}
	require.NoError(t, ApplyOrderPaid(t.Context(), db, order, nil))
	saved, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	require.Equal(t, plan.ID, saved.PlanID)
	require.True(t, saved.AllowsCreditPurchase())
	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	require.Zero(t, summary.IncludedRemainingMicros)
}

func TestSubscriptionSeatSnapshots(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	initial := polarSubscriptionJSON("sub_seats", "active", org.ID.String(), now, end)
	initial["product_id"], initial["seats"], initial["modified_at"] = product, 3, now.Format(time.RFC3339)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{initial})
	require.NoError(t, SyncOrganizationSubscription(t.Context(), db, org.ID))
	saved, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	require.Equal(t, intPointer(3), saved.PurchasedSeats)

	for _, tc := range []struct {
		name           string
		modified       time.Time
		quantity, want int
	}{
		{"older", now.Add(-time.Minute), 1, 3},
		{"equal", now, 4, 4},
		{"absent", time.Time{}, 5, 5},
		{"newer zero", now.Add(time.Minute), 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
			event.Data.ProductID, event.Data.Seats = product, &tc.quantity
			event.Data.ModifiedAt = polarTime{Time: tc.modified}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, event))
			saved, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
			require.NoError(t, err)
			require.Equal(t, &tc.want, saved.PurchasedSeats)
			wantModified := tc.modified
			if wantModified.IsZero() || wantModified.Before(now) {
				wantModified = now
			}
			require.NotNil(t, saved.PolarModifiedAt)
			require.True(t, saved.PolarModifiedAt.Equal(wantModified))
		})
	}

	before, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	settingsBefore, err := billingmodels.FindOrganizationLLMSettings(db, org.ID)
	require.NoError(t, err)
	_, err = billingmodels.AddIncludedLLMCreditGrant(db, org.ID, billingmodels.CentsToMicros(500), "previous_subscription", end)
	require.NoError(t, err)
	grantsBefore, err := billingmodels.ListOrganizationLLMCreditGrants(db, org.ID)
	require.NoError(t, err)
	for _, status := range []string{"active", "canceled", "incomplete"} {
		for _, seats := range []string{"", `,"seats":null`, `,"seats":-1`} {
			t.Run(status+seats, func(t *testing.T) {
				var snapshot SubscriptionData
				payload := fmt.Sprintf(`{"id":"sub_seats","status":%q,"product_id":%q,"modified_at":%q,"current_period_end":%q,"customer":{"id":"changed_customer","external_id":%q}%s}`, status, product, now.Add(time.Hour).Format(time.RFC3339), now.Add(-time.Hour).Format(time.RFC3339), org.ID.String(), seats)
				require.NoError(t, json.Unmarshal([]byte(payload), &snapshot))
				require.ErrorIs(t, ApplySubscription(t.Context(), db, snapshot), ErrPermanentApply)
				after, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
				require.NoError(t, err)
				require.Equal(t, before, after)
				grantsAfter, err := billingmodels.ListOrganizationLLMCreditGrants(db, org.ID)
				require.NoError(t, err)
				require.Equal(t, grantsBefore, grantsAfter)
				state, err := billingmodels.FindOrganizationLLMSettings(db, org.ID)
				require.NoError(t, err)
				require.Equal(t, settingsBefore, state)
			})
		}
	}
}

func intPointer(quantity int) *int { return &quantity }

// A delayed webhook can report seats that Polar no longer has. The stale
// snapshot proves nothing about an in-flight purchase, so it must not
// settle it.
func Test__ApplySubscriptionStaleSnapshotDoesNotSettleSeatPurchase(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)

	// Polar went from 5 seats down to 3. The snapshot with 3 seats applies.
	fresh := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	fresh.Data.ProductID, fresh.Data.Seats = product, intPointer(3)
	fresh.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, fresh))

	// A purchase from 3 to 4 seats lost its response.
	purchase := billingmodels.NewSeatPurchase(org.ID, nil, "sub_seats", 3, 4, nil)
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))

	// The delayed snapshot with 5 seats is older, so it is ignored.
	stale := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	stale.Data.ProductID, stale.Data.Seats = product, intPointer(5)
	stale.Data.ModifiedAt = polarTime{Time: now.Add(-time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stale))

	inFlight, err := billingmodels.FindInFlightSeatPurchase(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, inFlight, "a stale snapshot must not settle the purchase")
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, inFlight.Status)

	saved, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, intPointer(3), saved.PurchasedSeats)
}

// A snapshot fresh enough to apply proves the outcome of an in-flight
// purchase, so a webhook settles it.
func Test__ApplySubscriptionSettlesSeatPurchaseFromFreshSnapshot(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)

	initial := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	initial.Data.ProductID, initial.Data.Seats = product, intPointer(3)
	initial.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))

	// A purchase from 3 to 4 seats lost its response.
	purchase := billingmodels.NewSeatPurchase(org.ID, nil, "sub_seats", 3, 4, nil)
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))

	// A newer snapshot at the target proves that Polar applied the purchase.
	applied := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	applied.Data.ProductID, applied.Data.Seats = product, intPointer(4)
	applied.Data.ModifiedAt = polarTime{Time: now.Add(time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, applied))

	inFlight, err := billingmodels.FindInFlightSeatPurchase(db, org.ID)
	require.NoError(t, err)
	assert.Nil(t, inFlight)

	latest, err := billingmodels.FindLatestSeatPurchase(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, latest.Status)
}

func Test__ApplySubscriptionRenewalAdvancesPeriodAndGrantsIncluded(t *testing.T) {
	for _, eventType := range []string{subscriptionCycledType, subscriptionUpdatedType} {
		t.Run(eventType, func(t *testing.T) {
			org := setupBillingOrganization(t)
			require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
			db := database.Conn()
			firstStart := time.Now().UTC().Truncate(time.Second)
			firstEnd := firstStart.AddDate(0, 1, 0)
			renewedEnd := firstEnd.AddDate(0, 1, 0)
			// Included grants are keyed by subscription and period end across all orgs.
			subscriptionID := "sub_" + uuid.NewString()

			first := subscriptionEvent(org.ID, subscriptionID, "active", firstStart, firstEnd)
			first.Data.ModifiedAt = polarTime{Time: firstStart}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, first))

			renewal := subscriptionEvent(org.ID, subscriptionID, "active", firstEnd, renewedEnd)
			renewal.Type = eventType
			renewal.Data.ModifiedAt = polarTime{Time: firstEnd}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, renewal))

			requirePeriod(t, db, org.ID, firstEnd, renewedEnd)
			requireIncludedGrantExpiries(t, db, org.ID, firstEnd, renewedEnd)
		})
	}
}

func Test__ApplyOrderPaidSubscriptionCycleResyncsSubscription(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	firstStart := time.Now().UTC().Truncate(time.Second)
	firstEnd := firstStart.AddDate(0, 1, 0)
	renewedEnd := firstEnd.AddDate(0, 1, 0)
	subscriptionID := "sub_" + uuid.NewString()
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, subscriptionEvent(org.ID, subscriptionID, "active", firstStart, firstEnd)))
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON(subscriptionID, "active", org.ID.String(), firstEnd, renewedEnd),
	})
	order := &OrderWebhookEvent{
		Type: orderPaidType,
		Data: OrderData{
			ID:            uuid.NewString(),
			ProductID:     "prod_business",
			BillingReason: billingReasonSubscriptionCycle,
			Customer:      OrderCustomer{ID: "cust_polar_1", ExternalID: org.ID.String()},
		},
	}

	require.NoError(t, ApplyOrderPaid(t.Context(), db, order, nil))
	require.NoError(t, ApplyOrderPaid(t.Context(), db, order, nil))

	requirePeriod(t, db, org.ID, firstEnd, renewedEnd)
	requireIncludedGrantExpiries(t, db, org.ID, firstEnd, renewedEnd)
}

func Test__ApplyOrderPaidSubscriptionCycleRecoversFromStaleSnapshot(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	firstStart := time.Now().UTC().Truncate(time.Second)
	firstEnd := firstStart.AddDate(0, 1, 0)
	renewedEnd := firstEnd.AddDate(0, 1, 0)
	subscriptionID := "sub_" + uuid.NewString()

	first := subscriptionEvent(org.ID, subscriptionID, "active", firstStart, firstEnd)
	first.Data.ModifiedAt = polarTime{Time: firstStart}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, first))

	stale := subscriptionEvent(org.ID, subscriptionID, "active", firstEnd, renewedEnd)
	stale.Type = subscriptionCycledType
	stale.Data.ModifiedAt = polarTime{Time: firstStart.Add(-time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stale))

	requirePeriod(t, db, org.ID, firstStart, firstEnd)
	requireIncludedGrantExpiries(t, db, org.ID, firstEnd)

	fresh := polarSubscriptionJSON(subscriptionID, "active", org.ID.String(), firstEnd, renewedEnd)
	fresh["modified_at"] = firstEnd.Format(time.RFC3339)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{fresh})
	order := &OrderWebhookEvent{
		Type: orderPaidType,
		Data: OrderData{
			ID:            uuid.NewString(),
			ProductID:     "prod_business",
			BillingReason: billingReasonSubscriptionCycle,
			Customer:      OrderCustomer{ID: "cust_polar_1", ExternalID: org.ID.String()},
		},
	}

	require.NoError(t, ApplyOrderPaid(t.Context(), db, order, nil))
	require.NoError(t, ApplyOrderPaid(t.Context(), db, order, nil))

	requirePeriod(t, db, org.ID, firstEnd, renewedEnd)
	requireIncludedGrantExpiries(t, db, org.ID, firstEnd, renewedEnd)
}

func requirePeriod(t *testing.T, db *gorm.DB, orgID uuid.UUID, start, end time.Time) {
	t.Helper()
	plan, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
	require.NoError(t, err)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)
	assert.True(t, start.Equal(*plan.CurrentPeriodStart), "period start %s, want %s", plan.CurrentPeriodStart, start)
	assert.True(t, end.Equal(*plan.CurrentPeriodEnd), "period end %s, want %s", plan.CurrentPeriodEnd, end)
}

func requireIncludedGrantExpiries(t *testing.T, db *gorm.DB, orgID uuid.UUID, want ...time.Time) {
	t.Helper()
	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	var got []time.Time
	for _, grant := range grants {
		if grant.Kind != billingmodels.LLMCreditGrantKindIncluded {
			continue
		}
		require.NotNil(t, grant.ExpiresAt)
		got = append(got, grant.ExpiresAt.UTC())
	}
	require.Len(t, got, len(want))
	for _, expiry := range want {
		assert.True(t, slices.ContainsFunc(got, expiry.Equal), "missing included grant expiring %s", expiry)
	}
}

func Test__ApplySubscriptionIgnoresForeignCancellation(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	storedID, foreignID := "sub_team_"+uuid.NewString(), "sub_foreign_"+uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)
	product := createSeatSubscriptionProduct(t, db)
	initial := subscriptionEvent(org.ID, storedID, "active", now, now.AddDate(0, 1, 0))
	initial.Data.ProductID, initial.Data.Seats = product, intPointer(5)
	initial.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
	_, err := billingmodels.AddIncludedLLMCreditGrant(db, org.ID, billingmodels.CentsToMicros(500), storedID, now.AddDate(0, 1, 0))
	require.NoError(t, err)
	purchase := billingmodels.NewSeatPurchase(org.ID, nil, foreignID, 3, 4, nil)
	require.NoError(t, purchase.Create(db))
	before := captureSubscriptionState(t, db, org.ID)
	hook := logtest.NewGlobal()
	t.Cleanup(hook.Reset)

	foreign := subscriptionEvent(org.ID, foreignID, "canceled", now, now.AddDate(0, 1, 0))
	foreign.Data.ProductID, foreign.Data.Seats = product, intPointer(4)
	foreign.Data.Customer.ID = "cust_foreign"
	foreign.Data.ModifiedAt = polarTime{Time: now.Add(time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, foreign))

	after := captureSubscriptionState(t, db, org.ID)
	assert.Equal(t, before.plan, after.plan)
	assert.Equal(t, before.settings, after.settings)
	assert.Equal(t, before.grants, after.grants)
	require.NotNil(t, after.purchase)
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, after.purchase.Status,
		"the ignored snapshot still proves the outcome of a purchase on its own subscription")
	requireSubscriptionLog(t, hook, log.WarnLevel, "billing.foreign_subscription_snapshot_ignored", org.ID, storedID, foreignID, "active", "canceled")
}

// An ended snapshot of a replaced subscription cannot take the replacement
// back, but it can still prove the success of the replaced subscription's
// in-flight seat purchase. A snapshot that proves nothing leaves the purchase
// unknown: fewer seats than the target do not show that Polar did not charge.
func Test__ApplySubscriptionSettlesSeatPurchaseOnIgnoredForeignSnapshot(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, tc := range []struct {
		name            string
		endReplacement  bool
		foreignSeats    int
		foreignModified time.Time
		wantStatus      string
	}{
		{name: "replacement live and target reached", foreignSeats: 4, foreignModified: now.Add(time.Minute), wantStatus: billingmodels.SeatPurchaseStatusSucceeded},
		{name: "replacement ended and target reached", endReplacement: true, foreignSeats: 4, foreignModified: now.Add(time.Minute), wantStatus: billingmodels.SeatPurchaseStatusSucceeded},
		{name: "original seats after the purchase", foreignSeats: 3, foreignModified: now.Add(time.Minute), wantStatus: billingmodels.SeatPurchaseStatusUnknown},
		{name: "above-target snapshot after the purchase", foreignSeats: 5, foreignModified: now.Add(time.Minute), wantStatus: billingmodels.SeatPurchaseStatusUnknown},
		{name: "old target snapshot with live replacement", foreignSeats: 4, foreignModified: now.Add(-time.Hour), wantStatus: billingmodels.SeatPurchaseStatusUnknown},
		{name: "old target snapshot with ended replacement", endReplacement: true, foreignSeats: 4, foreignModified: now.Add(-time.Hour), wantStatus: billingmodels.SeatPurchaseStatusUnknown},
		{name: "target snapshot without timestamp", foreignSeats: 4, wantStatus: billingmodels.SeatPurchaseStatusUnknown},
		{name: "target snapshot at purchase creation", foreignSeats: 4, foreignModified: now, wantStatus: billingmodels.SeatPurchaseStatusUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			oldID, newID := "sub_old_"+uuid.NewString(), "sub_new_"+uuid.NewString()
			product := createSeatSubscriptionProduct(t, db)
			end := now.AddDate(0, 1, 0)
			replacement := subscriptionEvent(org.ID, newID, "active", now, end)
			replacement.Data.ProductID, replacement.Data.Seats = product, intPointer(5)
			replacement.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, replacement))
			if tc.endReplacement {
				replacement.Data.Status = "canceled"
				replacement.Data.ModifiedAt = polarTime{Time: now.Add(time.Second)}
				require.NoError(t, ApplySubscriptionEvent(t.Context(), db, replacement))
			}
			purchase := billingmodels.NewSeatPurchase(org.ID, nil, oldID, 3, 4, nil)
			purchase.Status = billingmodels.SeatPurchaseStatusUnknown
			purchase.CreatedAt = now
			require.NoError(t, purchase.Create(db))
			before := captureSubscriptionState(t, db, org.ID)
			hook := logtest.NewGlobal()
			t.Cleanup(hook.Reset)

			foreign := subscriptionEvent(org.ID, oldID, "canceled", now, end)
			foreign.Data.ProductID, foreign.Data.Seats = product, intPointer(tc.foreignSeats)
			foreign.Data.ModifiedAt = polarTime{Time: tc.foreignModified}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, foreign))

			after := captureSubscriptionState(t, db, org.ID)
			assert.Equal(t, before.plan, after.plan, "the ended snapshot must not replace the stored state")
			assert.Equal(t, before.settings, after.settings)
			assert.Equal(t, before.grants, after.grants)
			require.NotNil(t, after.purchase)
			assert.Equal(t, tc.wantStatus, after.purchase.Status)
			assert.Empty(t, after.purchase.ErrorCode)
			inFlight, err := billingmodels.FindInFlightSeatPurchase(db, org.ID)
			require.NoError(t, err)
			if tc.wantStatus == billingmodels.SeatPurchaseStatusUnknown {
				require.NotNil(t, inFlight)
				assert.Equal(t, purchase.ID, inFlight.ID)
			} else {
				assert.Nil(t, inFlight)
			}
			storedStatus := "active"
			if tc.endReplacement {
				storedStatus = "canceled"
			}
			requireSubscriptionLog(t, hook, log.WarnLevel, "billing.foreign_subscription_snapshot_ignored", org.ID, newID, oldID, storedStatus, "canceled")
		})
	}
}

// Reconciliation reads the subscription of an in-flight purchase from Polar.
// When that subscription was replaced and has ended, its snapshot cannot take
// the replacement back, but it settles the purchase so it stops blocking new
// purchases on the replacement.
func Test__ReconcileSeatPurchaseSettlesPurchaseOnReplacedSubscription(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	oldID, newID := "sub_old_"+uuid.NewString(), "sub_new_"+uuid.NewString()
	product := createSeatSubscriptionProduct(t, db)
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	replacement := subscriptionEvent(org.ID, newID, "active", now, end)
	replacement.Data.ProductID, replacement.Data.Seats = product, intPointer(5)
	replacement.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, replacement))
	purchase := billingmodels.NewSeatPurchase(org.ID, nil, oldID, 3, 4, nil)
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))
	before := captureSubscriptionState(t, db, org.ID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/subscriptions/"+oldID, req.URL.Path)
		w.Header().Set(apiVersionHeader, APIVersion())
		snapshot := polarSubscriptionJSON(oldID, "canceled", org.ID.String(), now, end)
		snapshot["product_id"] = product
		snapshot["seats"] = 4
		snapshot["modified_at"] = now.Add(time.Minute).Format(time.RFC3339)
		require.NoError(t, json.NewEncoder(w).Encode(snapshot))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, ReconcileSeatPurchase(t.Context(), db, NewClientFromEnv(), org.ID))

	after := captureSubscriptionState(t, db, org.ID)
	assert.Equal(t, before.plan, after.plan, "the ended snapshot must not replace the stored state")
	require.NotNil(t, after.purchase)
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, after.purchase.Status)
	inFlight, err := billingmodels.FindInFlightSeatPurchase(db, org.ID)
	require.NoError(t, err)
	assert.Nil(t, inFlight, "the settled purchase no longer blocks purchases on the replacement")
}

func Test__ApplySubscriptionReplacesCanceledSubscription(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing_timestamp_and_periods_%t", missing), func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			oldID, newID := "sub_old_"+uuid.NewString(), "sub_new_"+uuid.NewString()
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			old := subscriptionEvent(org.ID, oldID, "active", now, now.AddDate(0, 1, 0))
			old.Data.ModifiedAt = polarTime{Time: now.Add(time.Hour)}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, old))
			old.Data.Status = "canceled"
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, old))

			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests++
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "/subscriptions/"+newID, req.URL.Path)
				w.Header().Set(apiVersionHeader, APIVersion())
				snapshot := polarSubscriptionJSON(newID, "active", org.ID.String(), now, now.AddDate(0, 1, 0))
				if missing {
					delete(snapshot, "modified_at")
					delete(snapshot, "current_period_start")
					delete(snapshot, "current_period_end")
				} else {
					snapshot["modified_at"] = now.Format(time.RFC3339)
				}
				require.NoError(t, json.NewEncoder(w).Encode(snapshot))
			}))
			t.Cleanup(server.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
			t.Setenv("POLAR_API_BASE_URL", server.URL)

			replacement := subscriptionEvent(org.ID, newID, "active", now, now.AddDate(0, 1, 0))
			replacement.Data.ModifiedAt = polarTime{Time: now}
			if missing {
				replacement.Data.ModifiedAt = polarTime{}
				replacement.Data.CurrentPeriodStart, replacement.Data.CurrentPeriodEnd = polarTime{}, polarTime{}
			}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, replacement))
			assert.Equal(t, 1, requests)
			plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
			require.NoError(t, err)
			require.NotNil(t, plan.PolarSubscriptionID)
			assert.Equal(t, newID, *plan.PolarSubscriptionID)
			assert.True(t, plan.IsActiveBusiness())
			if missing {
				assert.Nil(t, plan.PolarModifiedAt)
				assert.Nil(t, plan.CurrentPeriodStart)
				assert.Nil(t, plan.CurrentPeriodEnd)
				return
			}
			require.NotNil(t, plan.PolarModifiedAt)
			assert.True(t, plan.PolarModifiedAt.Equal(now))
			summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
			require.NoError(t, err)
			assert.Equal(t, billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
		})
	}
}

// A delayed event for an ended foreign subscription must not restore access:
// the stored subscription has ended, so only a Polar API verification can
// prove the incoming subscription is live.
func Test__ApplySubscriptionVerifiesForeignSubscriptionAfterStoredEnded(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		responseStatus             int
		verifiedStatus             string
		wrongOrganization          bool
		wantError, wantReplacement bool
	}{
		{name: "verified ended foreign ignored", verifiedStatus: "canceled"},
		{name: "verified revoked foreign ignored", verifiedStatus: "revoked"},
		{name: "verified live foreign replaces", verifiedStatus: "active", wantReplacement: true},
		{name: "verification failed", responseStatus: http.StatusInternalServerError, wantError: true},
		{name: "verification wrong organization", verifiedStatus: "active", wrongOrganization: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			storedID, delayedID := "sub_stored_"+uuid.NewString(), "sub_delayed_"+uuid.NewString()
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			stored := subscriptionEvent(org.ID, storedID, "active", now, end)
			stored.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stored))
			stored.Data.Status = "canceled"
			stored.Data.ModifiedAt = polarTime{Time: now.Add(time.Minute)}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stored))
			before := captureSubscriptionState(t, db, org.ID)
			require.False(t, before.plan.IsActiveBusiness())

			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests++
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "/subscriptions/"+delayedID, req.URL.Path)
				w.Header().Set(apiVersionHeader, APIVersion())
				if tc.responseStatus != 0 {
					http.Error(w, "verification failed", tc.responseStatus)
					return
				}
				snapshot := polarSubscriptionJSON(delayedID, tc.verifiedStatus, org.ID.String(), now, end)
				snapshot["modified_at"] = now.Add(2 * time.Minute).Format(time.RFC3339)
				if tc.wrongOrganization {
					snapshot["customer"] = map[string]any{"id": "cust_other", "external_id": uuid.NewString()}
				}
				require.NoError(t, json.NewEncoder(w).Encode(snapshot))
			}))
			t.Cleanup(server.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
			t.Setenv("POLAR_API_BASE_URL", server.URL)
			hook := logtest.NewGlobal()
			t.Cleanup(hook.Reset)

			delayed := subscriptionEvent(org.ID, delayedID, "active", now, end)
			delayed.Data.ModifiedAt = polarTime{Time: now.Add(-time.Hour)}
			delayed.Data.Customer.ID = "cust_delayed"
			err := ApplySubscriptionEvent(t.Context(), db, delayed)
			assert.Equal(t, 1, requests)
			if tc.wantError {
				require.Error(t, err)
				assert.NotErrorIs(t, err, ErrPermanentApply)
				assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
				return
			}
			require.NoError(t, err)
			if !tc.wantReplacement {
				assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
				requireSubscriptionLog(t, hook, log.WarnLevel, "billing.foreign_subscription_snapshot_ignored", org.ID, storedID, delayedID, "canceled", tc.verifiedStatus)
				return
			}
			after := captureSubscriptionState(t, db, org.ID)
			require.NotNil(t, after.plan.PolarSubscriptionID)
			assert.Equal(t, delayedID, *after.plan.PolarSubscriptionID)
			assert.True(t, after.plan.IsActiveBusiness())
		})
	}
}

// A stored subscription can end while a delayed foreign event is in flight:
// the refresh records the stored cancellation, and the delayed event still
// needs its own verification before it can replace the stored subscription.
func Test__ApplySubscriptionVerifiesForeignSubscriptionAfterRefreshEndsStored(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	storedID, delayedID := "sub_stored_"+uuid.NewString(), "sub_delayed_"+uuid.NewString()
	require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	initial := subscriptionEvent(org.ID, storedID, "active", now, end)
	initial.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
	before := captureSubscriptionState(t, db, org.ID)
	require.True(t, before.plan.IsActiveBusiness())

	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests = append(requests, req.URL.Path)
		w.Header().Set(apiVersionHeader, APIVersion())
		assert.Equal(t, http.MethodGet, req.Method)
		snapshot := polarSubscriptionJSON(strings.TrimPrefix(req.URL.Path, "/subscriptions/"), "canceled", org.ID.String(), now, end)
		snapshot["modified_at"] = now.Add(time.Minute).Format(time.RFC3339)
		require.NoError(t, json.NewEncoder(w).Encode(snapshot))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	hook := logtest.NewGlobal()
	t.Cleanup(hook.Reset)

	delayed := subscriptionEvent(org.ID, delayedID, "active", now, end)
	delayed.Data.ModifiedAt = polarTime{Time: now.Add(-time.Hour)}
	delayed.Data.Customer.ID = "cust_delayed"
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, delayed))

	assert.Equal(t, []string{"/subscriptions/" + storedID, "/subscriptions/" + delayedID}, requests)
	after := captureSubscriptionState(t, db, org.ID)
	require.NotNil(t, after.plan.PolarSubscriptionID)
	assert.Equal(t, storedID, *after.plan.PolarSubscriptionID)
	assert.False(t, after.plan.IsActiveBusiness())
	requireSubscriptionLog(t, hook, log.WarnLevel, "billing.foreign_subscription_snapshot_ignored", org.ID, storedID, delayedID, "canceled", "canceled")
}

func Test__ApplySubscriptionReleasesLocksDuringPolarReads(t *testing.T) {
	for _, tc := range []struct {
		name          string
		storedEnded   bool
		changeStored  bool
		adoptIncoming bool
	}{
		{name: "refresh stored subscription"},
		{name: "verify foreign subscription", storedEnded: true},
		{name: "identity changes during refresh", changeStored: true},
		{name: "identity changes during verification", storedEnded: true, changeStored: true},
		{name: "incoming identity adopted and ended during verification", storedEnded: true, changeStored: true, adoptIncoming: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			storedID, incomingID, concurrentID := "sub_stored_"+uuid.NewString(), "sub_incoming_"+uuid.NewString(), "sub_concurrent_"+uuid.NewString()
			if tc.adoptIncoming {
				concurrentID = incomingID
			}
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			stored := subscriptionEvent(org.ID, storedID, "active", now, end)
			stored.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stored))
			if tc.storedEnded {
				stored.Data.Status = "canceled"
				stored.Data.ModifiedAt = polarTime{Time: now.Add(time.Second)}
				require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stored))
			}

			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests = append(requests, req.URL.Path)
				// A separate transaction must acquire the same row immediately,
				// while the subscription apply path waits for this HTTP response.
				err := db.Transaction(func(tx *gorm.DB) error {
					if err := tx.Exec("SELECT id FROM organizations WHERE id = ? FOR UPDATE NOWAIT", org.ID).Error; err != nil {
						return err
					}
					if !tc.changeStored || len(requests) != 1 {
						return nil
					}
					stored.Data.Status = "canceled"
					stored.Data.ModifiedAt = polarTime{Time: now.Add(2 * time.Second)}
					if _, err := StageOrganizationSubscription(req.Context(), tx, org.ID, stored.Data); err != nil {
						return err
					}
					concurrent := subscriptionEvent(org.ID, concurrentID, "active", now, end)
					concurrent.Data.ModifiedAt = polarTime{Time: now.Add(3 * time.Second)}
					if _, err := StageOrganizationSubscription(req.Context(), tx, org.ID, concurrent.Data); err != nil {
						return err
					}
					if tc.adoptIncoming {
						concurrent.Data.Status = "canceled"
						concurrent.Data.ModifiedAt = polarTime{Time: now.Add(4 * time.Second)}
						_, err := StageOrganizationSubscription(req.Context(), tx, org.ID, concurrent.Data)
						return err
					}
					return nil
				})
				if !assert.NoError(t, err, "Polar reads must not hold the organization lock") {
					http.Error(w, "organization locked", http.StatusInternalServerError)
					return
				}
				id := strings.TrimPrefix(req.URL.Path, "/subscriptions/")
				status := "active"
				if id == storedID || tc.adoptIncoming {
					status = "canceled"
				}
				w.Header().Set(apiVersionHeader, APIVersion())
				snapshot := polarSubscriptionJSON(id, status, org.ID.String(), now, end)
				snapshot["modified_at"] = now.Add(time.Minute).Format(time.RFC3339)
				assert.NoError(t, json.NewEncoder(w).Encode(snapshot))
			}))
			t.Cleanup(server.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
			t.Setenv("POLAR_API_BASE_URL", server.URL)

			incoming := subscriptionEvent(org.ID, incomingID, "active", now, end)
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, incoming))
			plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
			require.NoError(t, err)
			require.NotNil(t, plan.PolarSubscriptionID)
			if tc.changeStored {
				assert.Equal(t, concurrentID, *plan.PolarSubscriptionID)
				if tc.adoptIncoming {
					assert.False(t, plan.IsActiveBusiness(), "the original webhook must not override the verified ended snapshot")
					return
				}
				assert.Equal(t, "/subscriptions/"+concurrentID, requests[len(requests)-1], "recheck the new stored identity before applying")
				return
			}
			assert.Equal(t, incomingID, *plan.PolarSubscriptionID)
			want := []string{"/subscriptions/" + incomingID}
			if !tc.storedEnded {
				want = append([]string{"/subscriptions/" + storedID}, want...)
			}
			assert.Equal(t, want, requests)
		})
	}
}

func Test__ApplySubscriptionRequiresReadsOutsideCallerTransaction(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	stored := subscriptionEvent(org.ID, "sub_stored_"+uuid.NewString(), "active", now, end)
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stored))
	before := captureSubscriptionState(t, db, org.ID)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set(apiVersionHeader, APIVersion())
		assert.NoError(t, json.NewEncoder(w).Encode(polarSubscriptionJSON(stored.Data.ID, "active", org.ID.String(), now, end)))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	err := db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, org.ID); err != nil {
			return err
		}
		incoming := subscriptionEvent(org.ID, "sub_foreign_"+uuid.NewString(), "active", now, end)
		_, err := StageOrganizationSubscription(t.Context(), tx, org.ID, incoming.Data)
		return err
	})
	require.ErrorIs(t, err, billingmodels.ErrPolarSubscriptionRefreshFailed)
	assert.NotErrorIs(t, err, ErrPermanentApply)
	assert.Zero(t, requests)
	assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
}

func Test__ApplySubscriptionResolvesTwoLiveSubscriptions(t *testing.T) {
	for _, tc := range []struct {
		name                                                                                   string
		responseStatus                                                                         int
		refreshID, refreshStatus, incomingStatus                                               string
		modifiedOffset                                                                         time.Duration
		wantError, wantReplacement, wrongOrganization, scheduled, refreshedScheduledWithoutEnd bool
	}{
		{name: "both live", refreshID: "sub_stored", refreshStatus: "active"},
		{name: "incoming past due", refreshID: "sub_stored", refreshStatus: "active", incomingStatus: "past_due"},
		{name: "incoming trialing", refreshID: "sub_stored", refreshStatus: "active", incomingStatus: "trialing"},
		{name: "incoming scheduled cancellation", refreshID: "sub_stored", refreshStatus: "active", incomingStatus: "canceled", scheduled: true},
		{name: "refreshed scheduled cancellation without period", refreshID: "sub_stored", refreshStatus: "canceled", refreshedScheduledWithoutEnd: true},
		{name: "missed cancellation", refreshID: "sub_stored", refreshStatus: "canceled", wantReplacement: true},
		{name: "refresh failed", responseStatus: http.StatusInternalServerError, wantError: true},
		{name: "not found", responseStatus: http.StatusNotFound, wantError: true},
		{name: "wrong subscription", refreshID: "sub_wrong", refreshStatus: "canceled", wantError: true},
		{name: "wrong organization", refreshID: "sub_stored", refreshStatus: "canceled", wrongOrganization: true, wantError: true},
		{name: "stale cancellation", refreshID: "sub_stored", refreshStatus: "canceled", modifiedOffset: -time.Minute, wantError: true},
		{name: "empty refreshed status", refreshID: "sub_stored", wantError: true},
		{name: "unknown refreshed status", refreshID: "sub_stored", refreshStatus: "surprise", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			storedID, incomingID := "sub_stored_"+uuid.NewString(), "sub_incoming_"+uuid.NewString()
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			initial := subscriptionEvent(org.ID, storedID, "active", now, end)
			initial.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
			oldGrant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, billingmodels.IncludedGrantKey(storedID, end))
			require.NoError(t, err, "the stored subscription must start with an included credit grant")
			require.Equal(t, org.ID, oldGrant.OrganizationID)
			require.False(t, oldGrant.IsExpired(time.Now()))
			before := captureSubscriptionState(t, db, org.ID)
			incomingStatus := tc.incomingStatus
			if incomingStatus == "" {
				incomingStatus = "active"
			}
			incomingModified := now.Add(-time.Hour)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests++
				w.Header().Set(apiVersionHeader, APIVersion())
				assert.Equal(t, http.MethodGet, req.Method)
				if req.URL.Path == "/subscriptions/"+incomingID {
					snapshot := polarSubscriptionJSON(incomingID, incomingStatus, org.ID.String(), now, end)
					snapshot["modified_at"] = incomingModified.Format(time.RFC3339)
					snapshot["customer"] = map[string]any{"id": "cust_incoming", "external_id": org.ID.String()}
					require.NoError(t, json.NewEncoder(w).Encode(snapshot))
					return
				}
				assert.Equal(t, "/subscriptions/"+storedID, req.URL.Path)
				if tc.responseStatus != 0 {
					http.Error(w, "refresh failed", tc.responseStatus)
					return
				}
				refreshID := tc.refreshID
				if refreshID == "sub_stored" {
					refreshID = storedID
				}
				snapshot := polarSubscriptionJSON(refreshID, tc.refreshStatus, org.ID.String(), now, end)
				if tc.wrongOrganization {
					snapshot["customer"] = map[string]any{"id": "cust_other", "external_id": uuid.NewString()}
				}
				if tc.refreshedScheduledWithoutEnd {
					snapshot["cancel_at_period_end"] = true
					delete(snapshot, "current_period_end")
				}
				snapshot["modified_at"] = now.Add(tc.modifiedOffset).Format(time.RFC3339)
				require.NoError(t, json.NewEncoder(w).Encode(snapshot))
			}))
			t.Cleanup(server.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
			t.Setenv("POLAR_API_BASE_URL", server.URL)
			hook := logtest.NewGlobal()
			t.Cleanup(hook.Reset)

			incoming := subscriptionEvent(org.ID, incomingID, incomingStatus, now, end)
			incoming.Data.CancelAtPeriodEnd = tc.scheduled
			incoming.Data.ModifiedAt = polarTime{Time: incomingModified}
			incoming.Data.Customer.ID = "cust_incoming"
			err = ApplySubscriptionEvent(t.Context(), db, incoming)
			wantRequests := 1
			if tc.wantReplacement {
				wantRequests = 2
			}
			assert.Equal(t, wantRequests, requests)
			if tc.wantError {
				require.Error(t, err)
				assert.NotErrorIs(t, err, ErrPermanentApply)
				assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
				return
			}
			require.NoError(t, err)
			if !tc.wantReplacement {
				assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
				requireSubscriptionLog(t, hook, log.ErrorLevel, "billing.multiple_live_subscriptions", org.ID, storedID, incomingID, tc.refreshStatus, incomingStatus)
				return
			}
			after := captureSubscriptionState(t, db, org.ID)
			require.NotNil(t, after.plan.PolarSubscriptionID)
			assert.Equal(t, incomingID, *after.plan.PolarSubscriptionID)
			assert.True(t, after.plan.IsActiveBusiness())
			require.NotNil(t, after.plan.PolarModifiedAt)
			assert.True(t, after.plan.PolarModifiedAt.Equal(incoming.Data.ModifiedAt.Time))
			assert.Equal(t, "cust_incoming", *after.settings.PolarCustomerID)
			var oldExpired, newLive bool
			for _, grant := range after.grants {
				if grant.Kind != billingmodels.LLMCreditGrantKindIncluded || grant.PolarOrderID == nil {
					continue
				}
				oldExpired = oldExpired || (grant.ID == oldGrant.ID && grant.IsExpired(time.Now()))
				newLive = newLive || (*grant.PolarOrderID == billingmodels.IncludedGrantKey(incomingID, end) && !grant.IsExpired(time.Now()))
			}
			assert.True(t, oldExpired, "the refreshed cancellation must expire the old included grant")
			assert.True(t, newLive, "the replacement must grant its own included credits")
		})
	}
}

func Test__SyncOrganizationSubscriptionSelectsStableIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, stored, storedStatus, want string
		items                            []string
	}{
		{name: "stored live wins", stored: "sub_stored", storedStatus: "active", want: "sub_stored", items: []string{"sub_newer", "sub_stored"}},
		{name: "newest live", want: "sub_newer", items: []string{"sub_older", "sub_newer"}},
		{name: "newest live shuffled", want: "sub_newer", items: []string{"sub_newer", "sub_older"}},
		{name: "missing dates last", want: "sub_older", items: []string{"sub_missing", "sub_older"}},
		{name: "equal dates ascending id", want: "sub_a", items: []string{"sub_z", "sub_a"}},
		{name: "past due is live", want: "sub_due", items: []string{"sub_canceled", "sub_due"}},
		{name: "scheduled cancellation is live", want: "sub_scheduled", items: []string{"sub_canceled", "sub_scheduled"}},
		{name: "stored ended fallback", stored: "sub_canceled", storedStatus: "canceled", want: "sub_canceled", items: []string{"sub_other_canceled", "sub_canceled"}},
		{name: "foreign ended leaves state", want: "", items: []string{"sub_canceled"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			if tc.stored != "" {
				require.NoError(t, ApplySubscriptionEvent(t.Context(), db, subscriptionEvent(org.ID, tc.stored+"_"+org.ID.String(), tc.storedStatus, now, end)))
			}
			var snapshots []map[string]any
			for _, id := range tc.items {
				status := "active"
				if strings.Contains(id, "canceled") || id == "sub_scheduled" {
					status = "canceled"
				}
				if id == "sub_due" {
					status = "past_due"
				}
				snapshot := polarSubscriptionJSON(id+"_"+org.ID.String(), status, org.ID.String(), now, end)
				if id != "sub_missing" {
					created := now.Add(-time.Hour)
					if id == "sub_newer" {
						created = now
					}
					snapshot["created_at"] = created.Format(time.RFC3339)
				}
				if id == "sub_scheduled" {
					snapshot["cancel_at_period_end"] = true
				}
				snapshots = append(snapshots, snapshot)
			}
			usePolarSubscriptionServer(t, org.ID.String(), snapshots)
			require.NoError(t, SyncOrganizationSubscription(t.Context(), db, org.ID))
			plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
			require.NoError(t, err)
			if tc.want == "" {
				assert.Nil(t, plan.PolarSubscriptionID)
				return
			}
			require.NotNil(t, plan.PolarSubscriptionID)
			assert.Equal(t, tc.want+"_"+org.ID.String(), *plan.PolarSubscriptionID)
		})
	}
}

func Test__SyncOrganizationSubscriptionReportsConflictingLiveSubscriptions(t *testing.T) {
	for _, tc := range []struct {
		name, stored, want, conflict string
	}{
		{name: "stored live kept", stored: "sub_kept", want: "sub_kept", conflict: "sub_duplicate"},
		{name: "newest live kept", want: "sub_newer", conflict: "sub_older"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			if tc.stored != "" {
				require.NoError(t, ApplySubscriptionEvent(t.Context(), db, subscriptionEvent(org.ID, tc.stored+"_"+org.ID.String(), "active", now, end)))
			}
			snapshots := []map[string]any{}
			for _, id := range []string{tc.want, tc.conflict} {
				snapshot := polarSubscriptionJSON(id+"_"+org.ID.String(), "active", org.ID.String(), now, end)
				created := now.Add(-time.Hour)
				if id == "sub_newer" {
					created = now
				}
				snapshot["created_at"] = created.Format(time.RFC3339)
				snapshots = append(snapshots, snapshot)
			}
			usePolarSubscriptionServer(t, org.ID.String(), snapshots)
			hook := logtest.NewGlobal()
			t.Cleanup(hook.Reset)

			require.NoError(t, SyncOrganizationSubscription(t.Context(), db, org.ID))

			plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
			require.NoError(t, err)
			require.NotNil(t, plan.PolarSubscriptionID)
			assert.Equal(t, tc.want+"_"+org.ID.String(), *plan.PolarSubscriptionID)
			assert.True(t, plan.IsActiveBusiness())
			requireSubscriptionLog(t, hook, log.ErrorLevel, "billing.multiple_live_subscriptions", org.ID,
				tc.want+"_"+org.ID.String(), tc.conflict+"_"+org.ID.String(), "active", "active")
		})
	}
}

func createSeatSubscriptionProduct(t *testing.T, db *gorm.DB) string {
	t.Helper()
	product := "product_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	return product
}

type subscriptionState struct {
	plan     *billingmodels.OrganizationBillingPlan
	settings *billingmodels.OrganizationLLMSettings
	grants   []billingmodels.OrganizationLLMCreditGrant
	purchase *billingmodels.SeatPurchase
}

func captureSubscriptionState(t *testing.T, db *gorm.DB, orgID uuid.UUID) subscriptionState {
	t.Helper()
	var state subscriptionState
	var err error
	state.plan, err = billingmodels.FindOrganizationBillingPlan(db, orgID)
	require.NoError(t, err)
	state.settings, err = billingmodels.FindOrganizationLLMSettings(db, orgID)
	require.NoError(t, err)
	state.grants, err = billingmodels.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	state.purchase, err = billingmodels.FindLatestSeatPurchase(db, orgID)
	require.NoError(t, err)
	return state
}

func requireSubscriptionLog(t *testing.T, hook *logtest.Hook, level log.Level, message string, orgID uuid.UUID, storedID, incomingID, storedStatus, incomingStatus string) {
	t.Helper()
	for _, entry := range hook.AllEntries() {
		if entry.Message != message {
			continue
		}
		assert.Equal(t, level, entry.Level)
		assert.Equal(t, orgID.String(), entry.Data["organization_id"])
		assert.Equal(t, storedID, entry.Data["stored_subscription_id"])
		assert.Equal(t, incomingID, entry.Data["incoming_subscription_id"])
		assert.Equal(t, storedStatus, entry.Data["stored_subscription_status"])
		assert.Equal(t, incomingStatus, entry.Data["incoming_subscription_status"])
		return
	}
	t.Errorf("missing log %q", message)
}

func Test__ApplySubscriptionRollsBackCancellationWhenReplacementIsInvalid(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	storedID, incomingID := "sub_stored_"+uuid.NewString(), "sub_incoming_"+uuid.NewString()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	initial := subscriptionEvent(org.ID, storedID, "active", now, end)
	initial.Data.Seats = intPointer(3)
	initial.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
	oldGrant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, billingmodels.IncludedGrantKey(storedID, end))
	require.NoError(t, err)
	require.Equal(t, org.ID, oldGrant.OrganizationID)
	require.False(t, oldGrant.IsExpired(time.Now()))
	purchase := billingmodels.NewSeatPurchase(org.ID, nil, storedID, 3, 4, nil)
	require.NoError(t, purchase.Create(db))
	product := createSeatSubscriptionProduct(t, db)
	before := captureSubscriptionState(t, db, org.ID)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		w.Header().Set(apiVersionHeader, APIVersion())
		if req.URL.Path == "/subscriptions/"+incomingID {
			snapshot := polarSubscriptionJSON(incomingID, "active", org.ID.String(), now, end)
			snapshot["product_id"] = product
			snapshot["modified_at"] = now.Format(time.RFC3339)
			snapshot["customer"] = map[string]any{"id": "cust_incoming", "external_id": org.ID.String()}
			require.NoError(t, json.NewEncoder(w).Encode(snapshot))
			return
		}
		assert.Equal(t, "/subscriptions/"+storedID, req.URL.Path)
		snapshot := polarSubscriptionJSON(storedID, "canceled", org.ID.String(), now, end)
		snapshot["modified_at"], snapshot["seats"] = now.Add(time.Minute).Format(time.RFC3339), 4
		snapshot["customer"] = map[string]any{"id": "cust_replacement", "external_id": org.ID.String()}
		require.NoError(t, json.NewEncoder(w).Encode(snapshot))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	incoming := subscriptionEvent(org.ID, incomingID, "active", now, end)
	incoming.Data.ProductID = product
	incoming.Data.Customer.ID = "cust_incoming"
	err = ApplySubscriptionEvent(t.Context(), db, incoming)

	require.ErrorIs(t, err, ErrPermanentApply)
	assert.Equal(t, 2, requests)
	assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
}

func Test__ApplySubscriptionPreservesStateWhenStoredRefreshSeatsAreInvalid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		seats *int
	}{
		{name: "missing seats"},
		{name: "negative seats", seats: intPointer(-1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			product := createSeatSubscriptionProduct(t, db)
			storedID, incomingID := "sub_stored_"+uuid.NewString(), "sub_incoming_"+uuid.NewString()
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			initial := subscriptionEvent(org.ID, storedID, "active", now, end)
			initial.Data.ProductID, initial.Data.Seats = product, intPointer(5)
			initial.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
			purchase := billingmodels.NewSeatPurchase(org.ID, nil, storedID, 5, 6, nil)
			require.NoError(t, purchase.Create(db))
			before := captureSubscriptionState(t, db, org.ID)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests++
				assert.Equal(t, "/subscriptions/"+storedID, req.URL.Path)
				w.Header().Set(apiVersionHeader, APIVersion())
				snapshot := polarSubscriptionJSON(storedID, "canceled", org.ID.String(), now, end)
				snapshot["product_id"], snapshot["seats"] = product, tc.seats
				snapshot["modified_at"] = now.Add(time.Minute).Format(time.RFC3339)
				snapshot["customer"] = map[string]any{"id": "cust_refreshed", "external_id": org.ID.String()}
				require.NoError(t, json.NewEncoder(w).Encode(snapshot))
			}))
			t.Cleanup(server.Close)
			t.Setenv("POLAR_API_BASE_URL", server.URL)

			incoming := subscriptionEvent(org.ID, incomingID, "active", now, end)
			incoming.Data.Customer.ID = "cust_incoming"
			err := ApplySubscriptionEvent(t.Context(), db, incoming)

			require.Error(t, err)
			assert.NotErrorIs(t, err, ErrPermanentApply)
			assert.Equal(t, 1, requests)
			assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
		})
	}
}
