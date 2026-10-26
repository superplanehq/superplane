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
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/testsupport/polarfake"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
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
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
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

func (f *seatRenewalSyncFixture) entries(level log.Level, message string) []log.Entry {
	var found []log.Entry
	for _, entry := range f.logs.AllEntries() {
		if entry.Level == level && entry.Message == message {
			found = append(found, *entry)
		}
	}
	return found
}

func (f *seatRenewalSyncFixture) gets() []string {
	var ids []string
	for _, request := range f.polar.Requests() {
		if request.Method == http.MethodGet {
			ids = append(ids, request.SubscriptionID)
		}
	}
	return ids
}

func Test__SeatRenewalSyncReconcilesTheOrganizationThatRenewsFirstFirst(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	_, later := f.addOrganization(4, 20*24*time.Hour)
	_, soon := f.addOrganization(3, 12*time.Hour)
	_, sooner := f.addOrganization(1, 6*time.Hour)

	f.worker.tick(context.Background())

	assert.Equal(t, []string{sooner.ID, soon.ID, later.ID}, f.gets())
	assert.Empty(t, f.polar.Patches(), "the sync never writes a seat count, even with spare seats")
	assert.Nil(t, f.polar.Subscription(later.ID).PendingSeats, "renewal keeps the paid seats")
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

	assert.Equal(t, middleOrg, f.worker.resumeAfter)
	assert.Equal(t, []string{soon.ID, middle.ID}, f.gets(), "the rate limit stops the run before later organizations")

	requests := len(f.polar.Requests())
	f.worker.tick(context.Background())

	// The next run continues where the rate limit stopped it, so the
	// organizations later in the list are checked too.
	assert.Equal(t, []string{middle.ID, later.ID}, f.gets()[requests:], "the resumed run does not spend its budget again on earlier organizations")
	assert.Equal(t, uuid.Nil, f.worker.resumeAfter, "a completed run clears the position")

	requests = len(f.polar.Requests())
	f.worker.tick(context.Background())

	// After a completed run, the next run starts again with the
	// organization that renews first.
	assert.Equal(t, []string{soon.ID, middle.ID, later.ID}, f.gets()[requests:])
	assert.Empty(t, f.polar.Patches())
}

func Test__SeatRenewalSyncReportsAFailureCloseToRenewalAsAnError(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	soonOrg, soon := f.addOrganization(3, 12*time.Hour)
	f.addOrganization(3, 10*24*time.Hour)
	f.polar.FailNext(2, http.StatusServiceUnavailable, "")

	f.worker.tick(context.Background())

	failed := f.entries(log.ErrorLevel, seatRenewalSyncFailedKey)
	require.Len(t, failed, 1, "only a failure close to renewal is an error")
	assert.Equal(t, soonOrg.String(), failed[0].Data["organization_id"])
	assert.Equal(t, soon.ID, failed[0].Data["polar_subscription_id"])
}

func Test__SeatRenewalSyncKeepsReconcilingWhileAPurchaseIsInFlight(t *testing.T) {
	f := newSeatRenewalSyncFixture(t)
	orgID, sub := f.addOrganization(3, 12*time.Hour)
	purchase := billingmodels.NewSeatPurchase(orgID, nil, sub.ID, 3, 4, nil)
	require.NoError(t, purchase.Create(database.Conn()))
	// A renewal event was lost while the purchase is in its confirmation window.
	renewedStart := sub.PeriodEnd
	f.polar.Update(sub.ID, func(s *polarfake.Subscription) {
		s.PeriodStart = renewedStart
		s.PeriodEnd = renewedStart.AddDate(0, 1, 0)
	})

	f.worker.tick(context.Background())

	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), orgID)
	require.NoError(t, err)
	assert.True(t, plan.CurrentPeriodStart.Equal(renewedStart), "the stored period follows Polar")
	assert.Empty(t, f.polar.Patches())
	assert.Empty(t, f.entries(log.ErrorLevel, seatPurchaseUnresolvedKey), "a recent purchase is not unresolved yet")
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

