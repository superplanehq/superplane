package polar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
)

// VerifyOrganizationDeletion returns the clearance that Organization.SoftDelete
// needs. Call it before the deletion transaction: it takes no lock, and the
// Polar request must not hold the organization lock. SoftDelete checks again
// under the lock that the billing row did not change.
//
// A billing row that cannot bill gets a local clearance without a Polar
// request. A row that may still bill needs Polar to return the stored
// subscription and every other subscription in a terminal state. A live
// subscription returns billingmodels.ErrOrganizationSubscriptionLive. A Polar
// failure or an unclear answer returns
// billingmodels.ErrOrganizationSubscriptionUnverified that wraps the cause.
//
// A payable checkout from before intents existed is adopted first, and any
// unresolved checkout intent is verified. A checkout that can still
// activate returns billingmodels.ErrOrganizationCheckoutPending: a payment after the
// deletion would start a subscription for a deleted organization. A paid
// checkout applies its subscription before the billing row is read.
//
// The verifier does not write the Polar subscription snapshot outside the
// checkout and subscription apply paths. Webhooks and the subscription
// refresh own the billing row.
func VerifyOrganizationDeletion(ctx context.Context, orgID uuid.UUID) (*billingmodels.OrganizationDeletionClearance, error) {
	if err := settleCheckoutBeforeDeletion(ctx, orgID); err != nil {
		return nil, err
	}
	plan, err := billingmodels.FindOrganizationBillingPlan(database.DB(ctx), orgID)
	if err != nil {
		return nil, err
	}
	if !plan.MayBeBillable(time.Now()) {
		return billingmodels.NewLocalDeletionClearance(orgID, plan)
	}
	if !Configured() {
		return nil, unverifiedDeletion("Polar is not configured")
	}
	if err := confirmSubscriptionEnded(ctx, orgID, polarSubscriptionID(plan)); err != nil {
		return nil, err
	}
	return billingmodels.NewVerifiedDeletionClearance(plan), nil
}

// maxDeletionCheckoutAttempts bounds the verification loop. Each repeat means
// that another writer changed the intent in between.
const maxDeletionCheckoutAttempts = 3

// settleCheckoutBeforeDeletion makes sure that no checkout of the
// organization can still activate. SoftDelete checks again under the
// organization lock that no unresolved intent exists.
func settleCheckoutBeforeDeletion(ctx context.Context, orgID uuid.UUID) error {
	if err := adoptPreexistingCheckoutBeforeDeletion(ctx, orgID); err != nil {
		return err
	}
	for range maxDeletionCheckoutAttempts {
		intent, err := billingmodels.FindUnresolvedOrganizationBillingCheckout(database.DB(ctx), orgID)
		if err != nil || intent == nil {
			return err
		}
		if !Configured() {
			return unverifiedDeletion("Polar is not configured to verify checkout intent %s", intent.ID)
		}
		verification, err := NewCheckoutLifecycle(NewClientFromEnv()).Verify(ctx, intent)
		if err != nil {
			return fmt.Errorf("%w: verify checkout intent %s: %w", billingmodels.ErrOrganizationSubscriptionUnverified, intent.ID, err)
		}
		switch verification.Outcome {
		case CheckoutResolved, CheckoutChanged:
			continue
		case CheckoutSubscriptionLive:
			return fmt.Errorf("%w: checkout intent %s started a live subscription", billingmodels.ErrOrganizationSubscriptionLive, intent.ID)
		default:
			return fmt.Errorf("%w: checkout intent %s is %s", billingmodels.ErrOrganizationCheckoutPending, intent.ID, verification.Outcome)
		}
	}
	return billingmodels.ErrOrganizationCheckoutPending
}

func adoptPreexistingCheckoutBeforeDeletion(ctx context.Context, orgID uuid.UUID) error {
	db := database.DB(ctx)
	hasCustomer, err := billingmodels.OrganizationHasPolarCustomer(db, orgID)
	if err != nil || !hasCustomer {
		return err
	}
	known, err := billingmodels.OrganizationHasBillingCheckouts(db, orgID)
	if err != nil || known {
		return err
	}
	if !Configured() {
		return unverifiedDeletion("Polar is not configured to verify preexisting checkouts")
	}
	if err := NewCheckoutLifecycle(NewClientFromEnv()).AdoptPreexisting(ctx, orgID); err != nil {
		return fmt.Errorf("%w: verify preexisting checkouts: %w", billingmodels.ErrOrganizationSubscriptionUnverified, err)
	}
	return nil
}

func confirmSubscriptionEnded(ctx context.Context, orgID uuid.UUID, storedID string) error {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	items, err := NewClientFromEnv().ListSubscriptions(ctx, orgID.String(), "")
	if err != nil {
		return fmt.Errorf("%w: list Polar subscriptions: %w", billingmodels.ErrOrganizationSubscriptionUnverified, err)
	}

	now := time.Now()
	storedEnded := false
	for _, item := range items {
		if externalID := item.organizationExternalID(); externalID != "" && externalID != orgID.String() {
			return unverifiedDeletion("Polar subscription %q customer does not match the organization", item.ID)
		}
		status := strings.ToLower(strings.TrimSpace(item.Status))
		if !billingmodels.PolarSubscriptionStatusIsKnown(status) {
			return unverifiedDeletion("Polar subscription %q has unexpected status %q", item.ID, item.Status)
		}
		periodEnd := item.CurrentPeriodEnd.TimePtr()
		terminal := billingmodels.PolarSubscriptionIsTerminal(status, item.CancelAtPeriodEnd, periodEnd, now)
		if item.ID == storedID {
			storedEnded = terminal
		}
		if terminal {
			continue
		}
		if billingmodels.PolarSubscriptionIsLive(status, item.CancelAtPeriodEnd, periodEnd, now) {
			return fmt.Errorf("%w: Polar subscription %q is %s", billingmodels.ErrOrganizationSubscriptionLive, item.ID, status)
		}
		return unverifiedDeletion("Polar subscription %q is not in a terminal state: %s", item.ID, status)
	}
	if !storedEnded {
		return unverifiedDeletion("Polar did not return stored subscription %q in a terminal state", storedID)
	}
	return nil
}

func unverifiedDeletion(format string, args ...any) error {
	return fmt.Errorf("%w: %s", billingmodels.ErrOrganizationSubscriptionUnverified, fmt.Sprintf(format, args...))
}
