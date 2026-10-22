package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func Test__ApplySubscriptionActivatesBusinessAndGrantsIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_biz", "active", periodStart, periodEnd)

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)
	requireWelcomeGrantExpiresFourteenDaysAfterCreate(t, db, org.ID)
}

func Test__ApplySubscriptionCancelRestoresTrialAndExpiresIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_biz", "active", periodStart, periodEnd)))

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_biz", "canceled", periodStart, periodEnd)))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.RemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)
}

func Test__ApplySubscriptionCancelEndsPlanAfterTrialAndExpiresIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_lapsed", "active", periodStart, periodEnd)))
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", org.ID).
		Update("trial_ends_at", time.Now().Add(-time.Hour)).Error)
	require.NoError(t, db.Model(&models.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", org.ID, models.LLMCreditGrantKindWelcome).
		Update("expires_at", time.Now().Add(-time.Hour)).Error)

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_lapsed", "canceled", periodStart, periodEnd)))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.IsOpenTrial(time.Now()))

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, int64(0), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)
}

func Test__ApplySubscriptionResubscribeSamePeriodGrantsIncludedAgain(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_renew", "active", periodStart, periodEnd)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_renew", "canceled", periodStart, periodEnd)))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, int64(0), summary.PurchasedRemainingMicros)
	assertNoTrialConversionGrant(t, db, org.ID)

	grants, err := models.ListOrganizationLLMCreditGrants(db, org.ID)
	require.NoError(t, err)
	liveIncluded := 0
	canceledIncluded := 0
	for _, grant := range grants {
		if grant.Kind != models.LLMCreditGrantKindIncluded {
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
		assert.Equal(t, models.IncludedGrantKey("sub_renew", periodEnd), *grant.PolarOrderID)
	}
	assert.Equal(t, 1, liveIncluded)
	assert.Equal(t, 1, canceledIncluded)
}

func Test__ApplySubscriptionIncompleteKeepsTrial(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)

	require.NoError(t, ApplySubscriptionEvent(
		context.Background(),
		db,
		subscriptionEvent(org.ID, "sub_incomplete", "incomplete", periodStart, periodEnd),
	))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())
}

func Test__ApplySubscriptionCancelAtPeriodEndKeepsBusiness(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(org.ID, "sub_ending", "active", periodStart, periodEnd)
	event.Data.CancelAtPeriodEnd = true

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.CancelAtPeriodEnd)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__CancelOrganizationSubscriptionSchedulesPeriodEnd(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.CancelAtPeriodEnd)
	assert.True(t, plan.IsActiveBusiness())
}

func Test__ResumeOrganizationSubscriptionClearsPeriodEndCancel(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.False(t, plan.CancelAtPeriodEnd)
}

func Test__ResumeWithoutModifiedAtDoesNotBlockLaterWebhook(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.False(t, plan.CancelAtPeriodEnd)
	require.NotNil(t, plan.PolarModifiedAt)
	assert.True(t, plan.PolarModifiedAt.Equal(provider))

	webhook := subscriptionEvent(org.ID, "sub_resume", "active", periodStart, periodEnd)
	webhook.Data.ModifiedAt = polarTime{Time: laterWebhook}
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, webhook))

	plan, err = models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.False(t, plan.CancelAtPeriodEnd)
	require.NotNil(t, plan.PolarModifiedAt)
	assert.True(t, plan.PolarModifiedAt.Equal(laterWebhook))
}

func Test__ApplySubscriptionIgnoresStaleCancelAfterResume(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.False(t, plan.CancelAtPeriodEnd)
	require.NotNil(t, plan.PolarModifiedAt)
	assert.True(t, plan.PolarModifiedAt.Equal(newer))
}

func Test__CancelOrganizationSubscriptionRejectsTrial(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

	err := CancelOrganizationSubscription(context.Background(), db, org.ID)
	require.ErrorIs(t, err, ErrSubscriptionNotCancelable)
}

func Test__CancelOrganizationSubscriptionRejectsAdminPlan(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	_, err := models.SetAdminOrganizationPlan(db, org.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

	err = CancelOrganizationSubscription(context.Background(), db, org.ID)
	require.ErrorIs(t, err, ErrAdminPlanCannotCancel)
}

func Test__ApplySubscriptionIncludedGrantIsIdempotentWithoutPeriodStart(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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

	grants, err := models.ListOrganizationLLMCreditGrants(db, org.ID)
	require.NoError(t, err)
	included := 0
	for _, grant := range grants {
		if grant.Kind != models.LLMCreditGrantKindIncluded {
			continue
		}
		included++
		require.NotNil(t, grant.PolarOrderID)
		assert.Equal(t, models.IncludedGrantKey("sub_no_start", periodEnd), *grant.PolarOrderID)
	}
	assert.Equal(t, 1, included)
}

func Test__SyncOrganizationSubscriptionActivatesBusinessAndGrantsIncluded(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_canceled", "canceled", org.ID.String(), periodStart.AddDate(0, -1, 0), periodStart),
		polarSubscriptionJSON("sub_sync_biz", "active", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))
	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())
	require.NotNil(t, plan.PolarSubscriptionID)
	assert.Equal(t, "sub_sync_biz", *plan.PolarSubscriptionID)

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__SyncOrganizationSubscriptionAppliesCanceledWhenNoPaidExists(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(org.ID, "sub_sync_cancel", "active", periodStart, periodEnd)))

	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_cancel", "canceled", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())
}

