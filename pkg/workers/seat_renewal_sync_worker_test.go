package workers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/polarfake"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

type seatRenewalSyncFixture struct {
	t      *testing.T
	r      *support.ResourceRegistry
	polar  *polarfake.Server
	worker *SeatRenewalSyncWorker
	now    time.Time
	logs   *logtest.Hook
}

func newSeatRenewalSyncFixture(t *testing.T) *seatRenewalSyncFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(models.BillingEnabledEnv, "yes")
	f := &seatRenewalSyncFixture{t: t, r: r, polar: polarfake.NewServer(t), now: time.Now().UTC().Truncate(time.Second)}
	f.worker = NewSeatRenewalSyncWorker()
	f.worker.now = func() time.Time { return f.now }
	f.logs = logtest.NewGlobal()
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(make(log.LevelHooks)) })
	return f
}

// addOrganization adds an organization with only an owner on a per-seat
// subscription that renews after the given time.
func (f *seatRenewalSyncFixture) addOrganization(seats int, renewsIn time.Duration) (uuid.UUID, polarfake.Subscription) {
	f.t.Helper()
	org, err := models.CreateOrganizationInTransaction(database.Conn(), "Org "+uuid.NewString()[:8], "")
	require.NoError(f.t, err)
	account, err := models.CreateAccount("Owner", uuid.NewString()+"@example.com")
	require.NoError(f.t, err)
	_, err = models.CreateUser(org.ID, account.ID, account.Email, account.Name)
	require.NoError(f.t, err)
	return org.ID, f.polar.AddSeatPlan(database.Conn(), org.ID, seats, f.now.Add(renewsIn))
}

func (f *seatRenewalSyncFixture) patchedSubscriptions() []string {
	var ids []string
	for _, patch := range f.polar.Patches() {
		ids = append(ids, patch.SubscriptionID)
	}
	return ids
}

func (f *seatRenewalSyncFixture) entries(level log.Level, message string) []log.Entry {
	var found []log.Entry
	for _, entry := range f.logs.AllEntries() {
		if entry.Level == level && entry.Message == message {
			found = append(found, *entry)
		}
	}
	return found
}

func Test__SeatRenewalSyncFixesTheOrganizationThatRenewsFirstFirst(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	laterOrg, later := f.addOrganization(4, 20*24*time.Hour)
	soonOrg, soon := f.addOrganization(3, 12*time.Hour)
	_, inSync := f.addOrganization(1, 6*time.Hour)

	f.worker.tick(context.Background())

	// The in-sync organization renews first and already has the target
	// count. It still gets one idempotent update, because the plan records
	// only counts that SuperPlane sent.
	assert.Equal(t, []string{inSync.ID, soon.ID, later.ID}, f.patchedSubscriptions())
	for _, patch := range f.polar.Patches() {
		assert.Equal(t, map[string]any{"seats": float64(1), "proration_behavior": "next_period"}, patch.Body)
	}
	assert.Nil(t, f.polar.Subscription(inSync.ID).PendingSeats)
	assert.Equal(t, 3, f.polar.Subscription(soon.ID).Seats, "the current period keeps its seats")

	fixed := f.entries(log.WarnLevel, seatRenewalMismatchFixedKey)
	require.Len(t, fixed, 2, "a count that already matches is recorded, not fixed")
	assert.Equal(t, soonOrg.String(), fixed[0].Data["organization_id"])
	assert.Equal(t, soon.ID, fixed[0].Data["polar_subscription_id"])
	assert.Equal(t, 1, fixed[0].Data["renewal_target_seats"])
	assert.Equal(t, 3, fixed[0].Data["next_period_seats"])
	assert.Equal(t, 3, fixed[0].Data["current_seats"])
	assert.Equal(t, laterOrg.String(), fixed[1].Data["organization_id"])

	f.worker.tick(context.Background())
	assert.Len(t, f.polar.Patches(), 3, "counts and records that already match need no update")
}

