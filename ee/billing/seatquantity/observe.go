package seatquantity

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	CauseSubscriptionReplaced = "subscription_replaced"
	CauseMemberFloor          = "member_floor"
	CauseScheduleRemains      = "schedule_not_replaced"

	AlertRecoveryKey = "billing.seat_removal_recovery"
)

// Snapshot is the part of a provider subscription that can resolve a removal.
// Verified is true only for a direct provider read, not for a webhook.
type Snapshot struct {
	SubscriptionID   string
	Seats            *int
	PendingSeats     *int
	PendingAppliesAt *time.Time
	PendingUnrelated bool
	PeriodStart      *time.Time
	PeriodEnd        *time.Time
	ModifiedAt       *time.Time
	Status           string
	Verified         bool
}

// Alert is written only after the transaction that recorded the recovery commits.
type Alert struct {
	OrganizationID uuid.UUID
	OperationID    uuid.UUID
	SubscriptionID string
	Target         int
	Members        int
	Boundary       string
	RecoveryStatus string
}

// RecoveryAlert is the capacity-alert form of Alert. The caller supplies Emit.
type RecoveryAlert struct {
	Alert
	emit func(Alert)
}

func (a RecoveryAlert) Emit() {
	if a.emit != nil {
		a.emit(a.Alert)
	}
}

// EmitRecovery builds an alert the caller emits after commit.
func EmitRecovery(alert Alert, emit func(Alert)) billing.CapacityAlert {
	return RecoveryAlert{Alert: alert, emit: emit}
}

// Observe applies a snapshot to the open removals of an organization.
// It never sends a provider request and never removes a member or buys seats.
// The caller holds the organization transaction. Call Emit on the returned
// alerts only after that transaction commits.
func Observe(tx *gorm.DB, orgID uuid.UUID, snap Snapshot, emit func(Alert)) ([]billing.CapacityAlert, error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil || plan == nil {
		return nil, err
	}
	if plan.SeatQuantityOperationID == nil && plan.SeatProvedTarget == nil && plan.SeatAdmissionLimit == nil {
		return nil, nil
	}
	ops, err := billingmodels.ListOpenSeatQuantityOperations(tx, orgID)
	if err != nil {
		return nil, err
	}
	var alerts []billing.CapacityAlert
	for i := range ops {
		opAlerts, err := observeOperation(tx, plan, &ops[i], snap, emit)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, opAlerts...)
	}
	return alerts, nil
}

func observeOperation(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot, emit func(Alert)) ([]billing.CapacityAlert, error) {
	if snap.SubscriptionID != "" && snap.SubscriptionID != op.SubscriptionID {
		return observeReplacement(tx, plan, op, snap, emit)
	}
	switch op.Action {
	case billingmodels.SeatQuantityActionSupersede:
		return observeSupersession(tx, plan, op, snap, emit)
	default:
		return observeRemoval(tx, plan, op, snap, emit)
	}
}

