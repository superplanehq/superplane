package seatremoval

import (
	"errors"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/seatquantity"
	"gorm.io/gorm"
)

// PurchaseBinding is the removal context a seat purchase confirms.
// An empty PendingID is the versioned absence of a removal.
type PurchaseBinding struct {
	Version         int64
	PendingID       string
	PendingRevision int
}

// BindPurchase checks that the confirmed removal is still current.
// It returns the removal a successful purchase must supersede, or nil when
// the confirmation records that no removal is pending. It does not write.
func BindPurchase(tx *gorm.DB, orgID uuid.UUID, binding PurchaseBinding) (*billingmodels.SeatQuantityOperation, error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.SeatChangeVersion != binding.Version {
		return nil, statusError(ReasonStalePreview, "The seat removal changed. Review the purchase and confirm again.")
	}
	inFlight, err := billingmodels.FindInFlightSeatQuantityOperation(tx, orgID)
	if err != nil {
		return nil, err
	}
	if inFlight != nil {
		return nil, statusError(ReasonOutcomeUnknown, "A seat change is still unconfirmed. Wait for billing to finish checking it.")
	}
	if binding.PendingID == "" {
		if plan.SeatProvedTarget != nil || plan.SeatQuantityOperationID != nil {
			return nil, statusError(ReasonStalePreview, "The seat removal changed. Review the purchase and confirm again.")
		}
		return nil, nil
	}
	id, err := uuid.Parse(binding.PendingID)
	if err != nil {
		return nil, statusError(ReasonStalePreview, "The seat removal changed. Review the purchase and confirm again.")
	}
	op, err := billingmodels.FindSeatQuantityOperation(tx, orgID, id)
	if err != nil {
		return nil, err
	}
	if op == nil || op.Revision != binding.PendingRevision || plan.SeatQuantityOperationID == nil || *plan.SeatQuantityOperationID != op.ID {
		return nil, statusError(ReasonStalePreview, "The seat removal changed. Review the purchase and confirm again.")
	}
	if op.Status != billingmodels.SeatQuantityStatusScheduled && op.Status != billingmodels.SeatQuantityStatusRepairRequired {
		return nil, statusError(ReasonStalePreview, "The seat removal changed. Review the purchase and confirm again.")
	}
	return op, nil
}

// LinkPurchaseSupersession records that a purchase will replace a removal.
// The caller holds the organization lock and has already inserted the purchase.
// Admission stays at the proved lower count until the purchase and the provider
// schedule are both verified. This sends no provider request.
func LinkPurchaseSupersession(tx *gorm.DB, purchase *billingmodels.SeatPurchase, removal *billingmodels.SeatQuantityOperation) error {
	if purchase == nil || removal == nil {
		return nil
	}
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, purchase.OrganizationID)
	if err != nil {
		return err
	}
	if plan == nil {
		return statusError(ReasonNotAvailable, "Seat purchases need an active Team subscription.")
	}
	versionSeen := plan.SeatChangeVersion
	plan.SeatChangeVersion++
	periodStart, periodEnd := time.Time{}, time.Time{}
	if purchase.PeriodStart != nil {
		periodStart = purchase.PeriodStart.UTC()
	}
	if purchase.PeriodEnd != nil {
		periodEnd = purchase.PeriodEnd.UTC()
	}
	op := &billingmodels.SeatQuantityOperation{
		OrganizationID: purchase.OrganizationID, ActorUserID: purchase.UserID,
		Origin: billingmodels.SeatQuantityOriginPurchase, Action: billingmodels.SeatQuantityActionSupersede,
		SubscriptionID: purchase.PolarSubscriptionID, PeriodStart: periodStart, PeriodEnd: periodEnd,
		SourcePaidQuantity: purchase.FromSeats, TargetQuantity: purchase.ToSeats, EffectiveAt: periodEnd,
		OperationKey:       "purchase:" + purchase.ID.String(),
		RequestFingerprint: billingmodels.SeatQuantityFingerprint(billingmodels.SeatQuantityActionSupersede, purchase.ToSeats, purchase.PolarSubscriptionID, periodEnd, versionSeen, purchase.FromSeats),
		Revision:           1, PredecessorID: &removal.ID, SeatPurchaseID: &purchase.ID,
		Status: billingmodels.SeatQuantityStatusSubmitting, SeatChangeVersion: plan.SeatChangeVersion,
	}
	if err := op.Create(tx); err != nil {
		return err
	}
	plan.SeatQuantityOperationID = &op.ID
	return billingmodels.SaveSeatChange(tx, plan)
}

// FinishPurchase records a definitive purchase outcome on its supersession.
// Success does not clear the lower admission here. That happens only when a
// snapshot shows the provider replaced the scheduled removal. A rejection
// restores the proved removal. No restoration request is sent.
func FinishPurchase(tx *gorm.DB, purchase *billingmodels.SeatPurchase, status string) error {
	if purchase == nil {
		return nil
	}
	op, err := findSupersession(tx, purchase.ID)
	if err != nil || op == nil || !op.InFlight() {
		return err
	}
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, purchase.OrganizationID)
	if err != nil || plan == nil {
		return err
	}
	switch status {
	case billingmodels.SeatPurchaseStatusFailed, billingmodels.SeatPurchaseStatusActionRequired:
		if err := op.Transition(tx, billingmodels.SeatQuantityStatusRejected, status, nil); err != nil {
			return err
		}
		return restorePredecessor(tx, plan, op)
	case billingmodels.SeatPurchaseStatusSucceeded:
		if op.Status != billingmodels.SeatQuantityStatusSubmitting {
			return nil
		}
		return op.Transition(tx, billingmodels.SeatQuantityStatusUnknown, seatquantity.CauseScheduleRemains, nil)
	default:
		return op.Transition(tx, billingmodels.SeatQuantityStatusUnknown, "", nil)
	}
}

func restorePredecessor(tx *gorm.DB, plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation) error {
	if op.PredecessorID == nil {
		billingmodels.ClearSeatRestriction(plan)
		return billingmodels.SaveSeatChange(tx, plan)
	}
	predecessor, err := billingmodels.FindSeatQuantityOperation(tx, op.OrganizationID, *op.PredecessorID)
	if err != nil {
		return err
	}
	if predecessor != nil && predecessor.Status == billingmodels.SeatQuantityStatusScheduled {
		target := predecessor.TargetQuantity
		plan.SeatProvedTarget = &target
		plan.SeatAdmissionLimit = &target
		plan.SeatQuantityOperationID = &predecessor.ID
		end := predecessor.PeriodEnd.UTC()
		plan.SeatBoundPeriodEnd = &end
		return billingmodels.SaveSeatChange(tx, plan)
	}
	plan.SeatQuantityOperationID = op.PredecessorID
	return billingmodels.SaveSeatChange(tx, plan)
}

func findSupersession(tx *gorm.DB, purchaseID uuid.UUID) (*billingmodels.SeatQuantityOperation, error) {
	var op billingmodels.SeatQuantityOperation
	err := tx.Where("seat_purchase_id = ?", purchaseID).Order("created_at DESC").First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}
