package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/polarfake"
)

// Test__RenewalEstimateMatchesTheScheduledSeatCount runs the billing API and
// the renewal scheduler on the same organization state. The estimate that
// the customer sees must be the seat count that Polar receives.
func Test__RenewalEstimateMatchesTheScheduledSeatCount(t *testing.T) {
	tests := []struct {
		name      string
		paidSeats int
		// members includes the owner.
		members            int
		pendingInvitations int
		// change runs after the seed, before the API and the scheduler.
		change func(t *testing.T, state renewalEstimateState)
		// expected is the renewal seat count. Nil means that the paid seats
		// are invalid, so there is no estimate and no scheduled count.
		expected *int
	}{
		{name: "ordinary", paidSeats: 10, members: 3, expected: new(3)},
		{name: "no members renew one seat", paidSeats: 10, members: 1, expected: new(1), change: func(t *testing.T, state renewalEstimateState) {
			require.NoError(t, database.Conn().Delete(&models.User{}, "id = ?", state.r.User).Error)
		}},
		{name: "over capacity renews only the paid seats", paidSeats: 1, members: 3, expected: new(1)},
		{name: "pending invitations do not count", paidSeats: 10, members: 3, pendingInvitations: 2, expected: new(3)},
		{name: "plan maximum change creates no paid seats", paidSeats: 1, members: 3, expected: new(1), change: func(t *testing.T, state renewalEstimateState) {
			require.NoError(t, models.SetOrganizationSeatLimitOverride(database.Conn(), state.r.Organization.ID, new(80)))
		}},
		{name: "zero paid seats", paidSeats: 10, members: 3, change: func(t *testing.T, state renewalEstimateState) {
			state.storeInvalidPaidSeats(t, 0)
		}},
		{name: "missing paid seats", paidSeats: 10, members: 3, change: func(t *testing.T, state renewalEstimateState) {
			state.storeInvalidPaidSeats(t, nil)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newRenewalEstimateState(t, tt.paidSeats, tt.members, tt.pendingInvitations)
			if tt.change != nil {
				tt.change(t, state)
			}

			scheduler := seatrenewal.NewScheduler(database.Conn(), polar.NewClientFromEnv(), time.Now)
			_, scheduleErr := scheduler.Schedule(context.Background(), state.r.Organization.ID)
			billing, err := DescribeOrganizationBilling(context.Background(), state.r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
			require.NoError(t, err)
			require.True(t, billing.PlanPerSeat)

			if tt.expected == nil {
				require.ErrorIs(t, scheduleErr, entitlements.ErrInvalidPaidSeats)
				assert.Empty(t, state.polar.Patches(), "invalid paid seats get no scheduled count")
				assert.Nil(t, billing.ExpectedRenewalSeats, "invalid paid seats get no estimate")
				assert.Nil(t, billing.ExpectedRenewalCents)
				assert.Equal(t, int32(0), billing.GetPurchasedSeats(), "no paid seats are invented")
				return
			}

			require.NoError(t, scheduleErr)
			patches := state.polar.Patches()
			require.Len(t, patches, 1)
			assert.Equal(t, scheduledRenewal(*tt.expected), patches[0].Body)
			require.NotNil(t, billing.ExpectedRenewalSeats)
			assert.Equal(t, int(*billing.ExpectedRenewalSeats), int(patches[0].Body["seats"].(float64)),
				"the estimate is the scheduled seat count")
			require.NotNil(t, billing.ExpectedRenewalCents)
			assert.Equal(t, int64(*tt.expected)*billing.PlanPriceCents, *billing.ExpectedRenewalCents)
			assert.Equal(t, int32(tt.paidSeats), billing.GetPurchasedSeats(), "the paid seats do not change")
			assert.Equal(t, int32(tt.pendingInvitations), billing.GetPendingInvitationSeats())
		})
	}
}

type renewalEstimateState struct {
	r     *support.ResourceRegistry
	polar *polarfake.Server
	sub   polarfake.Subscription
}

func newRenewalEstimateState(t *testing.T, paidSeats, members, pendingInvitations int) renewalEstimateState {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(models.BillingEnabledEnv, "yes")
	state := renewalEstimateState{r: r, polar: polarfake.NewServer(t)}
	state.sub = state.polar.AddSeatPlan(database.Conn(), r.Organization.ID, paidSeats, time.Now().Add(10*24*time.Hour))
	for range members - 1 {
		support.CreateUser(t, r, r.Organization.ID)
	}
	now := time.Now()
	for i := range pendingInvitations {
		invitation := models.OrganizationInvitation{
			OrganizationID: r.Organization.ID, Email: support.RandomName("pending") + "@example.com",
			Role: models.RoleOrgOperator, InvitedBy: r.User, State: models.InvitationStatePending,
			SentAt: now.Add(time.Duration(i) * time.Second), ExpiresAt: now.Add(time.Hour),
		}
		require.NoError(t, invitation.Save(database.Conn()))
	}
	return state
}

// storeInvalidPaidSeats stores paid seats that need repair. Polar answers
// with an older snapshot, so the scheduler keeps the stored value.
func (s renewalEstimateState) storeInvalidPaidSeats(t *testing.T, seats any) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, database.Conn().Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", s.r.Organization.ID).
		Updates(map[string]any{"purchased_seats": seats, "polar_modified_at": now}).Error)
	s.polar.Update(s.sub.ID, func(sub *polarfake.Subscription) { sub.ModifiedAt = new(now.Add(-time.Hour)) })
}
