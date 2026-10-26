package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func createSeatPurchase(t *testing.T, orgID uuid.UUID, subscriptionID string, from, to int, status string) *billingmodels.SeatPurchase {
	t.Helper()
	purchase := billingmodels.NewSeatPurchase(orgID, nil, subscriptionID, from, to, nil)
	purchase.Status = status
	require.NoError(t, purchase.Create(database.Conn()))
	return purchase
}

func Test__FindExpectedSeatCountReturnsTheLatestExpectedTarget(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID
	periodStart := time.Now().Add(-time.Hour)

	expected, err := billingmodels.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	assert.Nil(t, expected)

	createSeatPurchase(t, orgID, "sub_team", 3, 5, billingmodels.SeatPurchaseStatusSucceeded)
	expected, err = billingmodels.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, 5, *expected)

	createSeatPurchase(t, orgID, "sub_team", 5, 9, billingmodels.SeatPurchaseStatusFailed)
	createSeatPurchase(t, orgID, "sub_team", 5, 8, billingmodels.SeatPurchaseStatusActionRequired)
	createSeatPurchase(t, orgID, "sub_other", 5, 12, billingmodels.SeatPurchaseStatusSucceeded)
	expected, err = billingmodels.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	assert.Equal(t, 5, *expected, "failed and action required purchases add no expectation")

	createSeatPurchase(t, orgID, "sub_team", 5, 7, billingmodels.SeatPurchaseStatusUnknown)
	expected, err = billingmodels.FindExpectedSeatCount(db, orgID, "sub_team", periodStart)
	require.NoError(t, err)
	assert.Equal(t, 7, *expected, "an in-flight purchase is the newest expectation")
}

func Test__SeatPurchaseAllowsOneInFlightPurchasePerOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID

	first := createSeatPurchase(t, orgID, "sub_team", 3, 5, billingmodels.SeatPurchaseStatusPending)
	require.Error(t, billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 6, nil).Create(db))

	require.NoError(t, first.UpdateStatus(db, billingmodels.SeatPurchaseStatusUnknown, ""))
	require.Error(t, billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 6, nil).Create(db))

	require.NoError(t, first.UpdateStatus(db, billingmodels.SeatPurchaseStatusFailed, billingmodels.SeatPurchaseErrorPaymentFailed))
	require.NoError(t, billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 6, nil).Create(db))

	inFlight, err := billingmodels.FindInFlightSeatPurchase(db, orgID)
	require.NoError(t, err)
	require.NotNil(t, inFlight)
	assert.Equal(t, 6, inFlight.ToSeats)
}

func Test__SeatPurchaseUpdateStatusKeepsASettledOutcome(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	purchase := createSeatPurchase(t, r.Organization.ID, "sub_team", 3, 5, billingmodels.SeatPurchaseStatusPending)

	late := *purchase
	require.NoError(t, purchase.UpdateStatus(db, billingmodels.SeatPurchaseStatusSucceeded, ""))
	require.NoError(t, late.UpdateStatus(db, billingmodels.SeatPurchaseStatusFailed, billingmodels.SeatPurchaseErrorRejected))

	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, late.Status, "a late result reloads the settled purchase")
	assert.Empty(t, late.ErrorCode)
}

