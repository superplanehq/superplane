package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func createSeatPurchase(t *testing.T, orgID uuid.UUID, subscriptionID string, from, to int, status string) *models.SeatPurchase {
	t.Helper()
	purchase := models.NewSeatPurchase(orgID, nil, subscriptionID, from, to, nil)
	purchase.Status = status
	require.NoError(t, purchase.Create(database.Conn()))
	return purchase
}

func Test__FindExpectedSeatCountReturnsTheLatestExpectedTarget(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID
	periodStart := time.Now().Add(-time.Hour)

	expected, err := models.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	assert.Nil(t, expected)

	createSeatPurchase(t, orgID, "sub_team", 3, 5, models.SeatPurchaseStatusSucceeded)
	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, 5, *expected)

	createSeatPurchase(t, orgID, "sub_team", 5, 9, models.SeatPurchaseStatusFailed)
	createSeatPurchase(t, orgID, "sub_team", 5, 8, models.SeatPurchaseStatusActionRequired)
	createSeatPurchase(t, orgID, "sub_other", 5, 12, models.SeatPurchaseStatusSucceeded)
	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	assert.Equal(t, 5, *expected, "failed and action required purchases add no expectation")

	createSeatPurchase(t, orgID, "sub_team", 5, 7, models.SeatPurchaseStatusUnknown)
	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	assert.Equal(t, 7, *expected, "an in-flight purchase is the newest expectation")
}

func Test__SeatPurchaseAllowsOneInFlightPurchasePerOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID

	first := createSeatPurchase(t, orgID, "sub_team", 3, 5, models.SeatPurchaseStatusPending)
	require.Error(t, models.NewSeatPurchase(orgID, nil, "sub_team", 3, 6, nil).Create(db))

	require.NoError(t, first.UpdateStatus(db, models.SeatPurchaseStatusUnknown, ""))
	require.Error(t, models.NewSeatPurchase(orgID, nil, "sub_team", 3, 6, nil).Create(db))

	require.NoError(t, first.UpdateStatus(db, models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorPaymentFailed))
	require.NoError(t, models.NewSeatPurchase(orgID, nil, "sub_team", 3, 6, nil).Create(db))

	inFlight, err := models.FindInFlightSeatPurchase(db, orgID)
	require.NoError(t, err)
	require.NotNil(t, inFlight)
	assert.Equal(t, 6, inFlight.ToSeats)
}

func Test__SeatPurchaseUpdateStatusKeepsASettledOutcome(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	purchase := createSeatPurchase(t, r.Organization.ID, "sub_team", 3, 5, models.SeatPurchaseStatusPending)

	late := *purchase
	require.NoError(t, purchase.UpdateStatus(db, models.SeatPurchaseStatusSucceeded, ""))
	require.NoError(t, late.UpdateStatus(db, models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorRejected))

	assert.Equal(t, models.SeatPurchaseStatusSucceeded, late.Status, "a late result reloads the settled purchase")
	assert.Empty(t, late.ErrorCode)
}

func Test__SeatPurchaseSettlementFollowsPolarSnapshots(t *testing.T) {
	now := time.Now()
	before, after := now.Add(-time.Hour), now.Add(time.Hour)
	periodStart := now.Add(-7 * 24 * time.Hour)
	renewed := now.Add(23 * 24 * time.Hour)
	seats := func(n int) *int { return &n }

	tests := []struct {
		name       string
		status     string
		seats      *int
		modifiedAt *time.Time
		// periodStart is the period of the snapshot. Nil means the period
		// in which the purchase started.
		periodStart *time.Time
		want        string
	}{
		{name: "target reached while pending", status: models.SeatPurchaseStatusPending, seats: seats(5), want: models.SeatPurchaseStatusSucceeded},
		{name: "above target while unknown", status: models.SeatPurchaseStatusUnknown, seats: seats(6), want: models.SeatPurchaseStatusSucceeded},
		{name: "unchanged after the purchase", status: models.SeatPurchaseStatusUnknown, seats: seats(3), modifiedAt: &after, want: models.SeatPurchaseStatusFailed},
		{name: "unchanged before the purchase", status: models.SeatPurchaseStatusUnknown, seats: seats(3), modifiedAt: &before},
		{name: "unchanged without a modification time", status: models.SeatPurchaseStatusUnknown, seats: seats(3)},
		{name: "pending request can still arrive", status: models.SeatPurchaseStatusPending, seats: seats(3), modifiedAt: &after},
		{name: "between the two counts", status: models.SeatPurchaseStatusUnknown, seats: seats(4), modifiedAt: &after},
		{name: "snapshot without seats", status: models.SeatPurchaseStatusUnknown, modifiedAt: &after},
		{name: "settled purchase", status: models.SeatPurchaseStatusFailed, seats: seats(5)},
		{name: "target reached in a later period", status: models.SeatPurchaseStatusUnknown, seats: seats(5), modifiedAt: &after, periodStart: &renewed},
		{name: "unchanged in a later period", status: models.SeatPurchaseStatusUnknown, seats: seats(3), modifiedAt: &after, periodStart: &renewed},
		{name: "pending target reached in a later period", status: models.SeatPurchaseStatusPending, seats: seats(5), periodStart: &renewed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purchase := models.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 5, &models.SeatPurchaseOperation{
				Key: "key", Fingerprint: models.SeatPurchaseFingerprint(2, 3), PeriodStart: periodStart, PeriodEnd: renewed,
			})
			purchase.Status, purchase.CreatedAt = tt.status, now
			snapshotPeriod := &periodStart
			if tt.periodStart != nil {
				snapshotPeriod = tt.periodStart
			}
			got, settled := purchase.SettlementFor(snapshotPeriod, tt.seats, tt.modifiedAt)
			assert.Equal(t, tt.want != "", settled)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Rows from before operation keys have no stored period. Only a snapshot
// whose period started before the purchase can be evidence for them.
func Test__SeatPurchaseWithoutAStoredPeriodSettlesOnlyFromItsCreationPeriod(t *testing.T) {
	now := time.Now()
	after := now.Add(time.Minute)
	seats := func(n int) *int { return &n }
	creationPeriod := now.Add(-7 * 24 * time.Hour)
	renewed := now.Add(time.Second)
	purchase := models.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 5, nil)
	purchase.Status, purchase.CreatedAt = models.SeatPurchaseStatusUnknown, now

	status, settled := purchase.SettlementFor(&creationPeriod, seats(5), &after)
	assert.True(t, settled)
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, status)

	for _, period := range []*time.Time{&renewed, nil} {
		_, settled = purchase.SettlementFor(period, seats(5), &after)
		assert.False(t, settled, "success needs a snapshot from the creation period")
		_, settled = purchase.SettlementFor(period, seats(3), &after)
		assert.False(t, settled, "failure needs a snapshot from the creation period")
	}
}

