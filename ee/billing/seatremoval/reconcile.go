package seatremoval

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/ee/billing/seatquantity"
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Recover reads provider state for an open removal and may cancel that removal
// when the member count no longer fits it. It never buys seats, never removes
// a member, and never sends a lower seat count.
func Recover(ctx context.Context, db *gorm.DB, client SubscriptionClient, orgID uuid.UUID) error {
	return NewRunner(db, client).Recover(ctx, orgID)
}

// Recover uses the runner clock. Tests move that clock across the renewal guard.
func (r *Runner) Recover(ctx context.Context, orgID uuid.UUID) error {
	ops, err := billingmodels.ListOpenSeatQuantityOperations(r.DB.WithContext(ctx), orgID)
	if err != nil {
		return err
	}
	if len(ops) == 0 {
		return nil
	}
	return billingmodels.WithSeatQuantityLock(r.DB.WithContext(ctx), orgID, func(*gorm.DB) error {
		return r.recoverLocked(ctx, orgID)
	})
}

func (r *Runner) recoverLocked(ctx context.Context, orgID uuid.UUID) error {
	plan, err := billingmodels.FindOrganizationBillingPlan(r.DB.WithContext(ctx), orgID)
	if err != nil || plan == nil || plan.PolarSubscriptionID == nil || *plan.PolarSubscriptionID == "" {
		return err
	}
	snap, err := r.Client.GetSubscription(ctx, *plan.PolarSubscriptionID)
	if err != nil {
		return err
	}
	var alerts []billing.CapacityAlert
	if err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, orgID); err != nil {
			return err
		}
		staged, err := seatquantity.Observe(tx, orgID, snapshotOf(snap, true), EmitAlert)
		if err != nil {
			return err
		}
		alerts = staged
		return nil
	}); err != nil {
		return err
	}
	for _, alert := range alerts {
		alert.Emit()
	}
	return r.enforceFloorLocked(ctx, orgID, snap)
}

func (r *Runner) enforceFloorLocked(ctx context.Context, orgID uuid.UUID, snap *polar.SubscriptionData) error {
	var scheduled *billingmodels.SeatQuantityOperation
	var plan *billingmodels.OrganizationBillingPlan
	var members int
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, orgID); err != nil {
			return err
		}
		var err error
		plan, members, err = loadPlan(tx, orgID)
		if err != nil || plan == nil || plan.PurchasedSeats == nil {
			return err
		}
		inFlight, err := billingmodels.FindInFlightSeatQuantityOperation(tx, orgID)
		if err != nil || inFlight != nil {
			return err
		}
		scheduled, err = billingmodels.FindScheduledSeatQuantityOperation(tx, orgID)
		return err
	})
	if err != nil || scheduled == nil || plan == nil || plan.PurchasedSeats == nil || plan.CurrentPeriodEnd == nil {
		return err
	}
	paid := *plan.PurchasedSeats
	if members <= scheduled.TargetQuantity {
		return nil
	}
	// A pending update that is not only seats must stay as Polar stored it.
	mixed := snap != nil && snap.PendingUpdate != nil && snap.PendingUpdate.HasUnrelatedChanges()
	if members > paid || mixed || !seatquantity.AllowsMutation(r.now(), *plan.CurrentPeriodEnd) {
		return r.requireRepair(ctx, scheduled, members, plan)
	}
	return r.protectiveCancel(ctx, plan, scheduled, members)
}

func (r *Runner) requireRepair(ctx context.Context, scheduled *billingmodels.SeatQuantityOperation, members int, plan *billingmodels.OrganizationBillingPlan) error {
	var alert seatquantity.Alert
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, scheduled.OrganizationID); err != nil {
			return err
		}
		current, err := billingmodels.FindSeatQuantityOperation(tx, scheduled.OrganizationID, scheduled.ID)
		if err != nil || current == nil || current.Status == billingmodels.SeatQuantityStatusRepairRequired {
			return err
		}
		if err := current.Transition(tx, billingmodels.SeatQuantityStatusRepairRequired, seatquantity.CauseMemberFloor, nil); err != nil {
			return err
		}
		limit := members
		if plan.PurchasedSeats != nil && *plan.PurchasedSeats < limit {
			limit = *plan.PurchasedSeats
		}
		stored, err := billingmodels.FindOrganizationBillingPlan(tx, scheduled.OrganizationID)
		if err != nil || stored == nil {
			return err
		}
		stored.SeatAdmissionLimit = &limit
		stored.SeatQuantityOperationID = &current.ID
		if err := billingmodels.SaveSeatChange(tx, stored); err != nil {
			return err
		}
		alert = seatquantity.Alert{
			OrganizationID: scheduled.OrganizationID, OperationID: current.ID,
			SubscriptionID: current.SubscriptionID, Target: current.TargetQuantity,
			Members: members, Boundary: boundary(stored), RecoveryStatus: "repair_required",
		}
		return nil
	})
	if err != nil {
		return err
	}
	if alert.OperationID != uuid.Nil {
		EmitAlert(alert)
	}
	return nil
}

