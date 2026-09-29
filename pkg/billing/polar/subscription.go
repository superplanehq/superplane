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
	ErrSubscriptionCheckoutDisabled      = errors.New("business checkout is not configured")
	ErrAdminPlanCannotCancel             = errors.New("an admin plan cannot be canceled from billing")
	ErrSubscriptionNotCancelable         = errors.New("this organization has no business subscription to cancel")
	ErrSubscriptionAlreadyCanceling      = errors.New("business is already set to end at the period end")
	ErrSubscriptionNotCanceling          = errors.New("business is not set to end at the period end")
	ErrSubscriptionChangedDuringDeletion = errors.New("subscription changed during organization deletion")
)

const maxDeletedOrganizationSubscriptionCancels = 3

func ApplySubscriptionEvent(ctx context.Context, tx *gorm.DB, event *SubscriptionWebhookEvent) error {
	if event == nil {
		return permanentApplyError("subscription event is required")
	}
	if !isSubscriptionEventType(event.Type) {
		return nil
	}
	data := event.Data
	if data.ModifiedAt.Time.IsZero() && !event.Timestamp.Time.IsZero() {
		data.ModifiedAt = event.Timestamp
	}
	return ApplySubscription(ctx, tx, data)
}

func ApplySubscription(ctx context.Context, tx *gorm.DB, data SubscriptionData) error {
	orgID, err := parseSubscriptionOrganizationID(data)
	if err != nil {
		return err
	}
	tx = tx.WithContext(ctx)

	var missing bool
	var deleted bool
	err = tx.Transaction(func(inner *gorm.DB) error {
		organization, err := models.LockOrganizationIncludingDeleted(inner, orgID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				missing = true
				return nil
			}
			return err
		}
		if organization.DeletedAt.Valid {
			deleted = true
			return nil
		}
		return applyOpenOrganizationSubscription(ctx, inner, orgID, data)
	})
	if err != nil {
		return err
	}
	if missing {
		return permanentApplyError("organization not found")
	}
	if deleted {
		return cancelSubscriptionForDeletedOrganization(ctx, tx, orgID, data)
	}
	return nil
}

func applyOpenOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
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
				ModifiedAt:        data.ModifiedAt.TimePtr(),
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
		return models.SyncIncludedLLMCreditGrant(inner, models.IncludedUsageSync{
			OrganizationID:   orgID,
			SubscriptionID:   data.ID,
			PeriodEnd:        periodEnd,
			GrantIncluded:    grantIncluded,
			IsActiveBusiness: plan != nil && plan.IsActiveBusiness(),
		})
	})
}

func cancelSubscriptionForDeletedOrganization(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
	if !subscriptionDataRenews(data) {
		return permanentApplyError("organization not found")
	}
	if !SubscriptionCheckoutEnabled() {
		return errors.New("cannot cancel Polar subscription for deleted organization: checkout is not configured")
	}

	if err := cancelDifferentRenewingSubscription(ctx, tx, orgID, data.ID); err != nil {
		return err
	}

	updated, err := NewClientFromEnv().CancelSubscriptionAtPeriodEnd(ctx, data.ID)
	if err != nil {
		return err
	}
	return persistDeletedOrganizationCancellation(ctx, tx, orgID, *updated)
}

func cancelDifferentRenewingSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, incomingID string) error {
	incomingID = strings.TrimSpace(incomingID)
	for attempt := 0; attempt < maxDeletedOrganizationSubscriptionCancels; attempt++ {
		plan, err := models.FindOrganizationBillingPlan(tx, orgID)
		if err != nil {
			return err
		}
		currentID := polarSubscriptionID(plan)
		if currentID == "" || currentID == incomingID || !subscriptionNeedsDeletionCancel(plan) {
			return nil
		}

		updated, err := NewClientFromEnv().CancelSubscriptionAtPeriodEnd(ctx, currentID)
		if err != nil {
			return err
		}
		if err := persistDeletedOrganizationCancellation(ctx, tx, orgID, *updated); err != nil {
			return err
		}
	}

	plan, err := models.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	currentID := polarSubscriptionID(plan)
	if currentID != "" && currentID != incomingID && subscriptionNeedsDeletionCancel(plan) {
		return errors.New("deleted organization still has a renewing subscription")
	}
	return nil
}

func persistDeletedOrganizationCancellation(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
	if strings.TrimSpace(data.organizationExternalID()) == "" {
		data.ExternalCustomerID = orgID.String()
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		if _, err := models.LockOrganizationIncludingDeleted(inner, orgID); err != nil {
			return err
		}
		plan, err := models.FindOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		currentID := polarSubscriptionID(plan)
		incomingID := strings.TrimSpace(data.ID)
		if currentID != "" && currentID != incomingID {
			return nil
		}
		return applyOpenOrganizationSubscription(ctx, inner, orgID, data)
	})
}

func subscriptionDataRenews(data SubscriptionData) bool {
	if strings.TrimSpace(data.ID) == "" || data.CancelAtPeriodEnd {
		return false
	}
	return models.PolarSubscriptionIsPaid(data.Status)
}

func SyncOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) error {
	if !SubscriptionCheckoutEnabled() {
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

func CancelOrganizationSubscriptionForDeletion(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	if !SubscriptionCheckoutEnabled() {
		err := deletionBlockedByActiveSubscription(tx, orgID)
		return false, err
	}

	err := CancelOrganizationSubscription(ctx, tx, orgID)
	if err == nil {
		return true, nil
	}
	if subscriptionCancelCanBeSkipped(err) {
		return false, nil
	}
	return false, err
}

func SubscriptionChangedSinceDeletionCancel(tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	plan, err := models.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return false, err
	}
	return subscriptionNeedsDeletionCancel(plan), nil
}

func deletionBlockedByActiveSubscription(tx *gorm.DB, orgID uuid.UUID) error {
	plan, err := models.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if subscriptionNeedsDeletionCancel(plan) {
		return ErrSubscriptionCheckoutDisabled
	}
	return nil
}

func subscriptionNeedsDeletionCancel(plan *models.OrganizationBillingPlan) bool {
	if plan == nil || plan.PlanSource == models.BillingPlanSourceAdmin {
		return false
	}
	if polarSubscriptionID(plan) == "" || plan.CancelAtPeriodEnd || !plan.IsActiveBusiness() {
		return false
	}
	return true
}

func subscriptionCancelCanBeSkipped(err error) bool {
	return errors.Is(err, ErrSubscriptionNotCancelable) ||
		errors.Is(err, ErrSubscriptionAlreadyCanceling) ||
		errors.Is(err, ErrAdminPlanCannotCancel)
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
