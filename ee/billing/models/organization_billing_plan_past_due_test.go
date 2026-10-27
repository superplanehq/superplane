package models_test

import (
	"fmt"
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

const day = 24 * time.Hour

// pastDueProduct is a paid Polar product. The legacy Business product has no
// seats. The Team product is per seat.
type pastDueProduct struct {
	name    string
	perSeat bool
}

var pastDueProducts = []pastDueProduct{
	{name: "Business"},
	{name: "Team", perSeat: true},
}

// pastDueReplay delivers Polar subscription snapshots for one organization in
// order. The first snapshot activates the paid plan.
type pastDueReplay struct {
	t          *testing.T
	db         *gorm.DB
	orgID      uuid.UUID
	productID  string
	seats      *int
	modifiedAt time.Time
}

func newPastDueReplay(t *testing.T, product pastDueProduct) *pastDueReplay {
	t.Helper()
	r := support.Setup(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	replay := &pastDueReplay{t: t, db: db, orgID: r.Organization.ID, modifiedAt: time.Now().Add(-time.Hour)}
	if !product.perSeat {
		replay.productID = "prod_business"
		require.NoError(t, billingmodels.BindLegacyBillingProduct(db, replay.productID))
		return replay
	}
	replay.productID = "prod_team_" + uuid.NewString()
	replay.seats = new(3)
	catalog := billingmodels.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public", PerSeat: true,
		PriceCents: 2000, SeatLimit: new(50), PolarProductID: &replay.productID,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))
	return replay
}

func (p *pastDueReplay) apply(status string, mutate ...func(*billingmodels.PolarSubscriptionApply)) *billingmodels.OrganizationBillingPlan {
	p.t.Helper()
	p.modifiedAt = p.modifiedAt.Add(time.Minute)
	start := time.Now().Add(-7 * day)
	end := time.Now().Add(23 * day)
	sub := billingmodels.PolarSubscriptionApply{
		ID: "sub_past_due", ProductID: p.productID, Status: status,
		PeriodStart: &start, PeriodEnd: &end, ModifiedAt: &p.modifiedAt, PurchasedSeats: p.seats,
	}
	for _, change := range mutate {
		change(&sub)
	}
	result, err := billingmodels.ApplyPolarSubscription(p.db, p.orgID, sub, nil)
	require.NoError(p.t, err)
	require.Equal(p.t, billingmodels.PolarSubscriptionApplied, result.Disposition)
	return p.resolve()
}

func (p *pastDueReplay) resolve() *billingmodels.OrganizationBillingPlan {
	p.t.Helper()
	plan, err := billingmodels.ResolveOrganizationBillingPlan(p.db, p.orgID)
	require.NoError(p.t, err)
	return plan
}

// endTrial removes the open trial, so a lapse shows as None.
func (p *pastDueReplay) endTrial() {
	p.t.Helper()
	require.NoError(p.t, p.db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", p.orgID).
		Update("trial_ends_at", time.Now().Add(-day)).Error)
}

// failedAgo moves the stored start of the failure into the past.
func (p *pastDueReplay) failedAgo(age time.Duration) {
	p.t.Helper()
	require.NoError(p.t, p.db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", p.orgID).
		Update("past_due_since", time.Now().Add(-age)).Error)
}

func (p *pastDueReplay) setGraceDays(days int) {
	p.t.Helper()
	require.NoError(p.t, billingmodels.UpdatePastDueGraceDays(p.db, days))
}

func failedAt(at time.Time) func(*billingmodels.PolarSubscriptionApply) {
	return func(sub *billingmodels.PolarSubscriptionApply) { sub.PastDueAt = &at }
}

func cancelAtPeriodEnd(sub *billingmodels.PolarSubscriptionApply) { sub.CancelAtPeriodEnd = true }

func requireLapsed(t *testing.T, plan *billingmodels.OrganizationBillingPlan) {
	t.Helper()
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.AllowsHostedExecution(time.Now()))
}

func requirePaid(t *testing.T, plan *billingmodels.OrganizationBillingPlan, product pastDueProduct) {
	t.Helper()
	assert.Equal(t, product.perSeat, plan.Catalog.PerSeat)
	assert.Equal(t, "paid", plan.Catalog.Category)
	assert.True(t, plan.IsActiveBusiness())
	assert.True(t, plan.AllowsHostedExecution(time.Now()))
}

