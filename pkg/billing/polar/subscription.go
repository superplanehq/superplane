package polar

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func ApplySubscriptionEvent(ctx context.Context, tx *gorm.DB, event *SubscriptionWebhookEvent) error {
	if event == nil {
		return permanentApplyError("subscription event is required")
	}
	if !isSubscriptionEventType(event.Type) {
		return nil
	}
	return ApplySubscription(ctx, tx, event.Data)
}

func ApplySubscription(ctx context.Context, tx *gorm.DB, data SubscriptionData) error {
	orgID, err := parseSubscriptionOrganizationID(data)
	if err != nil {
		return err
	}
	tx = tx.WithContext(ctx)
	if _, err := models.FindOrganizationByIDInTransaction(tx, orgID.String()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return permanentApplyError("organization not found")
		}
		return err
	}

	status := strings.ToLower(strings.TrimSpace(data.Status))
	var periodStart, periodEnd *time.Time
	if !data.CurrentPeriodStart.Time.IsZero() {
		start := data.CurrentPeriodStart.Time
		periodStart = &start
	}
	if !data.CurrentPeriodEnd.Time.IsZero() {
		end := data.CurrentPeriodEnd.Time
		periodEnd = &end
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		plan, grantIncluded, err := models.ApplyPolarSubscription(
			inner,
			orgID,
			data.ID,
			status,
			periodStart,
			periodEnd,
		)
		if err != nil {
			return err
		}
		if customerID := firstNonEmpty(data.Customer.ID, data.CustomerID); customerID != "" {
			if err := models.SetOrganizationPolarCustomerID(inner, orgID, customerID); err != nil {
				return err
			}
		}
		if plan != nil && plan.PlanSource == models.BillingPlanSourceAdmin {
			return nil
		}
		if !grantIncluded {
			return nil
		}
		if err := models.ConvertOpenTrialAllowanceToTopup(inner, orgID, data.ID); err != nil {
			return err
		}
		if periodEnd == nil {
			return nil
		}
		start := time.Now()
		if periodStart != nil {
			start = *periodStart
		}
		_, err = models.AddIncludedLLMCreditGrant(
			inner,
			orgID,
			models.CentsToMicros(models.DefaultIncludedGrantCents),
			models.IncludedGrantKey(data.ID, start),
			*periodEnd,
		)
		return err
	})
}

func SyncOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) error {
	if !SubscriptionCheckoutEnabled() {
		return nil
	}

	plan, err := models.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if plan != nil && plan.PlanSource == models.BillingPlanSourceAdmin {
		return nil
	}

	items, err := NewClientFromEnv().ListSubscriptions(ctx, orgID.String(), BusinessProductID())
	if err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("failed to list Polar subscriptions")
		return nil
	}

	selected := selectBusinessSubscription(items)
	if selected == nil {
		return nil
	}
	if strings.TrimSpace(selected.organizationExternalID()) == "" {
		selected.ExternalCustomerID = orgID.String()
	}
	return ApplySubscription(ctx, tx, *selected)
}

func selectBusinessSubscription(items []SubscriptionData) *SubscriptionData {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if polarSubscriptionIsPaid(items[i].Status) {
			return &items[i]
		}
	}
	return &items[0]
}

func polarSubscriptionIsPaid(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case models.PolarSubscriptionStatusActive, models.PolarSubscriptionStatusTrialing:
		return true
	default:
		return false
	}
}

func parseSubscriptionOrganizationID(data SubscriptionData) (uuid.UUID, error) {
	orgID, err := uuid.Parse(data.organizationExternalID())
	if err != nil {
		return uuid.Nil, permanentApplyError("subscription customer external id is not an organization id")
	}
	return orgID, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