func Test__SeatPurchaseSucceedsOnlyFromEvidenceAboutThePurchase(t *testing.T) {
	now := time.Now()
	before, after := now.Add(-time.Hour), now.Add(time.Hour)
	periodStart := now.Add(-7 * 24 * time.Hour)
	renewed := now.Add(23 * 24 * time.Hour)
	seats := func(n int) *int { return &n }
	evidence := func(change func(e *billingmodels.SeatPurchaseEvidence)) billingmodels.SeatPurchaseEvidence {
		e := billingmodels.SeatPurchaseEvidence{SubscriptionID: "sub_team", PeriodStart: &periodStart, Seats: seats(5), ModifiedAt: &after}
		change(&e)
		return e
	}

	tests := []struct {
		name      string
		status    string
		evidence  billingmodels.SeatPurchaseEvidence
		succeeded bool
	}{
		{name: "response with the target", status: billingmodels.SeatPurchaseStatusPending, succeeded: true,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Response, e.ModifiedAt = true, nil })},
		{name: "snapshot with the target after the purchase", status: billingmodels.SeatPurchaseStatusUnknown, succeeded: true,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) {})},
		{name: "pending purchase and the target after the purchase", status: billingmodels.SeatPurchaseStatusPending, succeeded: true,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) {})},
		{name: "snapshot with the target before the purchase", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.ModifiedAt = &before })},
		{name: "snapshot with the target at the purchase time", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.ModifiedAt = &now })},
		{name: "snapshot without a modification time", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.ModifiedAt = nil })},
		{name: "above the target", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Seats = seats(6) })},
		{name: "original seats after the purchase", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Seats = seats(3) })},
		{name: "between the two counts", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Seats = seats(4) })},
		{name: "response with the original seats", status: billingmodels.SeatPurchaseStatusPending,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Response, e.Seats = true, seats(3) })},
		{name: "no seats", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Seats = nil })},
		{name: "another subscription", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.SubscriptionID = "sub_other" })},
		{name: "response from another subscription", status: billingmodels.SeatPurchaseStatusPending,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Response, e.SubscriptionID = true, "sub_other" })},
		{name: "target in a later period", status: billingmodels.SeatPurchaseStatusUnknown,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.PeriodStart = &renewed })},
		{name: "response in a later period", status: billingmodels.SeatPurchaseStatusPending,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Response, e.PeriodStart = true, &renewed })},
		{name: "settled purchase", status: billingmodels.SeatPurchaseStatusFailed,
			evidence: evidence(func(e *billingmodels.SeatPurchaseEvidence) { e.Response = true })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purchase := billingmodels.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 5, &billingmodels.SeatPurchaseOperation{
				Key: "key", Fingerprint: billingmodels.SeatPurchaseFingerprint(2, 3), PeriodStart: periodStart, PeriodEnd: renewed,
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
	purchase := billingmodels.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 5, nil)
	purchase.Status, purchase.CreatedAt = billingmodels.SeatPurchaseStatusUnknown, now
	evidence := func(periodStart *time.Time) billingmodels.SeatPurchaseEvidence {
		return billingmodels.SeatPurchaseEvidence{SubscriptionID: "sub_team", PeriodStart: periodStart, Seats: &target, ModifiedAt: &after}
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
	purchase := billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, &billingmodels.SeatPurchaseOperation{
		Key: "key-1", Fingerprint: billingmodels.SeatPurchaseFingerprint(1, 3), PeriodStart: periodStart, PeriodEnd: periodEnd,
	})
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(db))
	settle := func(seats int, periodStart time.Time, modifiedAt time.Time) {
		t.Helper()
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return billingmodels.SettleSeatPurchase(tx, orgID, billingmodels.SeatPurchaseEvidence{
				SubscriptionID: "sub_team", PeriodStart: &periodStart, Seats: &seats, ModifiedAt: &modifiedAt,
			})
		}))
		require.NoError(t, db.First(purchase, "id = ?", purchase.ID).Error)
	}
	later := time.Now().Add(time.Minute)

	settle(3, periodStart, later)
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, purchase.Status, "the original seats after the purchase prove nothing")
	settle(4, periodEnd, later)
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, purchase.Status, "the target in a renewed period proves nothing")

	settle(4, periodStart, later)
	require.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, purchase.Status)

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
		assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, purchase.Status, "a %s event keeps the success", event.name)
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
	operation := &billingmodels.SeatPurchaseOperation{
		Key: "key-1", Fingerprint: billingmodels.SeatPurchaseFingerprint(1, 3),
		PeriodStart: periodStart, PeriodEnd: periodStart.Add(30 * 24 * time.Hour),
	}

	first := billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, operation)
	first.Status = billingmodels.SeatPurchaseStatusFailed
	require.NoError(t, first.Create(db))

	err = billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, operation).Create(db)
	assert.True(t, billingmodels.IsSeatPurchaseOperationKeyConflict(err), "the same key cannot record a second purchase: %v", err)
	require.NoError(t, billingmodels.NewSeatPurchase(other.ID, nil, "sub_other", 3, 4, operation).Create(db), "keys are scoped to the organization")

	stored, err := billingmodels.FindSeatPurchaseByOperationKey(db, orgID, "key-1")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, first.ID, stored.ID)
	assert.True(t, stored.MatchesRequest(billingmodels.SeatPurchaseFingerprints{Counts: billingmodels.SeatPurchaseFingerprint(1, 3)}))
	assert.False(t, stored.MatchesRequest(billingmodels.SeatPurchaseFingerprints{Counts: billingmodels.SeatPurchaseFingerprint(2, 3)}))
	require.NotNil(t, stored.PeriodStart)
	assert.True(t, stored.PeriodStart.Equal(operation.PeriodStart))

	missing, err := billingmodels.FindSeatPurchaseByOperationKey(db, orgID, "key-2")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func Test__FindExpectedSeatCountIgnoresPurchasesFromAnEarlierPeriod(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orgID := r.Organization.ID
	previous := time.Now().Add(-31 * 24 * time.Hour).UTC().Truncate(time.Second)
	current := previous.Add(30 * 24 * time.Hour)
	purchase := billingmodels.NewSeatPurchase(orgID, nil, "sub_team", 3, 4, &billingmodels.SeatPurchaseOperation{
		Key: "key-1", Fingerprint: billingmodels.SeatPurchaseFingerprint(1, 3), PeriodStart: previous, PeriodEnd: current,
	})
	purchase.Status = billingmodels.SeatPurchaseStatusSucceeded
	require.NoError(t, purchase.Create(db))

	expected, err := billingmodels.FindExpectedSeatCount(db, orgID, "sub_team", previous)
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, 4, *expected)

	expected, err = billingmodels.FindExpectedSeatCount(db, orgID, "sub_team", current)
	require.NoError(t, err)
	assert.Nil(t, expected, "a renewal reduction back to 3 seats is not compared with the old target")
}