func Test__SeatRenewalSyncResumesWhereARateLimitStopped(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	_, soon := f.addOrganization(3, 12*time.Hour)
	middleOrg, middle := f.addOrganization(3, 2*24*time.Hour)
	_, later := f.addOrganization(3, 10*24*time.Hour)
	// A server error skips the first organization, and a rate limit then
	// stops the run at the second one.
	f.polar.FailNext(1, http.StatusServiceUnavailable, "")
	f.polar.FailNext(1, http.StatusTooManyRequests, "")

	f.worker.tick(context.Background())

	assert.Empty(t, f.polar.Patches())
	assert.Equal(t, middleOrg, f.worker.resumeAfter)
	for _, request := range f.polar.Requests() {
		assert.NotEqual(t, later.ID, request.SubscriptionID, "the rate limit stops the run before later organizations")
	}

	requests := len(f.polar.Requests())
	f.worker.tick(context.Background())

	// The next run continues where the rate limit stopped it, so the
	// organizations later in the list are checked too.
	assert.Equal(t, []string{middle.ID, later.ID}, f.patchedSubscriptions())
	for _, request := range f.polar.Requests()[requests:] {
		assert.NotEqual(t, soon.ID, request.SubscriptionID, "the resumed run does not spend its budget again on earlier organizations")
	}
	assert.Equal(t, uuid.Nil, f.worker.resumeAfter, "a completed run clears the position")

	requests = len(f.polar.Requests())
	f.worker.tick(context.Background())

	// After a completed run, the next run starts again with the
	// organization that renews first.
	require.Greater(t, len(f.polar.Requests()), requests)
	assert.Equal(t, soon.ID, f.polar.Requests()[requests].SubscriptionID)
	assert.Equal(t, []string{middle.ID, later.ID, soon.ID}, f.patchedSubscriptions())
}

func Test__SeatRenewalSyncFixesTheCountAfterAPolarOutage(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	soonOrg, soon := f.addOrganization(3, 12*time.Hour)
	_, later := f.addOrganization(3, 10*24*time.Hour)
	f.polar.FailNext(2, http.StatusServiceUnavailable, "")

	f.worker.tick(context.Background())

	assert.Empty(t, f.polar.Patches())
	failed := f.entries(log.ErrorLevel, seatRenewalSyncFailedKey)
	require.Len(t, failed, 1, "only a failure close to renewal is an error")
	assert.Equal(t, soonOrg.String(), failed[0].Data["organization_id"])
	assert.Equal(t, soon.ID, failed[0].Data["polar_subscription_id"])

	f.worker.tick(context.Background())

	assert.Equal(t, []string{soon.ID, later.ID}, f.patchedSubscriptions())
	assert.Len(t, f.entries(log.WarnLevel, seatRenewalMismatchFixedKey), 2)
}

func Test__SeatRenewalSyncWaitsUntilTheSeatPurchaseSettles(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	orgID, sub := f.addOrganization(3, 12*time.Hour)
	purchase := models.NewSeatPurchase(orgID, nil, sub.ID, 3, 4, nil)
	require.NoError(t, purchase.Create(database.Conn()))

	f.worker.tick(context.Background())
	assert.Empty(t, f.polar.Patches(), "a purchase in its confirmation window owns the next seat change")
	blocked := f.entries(log.WarnLevel, seatRenewalBlockedByPurchaseKey)
	require.Len(t, blocked, 1)
	assert.Equal(t, orgID.String(), blocked[0].Data["organization_id"])
	assert.Equal(t, purchase.ID.String(), blocked[0].Data["seat_purchase_id"])
	assert.Equal(t, sub.ID, blocked[0].Data["polar_subscription_id"])
	assert.Empty(t, f.entries(log.ErrorLevel, seatPurchaseUnresolvedKey), "a recent purchase is not unresolved yet")

	f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.Seats = 4 })
	require.NoError(t, purchase.UpdateStatus(database.Conn(), models.SeatPurchaseStatusSucceeded, ""))

	f.worker.tick(context.Background())
	require.Len(t, f.polar.Patches(), 1)
	assert.Equal(t, map[string]any{"seats": float64(1), "proration_behavior": "next_period"}, f.polar.Patches()[0].Body)
}

// addSetupOrganization puts the organization of the test setup on a per-seat
// subscription with the owner and one more member.
func (f *seatRenewalSyncFixture) addSetupOrganization(seats int, renewsIn time.Duration) polarfake.Subscription {
	f.t.Helper()
	sub := f.polar.AddSeatPlan(database.Conn(), f.r.Organization.ID, seats, f.now.Add(renewsIn))
	support.CreateUser(f.t, f.r, f.r.Organization.ID)
	return sub
}

