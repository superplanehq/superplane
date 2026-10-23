package models_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func createSeatPurchase(t *testing.T, orgID uuid.UUID, subscriptionID string, from, to int, status string) *models.SeatPurchase {
	t.Helper()
	purchase := models.NewSeatPurchase(orgID, nil, subscriptionID, from, to)
	purchase.Status = status
	require.NoError(t, purchase.Create(database.Conn()))
	return purchase
}

func Test__FindExpectedSeatCountReturnsTheLatestExpectedTarget(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID

	expected, err := models.FindExpectedSeatCount(db, orgID, "sub_team")
	require.NoError(t, err)
	assert.Nil(t, expected)

	createSeatPurchase(t, orgID, "sub_team", 3, 5, models.SeatPurchaseStatusSucceeded)
	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team")
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, 5, *expected)

	createSeatPurchase(t, orgID, "sub_team", 5, 9, models.SeatPurchaseStatusFailed)
	createSeatPurchase(t, orgID, "sub_team", 5, 8, models.SeatPurchaseStatusActionRequired)
	createSeatPurchase(t, orgID, "sub_other", 5, 12, models.SeatPurchaseStatusSucceeded)
	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team")
	require.NoError(t, err)
	assert.Equal(t, 5, *expected, "failed and action required purchases add no expectation")

	createSeatPurchase(t, orgID, "sub_team", 5, 7, models.SeatPurchaseStatusUnknown)
	expected, err = models.FindExpectedSeatCount(db, orgID, "sub_team")
	require.NoError(t, err)
	assert.Equal(t, 7, *expected, "an in-flight purchase is the newest expectation")
}

func Test__SeatPurchaseAllowsOneInFlightPurchasePerOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID

	first := createSeatPurchase(t, orgID, "sub_team", 3, 5, models.SeatPurchaseStatusPending)
	require.Error(t, models.NewSeatPurchase(orgID, nil, "sub_team", 3, 6).Create(db))

	require.NoError(t, first.UpdateStatus(db, models.SeatPurchaseStatusUnknown, ""))
	require.Error(t, models.NewSeatPurchase(orgID, nil, "sub_team", 3, 6).Create(db))

	require.NoError(t, first.UpdateStatus(db, models.SeatPurchaseStatusFailed, models.SeatPurchaseErrorPaymentFailed))
	require.NoError(t, models.NewSeatPurchase(orgID, nil, "sub_team", 3, 6).Create(db))

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
	seats := func(n int) *int { return &n }

	tests := []struct {
		name       string
		status     string
		seats      *int
		modifiedAt *time.Time
		want       string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purchase := &models.SeatPurchase{FromSeats: 3, ToSeats: 5, Status: tt.status, CreatedAt: now}
			got, settled := purchase.SettlementFor(tt.seats, tt.modifiedAt)
			assert.Equal(t, tt.want != "", settled)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test__SeatPurchaseDownMigrationDropsTheTable(t *testing.T) {
	up, err := os.ReadFile("../../db/migrations/20261023195812_add-organization-seat-purchases.up.sql")
	require.NoError(t, err)
	down, err := os.ReadFile("../../db/migrations/20261023195812_add-organization-seat-purchases.down.sql")
	require.NoError(t, err)

	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	tableExists := func() bool {
		var exists bool
		require.NoError(t, tx.Raw(`SELECT to_regclass('organization_seat_purchases') IS NOT NULL`).Scan(&exists).Error)
		return exists
	}

	require.True(t, tableExists(), "the test database is migrated")
	require.NoError(t, tx.Exec(string(down)).Error)
	assert.False(t, tableExists())
	require.NoError(t, tx.Exec(string(up)).Error)
	assert.True(t, tableExists(), "the up migration recreates the table after a rollback")
}
