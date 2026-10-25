package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
)

// seatLogs captures the process logger for one test.
type seatLogs struct {
	t    *testing.T
	hook *logtest.Hook
}

func captureSeatLogs(t *testing.T) *seatLogs {
	t.Helper()
	hook := &logtest.Hook{}
	previous := log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	log.AddHook(hook)
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(previous) })
	return &seatLogs{t: t, hook: hook}
}

// entries returns the lines for one stable key. Every seat line must carry
// the billing.seats component, so log-based alerts can filter on it.
func (l *seatLogs) entries(message string) []*log.Entry {
	l.t.Helper()
	var matched []*log.Entry
	for _, entry := range l.hook.AllEntries() {
		if !strings.HasPrefix(entry.Message, "billing.seats") {
			continue
		}
		require.Equal(l.t, logging.ComponentBillingSeats, entry.Data["component"], entry.Message)
		if entry.Message == message {
			matched = append(matched, entry)
		}
	}
	return matched
}

func (l *seatLogs) overCapacity() []*log.Entry {
	l.t.Helper()
	return l.entries(logging.SeatsOverCapacityMessage)
}

func (l *seatLogs) resolved() []*log.Entry {
	l.t.Helper()
	return l.entries(logging.SeatsOverCapacityResolvedMessage)
}

func (l *seatLogs) requireNone() {
	l.t.Helper()
	require.Empty(l.t, l.overCapacity())
	require.Empty(l.t, l.resolved())
}

func requireOverCapacityEntry(t *testing.T, entry *log.Entry, f *seatPurchaseFixture, cause string, purchased, occupied int) {
	t.Helper()
	require.Equal(t, log.ErrorLevel, entry.Level)
	require.Equal(t, f.r.Organization.ID.String(), entry.Data["organization_id"])
	require.Equal(t, f.subscriptionID, entry.Data["subscription_id"])
	require.Equal(t, cause, entry.Data["cause"])
	require.Equal(t, purchased, entry.Data["purchased"])
	require.Equal(t, occupied, entry.Data["occupied"])
}

func (f *seatPurchaseFixture) deliver(eventType string, data map[string]any) {
	f.t.Helper()
	f.replays++
	data["modified_at"] = time.Now().Add(time.Duration(f.replays) * time.Minute).UTC().Format(time.RFC3339)
	body, err := json.Marshal(map[string]any{"type": eventType, "data": data})
	require.NoError(f.t, err)
	var event polar.SubscriptionWebhookEvent
	require.NoError(f.t, json.Unmarshal(body, &event))
	require.NoError(f.t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
}

// replayCycled delivers the renewal webhook for a new billing period.
func (f *seatPurchaseFixture) replayCycled(seats int) {
	f.t.Helper()
	data := f.snapshot(seats)
	data["current_period_start"] = f.periodEnd.Format(time.RFC3339)
	data["current_period_end"] = f.periodEnd.AddDate(0, 1, 0).Format(time.RFC3339)
	f.deliver("subscription.cycled", data)
}

func (f *seatPurchaseFixture) addMembers(count int) []*models.User {
	f.t.Helper()
	users := make([]*models.User, 0, count)
	for range count {
		users = append(users, support.CreateUser(f.t, f.r, f.r.Organization.ID))
	}
	return users
}

func (f *seatPurchaseFixture) removeMember(user *models.User) {
	f.t.Helper()
	_, err := RemoveUser(context.Background(), f.r.AuthService, f.r.Organization.ID.String(), user.ID.String())
	require.NoError(f.t, err)
}

func (f *seatPurchaseFixture) overCapacitySince() *time.Time {
	f.t.Helper()
	return f.plan().OverCapacitySince
}

// returnToTrial puts the organization back on the plan that it has before it
// buys Team.
func (f *seatPurchaseFixture) returnToTrial() {
	f.t.Helper()
	db := database.Conn()
	trial, err := models.FindBillingPlan(db, models.BillingPlanTrial)
	require.NoError(f.t, err)
	f.updateBillingPlan(map[string]any{
		"plan_id": trial.ID, "plan": trial.Key, "plan_source": models.BillingPlanSourceSystem,
		"polar_subscription_id": nil, "polar_subscription_status": "", "purchased_seats": nil,
		"current_period_start": nil, "current_period_end": nil,
		"trial_ends_at": time.Now().Add(24 * time.Hour),
	})
	require.NoError(f.t, db.Model(&models.BillingPlan{}).Where("id = ?", f.catalog.ID).Update("visibility", "public").Error)
}

// useCheckoutServer lets SelectBillingPlan reach a fake checkout. Other
// requests go to the subscription server of the fixture.
func (f *seatPurchaseFixture) useCheckoutServer() {
	f.t.Helper()
	orgID := f.r.Organization.ID.String()
	server := polarAPIServer(f.t, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/customers/external/" + orgID:
			f.writeJSON(w, http.StatusOK, map[string]any{"id": "cust_1", "external_id": orgID})
		case "/checkouts/":
			f.writeJSON(w, http.StatusOK, map[string]any{"url": "https://polar.sh/checkout/team"})
		default:
			f.serve(w, req)
		}
	})
	usePolarTestServer(f.t, server)
}

