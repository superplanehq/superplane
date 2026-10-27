package seatremoval

import (
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/seatquantity"
	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
)

// Describe reads the removal state for Billing controls. It does not write.
func Describe(tx *gorm.DB, orgID uuid.UUID, now time.Time) (billing.SeatQuantityView, error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return billing.SeatQuantityView{}, err
	}
	members, err := humanMembers(tx, orgID)
	if err != nil {
		return billing.SeatQuantityView{}, err
	}
	var op *billingmodels.SeatQuantityOperation
	if plan != nil && plan.SeatQuantityOperationID != nil {
		op, err = billingmodels.FindSeatQuantityOperation(tx, orgID, *plan.SeatQuantityOperationID)
		if err != nil {
			return billing.SeatQuantityView{}, err
		}
	}
	return view(plan, op, members, now), nil
}

func view(plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, members int, now time.Time) billing.SeatQuantityView {
	floor := seatquantity.MemberFloor(members)
	result := billing.SeatQuantityView{OrganizationID: organizationID(plan), MemberFloor: floor}
	if plan == nil {
		result.Reasons = []string{ReasonNotAvailable}
		return result
	}
	result.SeatChangeVersion = plan.SeatChangeVersion
	result.Binding = binding(plan, op, 0, floor)
	if plan.CurrentPeriodEnd != nil {
		cutoff := seatquantity.Cutoff(*plan.CurrentPeriodEnd)
		result.Cutoff = &cutoff
		result.Binding.Cutoff = cutoff
		result.Binding.EffectiveAt = plan.CurrentPeriodEnd.UTC()
	}
	if op != nil {
		result.OperationID = op.ID.String()
		result.Revision = op.Revision
		result.Origin = op.Origin
		result.Action = op.Action
		target := op.TargetQuantity
		result.Target = &target
		effective := op.EffectiveAt.UTC()
		result.EffectiveAt = &effective
		result.Status = op.Status
		paid := op.SourcePaidQuantity
		result.SourcePaidQuantity = &paid
		result.SubscriptionID = op.SubscriptionID
		start, end := op.PeriodStart.UTC(), op.PeriodEnd.UTC()
		result.PeriodStart, result.PeriodEnd = &start, &end
	}
	result.AllowedActions, result.Reasons = allowed(plan, op, now)
	return result
}

func preview(plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, members, target int, now time.Time) billing.SeatQuantityPreview {
	floor := seatquantity.MemberFloor(members)
	result := billing.SeatQuantityPreview{Allowed: false, Message: "Seat removal needs an active Team subscription."}
	result.MemberFloor = floor
	result.Target = target
	result.NoCurrentCharge = true
	if plan == nil || plan.PurchasedSeats == nil || plan.PolarSubscriptionID == nil || plan.CurrentPeriodEnd == nil || plan.CurrentPeriodStart == nil {
		result.Reason = ReasonNotAvailable
		return result
	}
	paid := *plan.PurchasedSeats
	result.OrganizationID = plan.OrganizationID
	result.SubscriptionID = *plan.PolarSubscriptionID
	result.PeriodStart = plan.CurrentPeriodStart.UTC()
	result.PeriodEnd = plan.CurrentPeriodEnd.UTC()
	result.EffectiveAt = result.PeriodEnd
	result.Cutoff = seatquantity.Cutoff(result.PeriodEnd)
	result.SourcePaidQuantity = paid
	result.SeatChangeVersion = plan.SeatChangeVersion
	if op != nil && op.Open() {
		result.PendingOperationID = op.ID.String()
		result.PendingRevision = op.Revision
	}
	current := paid
	if plan.SeatAdmissionLimit != nil && *plan.SeatAdmissionLimit < current {
		current = *plan.SeatAdmissionLimit
	}
	result.ImmediateAdmission = current
	if target > 0 && target < current {
		result.ImmediateAdmission = target
	}
	result.RenewalQuantity = paid
	if target > 0 && target < paid {
		result.RenewalQuantity = target
	}
	switch {
	case billingmodels.PolarSubscriptionIsPaymentBlocked(plan.PolarSubscriptionStatus):
		result.Reason = ReasonPaymentBlocked
		result.Message = "Update the payment method before you change seats."
	case !seatquantity.AllowsMutation(now, result.PeriodEnd):
		result.Reason = ReasonRenewalBoundary
		result.Message = "The billing period renews soon. Confirm the seat removal after the renewal."
	case target == paid && plan.SeatProvedTarget != nil:
		result.Allowed = true
		result.Reason = ""
		result.Message = "Cancellation keeps the paid seats at renewal. SuperPlane does not charge, refund, or credit the current period."
	case target < floor || target >= paid:
		result.Reason = ReasonInvalidTarget
		result.Message = "Choose a seat count from the current members up to, but not including, the paid seats."
	default:
		result.Allowed = true
		result.Message = "The lower seat count starts at the next renewal. SuperPlane does not charge, refund, or credit the current period."
	}
	return result
}

