package entitlements

import (
	"time"

	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
)

// AdmissionBlockUnverified stops new seat use until a snapshot verifies the
// billing period a removal was bound to. The clock does not clear the removal.
const AdmissionBlockUnverified = "unverified"

// admissionLimit is min(paid seats, the active restrictive target). An open
// removal whose bound period is expired or no longer matches the stored period
// admits no one new until that period is verified.
func admissionLimit(plan *billingmodels.OrganizationBillingPlan, paid, occupied int, now time.Time) int {
	limit := paid
	if plan.SeatAdmissionLimit != nil && *plan.SeatAdmissionLimit < limit {
		limit = *plan.SeatAdmissionLimit
	}
	if removalPeriodUnverified(plan, now) {
		return occupied
	}
	return limit
}

func removalPeriodUnverified(plan *billingmodels.OrganizationBillingPlan, now time.Time) bool {
	if plan.SeatQuantityOperationID == nil && plan.SeatAdmissionLimit == nil && plan.SeatProvedTarget == nil {
		return false
	}
	if plan.SeatBoundPeriodEnd == nil || plan.CurrentPeriodEnd == nil {
		return true
	}
	if !now.Before(*plan.SeatBoundPeriodEnd) {
		return true
	}
	return !plan.CurrentPeriodEnd.Equal(*plan.SeatBoundPeriodEnd)
}
