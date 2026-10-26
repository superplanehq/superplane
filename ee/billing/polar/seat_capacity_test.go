package polar

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
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
	catalog := billingmodels.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public",
		PerSeat: true, SeatLimit: new(50), PriceCents: 2000, PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", org.ID).
		Updates(map[string]any{
			"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
			"polar_subscription_id":     "sub_" + uuid.NewString(),
			"polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
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
	assert.Equal(t, billingmodels.SeatCapacityUnchanged, checkA.Transition.Change)
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
	assert.Equal(t, billingmodels.SeatCapacityResolved, result.check.Transition.Change)
	assert.Equal(t, 2, result.check.Transition.Purchased)
	assert.Equal(t, 2, result.check.Transition.Occupied)
	require.NoError(t, txB.Commit().Error)

	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Nil(t, plan.OverCapacitySince)
}

// overCapacityTeam puts the organization on a Team subscription with two paid
// seats and four members, so a snapshot with four seats resolves the
// over-capacity state. It returns that snapshot.
func overCapacityTeam(t *testing.T, org *models.Organization) SubscriptionData {
	t.Helper()
	db := database.Conn()
	product := "prod_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public",
		PerSeat: true, SeatLimit: new(50), PriceCents: 2000, PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	now := time.Now().UTC().Truncate(time.Second)
	subscriptionID := "sub_" + uuid.NewString()
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", org.ID).
		Updates(map[string]any{
			"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
			"polar_subscription_id": subscriptionID, "polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
			"purchased_seats": 2, "current_period_start": now.AddDate(0, 0, -7), "current_period_end": now.AddDate(0, 0, 23),
			"polar_modified_at": now.Add(-time.Hour), "over_capacity_since": now.Add(-time.Hour),
		}).Error)
	for range 4 {
		account, err := models.CreateAccount("Member", uuid.NewString()+"@example.com")
		require.NoError(t, err)
		_, err = models.CreateUser(org.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)
	}
	return SubscriptionData{
		ID: subscriptionID, ProductID: product, Status: billingmodels.PolarSubscriptionStatusActive, Seats: new(4),
		CurrentPeriodStart: polarTime{Time: now.AddDate(0, 0, -7)}, CurrentPeriodEnd: polarTime{Time: now.AddDate(0, 0, 23)},
		ModifiedAt: polarTime{Time: now}, ExternalCustomerID: org.ID.String(),
	}
}

func capacityLogs(hook *logtest.Hook) []string {
	var messages []string
	for _, entry := range hook.AllEntries() {
		if entry.Message == logging.SeatsOverCapacityMessage || entry.Message == logging.SeatsOverCapacityResolvedMessage {
			messages = append(messages, entry.Message)
		}
	}
	return messages
}

func overCapacitySince(t *testing.T, orgID uuid.UUID) *time.Time {
	t.Helper()
	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), orgID)
	require.NoError(t, err)
	return plan.OverCapacitySince
}

func TestSubscriptionApplyEmitsOnlyCommittedCapacityTransitions(t *testing.T) {
	t.Run("rollback after a staged resolution", func(t *testing.T) {
		org := setupBillingOrganization(t)
		snapshot := overCapacityTeam(t, org)
		hook := logtest.NewGlobal()
		t.Cleanup(hook.Reset)

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			alerts, err := StageOrganizationSubscription(t.Context(), tx, org.ID, snapshot)
			require.NoError(t, err)
			require.Len(t, alerts, 1)
			assert.Equal(t, billingmodels.SeatCapacityResolved, alerts[0].Transition.Change)
			return errors.New("local settlement failed")
		})

		require.Error(t, err)
		assert.Empty(t, capacityLogs(hook), "a rolled back resolution is never reported")
		assert.NotNil(t, overCapacitySince(t, org.ID))
	})

	t.Run("savepoint completion inside an outer transaction", func(t *testing.T) {
		org := setupBillingOrganization(t)
		snapshot := overCapacityTeam(t, org)
		hook := logtest.NewGlobal()
		t.Cleanup(hook.Reset)

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			require.NoError(t, tx.Transaction(func(savepoint *gorm.DB) error {
				_, err := StageOrganizationSubscription(t.Context(), savepoint, org.ID, snapshot)
				return err
			}))
			assert.Empty(t, capacityLogs(hook), "a released savepoint is not a commit")
			return errors.New("outer transaction failed")
		})

		require.Error(t, err)
		assert.Empty(t, capacityLogs(hook))
		assert.NotNil(t, overCapacitySince(t, org.ID))
	})

	t.Run("top-level apply and replay", func(t *testing.T) {
		org := setupBillingOrganization(t)
		snapshot := overCapacityTeam(t, org)
		hook := logtest.NewGlobal()
		t.Cleanup(hook.Reset)

		require.NoError(t, ApplyOrganizationSubscription(t.Context(), database.Conn(), org.ID, snapshot))
		require.NoError(t, ApplyOrganizationSubscription(t.Context(), database.Conn(), org.ID, snapshot))

		assert.Equal(t, []string{logging.SeatsOverCapacityResolvedMessage}, capacityLogs(hook), "an unchanged replay reports nothing")
		assert.Nil(t, overCapacitySince(t, org.ID))
	})

	t.Run("top-level apply inside a caller transaction", func(t *testing.T) {
		org := setupBillingOrganization(t)
		snapshot := overCapacityTeam(t, org)
		hook := logtest.NewGlobal()
		t.Cleanup(hook.Reset)

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			return ApplyOrganizationSubscription(t.Context(), tx, org.ID, snapshot)
		})

		require.Error(t, err, "only the transaction owner can report capacity")
		assert.Empty(t, capacityLogs(hook))
		assert.NotNil(t, overCapacitySince(t, org.ID))
	})
}
