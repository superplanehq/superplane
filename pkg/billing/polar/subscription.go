package polar

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
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

func BusinessCheckoutProductID(tx *gorm.DB) (string, error) {
	plan, err := models.FindBillingPlan(tx, models.BillingPlanBusiness)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	if err == nil && plan.PolarProductID != nil {
		if id := strings.TrimSpace(*plan.PolarProductID); id != "" {
			return id, nil
		}
	}
	return BusinessProductID(), nil
}

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
	err = tx.WithContext(ctx).Transaction(func(inner *gorm.DB) error {
		if _, err := models.LockOrganization(inner, orgID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return permanentApplyError("organization not found")
			}
			return err
		}
		if err := models.BindLegacyBillingProduct(inner, BusinessProductID()); err != nil {
			return err
		}
		incoming := subscriptionSnapshot(data)
		result, err := models.ApplyPolarSubscription(inner, orgID, incoming, nil)
		if err != nil {
			return err
		}
		if result.Disposition == models.PolarSubscriptionRefreshRequired {
			storedID := polarSubscriptionID(result.Plan)
			refreshCtx, cancel := context.WithTimeout(ctx, httpTimeout)
			defer cancel()
			refreshed, err := NewClientFromEnv().GetSubscription(refreshCtx, storedID)
			if err != nil {
				return fmt.Errorf("refresh stored Polar subscription %q: %w", storedID, err)
			}
			if externalID := refreshed.organizationExternalID(); externalID != "" && externalID != orgID.String() {
				return fmt.Errorf("refreshed Polar subscription customer does not match organization")
			}
			stored := subscriptionSnapshot(*refreshed)
			result, err = models.ApplyPolarSubscription(inner, orgID, incoming, &stored)
			if err != nil {
				return err
			}
			if result.RefreshedStored != nil {
				if err := syncSubscriptionSideEffects(inner, orgID, *refreshed, result.RefreshedStored); err != nil {
					return err
				}
			}
		}
		return syncSubscriptionSideEffects(inner, orgID, data, result)
	})
	if errors.Is(err, models.ErrInvalidPurchasedSeats) && !errors.Is(err, models.ErrPolarSubscriptionRefreshFailed) {
		return permanentApplyError(err.Error())
	}
	return err
}

func subscriptionSnapshot(data SubscriptionData) models.PolarSubscriptionApply {
	return models.PolarSubscriptionApply{
		ID:                data.ID,
		ProductID:         data.ProductID,
		Status:            strings.ToLower(strings.TrimSpace(data.Status)),
		PeriodStart:       data.CurrentPeriodStart.TimePtr(),
		PeriodEnd:         data.CurrentPeriodEnd.TimePtr(),
		CancelAtPeriodEnd: data.CancelAtPeriodEnd,
		ModifiedAt:        data.ModifiedAt.TimePtr(),
		PurchasedSeats:    data.Seats,
	}
}

func syncSubscriptionSideEffects(tx *gorm.DB, orgID uuid.UUID, data SubscriptionData, result *models.PolarSubscriptionApplyResult) error {
	if result.Disposition != models.PolarSubscriptionApplied {
		return nil
	}
	if customerID := firstNonEmpty(data.Customer.ID, data.CustomerID); customerID != "" {
		if err := models.SetOrganizationPolarCustomerID(tx, orgID, customerID); err != nil {
			return err
		}
	}
	return models.SyncIncludedLLMCreditGrant(tx, models.IncludedUsageSync{
		OrganizationID:   orgID,
		SubscriptionID:   data.ID,
		PeriodEnd:        data.CurrentPeriodEnd.TimePtr(),
		GrantIncluded:    result.GrantIncluded,
		IsActiveBusiness: result.Plan != nil && result.Plan.IsActiveBusiness(),
	})
}

// ApplyOrganizationSubscription applies a snapshot that a direct Polar API call
// returned. Those snapshots can omit the customer, so the organization fills it in.
func ApplyOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
	if strings.TrimSpace(data.organizationExternalID()) == "" {
		data.ExternalCustomerID = orgID.String()
	}
	return ApplySubscription(ctx, tx, data)
}

func SyncOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) error {
	if !Configured() {
		return nil
	}

	if err := models.BindLegacyBillingProduct(tx, BusinessProductID()); err != nil {
		return err
	}
	items, err := NewClientFromEnv().ListSubscriptions(ctx, orgID.String(), "")
	if err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("failed to list Polar subscriptions")
		return nil
	}

	known := make([]SubscriptionData, 0, len(items))
	for _, item := range items {
		_, err := models.FindBillingPlanByProduct(tx, item.ProductID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		known = append(known, item)
	}
	plan, err := models.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	selected := selectBusinessSubscription(known, polarSubscriptionID(plan), time.Now())
	if selected == nil {
		return nil
	}
	return ApplyOrganizationSubscription(ctx, tx, orgID, *selected)
}

func selectBusinessSubscription(items []SubscriptionData, storedID string, now time.Time) *SubscriptionData {
	slices.SortFunc(items, func(a, b SubscriptionData) int {
		if a.CreatedAt.Time.IsZero() != b.CreatedAt.Time.IsZero() {
			if a.CreatedAt.Time.IsZero() {
				return 1
			}
			return -1
		}
		if order := b.CreatedAt.Time.Compare(a.CreatedAt.Time); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	var stored, newestLive *SubscriptionData
	for i := range items {
		item := &items[i]
		live := models.PolarSubscriptionIsLive(item.Status, item.CancelAtPeriodEnd, item.CurrentPeriodEnd.TimePtr(), now)
		if item.ID == storedID {
			stored = item
			if live {
				return stored
			}
		}
		if live && newestLive == nil {
			newestLive = item
		}
	}
	if newestLive != nil {
		return newestLive
	}
	return stored
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
	if !Configured() {
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
	return ApplyOrganizationSubscription(ctx, tx, orgID, *updated)
}

func polarSubscriptionID(plan *models.OrganizationBillingPlan) string {
	if plan == nil || plan.PolarSubscriptionID == nil {
		return ""
	}
	return strings.TrimSpace(*plan.PolarSubscriptionID)
}

// VerifyOrganizationSubscription checks every product without changing local billing.
func VerifyOrganizationSubscription(ctx context.Context, orgID uuid.UUID) (string, error) {
	if !Configured() {
		return "not_configured", nil
	}
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	items, err := NewClientFromEnv().ListSubscriptions(ctx, orgID.String(), "")
	if err != nil {
		return "failed", err
	}
	active := false
	now := time.Now()
	for _, item := range items {
		if externalID := item.organizationExternalID(); externalID != "" && externalID != orgID.String() {
			return "failed", fmt.Errorf("polar subscription customer does not match organization")
		}
		status := strings.ToLower(strings.TrimSpace(item.Status))
		if !models.PolarSubscriptionStatusIsKnown(status) {
			return "failed", fmt.Errorf("unexpected Polar subscription status %q", item.Status)
		}
		if !item.EndedAt.Time.IsZero() && !item.EndedAt.Time.After(now) {
			continue
		}
		if models.PolarSubscriptionIsLive(status, item.CancelAtPeriodEnd, item.CurrentPeriodEnd.TimePtr(), now) {
			active = true
		}
	}
	if active {
		return "active", nil
	}
	return "none", nil
}

func SubscriptionDashboardURL(subscriptionID string) string {
	if strings.TrimSpace(subscriptionID) == "" {
		return ""
	}
	slug := strings.TrimSpace(os.Getenv("POLAR_DASHBOARD_ORGANIZATION_SLUG"))
	if slug == "" {
		slug = "superplane"
		if strings.EqualFold(strings.TrimSpace(os.Getenv("POLAR_ENVIRONMENT")), "production") {
			slug = "superplane-inc"
		}
	}
	host := "https://sandbox.polar.sh"
	if strings.EqualFold(strings.TrimSpace(os.Getenv("POLAR_ENVIRONMENT")), "production") {
		host = "https://polar.sh"
	}
	return host + "/dashboard/" + url.PathEscape(slug) + "/sales/subscriptions/" + url.PathEscape(subscriptionID)
}