func (r *Runner) protectiveCancel(ctx context.Context, plan *billingmodels.OrganizationBillingPlan, scheduled *billingmodels.SeatQuantityOperation, members int) error {
	key := fmt.Sprintf("protective:%s:%d", scheduled.ID.String(), scheduled.Revision)
	req := Request{
		OrganizationID: scheduled.OrganizationID, OperationKey: key, Target: *plan.PurchasedSeats,
		SubscriptionID: scheduled.SubscriptionID, PeriodStart: scheduled.PeriodStart, PeriodEnd: scheduled.PeriodEnd,
		SourcePaidQuantity: scheduled.SourcePaidQuantity, SeatChangeVersion: plan.SeatChangeVersion,
	}
	// Protective recovery has no admin actor. The origin allows that.
	var staged *billingmodels.SeatQuantityOperation
	var prior seatState
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, plan.OrganizationID); err != nil {
			return err
		}
		existing, err := billingmodels.FindSeatQuantityOperationByKey(tx, plan.OrganizationID, key)
		if err != nil {
			return err
		}
		if existing != nil {
			// A protective cancellation that may still be executing is not sent again.
			staged = existing
			return errStop
		}
		currentPlan, _, err := loadPlan(tx, plan.OrganizationID)
		if err != nil {
			return err
		}
		prior = capture(currentPlan)
		currentPlan.SeatChangeVersion++
		op := &billingmodels.SeatQuantityOperation{
			OrganizationID: plan.OrganizationID, Origin: billingmodels.SeatQuantityOriginProtective,
			Action: billingmodels.SeatQuantityActionProtectiveCancel, SubscriptionID: scheduled.SubscriptionID,
			PeriodStart: scheduled.PeriodStart.UTC(), PeriodEnd: scheduled.PeriodEnd.UTC(),
			SourcePaidQuantity: *currentPlan.PurchasedSeats, TargetQuantity: *currentPlan.PurchasedSeats,
			EffectiveAt: scheduled.EffectiveAt.UTC(), OperationKey: key,
			RequestFingerprint: billingmodels.SeatQuantityFingerprint(billingmodels.SeatQuantityActionProtectiveCancel, *currentPlan.PurchasedSeats, scheduled.SubscriptionID, scheduled.PeriodEnd, req.SeatChangeVersion, scheduled.SourcePaidQuantity),
			Revision:           1, Status: billingmodels.SeatQuantityStatusSubmitting,
			Cause: seatquantity.CauseMemberFloor, SeatChangeVersion: currentPlan.SeatChangeVersion,
			PredecessorID: &scheduled.ID,
		}
		if err := op.Create(tx); err != nil {
			return err
		}
		currentPlan.SeatQuantityOperationID = &op.ID
		if err := billingmodels.SaveSeatChange(tx, currentPlan); err != nil {
			return err
		}
		staged = op
		return nil
	})
	if errors.Is(err, errStop) {
		return nil
	}
	if err != nil || staged == nil || !staged.InFlight() {
		return err
	}
	// The guard can start after registration and before the provider call.
	// Close the unsent operation. A submitting row would block every later
	// seat change even though Polar received nothing.
	if !seatquantity.AllowsMutation(r.now(), staged.EffectiveAt) {
		if err := r.reject(ctx, staged, prior, ReasonRenewalBoundary); err != nil {
			return err
		}
		return r.requireRepair(ctx, scheduled, members, plan)
	}
	if err := staged.MarkDispatched(r.DB.WithContext(ctx), r.now()); err != nil {
		return err
	}
	updated, updateErr := r.Client.ScheduleSubscriptionSeats(ctx, staged.SubscriptionID, staged.TargetQuantity)
	if err := r.record(ctx, staged, prior, updated, updateErr); err != nil {
		return err
	}
	EmitAlert(seatquantity.Alert{
		OrganizationID: plan.OrganizationID, OperationID: staged.ID, SubscriptionID: staged.SubscriptionID,
		Target: staged.TargetQuantity, Members: members, Boundary: "before_cutoff", RecoveryStatus: "protective_cancel",
	})
	return nil
}

func snapshotOf(data *polar.SubscriptionData, verified bool) seatquantity.Snapshot {
	snap := seatquantity.Snapshot{Verified: verified, SubscriptionID: data.ID, Seats: data.Seats, Status: data.Status}
	if !data.CurrentPeriodStart.Time.IsZero() {
		start := data.CurrentPeriodStart.Time.UTC()
		snap.PeriodStart = &start
	}
	if !data.CurrentPeriodEnd.Time.IsZero() {
		end := data.CurrentPeriodEnd.Time.UTC()
		snap.PeriodEnd = &end
	}
	if !data.ModifiedAt.Time.IsZero() {
		modified := data.ModifiedAt.Time.UTC()
		snap.ModifiedAt = &modified
	}
	if data.PendingUpdate != nil {
		snap.PendingSeats = data.PendingUpdate.Seats
		snap.PendingUnrelated = data.PendingUpdate.HasUnrelatedChanges()
		if !data.PendingUpdate.AppliesAt.Time.IsZero() {
			applies := data.PendingUpdate.AppliesAt.Time.UTC()
			snap.PendingAppliesAt = &applies
		}
	}
	return snap
}

func boundary(plan *billingmodels.OrganizationBillingPlan) string {
	if plan.CurrentPeriodEnd != nil && !seatquantity.AllowsMutation(time.Now(), *plan.CurrentPeriodEnd) {
		return "inside_guard"
	}
	return "before_cutoff"
}

var errStop = errors.New("seat removal recovery already recorded")

// EmitAlert writes a removal recovery line. Call it only after the transaction commits.
func EmitAlert(alert seatquantity.Alert) { seatquantity.LogAlert(alert) }