func (f *seatRenewalSyncFixture) invitations() *organizations.InvitationService {
	return &organizations.InvitationService{AuthService: f.r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
}

// invite creates a pending invitation and returns the accepting account and token.
func (f *seatRenewalSyncFixture) invite() (*models.Account, string) {
	f.t.Helper()
	invitee, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
	require.NoError(f.t, err)
	created, err := f.invitations().Create(context.Background(), f.r.Organization.ID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: invitee.Email})
	require.NoError(f.t, err)
	return invitee, strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
}

func (f *seatRenewalSyncFixture) plan() *models.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := models.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return plan
}

func requireInvitationReason(t *testing.T, err error, reason string) {
	t.Helper()
	require.Error(t, err)
	details := status.Convert(err).Details()
	require.Len(t, details, 1)
	require.Equal(t, reason, details[0].(*errdetails.ErrorInfo).Reason)
}

func Test__SeatRenewalSyncRepairsADroppedRenewalEvent(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	sub := f.addSetupOrganization(3, 12*time.Hour)
	invitee, token := f.invite()

	// Polar renewed the subscription with fewer seats, and no event arrived.
	renewedStart := sub.PeriodEnd
	renewedEnd := renewedStart.AddDate(0, 1, 0)
	f.polar.Update(sub.ID, func(s *polarfake.Subscription) {
		s.Seats = 2
		s.PeriodStart = renewedStart
		s.PeriodEnd = renewedEnd
	})

	f.worker.tick(context.Background())

	plan := f.plan()
	assert.Equal(t, new(2), plan.PurchasedSeats)
	assert.True(t, plan.CurrentPeriodStart.Equal(renewedStart))
	assert.True(t, plan.CurrentPeriodEnd.Equal(renewedEnd))
	_, err := f.invitations().Accept(context.Background(), invitee.ID.String(), token)
	requireInvitationReason(t, err, "NO_SEAT_AVAILABLE")
}

func Test__SeatRenewalSyncRepairsDroppedFailureAndRecoveryEvents(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	sub := f.addSetupOrganization(3, 12*time.Hour)
	invitee, token := f.invite()
	seatUsage := func() *entitlements.SeatUsage {
		usage, err := entitlements.ResolveSeats(database.Conn(), f.r.Organization.ID)
		require.NoError(t, err)
		return usage
	}

	f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.Status = models.PolarSubscriptionStatusPastDue })
	f.worker.tick(context.Background())

	assert.Equal(t, models.PolarSubscriptionStatusPastDue, f.plan().PolarSubscriptionStatus)
	assert.True(t, seatUsage().PaymentBlocked)
	_, err := f.invitations().Accept(context.Background(), invitee.ID.String(), token)
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
	assert.Equal(t, []string{sub.ID}, f.patchedSubscriptions(), "a past-due plan keeps its renewal seat count")

	f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.Status = models.PolarSubscriptionStatusActive })
	f.worker.tick(context.Background())

	plan := f.plan()
	assert.Equal(t, models.PolarSubscriptionStatusActive, plan.PolarSubscriptionStatus)
	assert.True(t, plan.Catalog.PerSeat, "the plan returns to the per-seat catalog plan")
	assert.False(t, seatUsage().PaymentBlocked)
	assert.Equal(t, []string{sub.ID}, f.patchedSubscriptions(), "the recovered plan already renews with its seat count")
}

// agedPurchase records an unknown purchase that started two hours ago in the
// given billing period of the subscription.
func (f *seatRenewalSyncFixture) agedPurchase(orgID uuid.UUID, subscriptionID string, periodStart, periodEnd time.Time) *models.SeatPurchase {
	f.t.Helper()
	purchase := models.NewSeatPurchase(orgID, nil, subscriptionID, 3, 4, &models.SeatPurchaseOperation{
		Key: uuid.NewString(), Fingerprint: models.SeatPurchaseFingerprint(1, 3), PeriodStart: periodStart, PeriodEnd: periodEnd,
	})
	purchase.Status = models.SeatPurchaseStatusUnknown
	purchase.CreatedAt = f.now.Add(-2 * time.Hour)
	require.NoError(f.t, purchase.Create(database.Conn()))
	return purchase
}

func (f *seatRenewalSyncFixture) assertStillUnresolved(purchase *models.SeatPurchase) {
	f.t.Helper()
	var stored models.SeatPurchase
	require.NoError(f.t, database.Conn().First(&stored, "id = ?", purchase.ID).Error)
	assert.Equal(f.t, models.SeatPurchaseStatusUnknown, stored.Status, "the report and the schedule never settle a purchase")
}

