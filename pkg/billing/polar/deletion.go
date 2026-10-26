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
// The verifier does not write the Polar snapshot. Webhooks and the
// subscription refresh own the billing row.
func VerifyOrganizationDeletion(ctx context.Context, orgID uuid.UUID) (*models.OrganizationDeletionClearance, error) {
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