func Test__FailedPaymentWithCancelAtPeriodEndGivesNoPaidAccess(t *testing.T) {
	for _, status := range []string{billingmodels.PolarSubscriptionStatusPastDue, billingmodels.PolarSubscriptionStatusUnpaid} {
		for _, product := range pastDueProducts {
			t.Run(status+" "+product.name, func(t *testing.T) {
				replay := newPastDueReplay(t, product)
				replay.endTrial()
				requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), product)

				requireLapsed(t, replay.apply(status, cancelAtPeriodEnd))
				stored, err := billingmodels.FindOrganizationBillingPlan(replay.db, replay.orgID)
				require.NoError(t, err)
				requireLapsed(t, stored)
			})
		}
	}
}

func Test__ActiveSubscriptionSetToCancelKeepsPaidAccessUntilPeriodEnd(t *testing.T) {
	for _, product := range pastDueProducts {
		t.Run(product.name, func(t *testing.T) {
			replay := newPastDueReplay(t, product)
			replay.endTrial()

			requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive, cancelAtPeriodEnd), product)
		})
	}
}

func Test__PastDueGraceKeepsPaidPlanUntilItEnds(t *testing.T) {
	for _, product := range pastDueProducts {
		t.Run(product.name, func(t *testing.T) {
			replay := newPastDueReplay(t, product)
			replay.endTrial()
			replay.setGraceDays(7)
			requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), product)

			failure := time.Now().Add(-day).Truncate(time.Second)
			plan := replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(failure), cancelAtPeriodEnd)
			requirePaid(t, plan, product)
			require.NotNil(t, plan.PastDueSince)
			assert.True(t, plan.PastDueSince.Equal(failure))
			require.NotNil(t, plan.PastDueAccessEndsAt(7))
			assert.True(t, plan.PastDueAccessEndsAt(7).Equal(failure.Add(7*day)))

			replay.failedAgo(8 * day)
			requireLapsed(t, replay.resolve())
			stored, err := billingmodels.FindOrganizationBillingPlan(replay.db, replay.orgID)
			require.NoError(t, err)
			requireLapsed(t, stored)
			require.NotNil(t, stored.PastDueSince, "the failure start stays for the billing page")
			assert.Equal(t, billingmodels.PolarSubscriptionStatusPastDue, stored.PolarSubscriptionStatus)
		})
	}
}

func Test__PastDueGraceAlreadyEndedLapsesOnApply(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.endTrial()
	replay.setGraceDays(7)
	replay.apply(billingmodels.PolarSubscriptionStatusActive)

	requireLapsed(t, replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(time.Now().Add(-8*day))))
}

func Test__PastDueGraceFollowsTheCurrentSetting(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.endTrial()
	replay.setGraceDays(7)
	replay.apply(billingmodels.PolarSubscriptionStatusActive)
	requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(time.Now().Add(-2*day))), pastDueProducts[1])

	replay.setGraceDays(1)

	requireLapsed(t, replay.resolve())
}

func Test__PastDueGraceKeepsOpenTrialAfterLapse(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.setGraceDays(7)
	replay.apply(billingmodels.PolarSubscriptionStatusActive)
	replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(time.Now().Add(-day)))
	replay.failedAgo(8 * day)

	plan := replay.resolve()

	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.AllowsHostedExecution(time.Now()))
}

func Test__ActiveSnapshotRestoresPaidPlanAfterPastDue(t *testing.T) {
	for _, graceDays := range []int{0, 7} {
		for _, product := range pastDueProducts {
			t.Run(fmt.Sprintf("%s grace %d", product.name, graceDays), func(t *testing.T) {
				replay := newPastDueReplay(t, product)
				replay.endTrial()
				replay.setGraceDays(graceDays)
				replay.apply(billingmodels.PolarSubscriptionStatusActive)
				replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(time.Now().Add(-8*day)))
				requireLapsed(t, replay.resolve())

				replay.seats = new(5)
				restored := replay.apply(billingmodels.PolarSubscriptionStatusActive)

				requirePaid(t, restored, product)
				assert.Nil(t, restored.PastDueSince)
				if product.perSeat {
					assert.Equal(t, 5, *restored.PurchasedSeats)
				}
			})
		}
	}
}

