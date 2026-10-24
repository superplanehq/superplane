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

func TestSubscriptionSeatSnapshots(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	catalog := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	initial := polarSubscriptionJSON("sub_seats", "active", org.ID.String(), now, end)
	initial["product_id"], initial["seats"], initial["modified_at"] = product, 3, now.Format(time.RFC3339)
	usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{initial})
	require.NoError(t, SyncOrganizationSubscription(t.Context(), db, org.ID))
	saved, err := models.FindOrganizationBillingPlan(db, org.ID)
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
			saved, err := models.FindOrganizationBillingPlan(db, org.ID)
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

	before, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	settingsBefore, err := models.FindOrganizationLLMSettings(db, org.ID)
	require.NoError(t, err)
	_, err = models.AddIncludedLLMCreditGrant(db, org.ID, models.CentsToMicros(500), "previous_subscription", end)
	require.NoError(t, err)
	grantsBefore, err := models.ListOrganizationLLMCreditGrants(db, org.ID)
	require.NoError(t, err)
	for _, status := range []string{"active", "canceled", "incomplete"} {
		for _, seats := range []string{"", `,"seats":null`, `,"seats":-1`} {
			t.Run(status+seats, func(t *testing.T) {
				var snapshot SubscriptionData
				payload := fmt.Sprintf(`{"id":"sub_seats","status":%q,"product_id":%q,"modified_at":%q,"current_period_end":%q,"customer":{"id":"changed_customer","external_id":%q}%s}`, status, product, now.Add(time.Hour).Format(time.RFC3339), now.Add(-time.Hour).Format(time.RFC3339), org.ID.String(), seats)
				require.NoError(t, json.Unmarshal([]byte(payload), &snapshot))
				require.ErrorIs(t, ApplySubscription(t.Context(), db, snapshot), ErrPermanentApply)
				after, err := models.FindOrganizationBillingPlan(db, org.ID)
				require.NoError(t, err)
				require.Equal(t, before, after)
				grantsAfter, err := models.ListOrganizationLLMCreditGrants(db, org.ID)
				require.NoError(t, err)
				require.Equal(t, grantsBefore, grantsAfter)
				state, err := models.FindOrganizationLLMSettings(db, org.ID)
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
	catalog := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)

	// Polar went from 5 seats down to 3. The snapshot with 3 seats applies.
	fresh := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	fresh.Data.ProductID, fresh.Data.Seats = product, intPointer(3)
	fresh.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, fresh))

	// A purchase from 3 to 4 seats lost its response.
	purchase := models.NewSeatPurchase(org.ID, nil, "sub_seats", 3, 4)
	purchase.Status = models.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))

	// The delayed snapshot with 5 seats is older, so it is ignored.
	stale := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	stale.Data.ProductID, stale.Data.Seats = product, intPointer(5)
	stale.Data.ModifiedAt = polarTime{Time: now.Add(-time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, stale))

	inFlight, err := models.FindInFlightSeatPurchase(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, inFlight, "a stale snapshot must not settle the purchase")
	assert.Equal(t, models.SeatPurchaseStatusUnknown, inFlight.Status)

	saved, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, intPointer(3), saved.PurchasedSeats)
}

// A snapshot fresh enough to apply proves the outcome of an in-flight
// purchase, so a webhook settles it.
func Test__ApplySubscriptionSettlesSeatPurchaseFromFreshSnapshot(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "product_" + uuid.NewString()
	catalog := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)

	initial := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	initial.Data.ProductID, initial.Data.Seats = product, intPointer(3)
	initial.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))

	// A purchase from 3 to 4 seats lost its response.
	purchase := models.NewSeatPurchase(org.ID, nil, "sub_seats", 3, 4)
	purchase.Status = models.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))

	// A newer snapshot at the target proves that Polar applied the purchase.
	applied := subscriptionEvent(org.ID, "sub_seats", "active", now, end)
	applied.Data.ProductID, applied.Data.Seats = product, intPointer(4)
	applied.Data.ModifiedAt = polarTime{Time: now.Add(time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, applied))

	inFlight, err := models.FindInFlightSeatPurchase(db, org.ID)
	require.NoError(t, err)
	assert.Nil(t, inFlight)

	latest, err := models.FindLatestSeatPurchase(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, latest.Status)
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
	_, err := models.AddIncludedLLMCreditGrant(db, org.ID, models.CentsToMicros(500), storedID, now.AddDate(0, 1, 0))
	require.NoError(t, err)
	purchase := models.NewSeatPurchase(org.ID, nil, foreignID, 3, 4)
	require.NoError(t, purchase.Create(db))
	before := captureSubscriptionState(t, db, org.ID)
	hook := logtest.NewGlobal()
	t.Cleanup(hook.Reset)

	foreign := subscriptionEvent(org.ID, foreignID, "canceled", now, now.AddDate(0, 1, 0))
	foreign.Data.ProductID, foreign.Data.Seats = product, intPointer(4)
	foreign.Data.Customer.ID = "cust_foreign"
	foreign.Data.ModifiedAt = polarTime{Time: now.Add(time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, foreign))

	assert.Equal(t, before, captureSubscriptionState(t, db, org.ID))
	requireSubscriptionLog(t, hook, log.WarnLevel, "billing.foreign_subscription_snapshot_ignored", org.ID, storedID, foreignID, "active", "canceled")
}

