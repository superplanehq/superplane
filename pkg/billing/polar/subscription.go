package polar

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
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
	tx = tx.WithContext(ctx)
	if err := models.BindLegacyBillingProduct(tx, BusinessProductID()); err != nil {
		return err
	}
	if _, err := models.FindOrganizationByIDInTransaction(tx, orgID.String()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := logSubscriptionForDeletedOrganization(tx, orgID, data); err != nil {
				return err
			}
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
				ProductID:         data.ProductID,
				Status:            status,
				PeriodStart:       periodStart,
				PeriodEnd:         periodEnd,
				CancelAtPeriodEnd: data.CancelAtPeriodEnd,
				ModifiedAt:        data.ModifiedAt.TimePtr(),
				PurchasedSeats:    data.Seats,
			},
		)
		if err != nil {
			if errors.Is(err, models.ErrInvalidPurchasedSeats) {
				return permanentApplyError(err.Error())
			}
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

// subscriptionForDeletedOrganizationEvent is the stable log key for a
// subscription webhook that targets a soft-deleted organization. Polar can
// still charge for that subscription, so an operator must cancel it in Polar.
const subscriptionForDeletedOrganizationEvent = "billing.subscription_for_deleted_organization"

func logSubscriptionForDeletedOrganization(tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
	organization, err := models.FindOrganizationByIDIncludingDeleted(tx, orgID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !organization.DeletedAt.Valid {
		return nil
	}

	log.WithFields(log.Fields{
		"event":                     subscriptionForDeletedOrganizationEvent,
		"organization_id":           orgID.String(),
		"polar_subscription_id":     data.ID,
		"polar_subscription_status": strings.ToLower(strings.TrimSpace(data.Status)),
	}).Error(subscriptionForDeletedOrganizationEvent)
	return nil
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
	selected := selectBusinessSubscription(known)
	if selected == nil {
		return nil
	}
	return ApplyOrganizationSubscription(ctx, tx, orgID, *selected)
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
		switch status {
		case "active", "trialing", "canceled", "unpaid", "revoked", "past_due", "incomplete", "incomplete_expired":
		default:
			return "failed", fmt.Errorf("unexpected Polar subscription status %q", item.Status)
		}
		if !item.EndedAt.Time.IsZero() && !item.EndedAt.Time.After(now) {
			continue
		}
		if models.PolarSubscriptionIsPaid(status) || (item.CancelAtPeriodEnd && item.CurrentPeriodEnd.Time.After(now)) {
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