func Test__PastDueSinceMarksOneFailure(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.setGraceDays(7)
	replay.apply(billingmodels.PolarSubscriptionStatusActive)

	first := replay.apply(billingmodels.PolarSubscriptionStatusPastDue)
	require.NotNil(t, first.PastDueSince)
	assert.WithinDuration(t, time.Now(), *first.PastDueSince, time.Minute, "a snapshot without past_due_at starts the failure now")

	retried := replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(time.Now().Add(-2*day)))
	assert.True(t, retried.PastDueSince.Equal(*first.PastDueSince), "a retry does not move the failure start")

	replay.apply(billingmodels.PolarSubscriptionStatusActive)
	failure := time.Now().Add(-3 * day).Truncate(time.Second)
	next := replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(failure))
	assert.True(t, next.PastDueSince.Equal(failure), "a later failure starts a new period")
}

func Test__NonPayingStatusesLapseDespiteGrace(t *testing.T) {
	for _, status := range []string{
		billingmodels.PolarSubscriptionStatusCanceled, billingmodels.PolarSubscriptionStatusUnpaid, billingmodels.PolarSubscriptionStatusRevoked,
	} {
		for _, product := range pastDueProducts {
			t.Run(status+" "+product.name, func(t *testing.T) {
				replay := newPastDueReplay(t, product)
				replay.endTrial()
				replay.setGraceDays(7)
				replay.apply(billingmodels.PolarSubscriptionStatusActive)
				replay.apply(billingmodels.PolarSubscriptionStatusPastDue, failedAt(time.Now().Add(-day)))

				plan := replay.apply(status)

				requireLapsed(t, plan)
				assert.Nil(t, plan.PastDueSince)
			})
		}
	}
}

func Test__PausedSubscriptionLapsesPaidPlan(t *testing.T) {
	for _, product := range pastDueProducts {
		t.Run(product.name, func(t *testing.T) {
			replay := newPastDueReplay(t, product)
			replay.endTrial()
			requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), product)

			for _, mutate := range [][]func(*billingmodels.PolarSubscriptionApply){nil, {cancelAtPeriodEnd}} {
				plan := replay.apply(billingmodels.PolarSubscriptionStatusPaused, mutate...)

				requireLapsed(t, plan)
				assert.Nil(t, plan.PastDueSince)
				assert.Equal(t, billingmodels.PolarSubscriptionStatusPaused, plan.PolarSubscriptionStatus, "the status stays for the billing page and the seat lock")
				if product.perSeat {
					require.NotNil(t, plan.PurchasedSeats, "the stored seats stay for the seat lock")
					assert.Equal(t, 3, *plan.PurchasedSeats)
				}
			}

			stored, err := billingmodels.FindOrganizationBillingPlan(replay.db, replay.orgID)
			require.NoError(t, err)
			requireLapsed(t, stored)
		})
	}
}

// Rows written before past_due_since existed, or before paused and unpaid
// subscriptions lapsed the plan, still sit on the paid catalog. Resolve lapses
// them, and the stored subscription details and seats stay for the billing page
// and the seat lock.
func Test__PaymentBlockedRowWithoutGraceLapsesOnResolve(t *testing.T) {
	for _, status := range []string{billingmodels.PolarSubscriptionStatusPastDue, billingmodels.PolarSubscriptionStatusPaused, billingmodels.PolarSubscriptionStatusUnpaid} {
		for _, product := range pastDueProducts {
			t.Run(status+" "+product.name, func(t *testing.T) {
				replay := newPastDueReplay(t, product)
				replay.endTrial()
				requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), product)

				require.NoError(t, replay.db.Model(&billingmodels.OrganizationBillingPlan{}).
					Where("organization_id = ?", replay.orgID).
					Updates(map[string]any{"polar_subscription_status": status, "cancel_at_period_end": true}).Error)

				plan := replay.resolve()

				requireLapsed(t, plan)
				assert.Equal(t, status, plan.PolarSubscriptionStatus, "the status stays for the billing page and the seat lock")
				assert.Nil(t, plan.PastDueSince)
				if product.perSeat {
					require.NotNil(t, plan.PurchasedSeats, "the stored seats stay for the seat lock")
					assert.Equal(t, 3, *plan.PurchasedSeats)
				}
				stored, err := billingmodels.FindOrganizationBillingPlan(replay.db, replay.orgID)
				require.NoError(t, err)
				requireLapsed(t, stored)
			})
		}
	}
}

