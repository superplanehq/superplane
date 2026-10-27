package seatremoval

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/ee/billing/seatquantity"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestScheduleLowerRestrictsAdmissionBeforePolarConfirms(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	fx.client.onSchedule = func() {
		plan := fx.plan()
		require.NotNil(t, plan.SeatAdmissionLimit)
		assert.Equal(t, 7, *plan.SeatAdmissionLimit)
		assert.Nil(t, plan.SeatProvedTarget, "the lower limit is not proved until Polar confirms it")
	}

	result, err := fx.runner.Schedule(context.Background(), fx.request(7))

	require.NoError(t, err)
	require.Equal(t, []int{7}, fx.client.patches)
	assert.Equal(t, billingmodels.SeatQuantityStatusScheduled, result.Operation.Status)
	plan := fx.plan()
	require.NotNil(t, plan.SeatProvedTarget)
	assert.Equal(t, 7, *plan.SeatProvedTarget)
	assert.Equal(t, 7, *plan.SeatAdmissionLimit)
	assert.Equal(t, 10, *plan.PurchasedSeats)
	assert.Equal(t, int64(1), plan.SeatChangeVersion)
}

func TestRemovalReplayDoesNotSendASecondUpdate(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	req := fx.request(7)
	first, err := fx.runner.Schedule(context.Background(), req)
	require.NoError(t, err)

	second, err := fx.runner.Schedule(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, first.Operation.ID, second.Operation.ID)
	assert.Equal(t, []int{7}, fx.client.patches)
}

func TestRemovalRejectsInsideTheCutoffAndRestoresNothingNew(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	now := fx.periodEnd.Add(-seatquantity.MutationGuard)
	fx.runner.Now = func() time.Time { return now }

	_, err := fx.runner.Schedule(context.Background(), fx.request(7))

	require.Error(t, err)
	assert.Equal(t, ReasonRenewalBoundary, removalReason(t, err))
	assert.Empty(t, fx.client.patches)
	assert.Equal(t, int64(0), fx.plan().SeatChangeVersion)
}

func TestRemovalAllowsTheInstantBeforeTheCutoff(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	fx.runner.Now = func() time.Time { return fx.periodEnd.Add(-seatquantity.MutationGuard - time.Second) }

	_, err := fx.runner.Schedule(context.Background(), fx.request(7))

	require.NoError(t, err)
	assert.Equal(t, []int{7}, fx.client.patches)
}

func TestTimeAdvancingBeforeDispatchRejectsWithoutAPatch(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	calls := 0
	allowed := fx.periodEnd.Add(-seatquantity.MutationGuard - time.Minute)
	denied := fx.periodEnd.Add(-seatquantity.MutationGuard)
	fx.runner.Now = func() time.Time {
		calls++
		if calls >= 3 {
			return denied
		}
		return allowed
	}

	_, err := fx.runner.Schedule(context.Background(), fx.request(7))

	require.Error(t, err)
	assert.Equal(t, ReasonRenewalBoundary, removalReason(t, err))
	assert.Empty(t, fx.client.patches)
	plan := fx.plan()
	assert.Nil(t, plan.SeatAdmissionLimit)
	assert.Nil(t, plan.SeatProvedTarget)
	assert.Greater(t, plan.SeatChangeVersion, int64(0), "the failed attempt still advances the version")
}

func TestDefinitiveRejectionRestoresThePreviousAdmission(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(7))
	require.NoError(t, err)
	fx.client.fail = polar.ErrInvalidRequest

	_, err = fx.runner.Change(context.Background(), fx.request(5))

	require.Error(t, err)
	plan := fx.plan()
	require.NotNil(t, plan.SeatAdmissionLimit)
	assert.Equal(t, 7, *plan.SeatAdmissionLimit)
	assert.Equal(t, 7, *plan.SeatProvedTarget)
	assert.Equal(t, []int{7, 5}, fx.client.patches)
	assert.Equal(t, 7, *fx.client.pending, "Polar keeps the proved schedule when it rejects the change")
}

func TestUnknownChangeKeepsTheLowerLimitAndBlocksAnotherWrite(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(7))
	require.NoError(t, err)
	fx.client.fail = polar.ErrOutcomeUnknown

	unknown, err := fx.runner.Change(context.Background(), fx.request(5))

	require.NoError(t, err)
	assert.Equal(t, billingmodels.SeatQuantityStatusUnknown, unknown.Operation.Status)
	plan := fx.plan()
	assert.Equal(t, 5, *plan.SeatAdmissionLimit, "a lower unconfirmed target restricts admission immediately")
	_, err = fx.runner.Change(context.Background(), fx.request(6))
	assert.Equal(t, ReasonOutcomeUnknown, removalReason(t, err))
	assert.Equal(t, []int{7, 5}, fx.client.patches, "the unconfirmed change is not sent again")
}

