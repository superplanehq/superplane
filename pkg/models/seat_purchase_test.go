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

func Test__SeatPurchaseSucceedsOnlyFromEvidenceAboutThePurchase(t *testing.T) {
	now := time.Now()
	before, after := now.Add(-time.Hour), now.Add(time.Hour)
	periodStart := now.Add(-7 * 24 * time.Hour)
	renewed := now.Add(23 * 24 * time.Hour)
	seats := func(n int) *int { return &n }
	evidence := func(change func(e *models.SeatPurchaseEvidence)) models.SeatPurchaseEvidence {
		e := models.SeatPurchaseEvidence{SubscriptionID: "sub_team", PeriodStart: &periodStart, Seats: seats(5), ModifiedAt: &after}
		change(&e)
		return e
	}

	tests := []struct {
		name      string
		status    string
		evidence  models.SeatPurchaseEvidence
		succeeded bool
	}{
		{name: "response with the target", status: models.SeatPurchaseStatusPending, succeeded: true,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Response, e.ModifiedAt = true, nil })},
		{name: "snapshot with the target after the purchase", status: models.SeatPurchaseStatusUnknown, succeeded: true,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) {})},
		{name: "pending purchase and the target after the purchase", status: models.SeatPurchaseStatusPending, succeeded: true,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) {})},
		{name: "snapshot with the target before the purchase", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.ModifiedAt = &before })},
		{name: "snapshot with the target at the purchase time", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.ModifiedAt = &now })},
		{name: "snapshot without a modification time", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.ModifiedAt = nil })},
		{name: "above the target", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Seats = seats(6) })},
		{name: "original seats after the purchase", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Seats = seats(3) })},
		{name: "between the two counts", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Seats = seats(4) })},
		{name: "response with the original seats", status: models.SeatPurchaseStatusPending,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Response, e.Seats = true, seats(3) })},
		{name: "no seats", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Seats = nil })},
		{name: "another subscription", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.SubscriptionID = "sub_other" })},
		{name: "response from another subscription", status: models.SeatPurchaseStatusPending,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Response, e.SubscriptionID = true, "sub_other" })},
		{name: "target in a later period", status: models.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.PeriodStart = &renewed })},
		{name: "response in a later period", status: models.SeatPurchaseStatusPending,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Response, e.PeriodStart = true, &renewed })},
		{name: "settled purchase", status: models.SeatPurchaseStatusFailed,
			evidence: evidence(func(e *models.SeatPurchaseEvidence) { e.Response = true })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purchase := models.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 5, &models.SeatPurchaseOperation{
				Key: "key", Fingerprint: models.SeatPurchaseFingerprint(2, 3), PeriodStart: periodStart, PeriodEnd: renewed,
			})
			purchase.Status, purchase.CreatedAt = tt.status, now

			assert.Equal(t, tt.succeeded, purchase.SucceededBy(tt.evidence))
		})
	}
}

// Rows from before operation keys have no stored period. Only a snapshot
// whose period started before the purchase can be evidence for them.
func Test__SeatPurchaseWithoutAStoredPeriodSucceedsOnlyFromItsCreationPeriod(t *testing.T) {
	now := time.Now()
	after := now.Add(time.Minute)
	target := 5
	creationPeriod := now.Add(-7 * 24 * time.Hour)
	renewed := now.Add(time.Second)
	purchase := models.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 5, nil)
	purchase.Status, purchase.CreatedAt = models.SeatPurchaseStatusUnknown, now
	evidence := func(periodStart *time.Time) models.SeatPurchaseEvidence {
		return models.SeatPurchaseEvidence{SubscriptionID: "sub_team", PeriodStart: periodStart, Seats: &target, ModifiedAt: &after}
	}

	assert.True(t, purchase.SucceededBy(evidence(&creationPeriod)))
	assert.False(t, purchase.SucceededBy(evidence(&renewed)), "success needs a snapshot from the creation period")
	assert.False(t, purchase.SucceededBy(evidence(nil)), "success needs a snapshot with a period")
}

func Test__SettleSeatPurchaseKeepsAnUnprovedOutcomeAndAProvedOne(t *testing.T) {
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
	settle := func(seats int, periodStart time.Time, modifiedAt time.Time) {
		t.Helper()
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return models.SettleSeatPurchase(tx, orgID, models.SeatPurchaseEvidence{
				SubscriptionID: "sub_team", PeriodStart: &periodStart, Seats: &seats, ModifiedAt: &modifiedAt,
			})
		}))
		require.NoError(t, db.First(purchase, "id = ?", purchase.ID).Error)
	}
	later := time.Now().Add(time.Minute)

	settle(3, periodStart, later)
	assert.Equal(t, models.SeatPurchaseStatusUnknown, purchase.Status, "the original seats after the purchase prove nothing")
	settle(4, periodEnd, later)
	assert.Equal(t, models.SeatPurchaseStatusUnknown, purchase.Status, "the target in a renewed period proves nothing")

	settle(4, periodStart, later)
	require.Equal(t, models.SeatPurchaseStatusSucceeded, purchase.Status)

	for _, event := range []struct {
		name       string
		seats      int
		modifiedAt time.Time
	}{
		{name: "duplicate", seats: 4, modifiedAt: later},
		{name: "reordered", seats: 3, modifiedAt: later.Add(-30 * time.Second)},
		{name: "equal-timestamp", seats: 3, modifiedAt: later},
		{name: "newer", seats: 3, modifiedAt: later.Add(time.Minute)},
	} {
		settle(event.seats, periodStart, event.modifiedAt)
		assert.Equal(t, models.SeatPurchaseStatusSucceeded, purchase.Status, "a %s event keeps the success", event.name)
		assert.Empty(t, purchase.ErrorCode)
	}
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
	assert.True(t, stored.MatchesRequest(models.SeatPurchaseFingerprints{Counts: models.SeatPurchaseFingerprint(1, 3)}))
	assert.False(t, stored.MatchesRequest(models.SeatPurchaseFingerprints{Counts: models.SeatPurchaseFingerprint(2, 3)}))
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

