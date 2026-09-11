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

var (
	ErrSubscriptionCheckoutDisabled = errors.New("business checkout is not configured")
	ErrAdminPlanCannotCancel        = errors.New("an admin plan cannot be canceled from billing")
	ErrSubscriptionNotCancelable    = errors.New("this organization has no business subscription to cancel")
	ErrSubscriptionAlreadyCanceling = errors.New("business is already set to end at the period end")
	ErrSubscriptionNotCanceling     = errors.New("business is not set to end at the period end")
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
			models.PolarSubscriptionApply{
				ID:                data.ID,
				Status:            status,
				PeriodStart:       periodStart,
				PeriodEnd:         periodEnd,
				CancelAtPeriodEnd: data.CancelAtPeriodEnd,
			},
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
		return models.SyncIncludedLLMCreditGrant(inner, models.IncludedUsageSync{
			OrganizationID:   orgID,
			SubscriptionID:   data.ID,
			PeriodEnd:        periodEnd,
			GrantIncluded:    grantIncluded,
			IsActiveBusiness: plan != nil && plan.IsActiveBusiness(),
		})
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
		if models.PolarSubscriptionIsPaid(items[i].Status) {
			return &items[i]
		}
	}
	return &items[0]
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

func CancelOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) error {
	return setOrganizationSubscriptionCancelAtPeriodEnd(ctx, tx, orgID, true)
}

func ResumeOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) error {
	return setOrganizationSubscriptionCancelAtPeriodEnd(ctx, tx, orgID, false)
}

func setOrganizationSubscriptionCancelAtPeriodEnd(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, cancelAtPeriodEnd bool) error {
	if !SubscriptionCheckoutEnabled() {
		return ErrSubscriptionCheckoutDisabled
	}

	plan, err := models.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if plan == nil {
		return ErrSubscriptionNotCancelable
	}
	if plan.PlanSource == models.BillingPlanSourceAdmin {
		return ErrAdminPlanCannotCancel
	}

	subscriptionID := polarSubscriptionID(plan)
	if subscriptionID == "" || !plan.IsActiveBusiness() {
		if cancelAtPeriodEnd {
			return ErrSubscriptionNotCancelable
		}
		return ErrSubscriptionNotCanceling
	}
	if cancelAtPeriodEnd && plan.CancelAtPeriodEnd {
		return ErrSubscriptionAlreadyCanceling
	}
	if !cancelAtPeriodEnd && !plan.CancelAtPeriodEnd {
		return ErrSubscriptionNotCanceling
	}

	client := NewClientFromEnv()
	var updated *SubscriptionData
	if cancelAtPeriodEnd {
		updated, err = client.CancelSubscriptionAtPeriodEnd(ctx, subscriptionID)
	} else {
		updated, err = client.ResumeSubscription(ctx, subscriptionID)
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(updated.organizationExternalID()) == "" {
		updated.ExternalCustomerID = orgID.String()
	}
	return ApplySubscription(ctx, tx, *updated)
}

func polarSubscriptionID(plan *models.OrganizationBillingPlan) string {
	if plan == nil || plan.PolarSubscriptionID == nil {
		return ""
	}
	return strings.TrimSpace(*plan.PolarSubscriptionID)
}