func TestRaiseKeepsTheOldLimitUntilPolarConfirms(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(5))
	require.NoError(t, err)
	fx.client.onSchedule = func() {
		assert.Equal(t, 5, *fx.plan().SeatAdmissionLimit)
	}

	_, err = fx.runner.Change(context.Background(), fx.request(8))

	require.NoError(t, err)
	assert.Equal(t, 8, *fx.plan().SeatAdmissionLimit)
	assert.Equal(t, 8, *fx.plan().SeatProvedTarget)
}

func TestCancelClearsOnlyTheSeatSchedule(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(7))
	require.NoError(t, err)

	result, err := fx.runner.Cancel(context.Background(), fx.request(10))

	require.NoError(t, err)
	assert.Equal(t, billingmodels.SeatQuantityStatusCanceled, result.Operation.Status)
	assert.Nil(t, fx.plan().SeatAdmissionLimit)
	assert.Nil(t, fx.plan().SeatProvedTarget)
	assert.Equal(t, []int{7, 10}, fx.client.patches)
	assert.Equal(t, 10, fx.client.seats, "the current period keeps its paid seats")
}

func TestSelectingPaidSeatsCancelsTheRemoval(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(7))
	require.NoError(t, err)

	_, err = fx.runner.Change(context.Background(), fx.request(10))

	require.NoError(t, err)
	assert.Nil(t, fx.plan().SeatProvedTarget)
}

func TestTargetBelowTheMemberFloorIsRejected(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	support.CreateUser(t, fx.resources, fx.resources.Organization.ID)

	_, err := fx.runner.Schedule(context.Background(), fx.request(1))

	require.Error(t, err)
	assert.Equal(t, ReasonInvalidTarget, removalReason(t, err))
	assert.Empty(t, fx.client.patches)
}

func TestMixedPendingUpdateDoesNotWrite(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	fx.client.extraPending = map[string]any{"product_id": "prod_other"}

	_, err := fx.runner.Schedule(context.Background(), fx.request(7))

	require.Error(t, err)
	assert.Equal(t, ReasonMixedPending, removalReason(t, err))
	assert.Empty(t, fx.client.patches)
	assert.Equal(t, int64(0), fx.plan().SeatChangeVersion)
}

func TestPreviewInsideTheCutoffDoesNotMutate(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	fx.runner.Now = func() time.Time { return fx.periodEnd.Add(-time.Minute) }

	preview, err := fx.runner.Preview(context.Background(), fx.resources.Organization.ID, 7)

	require.NoError(t, err)
	assert.False(t, preview.Allowed)
	assert.Equal(t, ReasonRenewalBoundary, preview.Reason)
	assert.True(t, preview.NoCurrentCharge)
	assert.Equal(t, int64(0), preview.CurrentChargeCents)
	assert.Empty(t, fx.client.patches)
	view, err := Describe(database.Conn(), fx.resources.Organization.ID, fx.runner.now())
	require.NoError(t, err)
	assert.Contains(t, view.Reasons, ReasonRenewalBoundary)
}

func TestAcceptanceAfterSchedulePreservesTheFloor(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(1))
	require.NoError(t, err)
	support.CreateUser(t, fx.resources, fx.resources.Organization.ID)

	_, err = fx.runner.Change(context.Background(), fx.request(1))

	assert.Equal(t, ReasonInvalidTarget, removalReason(t, err))
}

func TestProtectiveCancelRestoresPaidSeatsWithoutBuyingOrRemovingMembers(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	_, err := fx.runner.Schedule(context.Background(), fx.request(7))
	require.NoError(t, err)
	for range 7 {
		support.CreateUser(t, fx.resources, fx.resources.Organization.ID)
	}

	require.NoError(t, Recover(context.Background(), database.Conn(), fx.client, fx.resources.Organization.ID))

	assert.Nil(t, fx.plan().SeatProvedTarget)
	assert.Nil(t, fx.plan().SeatAdmissionLimit)
	assert.Equal(t, []int{7, 10}, fx.client.patches)
	assert.Equal(t, 10, fx.client.seats)
	count, err := models.CountActiveHumanUsersByOrganization(database.Conn(), fx.resources.Organization.ID.String())
	require.NoError(t, err)
	assert.EqualValues(t, 8, count)
}

func TestMembersAbovePaidSeatsStayAndBlockNewUse(t *testing.T) {
	fx := newRemovalFixture(t, 2)
	_, err := fx.runner.Schedule(context.Background(), fx.request(1))
	require.NoError(t, err)
	support.CreateUser(t, fx.resources, fx.resources.Organization.ID)
	support.CreateUser(t, fx.resources, fx.resources.Organization.ID)

	require.NoError(t, Recover(context.Background(), database.Conn(), fx.client, fx.resources.Organization.ID))

	assert.Equal(t, []int{1}, fx.client.patches, "recovery does not buy seats")
	assert.Equal(t, 2, fx.client.seats)
	view, err := Describe(database.Conn(), fx.resources.Organization.ID, time.Now())
	require.NoError(t, err)
	assert.Equal(t, billingmodels.SeatQuantityStatusRepairRequired, view.Status)
	count, err := models.CountActiveHumanUsersByOrganization(database.Conn(), fx.resources.Organization.ID.String())
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)
}

