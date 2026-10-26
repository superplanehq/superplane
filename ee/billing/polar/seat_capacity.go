package polar

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/logging"
	"gorm.io/gorm"
)

// SeatCapacityCause says what made a capacity check run. It is a log field.
type SeatCapacityCause string

const (
	SeatCapacityCauseActivation     SeatCapacityCause = "activation"
	SeatCapacityCauseUpdate         SeatCapacityCause = "update"
	SeatCapacityCauseCycle          SeatCapacityCause = "cycle"
	SeatCapacityCauseRecovery       SeatCapacityCause = "recovery"
	SeatCapacityCauseRemoval        SeatCapacityCause = "removal"
	SeatCapacityCauseAccountDeleted SeatCapacityCause = "account_deletion"
)

// SeatCapacityCheck is a capacity transition and the cause to log with it.
type SeatCapacityCheck struct {
	OrganizationID uuid.UUID
	Cause          SeatCapacityCause
	Transition     billingmodels.SeatCapacityTransition
}

// SeatCapacityAlerts are the capacity transitions that a transaction staged.
// Only the owner of the outermost transaction emits them, after its commit
// succeeded. A rollback, a failed commit, or the end of a nested transaction
// or savepoint emits nothing, so an alert never describes discarded state.
type SeatCapacityAlerts []SeatCapacityCheck

// Emit writes each staged transition. Call it only after the outermost
// transaction committed.
func (a SeatCapacityAlerts) Emit() {
	for _, check := range a {
		check.Log()
	}
}

// RecheckSeatCapacity updates the over-capacity record of the organization
// from its current member count. Call it inside the transaction that changed
// membership or the plan, and call Log after that transaction commits. The
// count runs under the billing plan lock, so a concurrent membership change
// cannot make a check miss the final count. It does nothing when billing is
// off.
func RecheckSeatCapacity(tx *gorm.DB, orgID uuid.UUID, cause SeatCapacityCause) (SeatCapacityCheck, error) {
	check := SeatCapacityCheck{
		OrganizationID: orgID,
		Cause:          cause,
		Transition:     billingmodels.SeatCapacityTransition{Change: billingmodels.SeatCapacityUnchanged},
	}
	if !billingmodels.BillingEnabled() {
		return check, nil
	}
	var err error
	check.Transition, err = billingmodels.UpdateSeatCapacityState(tx, orgID, time.Now())
	return check, err
}

// Log writes the alert for a started state or the resolution for an ended
// state. It writes nothing when the state did not change.
func (c SeatCapacityCheck) Log() {
	fields := log.Fields{
		"organization_id": c.OrganizationID.String(),
		"subscription_id": c.Transition.SubscriptionID,
		"purchased":       c.Transition.Purchased,
		"occupied":        c.Transition.Occupied,
		"cause":           string(c.Cause),
	}
	switch c.Transition.Change {
	case billingmodels.SeatCapacityStarted:
		logging.LogSeatsOverCapacity(fields)
	case billingmodels.SeatCapacityResolved:
		logging.LogSeatsOverCapacityResolved(fields)
	}
}

// snapshotSeatCapacityCause says why a Polar snapshot changed the seat state.
// previous is the plan before the snapshot. saved is the plan after it.
func snapshotSeatCapacityCause(previous, saved *billingmodels.OrganizationBillingPlan) SeatCapacityCause {
	switch {
	case previous != nil && billingmodels.PolarSubscriptionIsPastDue(previous.PolarSubscriptionStatus) &&
		billingmodels.PolarSubscriptionIsPaid(saved.PolarSubscriptionStatus):
		return SeatCapacityCauseRecovery
	case !previous.IsPaidPerSeat():
		return SeatCapacityCauseActivation
	case periodStartChanged(previous.CurrentPeriodStart, saved.CurrentPeriodStart):
		return SeatCapacityCauseCycle
	default:
		return SeatCapacityCauseUpdate
	}
}

func periodStartChanged(previous, next *time.Time) bool {
	if previous == nil || next == nil {
		return previous != next
	}
	return !previous.Equal(*next)
}
