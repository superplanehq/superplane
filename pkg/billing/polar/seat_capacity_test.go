package polar

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// Two concurrent removals can each count three members when the count runs
// before the billing plan lock, because neither transaction sees the uncommitted
// deletion of the other. Both checks would keep the over-capacity record after
// two members remain. The count must run under the lock, so the second check
// sees the committed removals of the first one and records the resolution.
func TestRecheckSeatCapacityCountsMembersUnderThePlanLock(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	product := "prod_" + uuid.NewString()
	catalog := models.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public",
		PerSeat: true, SeatLimit: new(50), PriceCents: 2000, PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, models.SaveBillingPlan(db, &catalog))
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", org.ID).
		Updates(map[string]any{
			"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": models.BillingPlanSourcePolar,
			"polar_subscription_id":     "sub_" + uuid.NewString(),
			"polar_subscription_status": models.PolarSubscriptionStatusActive,
			"purchased_seats":           2,
			"over_capacity_since":       time.Now().Add(-time.Hour),
		}).Error)

	// Four members occupy two paid seats.
	var members []*models.User
	for range 4 {
		account, err := models.CreateAccount("Member", uuid.NewString()+"@example.com")
		require.NoError(t, err)
		user, err := models.CreateUser(org.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)
		members = append(members, user)
	}

	// Two overlapping transactions remove one member each.
	txA := db.Begin()
	require.NoError(t, txA.Error)
	t.Cleanup(func() { txA.Rollback() })
	txB := db.Begin()
	require.NoError(t, txB.Error)
	t.Cleanup(func() { txB.Rollback() })
	require.NoError(t, members[0].SoftDelete(txA, time.Now(), ""))
	require.NoError(t, members[1].SoftDelete(txB, time.Now(), ""))

	// The first check takes the plan lock and still counts three members.
	checkA, err := RecheckSeatCapacity(txA, org.ID, SeatCapacityCauseRemoval)
	require.NoError(t, err)
	assert.Equal(t, models.SeatCapacityUnchanged, checkA.Transition.Change)
	assert.Equal(t, 3, checkA.Transition.Occupied)

	// The second check waits for the lock, so it cannot finish before the
	// first transaction commits.
	type outcome struct {
		check SeatCapacityCheck
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		check, err := RecheckSeatCapacity(txB, org.ID, SeatCapacityCauseRemoval)
		done <- outcome{check, err}
	}()
	select {
	case <-done:
		t.Fatal("the second check finished while the first transaction held the billing plan lock")
	case <-time.After(200 * time.Millisecond):
	}

	require.NoError(t, txA.Commit().Error)
	var result outcome
	select {
	case result = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the second check did not finish after the first transaction committed")
	}
	require.NoError(t, result.err)
	assert.Equal(t, models.SeatCapacityResolved, result.check.Transition.Change)
	assert.Equal(t, 2, result.check.Transition.Purchased)
	assert.Equal(t, 2, result.check.Transition.Occupied)
	require.NoError(t, txB.Commit().Error)

	plan, err := models.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Nil(t, plan.OverCapacitySince)
}