func Test__ApplySubscriptionReplacesCanceledSubscription(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing_timestamp_and_periods_%t", missing), func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			oldID, newID := "sub_old_"+uuid.NewString(), "sub_new_"+uuid.NewString()
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
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
			plan, err := models.FindOrganizationBillingPlan(db, org.ID)
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
			summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
			require.NoError(t, err)
			assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
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
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
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
	require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
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
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			initial := subscriptionEvent(org.ID, storedID, "active", now, end)
			initial.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
			oldGrant, err := models.FindLLMCreditGrantByPolarOrderID(db, models.IncludedGrantKey(storedID, end))
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
				if grant.Kind != models.LLMCreditGrantKindIncluded || grant.PolarOrderID == nil {
					continue
				}
				oldExpired = oldExpired || (grant.ID == oldGrant.ID && grant.IsExpired(time.Now()))
				newLive = newLive || (*grant.PolarOrderID == models.IncludedGrantKey(incomingID, end) && !grant.IsExpired(time.Now()))
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
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
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
			plan, err := models.FindOrganizationBillingPlan(db, org.ID)
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
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
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

			plan, err := models.FindOrganizationBillingPlan(db, org.ID)
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
	catalog := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &catalog))
	return product
}

type subscriptionState struct {
	plan     *models.OrganizationBillingPlan
	settings *models.OrganizationLLMSettings
	grants   []models.OrganizationLLMCreditGrant
	purchase *models.SeatPurchase
}

func captureSubscriptionState(t *testing.T, db *gorm.DB, orgID uuid.UUID) subscriptionState {
	t.Helper()
	var state subscriptionState
	var err error
	state.plan, err = models.FindOrganizationBillingPlan(db, orgID)
	require.NoError(t, err)
	state.settings, err = models.FindOrganizationLLMSettings(db, orgID)
	require.NoError(t, err)
	state.grants, err = models.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	state.purchase, err = models.FindLatestSeatPurchase(db, orgID)
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
	require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	initial := subscriptionEvent(org.ID, storedID, "active", now, end)
	initial.Data.Seats = intPointer(3)
	initial.Data.ModifiedAt = polarTime{Time: now}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
	oldGrant, err := models.FindLLMCreditGrantByPolarOrderID(db, models.IncludedGrantKey(storedID, end))
	require.NoError(t, err)
	require.Equal(t, org.ID, oldGrant.OrganizationID)
	require.False(t, oldGrant.IsExpired(time.Now()))
	purchase := models.NewSeatPurchase(org.ID, nil, storedID, 3, 4)
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
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
			product := createSeatSubscriptionProduct(t, db)
			storedID, incomingID := "sub_stored_"+uuid.NewString(), "sub_incoming_"+uuid.NewString()
			now := time.Now().UTC().Truncate(time.Second)
			end := now.AddDate(0, 1, 0)
			initial := subscriptionEvent(org.ID, storedID, "active", now, end)
			initial.Data.ProductID, initial.Data.Seats = product, intPointer(5)
			initial.Data.ModifiedAt = polarTime{Time: now}
			require.NoError(t, ApplySubscriptionEvent(t.Context(), db, initial))
			purchase := models.NewSeatPurchase(org.ID, nil, storedID, 5, 6)
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
