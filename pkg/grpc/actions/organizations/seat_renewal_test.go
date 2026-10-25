package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
)

func scheduledRenewal(seats int) map[string]any {
	return map[string]any{"seats": float64(seats), "proration_behavior": "next_period"}
}

// renewalSchedules waits for the background renewal schedules and returns
// the next_period updates that Polar received.
func (f *seatPurchaseFixture) renewalSchedules() []map[string]any {
	seatrenewal.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]map[string]any(nil), f.schedules...)
}

func (f *seatPurchaseFixture) pendingRenewalSeats() *int {
	seatrenewal.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pendingSeats
}

func (f *seatPurchaseFixture) addMembers(count int) []*models.User {
	f.t.Helper()
	members := make([]*models.User, 0, count)
	for range count {
		account, err := models.CreateAccount("Member", uuid.NewString()+"@example.com")
		require.NoError(f.t, err)
		member, err := models.CreateUser(f.r.Organization.ID, account.ID, account.Email, account.Name)
		require.NoError(f.t, err)
		require.NoError(f.t, f.r.AuthService.AssignRole(member.ID.String(), models.RoleOrgOperator, f.r.Organization.ID.String(), models.DomainTypeOrganization))
		members = append(members, member)
	}
	return members
}

func (f *seatPurchaseFixture) removeMember(user *models.User) {
	f.t.Helper()
	_, err := RemoveUser(context.Background(), f.r.AuthService, f.r.Organization.ID.String(), user.ID.String())
	require.NoError(f.t, err)
}

func (f *seatPurchaseFixture) invitations() *InvitationService {
	return &InvitationService{AuthService: f.r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
}

// invite returns the acceptance token of a new invitation.
func (f *seatPurchaseFixture) invite(email string) string {
	f.t.Helper()
	created, err := f.invitations().Create(context.Background(), f.r.Organization.ID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: email})
	require.NoError(f.t, err)
	return strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
}

func (f *seatPurchaseFixture) accept(accountID uuid.UUID, token string) string {
	f.t.Helper()
	response, err := f.invitations().Accept(context.Background(), accountID.String(), token)
	require.NoError(f.t, err)
	return response.Fields["status"].GetStringValue()
}

func (f *seatPurchaseFixture) seatUsage() *entitlements.SeatUsage {
	f.t.Helper()
	seats, err := entitlements.ResolveSeats(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return seats
}

// applyCycled delivers subscription.cycled for the next period with the seats
// that Polar renewed.
func (f *seatPurchaseFixture) applyCycled(seats int) {
	f.t.Helper()
	f.mu.Lock()
	f.periodStart = f.periodEnd
	f.periodEnd = f.periodEnd.AddDate(0, 1, 0)
	f.polarSeats = seats
	f.pendingSeats = nil
	data := f.snapshot(seats)
	f.mu.Unlock()
	data["modified_at"] = time.Now().Add(time.Minute).UTC().Format(time.RFC3339)
	body, err := json.Marshal(map[string]any{"type": "subscription.cycled", "data": data})
	require.NoError(f.t, err)
	var event polar.SubscriptionWebhookEvent
	require.NoError(f.t, json.Unmarshal(body, &event))
	require.NoError(f.t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
}

func Test__RemovingAMemberSchedulesOneSeatLessForRenewal(t *testing.T) {
	f := newSeatPurchaseFixture(t, 10, new(50))
	members := f.addMembers(9)

	f.removeMember(members[0])

	assert.Equal(t, []map[string]any{scheduledRenewal(9)}, f.renewalSchedules())
	assert.Empty(t, f.patchBodies, "nothing is charged or credited now")
	assert.Equal(t, 10, f.purchasedSeats())
	assert.True(t, f.plan().CurrentPeriodEnd.Equal(f.periodEnd), "the renewal date does not change")
	assert.Equal(t, new(1), f.seatUsage().Available, "the freed seat stays usable until renewal")
	assert.Equal(t, new(9), f.plan().RenewalSeatsScheduled)
	assert.NotNil(t, f.plan().RenewalSeatsScheduledAt)
}

func Test__AMemberWhoJoinsSchedulesTheFullSeatCountAgain(t *testing.T) {
	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "new member", true: "restored former member"}[restore], func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 10, new(50))
			members := f.addMembers(9)
			f.removeMember(members[0])
			require.Equal(t, new(9), f.pendingRenewalSeats())

			email, accountID := members[0].GetEmail(), *members[0].AccountID
			if !restore {
				account, err := models.CreateAccount("Replacement", uuid.NewString()+"@example.com")
				require.NoError(t, err)
				email, accountID = account.Email, account.ID
			}
			token := f.invite(email)
			require.Equal(t, "joined", f.accept(accountID, token))

			assert.Equal(t, []map[string]any{scheduledRenewal(9), scheduledRenewal(10)}, f.renewalSchedules())
			assert.Nil(t, f.pendingRenewalSeats(), "no reduction remains")
			if restore {
				restored, err := models.FindActiveUserByEmail(f.r.Organization.ID.String(), email)
				require.NoError(t, err)
				assert.Equal(t, members[0].ID, restored.ID)
			}

			require.Equal(t, "already_member", f.accept(accountID, token))
			assert.Len(t, f.renewalSchedules(), 2, "a member who is already in the organization changes nothing")
		})
	}
}