func binding(plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, target, floor int) billing.SeatQuantityBinding {
	result := billing.SeatQuantityBinding{
		OrganizationID: plan.OrganizationID, SeatChangeVersion: plan.SeatChangeVersion,
		Target: target, MemberFloor: floor, NoCurrentCharge: true,
	}
	if plan.PolarSubscriptionID != nil {
		result.SubscriptionID = *plan.PolarSubscriptionID
	}
	if plan.CurrentPeriodStart != nil {
		result.PeriodStart = plan.CurrentPeriodStart.UTC()
	}
	if plan.CurrentPeriodEnd != nil {
		result.PeriodEnd = plan.CurrentPeriodEnd.UTC()
		result.EffectiveAt = result.PeriodEnd
		result.Cutoff = seatquantity.Cutoff(result.PeriodEnd)
	}
	if plan.PurchasedSeats != nil {
		result.SourcePaidQuantity = *plan.PurchasedSeats
		if target == 0 {
			if plan.SeatProvedTarget != nil {
				result.Target = *plan.SeatProvedTarget
			} else {
				result.Target = *plan.PurchasedSeats
			}
		}
		current := *plan.PurchasedSeats
		if plan.SeatAdmissionLimit != nil && *plan.SeatAdmissionLimit < current {
			current = *plan.SeatAdmissionLimit
		}
		result.ImmediateAdmission = current
		result.RenewalQuantity = result.Target
	}
	if op != nil && op.Open() {
		result.PendingOperationID = op.ID.String()
		result.PendingRevision = op.Revision
	}
	return result
}

func allowed(plan *billingmodels.OrganizationBillingPlan, op *billingmodels.SeatQuantityOperation, now time.Time) (actions, reasons []string) {
	if plan == nil || plan.PurchasedSeats == nil || plan.CurrentPeriodEnd == nil {
		return nil, []string{ReasonNotAvailable}
	}
	if op != nil && op.InFlight() {
		return nil, []string{ReasonOutcomeUnknown}
	}
	if op != nil && op.Status == billingmodels.SeatQuantityStatusRepairRequired {
		return nil, []string{ReasonRepairRequired}
	}
	if billingmodels.PolarSubscriptionIsPaymentBlocked(plan.PolarSubscriptionStatus) {
		return nil, []string{ReasonPaymentBlocked}
	}
	if !seatquantity.AllowsMutation(now, *plan.CurrentPeriodEnd) {
		return nil, []string{ReasonRenewalBoundary}
	}
	if plan.SeatProvedTarget != nil {
		return []string{billingmodels.SeatQuantityActionChange, billingmodels.SeatQuantityActionCancel}, nil
	}
	return []string{billingmodels.SeatQuantityActionSchedule}, nil
}

func organizationID(plan *billingmodels.OrganizationBillingPlan) uuid.UUID {
	if plan == nil {
		return uuid.Nil
	}
	return plan.OrganizationID
}