func (f *seatRenewalSyncFixture) plan() *billingmodels.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
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

	f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.Status = billingmodels.PolarSubscriptionStatusPastDue })
	f.worker.tick(context.Background())

	assert.Equal(t, billingmodels.PolarSubscriptionStatusPastDue, f.plan().PolarSubscriptionStatus)
	assert.True(t, seatUsage().PaymentBlocked)
	_, err := f.invitations().Accept(context.Background(), invitee.ID.String(), token)
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")

	f.polar.Update(sub.ID, func(s *polarfake.Subscription) { s.Status = billingmodels.PolarSubscriptionStatusActive })
	f.worker.tick(context.Background())

	plan := f.plan()
	assert.Equal(t, billingmodels.PolarSubscriptionStatusActive, plan.PolarSubscriptionStatus)
	assert.True(t, plan.Catalog.PerSeat, "the plan returns to the per-seat catalog plan")
	assert.False(t, seatUsage().PaymentBlocked)
	assert.Empty(t, f.polar.Patches())
}

// agedPurchase records an unknown purchase that started two hours ago in the
// given billing period of the subscription.
func (f *seatRenewalSyncFixture) agedPurchase(orgID uuid.UUID, subscriptionID string, periodStart, periodEnd time.Time) *billingmodels.SeatPurchase {
	f.t.Helper()
	purchase := billingmodels.NewSeatPurchase(orgID, nil, subscriptionID, 3, 4, &billingmodels.SeatPurchaseOperation{
		Key: uuid.NewString(), Fingerprint: billingmodels.SeatPurchaseFingerprint(1, 3), PeriodStart: periodStart, PeriodEnd: periodEnd,
	})
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	purchase.CreatedAt = f.now.Add(-2 * time.Hour)
	require.NoError(f.t, purchase.Create(database.Conn()))
	return purchase
}

func (f *seatRenewalSyncFixture) assertStillUnresolved(purchase *billingmodels.SeatPurchase) {
	f.t.Helper()
	var stored billingmodels.SeatPurchase
	require.NoError(f.t, database.Conn().First(&stored, "id = ?", purchase.ID).Error)
	assert.Equal(f.t, billingmodels.SeatPurchaseStatusUnknown, stored.Status, "the report and the reconciliation never settle a purchase")
}

func (f *seatRenewalSyncFixture) unresolvedLine(purchase *billingmodels.SeatPurchase) log.Entry {
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
	require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).
		Update("polar_subscription_status", billingmodels.PolarSubscriptionStatusCanceled).Error)
	purchase := f.agedPurchase(orgID, sub.ID, sub.PeriodStart, sub.PeriodEnd)

	f.worker.tick(context.Background())

	assert.Empty(t, f.polar.Requests(), "the canceled plan is not a renewal candidate")
	line := f.unresolvedLine(purchase)
	assert.Equal(t, orgID.String(), line.Data["organization_id"])
	assert.Equal(t, *purchase.OperationKey, line.Data["operation_key"])
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, line.Data["seat_purchase_status"])
	assert.Equal(t, sub.ID, line.Data["polar_subscription_id"])
	assert.Equal(t, sub.ID, line.Data["current_polar_subscription_id"])
	assert.True(t, sub.PeriodStart.Equal(line.Data["current_period_start"].(time.Time)))
	assert.True(t, sub.PeriodStart.Equal(line.Data["purchase_period_start"].(time.Time)))
	assert.True(t, sub.PeriodEnd.Equal(line.Data["purchase_period_end"].(time.Time)))
	assert.Equal(t, int64(2*time.Hour/time.Second), line.Data["age_seconds"])
	assert.NotContains(t, line.Data, "renewal_impact", "no purchase blocks renewal maintenance")
	f.assertStillUnresolved(purchase)

	f.worker.tick(context.Background())

	assert.Len(t, f.entries(log.ErrorLevel, seatPurchaseUnresolvedKey), 2, "the line repeats while the purchase stays unresolved")
}

func Test__SeatRenewalSyncReportsAnAgedPurchaseWhateverTheReconciliationDoes(t *testing.T) {
	tests := []struct {
		name string
		// setup prepares the organization and returns its purchase.
		setup func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *billingmodels.SeatPurchase
	}{
		{
			name: "reconciliation succeeds",
			setup: func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *billingmodels.SeatPurchase {
				return f.agedPurchase(orgID, sub.ID, sub.PeriodStart.AddDate(0, -1, 0), sub.PeriodStart)
			},
		},
		{
			name: "reconciliation fails",
			setup: func(f *seatRenewalSyncFixture, orgID uuid.UUID, sub polarfake.Subscription) *billingmodels.SeatPurchase {
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

			assert.Equal(t, orgID.String(), f.unresolvedLine(purchase).Data["organization_id"])
			assert.Empty(t, f.polar.Patches())
			f.assertStillUnresolved(purchase)
		})
	}
}