func (f *seatRenewalSyncFixture) unresolvedLine(purchase *models.SeatPurchase) log.Entry {
	f.t.Helper()
	var found []log.Entry
	for _, entry := range f.entries(log.ErrorLevel, seatPurchaseUnresolvedKey) {
		if entry.Data["seat_purchase_id"] == purchase.ID.String() {
			found = append(found, entry)
		}
	}
	require.Len(f.t, found, 1)
	return found[0]
}

func Test__SeatRenewalSyncReportsAnAgedPurchaseOutsideTheCandidateList(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	orgID, sub := f.addOrganization(3, 12*time.Hour)
	require.NoError(t, database.Conn().Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).
		Update("polar_subscription_status", models.PolarSubscriptionStatusCanceled).Error)
	purchase := f.agedPurchase(orgID, sub.ID, sub.PeriodStart, sub.PeriodEnd)

	f.worker.tick(context.Background())

	assert.Empty(t, f.polar.Requests(), "the canceled plan is not a renewal candidate")
	line := f.unresolvedLine(purchase)
	assert.Equal(t, orgID.String(), line.Data["organization_id"])
	assert.Equal(t, *purchase.OperationKey, line.Data["operation_key"])
	assert.Equal(t, models.SeatPurchaseStatusUnknown, line.Data["seat_purchase_status"])
	assert.Equal(t, sub.ID, line.Data["polar_subscription_id"])
	assert.Equal(t, sub.ID, line.Data["current_polar_subscription_id"])
	assert.True(t, sub.PeriodStart.Equal(line.Data["current_period_start"].(time.Time)))
	assert.True(t, sub.PeriodStart.Equal(line.Data["purchase_period_start"].(time.Time)))
	assert.True(t, sub.PeriodEnd.Equal(line.Data["purchase_period_end"].(time.Time)))
	assert.Equal(t, int64(2*time.Hour/time.Second), line.Data["age_seconds"])
	assert.Equal(t, "no_renewal_maintenance", line.Data["renewal_impact"])
	f.assertStillUnresolved(purchase)

	f.worker.tick(context.Background())

	assert.Len(t, f.entries(log.ErrorLevel, seatPurchaseUnresolvedKey), 2, "the line repeats while the purchase stays unresolved")
}

func Test__SeatRenewalSyncReportsAnAgedPurchaseWhateverTheScheduleDoes(t *testing.T) {
	tests := []struct {
		name string
		// setup prepares the organization and returns its purchase.
		setup  func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *models.SeatPurchase
		impact string
		// patched is true when the schedule sends the renewal count.
		patched bool
	}{
		{
			name: "schedule succeeds", impact: "renewal_maintenance_continues", patched: true,
			setup: func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *models.SeatPurchase {
				return f.agedPurchase(orgID, sub.ID, sub.PeriodStart.AddDate(0, -1, 0), sub.PeriodStart)
			},
		},
		{
			name: "schedule skips", impact: "no_renewal_maintenance",
			setup: func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *models.SeatPurchase {
				f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.CancelAtPeriodEnd = true })
				require.NoError(f.t, database.Conn().Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).
					Update("cancel_at_period_end", true).Error)
				return f.agedPurchase(orgID, sub.ID, sub.PeriodStart.AddDate(0, -1, 0), sub.PeriodStart)
			},
		},
		{
			name: "schedule fails", impact: "blocks_renewal_maintenance",
			setup: func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *models.SeatPurchase {
				f.polar.FailNext(1, http.StatusServiceUnavailable, "")
				return f.agedPurchase(orgID, sub.ID, sub.PeriodStart, sub.PeriodEnd)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSeatRenewalSyncFixture(t)
			orgID, sub := f.addOrganization(3, 12*time.Hour)
			purchase := tt.setup(f, orgID, sub)

			f.worker.tick(context.Background())

			assert.Equal(t, tt.impact, f.unresolvedLine(purchase).Data["renewal_impact"])
			if tt.patched {
				assert.Equal(t, []string{sub.ID}, f.patchedSubscriptions())
			} else {
				assert.Empty(t, f.polar.Patches())
			}
			f.assertStillUnresolved(purchase)
		})
	}
}