func Test__SeatPurchaseBlocksRenewalMaintenanceUntilVerifiedEvidenceReleasesIt(t *testing.T) {
	const window = 5 * time.Minute
	created := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	periodStart := created.Add(-10 * 24 * time.Hour)
	periodEnd := created.Add(20 * 24 * time.Hour)
	afterWindow := created.Add(window + time.Second)
	keyed := func(status string) *models.SeatPurchase {
		// A new purchase stores the bound subscription and period.
		purchase := models.NewSeatPurchase(uuid.New(), nil, "sub_team", 9, 10, &models.SeatPurchaseOperation{
			Key: "key", Fingerprint: models.SeatPurchaseBoundFingerprint(1, 9, "sub_team", periodStart, periodEnd),
			PeriodStart: periodStart, PeriodEnd: periodEnd,
		})
		purchase.Status, purchase.CreatedAt = status, created
		return purchase
	}
	legacy := func() *models.SeatPurchase {
		purchase := models.NewSeatPurchase(uuid.New(), nil, "sub_team", 9, 10, nil)
		purchase.Status, purchase.CreatedAt = models.SeatPurchaseStatusUnknown, created
		return purchase
	}

	tests := []struct {
		name           string
		purchase       *models.SeatPurchase
		subscriptionID string
		periodStart    *time.Time
		now            time.Time
		blocks         bool
	}{
		{name: "later period of the purchase subscription", purchase: keyed(models.SeatPurchaseStatusUnknown), subscriptionID: "sub_team", periodStart: &periodEnd, now: afterWindow},
		{name: "pending purchase in a later period", purchase: keyed(models.SeatPurchaseStatusPending), subscriptionID: "sub_team", periodStart: &periodEnd, now: afterWindow},
		{name: "replacement subscription", purchase: keyed(models.SeatPurchaseStatusUnknown), subscriptionID: "sub_new", periodStart: &periodStart, now: afterWindow},
		{name: "purchase period", purchase: keyed(models.SeatPurchaseStatusUnknown), subscriptionID: "sub_team", periodStart: &periodStart, now: afterWindow, blocks: true},
		{name: "period that starts before the purchase period ends", purchase: keyed(models.SeatPurchaseStatusUnknown), subscriptionID: "sub_team", periodStart: new(periodEnd.Add(-time.Second)), now: afterWindow, blocks: true},
		{name: "submission can still reach Polar", purchase: keyed(models.SeatPurchaseStatusUnknown), subscriptionID: "sub_team", periodStart: &periodEnd, now: created.Add(window), blocks: true},
		{name: "replacement subscription inside the submission window", purchase: keyed(models.SeatPurchaseStatusPending), subscriptionID: "sub_new", periodStart: &periodStart, now: created.Add(window), blocks: true},
		{name: "no verified period", purchase: keyed(models.SeatPurchaseStatusUnknown), subscriptionID: "sub_team", now: afterWindow.Add(90 * 24 * time.Hour), blocks: true},
		{name: "no verified subscription", purchase: keyed(models.SeatPurchaseStatusUnknown), periodStart: &periodEnd, now: afterWindow, blocks: true},
		{name: "legacy purchase in a period that started after its window", purchase: legacy(), subscriptionID: "sub_team", periodStart: new(created.Add(window + time.Second)), now: afterWindow},
		{name: "legacy purchase in a period that started at its window end", purchase: legacy(), subscriptionID: "sub_team", periodStart: new(created.Add(window)), now: afterWindow, blocks: true},
		{name: "legacy purchase in its creation period", purchase: legacy(), subscriptionID: "sub_team", periodStart: &periodStart, now: afterWindow, blocks: true},
		{name: "settled purchase", purchase: keyed(models.SeatPurchaseStatusFailed), subscriptionID: "sub_team", periodStart: &periodStart, now: created},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := *tt.purchase

			assert.Equal(t, tt.blocks, tt.purchase.BlocksRenewalMaintenance(tt.subscriptionID, tt.periodStart, tt.now, window))
			assert.Equal(t, before, *tt.purchase, "the release rule changes no purchase state")
		})
	}
}