func TestSeatsOverCapacityAfterCheckoutRaceAlertsOncePerTransition(t *testing.T) {
	f := newSeatPurchaseFixture(t, 1, new(50))
	logs := captureSeatLogs(t)
	orgID := f.r.Organization.ID
	f.returnToTrial()
	f.useCheckoutServer()
	service := InvitationService{AuthService: f.r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
	invite := func(email string) (*pb.CreateInvitationResponse, error) {
		return service.Create(context.Background(), orgID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: email})
	}

	// The checkout covers the current member. Two members join on the old plan
	// before the subscription activates.
	_, err := SelectBillingPlan(context.Background(), orgID.String(), &pb.SelectBillingPlanRequest{PlanKey: f.catalog.Key, Seats: new(int32(1))}, f.r.Account.ID.String(), "http://localhost")
	require.NoError(t, err)
	var joined []*models.User
	for range 2 {
		account, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
		require.NoError(t, err)
		invitation, err := invite(account.Email)
		require.NoError(t, err)
		_, err = service.Accept(context.Background(), account.ID.String(), strings.TrimPrefix(invitation.AcceptanceUrl, "https://example.com/invitations/"))
		require.NoError(t, err)
		joined = append(joined, findMemberByAccount(t, orgID, account.ID))
	}
	waiting, err := models.CreateAccount("Waiting", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	pending, err := invite(waiting.Email)
	require.NoError(t, err)
	logs.requireNone()

	f.replay(models.PolarSubscriptionStatusActive, 1)

	alerts := logs.overCapacity()
	require.Len(t, alerts, 1)
	requireOverCapacityEntry(t, alerts[0], f, "activation", 1, 3)
	require.NotNil(t, f.overCapacitySince())

	// A repeated snapshot does not alert again.
	f.replay(models.PolarSubscriptionStatusActive, 1)
	require.Len(t, logs.overCapacity(), 1)

	// Admission stays blocked.
	_, err = invite("late@example.com")
	requireInvitationReason(t, err, "SEAT_PURCHASE_REQUIRED")
	_, err = service.Accept(context.Background(), waiting.ID.String(), strings.TrimPrefix(pending.AcceptanceUrl, "https://example.com/invitations/"))
	requireInvitationReason(t, err, "NO_SEAT_AVAILABLE")

	// A removal that leaves more members than seats keeps the state.
	f.removeMember(joined[0])
	require.Empty(t, logs.resolved())
	require.NotNil(t, f.overCapacitySince())

	f.removeMember(joined[1])
	resolved := logs.resolved()
	require.Len(t, resolved, 1)
	require.Equal(t, log.InfoLevel, resolved[0].Level)
	require.Equal(t, "removal", resolved[0].Data["cause"])
	require.Equal(t, 1, resolved[0].Data["purchased"])
	require.Equal(t, 1, resolved[0].Data["occupied"])
	require.Nil(t, f.overCapacitySince())

	// The next transition alerts again.
	f.replay(models.PolarSubscriptionStatusActive, 0)
	alerts = logs.overCapacity()
	require.Len(t, alerts, 2)
	requireOverCapacityEntry(t, alerts[1], f, "update", 0, 1)
}

func findMemberByAccount(t *testing.T, orgID, accountID uuid.UUID) *models.User {
	t.Helper()
	var user models.User
	require.NoError(t, database.Conn().Where("organization_id = ? AND account_id = ?", orgID, accountID).First(&user).Error)
	return &user
}

func TestSeatsOverCapacityAlertsAgainWhenPaymentRecovers(t *testing.T) {
	for _, tc := range []struct {
		name      string
		graceDays int
	}{
		{name: "without grace"},
		{name: "during grace", graceDays: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 2, new(50))
			require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
				metadata.PastDueGraceDays = tc.graceDays
			}))
			logs := captureSeatLogs(t)
			f.addMembers(2)

			f.replay(models.PolarSubscriptionStatusActive, 2)
			alerts := logs.overCapacity()
			require.Len(t, alerts, 1)
			requireOverCapacityEntry(t, alerts[0], f, "update", 2, 3)

			// Past due is not a per-seat state, so it ends the record without a line.
			f.replay(models.PolarSubscriptionStatusPastDue, 2)
			require.Len(t, logs.overCapacity(), 1)
			require.Empty(t, logs.resolved())
			require.Nil(t, f.overCapacitySince())

			f.replay(models.PolarSubscriptionStatusActive, 2)
			alerts = logs.overCapacity()
			require.Len(t, alerts, 2)
			requireOverCapacityEntry(t, alerts[1], f, "recovery", 2, 3)
		})
	}
}

