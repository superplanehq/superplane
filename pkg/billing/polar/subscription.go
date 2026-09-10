package polar

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
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

	orgID, err := parseSubscriptionOrganizationID(event.Data)
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

	status := strings.ToLower(strings.TrimSpace(event.Data.Status))
	var periodStart, periodEnd *time.Time
	if !event.Data.CurrentPeriodStart.Time.IsZero() {
		start := event.Data.CurrentPeriodStart.Time
		periodStart = &start
	}
	if !event.Data.CurrentPeriodEnd.Time.IsZero() {
		end := event.Data.CurrentPeriodEnd.Time
		periodEnd = &end
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		plan, grantIncluded, err := models.ApplyPolarSubscription(
			inner,
			orgID,
			event.Data.ID,
			status,
			periodStart,
			periodEnd,
		)
		if err != nil {
			return err
		}
		if customerID := firstNonEmpty(event.Data.Customer.ID, event.Data.CustomerID); customerID != "" {
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
		if err := models.ConvertOpenTrialAllowanceToTopup(inner, orgID, event.Data.ID); err != nil {
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
			models.IncludedGrantKey(event.Data.ID, start),
			*periodEnd,
		)
		return err
	})
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