func Test__ListUnresolvedSeatPurchasesCreatedBeforeFindsAgedPurchasesOfAllOrganizations(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now().UTC().Truncate(time.Second)
	createdAt := func(orgID uuid.UUID, status string, age time.Duration) *models.SeatPurchase {
		purchase := models.NewSeatPurchase(orgID, nil, "sub_"+uuid.NewString(), 3, 4, nil)
		purchase.Status, purchase.CreatedAt = status, now.Add(-age)
		require.NoError(t, purchase.Create(db))
		return purchase
	}
	newOrganization := func() uuid.UUID {
		org, err := models.CreateOrganization("org-"+uuid.NewString()[:8], "")
		require.NoError(t, err)
		return org.ID
	}

	unknown := createdAt(r.Organization.ID, models.SeatPurchaseStatusUnknown, 2*time.Hour)
	pending := createdAt(newOrganization(), models.SeatPurchaseStatusPending, 3*time.Hour)
	createdAt(newOrganization(), models.SeatPurchaseStatusUnknown, 30*time.Minute)
	createdAt(newOrganization(), models.SeatPurchaseStatusSucceeded, 4*time.Hour)
	createdAt(newOrganization(), models.SeatPurchaseStatusFailed, 4*time.Hour)
	deleted := newOrganization()
	createdAt(deleted, models.SeatPurchaseStatusUnknown, 5*time.Hour)
	require.NoError(t, db.Model(&models.Organization{}).Where("id = ?", deleted).Update("deleted_at", now).Error)

	purchases, err := models.ListUnresolvedSeatPurchasesCreatedBefore(db, now.Add(-time.Hour))

	require.NoError(t, err)
	require.Len(t, purchases, 2)
	assert.Equal(t, pending.ID, purchases[0].ID, "the oldest purchase comes first")
	assert.Equal(t, unknown.ID, purchases[1].ID)
}

func Test__SeatPurchaseMatchesRequestByTheStoredFingerprintVersion(t *testing.T) {
	periodStart := time.Now().Add(-7 * 24 * time.Hour).UTC().Truncate(time.Second)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	request := func(additional, expected int, subscriptionID string, start, end time.Time) models.SeatPurchaseFingerprints {
		return models.SeatPurchaseFingerprints{
			Counts: models.SeatPurchaseFingerprint(additional, expected),
			Bound:  models.SeatPurchaseBoundFingerprint(additional, expected, subscriptionID, start, end),
		}
	}
	unbound := func(additional, expected int) models.SeatPurchaseFingerprints {
		return models.SeatPurchaseFingerprints{Counts: models.SeatPurchaseFingerprint(additional, expected)}
	}
	stored := func(fingerprint string) *models.SeatPurchase {
		return models.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 4, &models.SeatPurchaseOperation{
			Key: "key", Fingerprint: fingerprint, PeriodStart: periodStart, PeriodEnd: periodEnd,
		})
	}
	bound := stored(models.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart, periodEnd))
	legacy := stored(models.SeatPurchaseFingerprint(1, 3))

	tests := []struct {
		name     string
		purchase *models.SeatPurchase
		request  models.SeatPurchaseFingerprints
		matches  bool
	}{
		{name: "bound purchase and the same binding", purchase: bound, request: request(1, 3, "sub_team", periodStart, periodEnd), matches: true},
		{name: "bound purchase and no binding", purchase: bound, request: unbound(1, 3)},
		{name: "bound purchase and another subscription", purchase: bound, request: request(1, 3, "sub_new", periodStart, periodEnd)},
		{name: "bound purchase and another period start", purchase: bound, request: request(1, 3, "sub_team", periodEnd, periodEnd)},
		{name: "bound purchase and another period end", purchase: bound, request: request(1, 3, "sub_team", periodStart, periodEnd.Add(time.Second))},
		{name: "bound purchase and another quantity", purchase: bound, request: request(2, 3, "sub_team", periodStart, periodEnd)},
		{name: "legacy purchase and no binding", purchase: legacy, request: unbound(1, 3), matches: true},
		{name: "legacy purchase and a binding", purchase: legacy, request: request(1, 3, "sub_new", periodEnd, periodEnd), matches: true},
		{name: "legacy purchase and another quantity", purchase: legacy, request: request(2, 3, "sub_team", periodStart, periodEnd)},
		{name: "purchase without an operation", purchase: models.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 4, nil), request: unbound(1, 3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.matches, tt.purchase.MatchesRequest(tt.request))
		})
	}
}

func Test__SeatPurchaseBoundFingerprintIsStableForEqualInstants(t *testing.T) {
	periodStart := time.Now().Add(-7 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	elsewhere := time.FixedZone("UTC+9", 9*60*60)

	utc := models.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart, periodEnd)

	assert.Equal(t, utc, models.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart.In(elsewhere), periodEnd.In(elsewhere)))
	assert.NotEqual(t, utc, models.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart.Add(time.Microsecond), periodEnd))
	assert.NotEqual(t, utc, models.SeatPurchaseFingerprint(1, 3), "the v2 fingerprint never equals the v1 fingerprint")
}
