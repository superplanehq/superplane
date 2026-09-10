package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ApplySubscriptionActivatesBusinessAndGrantsIncluded(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	event := subscriptionEvent(r.Organization.ID, "sub_biz", "active", periodStart, periodEnd)

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.PurchasedRemainingMicros)
}

func Test__ApplySubscriptionCancelStopsBusiness(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(r.Organization.ID, "sub_biz", "active", periodStart, periodEnd)))

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(r.Organization.ID, "sub_biz", "canceled", periodStart, periodEnd)))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
}

func Test__ApplySubscriptionIncompleteKeepsTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)

	require.NoError(t, ApplySubscriptionEvent(
		context.Background(),
		db,
		subscriptionEvent(r.Organization.ID, "sub_incomplete", "incomplete", periodStart, periodEnd),
	))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())
}

func Test__ApplySubscriptionIncludedGrantIsIdempotentWithoutPeriodStart(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodEnd := time.Now().UTC().Truncate(time.Second).AddDate(0, 1, 0)
	event := &SubscriptionWebhookEvent{
		Type: subscriptionUpdatedType,
		Data: SubscriptionData{
			ID:               "sub_no_start",
			Status:           "active",
			CurrentPeriodEnd: polarTime{Time: periodEnd},
			Customer: OrderCustomer{
				ID:         "cust_polar_1",
				ExternalID: r.Organization.ID.String(),
			},
		},
	}

	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, event))

	grants, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
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
	r := support.Setup(t)
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, r.Organization.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_canceled", "canceled", r.Organization.ID.String(), periodStart.AddDate(0, -1, 0), periodStart),
		polarSubscriptionJSON("sub_sync_biz", "active", r.Organization.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))
	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())
	require.NotNil(t, plan.PolarSubscriptionID)
	assert.Equal(t, "sub_sync_biz", *plan.PolarSubscriptionID)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__SyncOrganizationSubscriptionAppliesCanceledWhenNoPaidExists(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	require.NoError(t, ApplySubscriptionEvent(context.Background(), db, subscriptionEvent(r.Organization.ID, "sub_sync_cancel", "active", periodStart, periodEnd)))

	usePolarSubscriptionServer(t, r.Organization.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_cancel", "canceled", r.Organization.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
}

func Test__SyncOrganizationSubscriptionLeavesTrialWhenPolarHasNone(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	usePolarSubscriptionServer(t, r.Organization.ID.String(), nil)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceSystem, plan.PlanSource)
}

func Test__SyncOrganizationSubscriptionLeavesTrialWhenPolarIsIncomplete(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	usePolarSubscriptionServer(t, r.Organization.ID.String(), []map[string]any{
		polarSubscriptionJSON("sub_sync_incomplete", "incomplete", r.Organization.ID.String(), periodStart, periodEnd),
	})

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
}

func Test__SyncOrganizationSubscriptionSkipsAdminPlan(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		http.Error(w, "should not list subscriptions for admin plans", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))
	assert.False(t, called)

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__SyncOrganizationSubscriptionKeepsLocalPlanWhenPolarFails(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "polar down", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", server.URL)

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
}

func Test__SyncOrganizationSubscriptionNoopsWhenCheckoutDisabled(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")

	require.NoError(t, SyncOrganizationSubscription(context.Background(), db, r.Organization.ID))

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
}

func subscriptionEvent(orgID uuid.UUID, subscriptionID, status string, start, end time.Time) *SubscriptionWebhookEvent {
	return &SubscriptionWebhookEvent{
		Type: subscriptionUpdatedType,
		Data: SubscriptionData{
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
		assert.Equal(t, "prod_business", req.URL.Query().Get("product_id"))
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