func TestScheduleAfterMemberJoinPreservesTheFloor(t *testing.T) {
	fx := newRemovalFixture(t, 10)
	support.CreateUser(t, fx.resources, fx.resources.Organization.ID)

	_, err := fx.runner.Schedule(context.Background(), fx.request(1))

	assert.Equal(t, ReasonInvalidTarget, removalReason(t, err))
	assert.Empty(t, fx.client.patches)
}

type removalFixture struct {
	t           *testing.T
	resources   *support.ResourceRegistry
	client      *fakeSeats
	runner      *Runner
	periodStart time.Time
	periodEnd   time.Time
}

func newRemovalFixture(t *testing.T, paid int) *removalFixture {
	t.Helper()
	resources := support.Setup(t)
	t.Cleanup(resources.Close)
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	now := time.Now().UTC().Truncate(time.Second)
	fx := &removalFixture{
		t: t, resources: resources,
		periodStart: now.Add(-7 * 24 * time.Hour),
		periodEnd:   now.Add(20 * 24 * time.Hour),
	}
	product := "prod_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
		PerSeat: true, PriceCents: 2000, SeatLimit: new(50), PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	subscriptionID := "sub_" + uuid.NewString()
	require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", resources.Organization.ID).Updates(map[string]any{
		"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
		"polar_subscription_id": subscriptionID, "polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
		"purchased_seats": paid, "current_period_start": fx.periodStart, "current_period_end": fx.periodEnd,
		"cancel_at_period_end": false,
	}).Error)
	fx.client = &fakeSeats{id: subscriptionID, seats: paid, status: "active", periodStart: fx.periodStart, periodEnd: fx.periodEnd}
	fx.runner = NewRunner(database.Conn(), fx.client)
	return fx
}

func (f *removalFixture) request(target int) Request {
	f.t.Helper()
	plan := f.plan()
	user := f.resources.User
	return Request{
		OrganizationID: f.resources.Organization.ID, ActorUserID: &user, OperationKey: uuid.NewString(),
		Target: target, SubscriptionID: *plan.PolarSubscriptionID, PeriodStart: *plan.CurrentPeriodStart,
		PeriodEnd: *plan.CurrentPeriodEnd, SourcePaidQuantity: *plan.PurchasedSeats, SeatChangeVersion: plan.SeatChangeVersion,
	}
}

func (f *removalFixture) plan() *billingmodels.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), f.resources.Organization.ID)
	require.NoError(f.t, err)
	return plan
}

func removalReason(t *testing.T, err error) string {
	t.Helper()
	var status *StatusError
	require.ErrorAs(t, err, &status)
	return status.Reason
}

type fakeSeats struct {
	id           string
	seats        int
	pending      *int
	status       string
	periodStart  time.Time
	periodEnd    time.Time
	patches      []int
	fail         error
	extraPending map[string]any
	onSchedule   func()
}

func (f *fakeSeats) GetSubscription(context.Context, string) (*polar.SubscriptionData, error) {
	return f.snapshot(), nil
}

func (f *fakeSeats) ScheduleSubscriptionSeats(_ context.Context, _ string, seats int) (*polar.SubscriptionData, error) {
	if f.onSchedule != nil {
		f.onSchedule()
	}
	f.patches = append(f.patches, seats)
	if f.fail != nil {
		err := f.fail
		f.fail = nil
		return nil, err
	}
	if seats == f.seats {
		f.pending = nil
	} else {
		f.pending = &seats
	}
	return f.snapshot(), nil
}

func (f *fakeSeats) snapshot() *polar.SubscriptionData {
	pending := "null"
	if len(f.extraPending) > 0 {
		body, err := json.Marshal(f.extraPending)
		if err != nil {
			panic(err)
		}
		pending = string(body)
	} else if f.pending != nil {
		pending = fmt.Sprintf(`{"seats":%d,"applies_at":%q}`, *f.pending, f.periodEnd.Format(time.RFC3339))
	}
	raw := fmt.Sprintf(`{"id":%q,"status":%q,"seats":%d,"current_period_start":%q,"current_period_end":%q,"modified_at":%q,"pending_update":%s}`,
		f.id, f.status, f.seats, f.periodStart.Format(time.RFC3339), f.periodEnd.Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339Nano), pending)
	var data polar.SubscriptionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		panic(err)
	}
	return &data
}