func Test__SyncOrganizationSubscriptionLeavesTrialWhenPolarHasNone(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	usePolarSubscriptionServer(t, org.ID.String(), nil)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceSystem, plan.PlanSource)
}

func Test__SyncOrganizationSubscriptionLeavesTrialWhenPolarIsIncomplete(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_incomplete", "incomplete", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
}

func Test__SyncOrganizationSubscriptionReplacesAdminTrialWithPolarPaid(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	_, err := models.SetAdminOrganizationPlan(db, org.ID, models.BillingPlanTrial, nil)
	require.NoError(t, err)
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_admin", "active", org.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourcePolar, plan.PlanSource)
	assert.True(t, plan.IsActiveBusiness())
}

func Test__SyncOrganizationSubscriptionLeavesAdminBusinessWhenPolarHasNone(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	_, err := models.SetAdminOrganizationPlan(db, org.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)
	usePolarSubscriptionServer(t, org.ID.String(), nil)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__SyncOrganizationSubscriptionKeepsLocalPlanWhenPolarFails(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "polar down", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
}

func Test__SyncOrganizationSubscriptionNoopsWhenCheckoutDisabled(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, org.ID))

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
}

func assertNoTrialConversionGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) {
	t.Helper()
	grants, err := models.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	for _, grant := range grants {
		if grant.PolarOrderID == nil {
			continue
		}
		assert.False(t, strings.HasPrefix(*grant.PolarOrderID, "trial-conversion:"))
	}
}

func requireWelcomeGrantExpiresFourteenDaysAfterCreate(t *testing.T, db *gorm.DB, orgID uuid.UUID) {
	t.Helper()
	var grant models.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", orgID, models.LLMCreditGrantKindWelcome).
		First(&grant).Error)
	require.NotNil(t, grant.ExpiresAt)
	assert.WithinDuration(t, grant.CreatedAt.Add(models.DefaultWelcomeGrantTTL), *grant.ExpiresAt, time.Second)
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
	plan, err := models.FindOrganizationBillingPlan(database.Conn(), org.ID)
	require.NoError(t, err)
	require.Equal(t, models.BillingPlanTrial, plan.Plan)
	require.Nil(t, plan.PolarSubscriptionID)
}

func TestCatalogSubscriptionDoesNotDeliverNewPolicies(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	plan := models.BillingPlan{Key: "paid_" + uuid.NewString()[:8], Name: "Paid", Category: "paid", Visibility: "private", PolarProductID: &product, CreditAmountCents: 7500, CreditFrequency: "billing_period", CreditExpiration: "period_end"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	now := time.Now()
	event := subscriptionEvent(org.ID, "sub_custom", "active", now, now.AddDate(0, 1, 0))
	event.Data.ProductID = product
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, event))
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, event))
	order := &OrderWebhookEvent{Type: orderPaidType, Data: OrderData{ID: uuid.NewString(), ProductID: product, BillingReason: billingReasonSubscriptionCycle, Customer: OrderCustomer{ExternalID: org.ID.String()}}}
	require.NoError(t, ApplyOrderPaid(t.Context(), db, order, nil))
	saved, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	require.Equal(t, plan.ID, saved.PlanID)
	require.True(t, saved.AllowsCreditPurchase())
	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	require.Zero(t, summary.IncludedRemainingMicros)
}

func Test__ApplySubscriptionRenewalAdvancesPeriodAndGrantsIncluded(t *testing.T) {
	for _, eventType := range []string{subscriptionCycledType, subscriptionUpdatedType} {
		t.Run(eventType, func(t *testing.T) {
			org := setupBillingOrganization(t)
			require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
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
	plan, err := models.FindOrganizationBillingPlan(db, orgID)
	require.NoError(t, err)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)
	assert.True(t, start.Equal(*plan.CurrentPeriodStart), "period start %s, want %s", plan.CurrentPeriodStart, start)
	assert.True(t, end.Equal(*plan.CurrentPeriodEnd), "period end %s, want %s", plan.CurrentPeriodEnd, end)
}

func requireIncludedGrantExpiries(t *testing.T, db *gorm.DB, orgID uuid.UUID, want ...time.Time) {
	t.Helper()
	grants, err := models.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	var got []time.Time
	for _, grant := range grants {
		if grant.Kind != models.LLMCreditGrantKindIncluded {
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
