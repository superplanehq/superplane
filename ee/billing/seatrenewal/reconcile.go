// Package seatrenewal reconciles the stored plan of an organization with its
// Polar subscription. Polar is the source of truth for the paid seats, the
// status, and the billing period. Reconciliation repairs lost webhooks: a
// missed renewal, a payment failure, or a recovery.
//
// Reconciliation only reads Polar. It never writes a seat count, so it never
// charges, credits, or schedules a renewal quantity. Paid seats renew
// unchanged until an Admin changes them.
//
// Reconciliation settles an in-flight seat purchase only when the verified
// Polar snapshot proves its success. It keeps the identity and the status of
// a purchase without such evidence until a person repairs it from Polar
// evidence.
package seatrenewal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"gorm.io/gorm"
)

// Client is the part of the Polar API that reconciliation uses. It only reads.
type Client interface {
	GetSubscription(ctx context.Context, subscriptionID string) (*polar.SubscriptionData, error)
}

// Action is what Reconcile did for an organization.
type Action string

const (
	// ActionSkipped means that Polar does not manage the plan, so there is
	// no subscription to read.
	ActionSkipped Action = "skipped"
	// ActionReconciled means that the stored plan received the verified Polar
	// subscription. The apply ignores a snapshot that is older than the
	// stored plan.
	ActionReconciled Action = "reconciled"
)

// Reasons for ActionSkipped.
const (
	SkipAdminPlan      = "admin_plan"
	SkipNoSubscription = "no_subscription"
)

// SeatPurchaseReconcileFailedKey is the log key for a purchase on another
// subscription than the stored plan that Reconcile could not reconcile.
// Reconcile continues with the stored plan subscription.
const SeatPurchaseReconcileFailedKey = "billing.seat_purchase_reconcile_failed"

// Result describes one Reconcile call. Fields that Reconcile did not reach
// are empty.
type Result struct {
	OrganizationID uuid.UUID
	Action         Action
	SkipReason     string
	SubscriptionID string
	// CurrentSeats is the seat count of the current Polar period.
	CurrentSeats *int
}

// LogFields are the stable log fields for a reconciliation result.
func (r Result) LogFields() log.Fields {
	fields := log.Fields{"organization_id": r.OrganizationID.String()}
	if r.SubscriptionID != "" {
		fields["polar_subscription_id"] = r.SubscriptionID
	}
	if r.CurrentSeats != nil {
		fields["current_seats"] = *r.CurrentSeats
	}
	return fields
}

// Reconciler applies the verified Polar subscription of an organization to
// its stored plan.
type Reconciler struct {
	db     *gorm.DB
	client Client
}

func NewReconciler(db *gorm.DB, client Client) *Reconciler {
	return &Reconciler{db: db, client: client}
}

// Reconcile settles a seat purchase that Polar proves, and then applies the
// current Polar subscription. Every Polar request runs outside a transaction
// and outside the organization lock. The subscription apply checks identity
// and ordering under the lock, and it writes capacity alerts only after its
// commit.
func (r *Reconciler) Reconcile(ctx context.Context, orgID uuid.UUID) (Result, error) {
	result := Result{OrganizationID: orgID}
	// A failure to read the subscription of a purchase on another
	// subscription does not stop the reconciliation of the current one.
	if err := polar.ReconcileSeatPurchase(ctx, r.db.WithContext(ctx), r.client, orgID); err != nil {
		if err := r.continueAfterReconcileFailure(ctx, orgID, err); err != nil {
			return result, err
		}
	}

	subscriptionID, reason, err := r.reconcilableSubscription(ctx, orgID)
	result.SubscriptionID = subscriptionID
	if err != nil {
		return result, err
	}
	if reason != "" {
		result.Action, result.SkipReason = ActionSkipped, reason
		return result, nil
	}
	snapshot, err := r.client.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return result, err
	}
	result.CurrentSeats = snapshot.Seats
	if strings.TrimSpace(snapshot.ID) == "" {
		return result, fmt.Errorf("polar returned no subscription for %s", subscriptionID)
	}
	if err := polar.ApplyOrganizationSubscription(ctx, r.db.WithContext(ctx), orgID, *snapshot); err != nil {
		return result, fmt.Errorf("apply polar subscription %s: %w", subscriptionID, err)
	}
	result.Action = ActionReconciled
	return result, nil
}

// SeatPurchaseLogFields are the stable log fields that identify a purchase.
func SeatPurchaseLogFields(purchase *billingmodels.SeatPurchase) log.Fields {
	fields := log.Fields{
		"organization_id":       purchase.OrganizationID.String(),
		"seat_purchase_id":      purchase.ID.String(),
		"seat_purchase_status":  purchase.Status,
		"polar_subscription_id": purchase.PolarSubscriptionID,
	}
	if purchase.OperationKey != nil {
		fields["operation_key"] = *purchase.OperationKey
	}
	if purchase.PeriodStart != nil {
		fields["purchase_period_start"] = purchase.PeriodStart.UTC()
	}
	if purchase.PeriodEnd != nil {
		fields["purchase_period_end"] = purchase.PeriodEnd.UTC()
	}
	return fields
}

// continueAfterReconcileFailure decides whether a failed purchase
// reconciliation stops the run. A purchase on another subscription than the
// stored plan cannot change the current subscription, so a failure to read
// its subscription, for example a replaced one, must not stop the
// reconciliation of the current subscription. It returns nil in that case and
// the reconciliation error otherwise. The purchase stays unchanged.
func (r *Reconciler) continueAfterReconcileFailure(ctx context.Context, orgID uuid.UUID, reconcileErr error) error {
	db := r.db.WithContext(ctx)
	purchase, err := billingmodels.FindInFlightSeatPurchase(db, orgID)
	if err != nil {
		return errors.Join(reconcileErr, err)
	}
	if purchase == nil {
		return reconcileErr
	}
	plan, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
	if err != nil {
		return errors.Join(reconcileErr, err)
	}
	current := planSubscriptionID(plan)
	if current == "" || current == purchase.PolarSubscriptionID {
		return reconcileErr
	}
	log.WithError(reconcileErr).
		WithFields(SeatPurchaseLogFields(purchase)).
		WithField("current_polar_subscription_id", current).
		Warn(SeatPurchaseReconcileFailedKey)
	return nil
}

// reconcilableSubscription reads the stored Polar subscription ID without a
// row lock. Only a plan that Polar manages has a subscription to reconcile.
// Otherwise, the reason names why not.
func (r *Reconciler) reconcilableSubscription(ctx context.Context, orgID uuid.UUID) (subscriptionID, skipReason string, err error) {
	plan, err := billingmodels.FindOrganizationBillingPlan(r.db.WithContext(ctx), orgID)
	if err != nil {
		return "", "", err
	}
	if plan == nil {
		return "", SkipNoSubscription, nil
	}
	subscriptionID = planSubscriptionID(plan)
	switch {
	case plan.PlanSource == billingmodels.BillingPlanSourceAdmin:
		return subscriptionID, SkipAdminPlan, nil
	case plan.PlanSource != billingmodels.BillingPlanSourcePolar || subscriptionID == "":
		return subscriptionID, SkipNoSubscription, nil
	default:
		return subscriptionID, "", nil
	}
}

func planSubscriptionID(plan *billingmodels.OrganizationBillingPlan) string {
	if plan == nil || plan.PolarSubscriptionID == nil {
		return ""
	}
	return strings.TrimSpace(*plan.PolarSubscriptionID)
}
