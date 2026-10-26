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
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
)

// assertNoAutomaticQuantityWrite fails when SuperPlane scheduled a seat count
// or sent a seat update other than the explicit purchases.
func (f *seatPurchaseFixture) assertNoAutomaticQuantityWrite(purchases int) {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Empty(f.t, f.schedules, "SuperPlane never schedules a renewal seat count")
	assert.Len(f.t, f.patchBodies, purchases, "only explicit purchases change the seat count")
	for _, body := range f.patchBodies {
		assert.Equal(f.t, "invoice", body["proration_behavior"])
	}
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

func Test__RemovingAMemberKeepsThePaidSeatWithoutAQuantityWrite(t *testing.T) {
	f := newSeatPurchaseFixture(t, 10, new(50))
	members := f.addMembers(9)

	f.removeMember(members[0])

	f.assertNoAutomaticQuantityWrite(0)
	assert.Equal(t, 10, f.purchasedSeats())
	assert.True(t, f.plan().CurrentPeriodEnd.Equal(f.periodEnd), "the renewal date does not change")
	assert.Equal(t, new(1), f.seatUsage().Available, "the freed seat stays paid and reusable")
}

func Test__AMemberWhoJoinsReusesAPaidSeatWithoutAQuantityWrite(t *testing.T) {
	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "new member", true: "restored former member"}[restore], func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 10, new(50))
			members := f.addMembers(9)
			f.removeMember(members[0])

			email, accountID := members[0].GetEmail(), *members[0].AccountID
			if !restore {
				account, err := models.CreateAccount("Replacement", uuid.NewString()+"@example.com")
				require.NoError(t, err)
				email, accountID = account.Email, account.ID
			}
			token := f.invite(email)
			require.Equal(t, "joined", f.accept(accountID, token))
			require.Equal(t, "already_member", f.accept(accountID, token))

			f.assertNoAutomaticQuantityWrite(0)
			assert.Equal(t, 10, f.purchasedSeats())
			assert.Equal(t, new(0), f.seatUsage().Available)
			if restore {
				restored, err := models.FindActiveUserByEmail(f.r.Organization.ID.String(), email)
				require.NoError(t, err)
				assert.Equal(t, members[0].ID, restored.ID)
			}
		})
	}
}

func Test__RenewalKeepsPaidSeatsAndPendingInvitations(t *testing.T) {
	f := newSeatPurchaseFixture(t, 5, new(50))
	members := f.addMembers(2)
	f.invite("first@example.com")
	f.invite("second@example.com")
	f.removeMember(members[0])

	// Polar renews the paid seats unchanged.
	f.applyCycled(5)

	f.assertNoAutomaticQuantityWrite(0)
	assert.Equal(t, 5, f.purchasedSeats(), "renewal keeps the spare paid seats")
	usage := f.seatUsage()
	assert.Equal(t, 2, usage.PendingInvitations, "pending invitations stay pending")
	assert.Equal(t, new(1), usage.Invitable)
}

// Ten paid seats, seven members, and one purchased seat give eleven paid
// seats. The purchase starts from the paid seats, not from the members.
func Test__SeatPurchaseStartsFromThePaidSeats(t *testing.T) {
	f := newSeatPurchaseFixture(t, 10, new(50))
	f.addMembers(6)

	_, err := f.purchase(1, 10)
	require.NoError(t, err)

	assert.Equal(t, []map[string]any{{"seats": float64(11), "proration_behavior": "invoice"}}, f.patchBodies)
	f.assertNoAutomaticQuantityWrite(1)
	assert.Equal(t, 11, f.purchasedSeats())
}

func Test__PreviewAndSyncSettleALostPurchaseResponseWithoutAQuantityWrite(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.polarSeats = seats
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}
	response, err := f.purchase(1, 3)
	require.NoError(t, err)
	require.Equal(t, seatPurchaseAwaitingConfirmation, response.Purchase.Status)

	_, err = f.preview(1)
	require.NoError(t, err)
	_, err = SyncOrganizationBilling(t.Context(), f.r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
	require.NoError(t, err)

	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
	assert.Equal(t, 4, f.purchasedSeats())
	f.assertNoAutomaticQuantityWrite(1)
}

func Test__ASeatPurchaseInFlightBlocksNoRemoval(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	members := f.addMembers(1)
	f.dropConnection()
	// Polar never answered the purchase, so the subscription has no
	// modification that proves whether Polar applied it.
	f.modifiedAt = time.Now().Add(-time.Hour)
	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	require.Equal(t, billingmodels.SeatPurchaseStatusUnknown, f.latestPurchase().Status)

	f.removeMember(members[0])

	f.assertNoAutomaticQuantityWrite(1)
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, f.latestPurchase().Status,
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

func Test__ResumeKeepsThePaidSeatsWithoutAQuantityWrite(t *testing.T) {
	f := newSeatPurchaseFixture(t, 5, new(50))
	f.addMembers(1)
	f.setPeriodEnd(time.Now().Add(time.Hour).UTC().Truncate(time.Second))
	f.setToCancel()

	response, err := ResumeOrganizationSubscription(context.Background(), f.r.Organization.ID.String(), &pb.ResumeOrganizationSubscriptionRequest{})
	require.NoError(t, err)

	assert.False(t, response.CancelAtPeriodEnd)
	f.assertNoAutomaticQuantityWrite(0)
	assert.Equal(t, 5, f.purchasedSeats(), "the subscription renews every paid seat")
}