func observeReplacement(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot, emit func(Alert)) ([]billing.CapacityAlert, error) {
	// An unverified event can be a delayed payload for the old subscription.
	// It must not clear or copy the removal onto the replacement.
	if !snap.Verified {
		return nil, nil
	}
	if op.InFlight() {
		if err := op.Transition(tx, billingmodels.SeatQuantityStatusRepairRequired, CauseSubscriptionReplaced, evidence(snap)); err != nil {
			return nil, err
		}
		members, membersErr := humanMembers(tx, plan.OrganizationID)
		if membersErr != nil {
			return nil, membersErr
		}
		return []billing.CapacityAlert{alert(plan, op, snap, "repair_required", members, emit)}, nil
	}
	if err := op.Transition(tx, billingmodels.SeatQuantityStatusCanceled, CauseSubscriptionReplaced, evidence(snap)); err != nil {
		return nil, err
	}
	if plan.SeatQuantityOperationID != nil && *plan.SeatQuantityOperationID == op.ID {
		billingmodels.ClearSeatRestriction(plan)
		if err := billingmodels.SaveSeatChange(tx, plan); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func observeRemoval(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot, emit func(Alert)) ([]billing.CapacityAlert, error) {
	if renewalApplied(op, snap) {
		return applyRenewal(tx, plan, op, snap, emit)
	}
	if op.InFlight() && scheduleVerified(op, snap) {
		return proveSchedule(tx, plan, op, snap)
	}
	if op.InFlight() && cancellationVerified(op, snap) {
		return proveCancellation(tx, plan, op, snap)
	}
	// Quantity or elapsed time alone does not resolve an unknown mutation.
	return nil, nil
}

func observeSupersession(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot, emit func(Alert)) ([]billing.CapacityAlert, error) {
	if op.SeatPurchaseID == nil {
		return nil, nil
	}
	purchase, err := findPurchase(tx, *op.SeatPurchaseID)
	if err != nil || purchase == nil {
		return nil, err
	}
	proved := purchase.Status == billingmodels.SeatPurchaseStatusSucceeded || purchase.SucceededBy(purchaseEvidence(snap))
	if !proved {
		return nil, nil
	}
	if !scheduleReplaced(snap, op.TargetQuantity) {
		if op.Status != billingmodels.SeatQuantityStatusUnknown {
			if err := op.Transition(tx, billingmodels.SeatQuantityStatusUnknown, CauseScheduleRemains, evidence(snap)); err != nil {
				return nil, err
			}
		}
		members, membersErr := humanMembers(tx, plan.OrganizationID)
		if membersErr != nil {
			return nil, membersErr
		}
		return []billing.CapacityAlert{alert(plan, op, snap, "schedule_remains", members, emit)}, nil
	}
	if err := supersedePredecessor(tx, op); err != nil {
		return nil, err
	}
	if err := op.Transition(tx, billingmodels.SeatQuantityStatusApplied, "", evidence(snap)); err != nil {
		return nil, err
	}
	if plan.SeatQuantityOperationID != nil && *plan.SeatQuantityOperationID == op.ID {
		billingmodels.ClearSeatRestriction(plan)
		if err := billingmodels.SaveSeatChange(tx, plan); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func applyRenewal(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot, emit func(Alert)) ([]billing.CapacityAlert, error) {
	members, err := humanMembers(tx, plan.OrganizationID)
	if err != nil {
		return nil, err
	}
	floor := MemberFloor(members)
	if snap.Seats != nil && *snap.Seats < floor {
		if err := op.Transition(tx, billingmodels.SeatQuantityStatusRepairRequired, CauseMemberFloor, evidence(snap)); err != nil {
			return nil, err
		}
		limit := *snap.Seats
		plan.SeatAdmissionLimit = &limit
		plan.SeatProvedTarget = nil
		plan.SeatQuantityOperationID = &op.ID
		if snap.PeriodEnd != nil {
			end := snap.PeriodEnd.UTC()
			plan.SeatBoundPeriodEnd = &end
		}
		if err := billingmodels.SaveSeatChange(tx, plan); err != nil {
			return nil, err
		}
		return []billing.CapacityAlert{alert(plan, op, snap, "repair_required", members, emit)}, nil
	}
	if err := op.Transition(tx, billingmodels.SeatQuantityStatusApplied, "", evidence(snap)); err != nil {
		return nil, err
	}
	if plan.SeatQuantityOperationID != nil && *plan.SeatQuantityOperationID == op.ID {
		billingmodels.ClearSeatRestriction(plan)
		if err := billingmodels.SaveSeatChange(tx, plan); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func proveCancellation(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot) ([]billing.CapacityAlert, error) {
	if err := supersedePredecessor(tx, op); err != nil {
		return nil, err
	}
	if err := op.Transition(tx, billingmodels.SeatQuantityStatusCanceled, "", evidence(snap)); err != nil {
		return nil, err
	}
	if plan.SeatQuantityOperationID != nil && *plan.SeatQuantityOperationID == op.ID {
		billingmodels.ClearSeatRestriction(plan)
		if err := billingmodels.SaveSeatChange(tx, plan); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func proveSchedule(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot) ([]billing.CapacityAlert, error) {
	if op.PredecessorID != nil {
		if err := supersedePredecessor(tx, op); err != nil {
			return nil, err
		}
	}
	if err := op.Transition(tx, billingmodels.SeatQuantityStatusScheduled, "", evidence(snap)); err != nil {
		return nil, err
	}
	target := op.TargetQuantity
	plan.SeatProvedTarget = &target
	plan.SeatAdmissionLimit = &target
	plan.SeatQuantityOperationID = &op.ID
	if snap.PeriodEnd != nil {
		end := snap.PeriodEnd.UTC()
		plan.SeatBoundPeriodEnd = &end
	}
	return nil, billingmodels.SaveSeatChange(tx, plan)
}

func renewalApplied(op *billingmodels.SeatQuantityOperation, snap Snapshot) bool {
	if snap.Seats == nil || *snap.Seats != op.TargetQuantity || snap.PeriodStart == nil || snap.PeriodEnd == nil || snap.ModifiedAt == nil {
		return false
	}
	if snap.PeriodEnd.Equal(op.PeriodEnd) {
		return false
	}
	if snap.PeriodStart.Before(op.EffectiveAt) {
		return false
	}
	return !snap.ModifiedAt.Before(op.CreatedAt)
}

func scheduleVerified(op *billingmodels.SeatQuantityOperation, snap Snapshot) bool {
	if snap.PendingUnrelated || snap.PendingSeats == nil || *snap.PendingSeats != op.TargetQuantity {
		return false
	}
	if snap.PendingAppliesAt == nil || !snap.PendingAppliesAt.Equal(op.EffectiveAt) {
		return false
	}
	if snap.PeriodEnd == nil || !snap.PeriodEnd.Equal(op.PeriodEnd) || snap.SubscriptionID != op.SubscriptionID {
		return false
	}
	if op.DispatchedAt == nil || snap.ModifiedAt == nil || snap.ModifiedAt.Before(*op.DispatchedAt) {
		return false
	}
	return true
}

// cancellationVerified reports that a verified read shows the cancel reached
// the provider. The current seats still equal the cancel target, and the
// pending reduction is gone. A missing dispatch time means the request was
// never sent, so absence of a pending update is not evidence.
func cancellationVerified(op *billingmodels.SeatQuantityOperation, snap Snapshot) bool {
	if !snap.Verified || snap.PendingUnrelated || snap.Seats == nil || *snap.Seats != op.TargetQuantity {
		return false
	}
	if op.Action != billingmodels.SeatQuantityActionCancel && op.Action != billingmodels.SeatQuantityActionProtectiveCancel {
		return false
	}
	if snap.PendingSeats != nil && *snap.PendingSeats < op.TargetQuantity {
		return false
	}
	if snap.SubscriptionID == "" || snap.SubscriptionID != op.SubscriptionID || snap.PeriodEnd == nil || !snap.PeriodEnd.Equal(op.PeriodEnd) {
		return false
	}
	if op.DispatchedAt == nil || snap.ModifiedAt == nil || snap.ModifiedAt.Before(*op.DispatchedAt) {
		return false
	}
	return true
}

func scheduleReplaced(snap Snapshot, target int) bool {
	if snap.Seats == nil || *snap.Seats != target || snap.PendingUnrelated {
		return false
	}
	return snap.PendingSeats == nil || *snap.PendingSeats >= target
}

func supersedePredecessor(tx *gorm.DB, op *billingmodels.SeatQuantityOperation) error {
	if op.PredecessorID == nil {
		return nil
	}
	predecessor, err := billingmodels.FindSeatQuantityOperation(tx, op.OrganizationID, *op.PredecessorID)
	if err != nil || predecessor == nil || !predecessor.Open() {
		return err
	}
	if err := predecessor.SetSuperseder(tx, op.ID); err != nil {
		return err
	}
	return predecessor.Transition(tx, billingmodels.SeatQuantityStatusSuperseded, "", nil)
}

func findPurchase(tx *gorm.DB, id uuid.UUID) (*billingmodels.SeatPurchase, error) {
	var purchase billingmodels.SeatPurchase
	err := tx.First(&purchase, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &purchase, err
}

func purchaseEvidence(snap Snapshot) billingmodels.SeatPurchaseEvidence {
	return billingmodels.SeatPurchaseEvidence{
		SubscriptionID: snap.SubscriptionID,
		PeriodStart:    snap.PeriodStart,
		Seats:          snap.Seats,
		ModifiedAt:     snap.ModifiedAt,
	}
}

func humanMembers(tx *gorm.DB, orgID uuid.UUID) (int, error) {
	count, err := coremodels.CountActiveHumanUsersByOrganization(tx, orgID.String())
	return int(count), err
}

func evidence(snap Snapshot) datatypes.JSON {
	body, err := json.Marshal(map[string]any{
		"subscription_id":    snap.SubscriptionID,
		"seats":              snap.Seats,
		"pending_seats":      snap.PendingSeats,
		"pending_applies_at": snap.PendingAppliesAt,
		"period_start":       snap.PeriodStart,
		"period_end":         snap.PeriodEnd,
		"modified_at":        snap.ModifiedAt,
		"status":             snap.Status,
		"verified":           snap.Verified,
	})
	if err != nil {
		return nil
	}
	return body
}

func alert(plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, snap Snapshot, status string, members int, emit func(Alert)) billing.CapacityAlert {
	boundary := "before_cutoff"
	if plan.SeatBoundPeriodEnd != nil && !AllowsMutation(time.Now(), *plan.SeatBoundPeriodEnd) {
		boundary = "inside_guard"
	}
	subscriptionID := op.SubscriptionID
	if snap.SubscriptionID != "" {
		subscriptionID = snap.SubscriptionID
	}
	return EmitRecovery(Alert{
		OrganizationID: plan.OrganizationID,
		OperationID:    op.ID,
		SubscriptionID: subscriptionID,
		Target:         op.TargetQuantity,
		Members:        members,
		Boundary:       boundary,
		RecoveryStatus: status,
	}, emit)
}
