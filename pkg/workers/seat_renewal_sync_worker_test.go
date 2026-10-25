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
	assert.Empty(t, f.polar.Patches(), "a payment-blocked plan gets no scheduled seat count")

	f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.Status = models.PolarSubscriptionStatusActive })
	f.worker.tick(context.Background())

	plan := f.plan()
	assert.Equal(t, models.PolarSubscriptionStatusActive, plan.PolarSubscriptionStatus)
	assert.True(t, plan.Catalog.PerSeat, "the plan returns to the per-seat catalog plan")
	assert.False(t, seatUsage().PaymentBlocked)
	assert.Equal(t, []string{sub.ID}, f.patchedSubscriptions(), "the recovered plan gets its renewal seat count again")
}