// A past-due row without a recorded failure start cannot measure a grace
// period, so it lapses even when the installation allows grace days.
func Test__PastDueRowWithoutFailureStartGetsNoGrace(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.endTrial()
	replay.setGraceDays(7)
	requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), pastDueProducts[1])

	require.NoError(t, replay.db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", replay.orgID).
		Update("polar_subscription_status", billingmodels.PolarSubscriptionStatusPastDue).Error)

	requireLapsed(t, replay.resolve())
}

// Paid catalog features must not survive the lapse: a payment-blocked row
// resolves features from the lapsed catalog, not from the paid one.
func Test__PaymentBlockedLapseDropsPaidCatalogFeatures(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.endTrial()

	feature := billingmodels.PlanFeatureDefinition{Key: "paid_feature_" + uuid.NewString()[:8], Label: "Paid feature", Gated: true}
	require.NoError(t, billingmodels.CreatePlanFeatureDefinition(replay.db, &feature))
	catalog, err := billingmodels.FindBillingPlanByProduct(replay.db, replay.productID)
	require.NoError(t, err)
	catalog.Features = append(catalog.Features, feature.Key)
	require.NoError(t, billingmodels.SaveBillingPlan(replay.db, catalog))

	features := func() map[string]bool {
		resolved, err := billingmodels.ResolveOrganizationProductFeatures(replay.db, replay.orgID)
		require.NoError(t, err)
		return resolved
	}

	requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), pastDueProducts[1])
	assert.True(t, features()[feature.Key], "the paid catalog grants the feature")

	replay.apply(billingmodels.PolarSubscriptionStatusPaused)
	assert.False(t, features()[feature.Key], "a paused snapshot lapses the paid catalog")

	for _, status := range []string{billingmodels.PolarSubscriptionStatusPastDue, billingmodels.PolarSubscriptionStatusUnpaid} {
		requirePaid(t, replay.apply(billingmodels.PolarSubscriptionStatusActive), pastDueProducts[1])
		require.True(t, features()[feature.Key])
		require.NoError(t, replay.db.Model(&billingmodels.OrganizationBillingPlan{}).
			Where("organization_id = ?", replay.orgID).
			Updates(map[string]any{"polar_subscription_status": status, "cancel_at_period_end": true}).Error)
		assert.False(t, features()[feature.Key], "%s without grace lapses on resolve", status)
	}
}

func Test__AgentRunGateNamesPaymentFailure(t *testing.T) {
	replay := newPastDueReplay(t, pastDueProducts[1])
	replay.endTrial()
	replay.apply(billingmodels.PolarSubscriptionStatusActive)
	require.NoError(t, billingmodels.AssertOrganizationPlanAllowsRuns(replay.db, replay.orgID))

	replay.apply(billingmodels.PolarSubscriptionStatusPastDue)
	err := billingmodels.AssertOrganizationPlanAllowsRuns(replay.db, replay.orgID)
	require.ErrorIs(t, err, billingmodels.ErrHostedPaymentPastDue)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired, "callers that match the broader error keep working")
	assert.Equal(t, models.SuperPlaneRunnerPaymentPastDueMessage, models.SuperPlaneRunnerReadinessMessage(err))
	assert.Equal(t, models.WorkOrderExecutionFailureReasonPaymentPastDue, models.WorkOrderExecutionFailureReasonFor(err))

	replay.apply(billingmodels.PolarSubscriptionStatusPaused)
	err = models.SuperPlaneRunnerReadinessError(replay.db, replay.orgID, nil)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
	require.NotErrorIs(t, err, billingmodels.ErrHostedPaymentPastDue)
	assert.Equal(t, "The subscription is paused. Resume the subscription in Billing to run agents.", models.SuperPlaneRunnerReadinessMessage(err))
	assert.Equal(t, "subscription_paused", models.WorkOrderExecutionFailureReasonFor(err))

	replay.apply(billingmodels.PolarSubscriptionStatusCanceled)
	err = billingmodels.AssertOrganizationPlanAllowsRuns(replay.db, replay.orgID)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
	require.NotErrorIs(t, err, billingmodels.ErrHostedPaymentPastDue)
	assert.Equal(t, models.SuperPlaneRunnerSubscribeMessage, models.SuperPlaneRunnerReadinessMessage(err))
}
