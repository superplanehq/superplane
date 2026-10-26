package polar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// VerifyOrganizationDeletion returns the clearance that Organization.SoftDelete
// needs. Call it before the deletion transaction: it takes no lock, and the
// Polar request must not hold the organization lock. SoftDelete checks again
// under the lock that the billing row did not change.
//
// A billing row that cannot bill gets a local clearance without a Polar
// request. A row that may still bill needs Polar to return the stored
// subscription and every other subscription in a terminal state. A live
// subscription returns models.ErrOrganizationSubscriptionLive. A Polar
// failure or an unclear answer returns
// models.ErrOrganizationSubscriptionUnverified that wraps the cause.
//
// An unresolved checkout intent is verified first. A checkout that can still
// activate returns models.ErrOrganizationCheckoutPending: a payment after the
// deletion would start a subscription for a deleted organization. A paid
// checkout applies its subscription before the billing row is read.
//
// The verifier does not write the Polar subscription snapshot outside the
// checkout and subscription apply paths. Webhooks and the subscription
// refresh own the billing row.
func VerifyOrganizationDeletion(ctx context.Context, orgID uuid.UUID) (*models.OrganizationDeletionClearance, error) {
	if err := settleCheckoutBeforeDeletion(ctx, orgID); err != nil {
		return nil, err
	}
	plan, err := models.FindOrganizationBillingPlan(database.DB(ctx), orgID)
	if err != nil {
		return nil, err
	}
	if !plan.MayBeBillable(time.Now()) {
		return models.NewLocalDeletionClearance(orgID, plan)
	}
	if !Configured() {
		return nil, unverifiedDeletion("Polar is not configured")
	}
	if err := confirmSubscriptionEnded(ctx, orgID, polarSubscriptionID(plan)); err != nil {
		return nil, err
	}
	return models.NewVerifiedDeletionClearance(plan), nil
}

// maxDeletionCheckoutAttempts bounds the verification loop. Each repeat means
// that another writer changed the intent in between.
const maxDeletionCheckoutAttempts = 3

// settleCheckoutBeforeDeletion makes sure that no checkout of the
// organization can still activate. SoftDelete checks again under the
// organization lock that no unresolved intent exists.
func settleCheckoutBeforeDeletion(ctx context.Context, orgID uuid.UUID) error {
	for range maxDeletionCheckoutAttempts {
		intent, err := models.FindUnresolvedOrganizationBillingCheckout(database.DB(ctx), orgID)
		if err != nil || intent == nil {
			return err
		}
		if !Configured() {
			return unverifiedDeletion("Polar is not configured to verify checkout intent %s", intent.ID)
		}
		verification, err := NewCheckoutLifecycle(NewClientFromEnv()).Verify(ctx, intent)
		if err != nil {
			return fmt.Errorf("%w: verify checkout intent %s: %w", models.ErrOrganizationSubscriptionUnverified, intent.ID, err)
		}
		switch verification.Outcome {
		case CheckoutResolved, CheckoutChanged:
			continue
		case CheckoutSubscriptionLive:
			return fmt.Errorf("%w: checkout intent %s started a live subscription", models.ErrOrganizationSubscriptionLive, intent.ID)
		default:
			return fmt.Errorf("%w: checkout intent %s is %s", models.ErrOrganizationCheckoutPending, intent.ID, verification.Outcome)
		}
	}
	return models.ErrOrganizationCheckoutPending
}

func confirmSubscriptionEnded(ctx context.Context, orgID uuid.UUID, storedID string) error {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	items, err := NewClientFromEnv().ListSubscriptions(ctx, orgID.String(), "")
	if err != nil {
		return fmt.Errorf("%w: list Polar subscriptions: %w", models.ErrOrganizationSubscriptionUnverified, err)
	}

	now := time.Now()
	storedEnded := false
	for _, item := range items {
		if externalID := item.organizationExternalID(); externalID != "" && externalID != orgID.String() {
			return unverifiedDeletion("Polar subscription %q customer does not match the organization", item.ID)
		}
		status := strings.ToLower(strings.TrimSpace(item.Status))
		if !models.PolarSubscriptionStatusIsKnown(status) {
			return unverifiedDeletion("Polar subscription %q has unexpected status %q", item.ID, item.Status)
		}
		periodEnd := item.CurrentPeriodEnd.TimePtr()
		terminal := models.PolarSubscriptionIsTerminal(status, item.CancelAtPeriodEnd, periodEnd, now)
		if item.ID == storedID {
			storedEnded = terminal
		}
		if terminal {
			continue
		}
		if models.PolarSubscriptionIsLive(status, item.CancelAtPeriodEnd, periodEnd, now) {
			return fmt.Errorf("%w: Polar subscription %q is %s", models.ErrOrganizationSubscriptionLive, item.ID, status)
		}
		return unverifiedDeletion("Polar subscription %q is not in a terminal state: %s", item.ID, status)
	}
	if !storedEnded {
		return unverifiedDeletion("Polar did not return stored subscription %q in a terminal state", storedID)
	}
	return nil
}

func unverifiedDeletion(format string, args ...any) error {
	return fmt.Errorf("%w: %s", models.ErrOrganizationSubscriptionUnverified, fmt.Sprintf(format, args...))
}
