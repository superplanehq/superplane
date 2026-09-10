package polar

import (
	"context"
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