func TestSeatsOverCapacityAfterRenewalNeverChangesPolar(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	logs := captureSeatLogs(t)
	f.addMembers(1)

	f.replayCycled(1)

	alerts := logs.overCapacity()
	require.Len(t, alerts, 1)
	requireOverCapacityEntry(t, alerts[0], f, "cycle", 1, 2)
	require.Zero(t, f.patchCount())
}

func TestSeatsOverCapacityResolvesWhenSeatsCoverMembers(t *testing.T) {
	f := newSeatPurchaseFixture(t, 1, new(50))
	logs := captureSeatLogs(t)
	f.addMembers(1)

	f.replay(models.PolarSubscriptionStatusActive, 1)
	require.Len(t, logs.overCapacity(), 1)

	f.replay(models.PolarSubscriptionStatusActive, 2)
	resolved := logs.resolved()
	require.Len(t, resolved, 1)
	require.Equal(t, "update", resolved[0].Data["cause"])
	require.Nil(t, f.overCapacitySince())
}

func TestSeatsOverCapacityUsesFinalVerifiedSubscription(t *testing.T) {
	for _, tc := range []struct {
		name               string
		storedStillLive    bool
		verificationFails  bool
		replacementStatus  string
		replacementSeats   int
		wantOverCapacity   bool
		wantResolutionLogs int
	}{
		{name: "conflicting live subscription is ignored", storedStillLive: true, replacementStatus: "active", replacementSeats: 2, wantOverCapacity: true},
		{name: "replacement remains over capacity", replacementStatus: "active", replacementSeats: 1, wantOverCapacity: true},
		{name: "replacement covers members", replacementStatus: "active", replacementSeats: 2, wantResolutionLogs: 1},
		{name: "ended replacement clears silently", replacementStatus: "canceled", replacementSeats: 2},
		{name: "failed verification rolls back", verificationFails: true, wantOverCapacity: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 1, new(50))
			logs := captureSeatLogs(t)
			f.addMembers(1)
			f.replay(models.PolarSubscriptionStatusActive, 1)
			since := f.overCapacitySince()
			require.NotNil(t, since)
			logs.hook.Reset()
			replacementID := "sub_" + uuid.NewString()
			server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodGet, req.Method)
				data := f.snapshot(1)
				data["modified_at"] = time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339)
				switch req.URL.Path {
				case "/subscriptions/" + f.subscriptionID:
					data["status"] = models.PolarSubscriptionStatusCanceled
					if tc.storedStillLive {
						data["status"] = models.PolarSubscriptionStatusActive
					}
				case "/subscriptions/" + replacementID:
					if tc.verificationFails {
						http.Error(w, "verification failed", http.StatusInternalServerError)
						return
					}
					data["id"] = replacementID
					data["status"] = tc.replacementStatus
					data["seats"] = tc.replacementSeats
				default:
					t.Errorf("unexpected Polar request: %s", req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				f.writeJSON(w, http.StatusOK, data)
			})
			usePolarTestServer(t, server)

			// The unverified event claims sufficient seats. Only the verified
			// final snapshot can change capacity or produce a resolution log.
			incoming := f.snapshot(2)
			incoming["id"] = replacementID
			body, err := json.Marshal(incoming)
			require.NoError(t, err)
			var data polar.SubscriptionData
			require.NoError(t, json.Unmarshal(body, &data))
			err = polar.ApplySubscription(t.Context(), database.Conn(), data)
			if tc.verificationFails {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			plan := f.plan()
			if tc.wantOverCapacity {
				require.NotNil(t, plan.OverCapacitySince)
				require.True(t, since.Equal(*plan.OverCapacitySince))
			} else {
				require.Nil(t, plan.OverCapacitySince)
			}
			if tc.storedStillLive || tc.verificationFails || tc.replacementStatus == "canceled" {
				require.Equal(t, f.subscriptionID, *plan.PolarSubscriptionID)
			} else {
				require.Equal(t, replacementID, *plan.PolarSubscriptionID)
			}
			require.Empty(t, logs.overCapacity())
			resolved := logs.resolved()
			require.Len(t, resolved, tc.wantResolutionLogs)
			if len(resolved) > 0 {
				require.Equal(t, replacementID, resolved[0].Data["subscription_id"])
				require.Equal(t, 2, resolved[0].Data["purchased"])
				require.Equal(t, 2, resolved[0].Data["occupied"])
			}
		})
	}
}

func TestSeatsOverCapacityClearsSilentlyWhenPlanHasNoSeats(t *testing.T) {
	f := newSeatPurchaseFixture(t, 1, new(50))
	logs := captureSeatLogs(t)
	f.addMembers(1)
	f.replay(models.PolarSubscriptionStatusActive, 1)
	require.Len(t, logs.overCapacity(), 1)

	// A plan without seats has no capacity state to report.
	business := saveBusinessPlan(t)
	data := f.snapshot(1)
	data["product_id"] = *business.PolarProductID
	data["seats"] = nil
	f.deliver("subscription.updated", data)

	require.Len(t, logs.overCapacity(), 1)
	require.Empty(t, logs.resolved())
	require.Nil(t, f.overCapacitySince())
}

func TestSeatsOverCapacityIsSilentWhenBillingIsOff(t *testing.T) {
	t.Setenv(models.BillingEnabledEnv, "no")
	f := newSeatPurchaseFixture(t, 1, new(50))
	logs := captureSeatLogs(t)
	members := f.addMembers(2)

	f.replay(models.PolarSubscriptionStatusActive, 1)
	f.removeMember(members[0])
	f.removeMember(members[1])

	logs.requireNone()
	require.Nil(t, f.overCapacitySince())
}