func Test__PendingInvitationsDoNotRenewAndTheCycleKeepsOnlyOccupiedSeats(t *testing.T) {
	f := newSeatPurchaseFixture(t, 5, new(50))
	members := f.addMembers(2)
	f.invite("first@example.com")
	f.invite("second@example.com")
	members = append(members, f.addMembers(1)...)

	f.removeMember(members[0])

	assert.Equal(t, []map[string]any{scheduledRenewal(3)}, f.renewalSchedules())
	assert.Equal(t, 2, f.seatUsage().PendingInvitations)

	f.applyCycled(3)

	assert.Equal(t, 3, f.purchasedSeats())
	plan := f.plan()
	assert.Equal(t, new(3), plan.RenewalSeatsScheduled, "a subscription snapshot keeps the scheduled record")
	assert.NotNil(t, plan.RenewalSeatsScheduledAt)
}

func Test__SeatPurchaseSchedulesTheRenewalSeatCountAgain(t *testing.T) {
	// Only the owner is a member, and Polar renews one seat.
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.pendingSeats = new(1)

	_, err := f.purchase(1, 3)
	require.NoError(t, err)

	assert.Equal(t, []map[string]any{{"seats": float64(4), "proration_behavior": "invoice"}}, f.patchBodies)
	assert.Equal(t, []map[string]any{scheduledRenewal(1)}, f.renewalSchedules(),
		"the purchase dropped the scheduled reduction, so it is scheduled again")
	assert.Equal(t, new(1), f.pendingRenewalSeats())
	assert.Equal(t, 4, f.purchasedSeats())
}

func Test__BillingSyncRestoresRenewalReductionAfterLostPurchaseResponse(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.pendingSeats = new(1)
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.polarSeats = seats
		f.pendingSeats = nil
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}
	response, err := f.purchase(1, 3)
	require.NoError(t, err)
	require.Equal(t, seatPurchaseAwaitingConfirmation, response.Purchase.Status)
	require.Nil(t, f.pendingRenewalSeats(), "the purchase cleared the reduction")
	f.mu.Lock()
	readsBeforeSync := f.getCount
	f.mu.Unlock()

	_, err = SyncOrganizationBilling(t.Context(), f.r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
	assert.Equal(t, new(1), f.pendingRenewalSeats())
	assert.Equal(t, new(1), f.plan().RenewalSeatsScheduled)
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Equal(t, 1, f.getCount-readsBeforeSync, "billing sync reads Polar for only one renewal schedule")
}

func Test__NoRenewalScheduleWhileASeatPurchaseIsInFlight(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	members := f.addMembers(1)
	f.dropConnection()
	// Polar never answered the purchase, so the subscription has no
	// modification that proves whether Polar applied it.
	f.modifiedAt = time.Now().Add(-time.Hour)
	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	require.Equal(t, models.SeatPurchaseStatusUnknown, f.latestPurchase().Status)

	f.removeMember(members[0])

	assert.Empty(t, f.renewalSchedules(), "the purchase replaces any scheduled count when it settles")
	assert.Equal(t, models.SeatPurchaseStatusUnknown, f.latestPurchase().Status,
		"nothing proves the outcome yet, so the purchase stays in flight")
}

// setToCancel marks the subscription to end at the period end, locally and
// in Polar.
func (f *seatPurchaseFixture) setToCancel() {
	f.t.Helper()
	f.mu.Lock()
	f.cancelAtPeriodEnd = true
	f.mu.Unlock()
	f.updateBillingPlan(map[string]any{"cancel_at_period_end": true})
}

func (f *seatPurchaseFixture) resume() *pb.DescribeOrganizationBillingResponse {
	f.t.Helper()
	response, err := ResumeOrganizationSubscription(context.Background(), f.r.Organization.ID.String(), &pb.ResumeOrganizationSubscriptionRequest{})
	require.NoError(f.t, err)
	return response
}

func Test__ResumeShortlyBeforeRenewalSchedulesTheMemberCount(t *testing.T) {
	f := newSeatPurchaseFixture(t, 5, new(50))
	f.addMembers(1)
	f.setPeriodEnd(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
	f.setToCancel()

	response := f.resume()

	assert.False(t, response.CancelAtPeriodEnd)
	assert.Equal(t, []map[string]any{scheduledRenewal(2)}, f.renewalSchedules())
	assert.Equal(t, new(2), f.pendingRenewalSeats())
	assert.Equal(t, new(2), f.plan().RenewalSeatsScheduled)
	assert.Equal(t, 5, f.purchasedSeats(), "the current period keeps every paid seat")
}

func Test__ResumeSucceedsWhenPolarRejectsTheRenewalSchedule(t *testing.T) {
	f := newSeatPurchaseFixture(t, 5, new(50))
	f.addMembers(1)
	f.setPeriodEnd(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
	f.setToCancel()
	f.scheduleRejection = &polarRejection{status: http.StatusUnprocessableEntity, code: "PolarRequestValidationError"}

	response := f.resume()

	assert.False(t, response.CancelAtPeriodEnd)
	assert.Len(t, f.renewalSchedules(), 1, "the schedule was sent once and is left to the sync worker")
	assert.False(t, f.plan().CancelAtPeriodEnd, "the resume stays applied")
	assert.Nil(t, f.pendingRenewalSeats())
	assert.Nil(t, f.plan().RenewalSeatsScheduled)
}