func Test__ListUnresolvedSeatPurchasesCreatedBeforeFindsAgedPurchasesOfAllOrganizations(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now().UTC().Truncate(time.Second)
	createdAt := func(orgID uuid.UUID, status string, age time.Duration) *billingmodels.SeatPurchase {
		purchase := billingmodels.NewSeatPurchase(orgID, nil, "sub_"+uuid.NewString(), 3, 4, nil)
		purchase.Status, purchase.CreatedAt = status, now.Add(-age)
		require.NoError(t, purchase.Create(db))
		return purchase
	}
	newOrganization := func() uuid.UUID {
		org, err := models.CreateOrganization("org-"+uuid.NewString()[:8], "")
		require.NoError(t, err)
		return org.ID
	}

	unknown := createdAt(r.Organization.ID, billingmodels.SeatPurchaseStatusUnknown, 2*time.Hour)
	pending := createdAt(newOrganization(), billingmodels.SeatPurchaseStatusPending, 3*time.Hour)
	createdAt(newOrganization(), billingmodels.SeatPurchaseStatusUnknown, 30*time.Minute)
	createdAt(newOrganization(), billingmodels.SeatPurchaseStatusSucceeded, 4*time.Hour)
	createdAt(newOrganization(), billingmodels.SeatPurchaseStatusFailed, 4*time.Hour)
	deleted := newOrganization()
	createdAt(deleted, billingmodels.SeatPurchaseStatusUnknown, 5*time.Hour)
	require.NoError(t, db.Model(&models.Organization{}).Where("id = ?", deleted).Update("deleted_at", now).Error)

	purchases, err := billingmodels.ListUnresolvedSeatPurchasesCreatedBefore(db, now.Add(-time.Hour))

	require.NoError(t, err)
	require.Len(t, purchases, 2)
	assert.Equal(t, pending.ID, purchases[0].ID, "the oldest purchase comes first")
	assert.Equal(t, unknown.ID, purchases[1].ID)
}

func Test__SeatPurchaseMatchesRequestByTheStoredFingerprintVersion(t *testing.T) {
	periodStart := time.Now().Add(-7 * 24 * time.Hour).UTC().Truncate(time.Second)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	request := func(additional, expected int, subscriptionID string, start, end time.Time) billingmodels.SeatPurchaseFingerprints {
		return billingmodels.SeatPurchaseFingerprints{
			Counts: billingmodels.SeatPurchaseFingerprint(additional, expected),
			Bound:  billingmodels.SeatPurchaseBoundFingerprint(additional, expected, subscriptionID, start, end),
		}
	}
	unbound := func(additional, expected int) billingmodels.SeatPurchaseFingerprints {
		return billingmodels.SeatPurchaseFingerprints{Counts: billingmodels.SeatPurchaseFingerprint(additional, expected)}
	}
	stored := func(fingerprint string) *billingmodels.SeatPurchase {
		return billingmodels.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 4, &billingmodels.SeatPurchaseOperation{
			Key: "key", Fingerprint: fingerprint, PeriodStart: periodStart, PeriodEnd: periodEnd,
		})
	}
	bound := stored(billingmodels.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart, periodEnd))
	legacy := stored(billingmodels.SeatPurchaseFingerprint(1, 3))

	tests := []struct {
		name     string
		purchase *billingmodels.SeatPurchase
		request  billingmodels.SeatPurchaseFingerprints
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
		{name: "purchase without an operation", purchase: billingmodels.NewSeatPurchase(uuid.New(), nil, "sub_team", 3, 4, nil), request: unbound(1, 3)},
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

	utc := billingmodels.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart, periodEnd)

	assert.Equal(t, utc, billingmodels.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart.In(elsewhere), periodEnd.In(elsewhere)))
	assert.NotEqual(t, utc, billingmodels.SeatPurchaseBoundFingerprint(1, 3, "sub_team", periodStart.Add(time.Microsecond), periodEnd))
	assert.NotEqual(t, utc, billingmodels.SeatPurchaseFingerprint(1, 3), "the v2 fingerprint never equals the v1 fingerprint")
}