func Test__SettleSeatPurchaseIgnoresSnapshotsFromAnotherPeriod(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID
	periodStart := time.Now().Add(-7 * 24 * time.Hour).UTC().Truncate(time.Second)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	purchase := models.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, &models.SeatPurchaseOperation{
		Key: "key-1", Fingerprint: models.SeatPurchaseFingerprint(1, 3), PeriodStart: periodStart, PeriodEnd: periodEnd,
	})
	purchase.Status = models.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))
	later := time.Now().Add(time.Minute)

	for _, seats := range []int{3, 4, 10} {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return models.SettleSeatPurchase(tx, orgID, "sub_team", &periodEnd, &seats, &later)
		}))
		require.NoError(t, db.First(purchase, "id = ?", purchase.ID).Error)
		assert.Equal(t, models.SeatPurchaseStatusUnknown, purchase.Status, "a renewed period with %d seats proves nothing", seats)
	}

	seats := 3
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return models.SettleSeatPurchase(tx, orgID, "sub_team", &periodStart, &seats, &later)
	}))
	require.NoError(t, db.First(purchase, "id = ?", purchase.ID).Error)
	assert.Equal(t, models.SeatPurchaseStatusFailed, purchase.Status, "a snapshot from the purchase period settles it")
	assert.Equal(t, models.SeatPurchaseErrorNotApplied, purchase.ErrorCode)
}

func Test__SeatPurchaseOperationKeyIsUniqueWithinTheOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID
	other, err := models.CreateOrganization("other-"+uuid.NewString()[:8], "")
	require.NoError(t, err)
	periodStart := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	operation := &models.SeatPurchaseOperation{
		Key: "key-1", Fingerprint: models.SeatPurchaseFingerprint(1, 3),
		PeriodStart: periodStart, PeriodEnd: periodStart.Add(30 * 24 * time.Hour),
	}

	first := models.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, operation)
	first.Status = models.SeatPurchaseStatusFailed
	require.NoError(t, first.Create(db))

	err = models.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, operation).Create(db)
	assert.True(t, models.IsSeatPurchaseOperationKeyConflict(err), "the same key cannot record a second purchase: %v", err)
	require.NoError(t, models.NewSeatPurchase(other.ID, nil, "sub_other", 3, 4, operation).Create(db), "keys are scoped to the organization")

	stored, err := models.FindSeatPurchaseByOperationKey(db, orgID, "key-1")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, first.ID, stored.ID)
	assert.True(t, stored.MatchesRequest(models.SeatPurchaseFingerprint(1, 3)))
	assert.False(t, stored.MatchesRequest(models.SeatPurchaseFingerprint(2, 3)))
	require.NotNil(t, stored.PeriodStart)
	assert.True(t, stored.PeriodStart.Equal(operation.PeriodStart))

	missing, err := models.FindSeatPurchaseByOperationKey(db, orgID, "key-2")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func Test__FindExpectedSeatCountIgnoresPurchasesFromAnEarlierPeriod(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID
	previous := time.Now().Add(-31 * 24 * time.Hour).UTC().Truncate(time.Second)
	current := previous.Add(30 * 24 * time.Hour)
	purchase := models.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, &models.SeatPurchaseOperation{
		Key: "key-1", Fingerprint: models.SeatPurchaseFingerprint(1, 3), PeriodStart: previous, PeriodEnd: current,
	})
	purchase.Status = models.SeatPurchaseStatusSucceeded
	require.NoError(t, purchase.Create(db))

	expected, err := models.FindExpectedSeatCount(db, orgID, "sub_team", previous)
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, 4, *expected)

	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team", current)
	require.NoError(t, err)
	assert.Nil(t, expected, "a renewal reduction back to 3 seats is not compared with the old target")
}
