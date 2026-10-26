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
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var (
	ErrSubscriptionCheckoutDisabled = errors.New("subscription checkout is not configured")
	ErrAdminPlanCannotCancel        = errors.New("an admin plan cannot be canceled from billing")
	ErrSubscriptionNotCancelable    = errors.New("this organization has no subscription to cancel")
	ErrSubscriptionAlreadyCanceling = errors.New("the subscription is already set to end at the period end")
	ErrSubscriptionNotCanceling     = errors.New("the subscription is not set to end at the period end")
	// ErrSubscriptionForDeletedOrganization marks a subscription webhook for a
	// soft-deleted organization. ApplySubscription already recorded it with the
	// billing.subscription_for_deleted_organization log key, so the webhook
	// handler must not write the generic permanent-apply error for it.
	ErrSubscriptionForDeletedOrganization = errors.New("subscription targets a deleted organization")
)

func BusinessCheckoutProductID(tx *gorm.DB) (string, error) {
	plan, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanBusiness)
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

// ApplySubscriptionEvent applies a subscription webhook in its own
// transaction and writes the capacity alerts after the commit.
func ApplySubscriptionEvent(ctx context.Context, db *gorm.DB, event *SubscriptionWebhookEvent) error {
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
	return ApplySubscription(ctx, db, data)
}

// ApplySubscription applies an unverified snapshot in its own transaction and
// writes the capacity alerts after the commit.
func ApplySubscription(ctx context.Context, db *gorm.DB, data SubscriptionData) error {
	return applyAndEmit(ctx, db, data, false)
}

// ApplyOrganizationSubscription applies a snapshot that a direct Polar API
// call returned, in its own transaction, and writes the capacity alerts after
// the commit. Those snapshots can omit the customer, so the organization fills
// it in. Inside a caller transaction, use StageOrganizationSubscription.
func ApplyOrganizationSubscription(ctx context.Context, db *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
	return applyAndEmit(ctx, db, withOrganizationCustomer(orgID, data), true)
}

// StageOrganizationSubscription applies a verified snapshot inside the
// transaction of the caller. It writes no alert. It returns the staged
// capacity transitions, and the caller emits them only after its outermost
// transaction committed.
func StageOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) (SeatCapacityAlerts, error) {
	if !inTransaction(tx) {
		return nil, errors.New("stage the Polar subscription inside a transaction")
	}
	return applySubscription(ctx, tx, withOrganizationCustomer(orgID, data), true)
}

// applyAndEmit applies the snapshot in transactions that it owns, so the
// capacity alerts that it writes describe committed state only.
func applyAndEmit(ctx context.Context, db *gorm.DB, data SubscriptionData, verified bool) error {
	if inTransaction(db) {
		return errors.New("apply the Polar subscription outside a caller transaction, or stage it")
	}
	alerts, err := applySubscription(ctx, db, data, verified)
	if err != nil {
		return err
	}
	alerts.Emit()
	return nil
}

func withOrganizationCustomer(orgID uuid.UUID, data SubscriptionData) SubscriptionData {
	if strings.TrimSpace(data.organizationExternalID()) == "" {
		data.ExternalCustomerID = orgID.String()
	}
	return data
}

func inTransaction(db *gorm.DB) bool {
	_, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok
}

// applySubscription applies a subscription snapshot. Snapshots that a direct
// Polar API call returned are verified; webhook payloads are not, because a
// delayed event can claim access that Polar already ended.
//
// It returns the capacity transitions of the applied state and never writes
// them. When db is a caller transaction, the transaction below is a savepoint
// and the caller still owns the commit.
func applySubscription(ctx context.Context, db *gorm.DB, data SubscriptionData, verified bool) (SeatCapacityAlerts, error) {
	orgID, err := data.OrganizationID()
	if err != nil {
		return nil, err
	}
	incoming := subscriptionSnapshot(data)
	incoming.Verified = verified
	var refreshedStored, verifiedIncoming *SubscriptionData
	needsSnapshot := errors.New("Polar subscription snapshot required")
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var alerts SeatCapacityAlerts
		var readID, readReason string
		err = db.WithContext(ctx).Transaction(func(inner *gorm.DB) error {
			if _, err := models.LockOrganization(inner, orgID); err != nil {
				return err
			}
			if err := billingmodels.BindLegacyBillingProduct(inner, BusinessProductID()); err != nil {
				return err
			}
			previous, err := billingmodels.FindOrganizationBillingPlan(inner, orgID)
			if err != nil {
				return err
			}
			attemptIncoming, appliedData := incoming, data
			if verifiedIncoming != nil {
				// Another apply can adopt this identity during verification.
				// Use the verified snapshot even when it is no longer foreign.
				if polarSubscriptionID(previous) == incoming.ID {
					attemptIncoming = subscriptionSnapshot(*verifiedIncoming)
					attemptIncoming.Verified = true
					appliedData = *verifiedIncoming
				}
			}
			result, err := billingmodels.ApplyPolarSubscription(inner, orgID, attemptIncoming, nil)
			if err != nil {
				return err
			}
			if result.Disposition == billingmodels.PolarSubscriptionRefreshRequired {
				storedID := polarSubscriptionID(result.Plan)
				if refreshedStored == nil || refreshedStored.ID != storedID {
					readID, readReason = storedID, "refresh stored"
					return needsSnapshot
				}
				stored := subscriptionSnapshot(*refreshedStored)
				result, err = billingmodels.ApplyPolarSubscription(inner, orgID, attemptIncoming, &stored)
				if err != nil {
					return err
				}
			}
			refreshedResult := result.RefreshedStored
			if result.Disposition == billingmodels.PolarSubscriptionVerificationRequired {
				if verifiedIncoming == nil {
					readID, readReason = incoming.ID, "verify"
					return needsSnapshot
				}
				verifiedSnapshot := subscriptionSnapshot(*verifiedIncoming)
				verifiedSnapshot.Verified = true
				result, err = billingmodels.ApplyPolarSubscription(inner, orgID, verifiedSnapshot, nil)
				if err != nil {
					return err
				}
				appliedData = *verifiedIncoming
			}
			if refreshedResult != nil {
				if err := syncSubscriptionSideEffects(inner, orgID, *refreshedStored, refreshedResult); err != nil {
					return err
				}
			}
			if err := syncSubscriptionSideEffects(inner, orgID, appliedData, result); err != nil {
				return err
			}
			if result.Disposition != billingmodels.PolarSubscriptionApplied && refreshedResult == nil {
				return nil
			}
			// Check the final plan once, after an atomic subscription replacement.
			capacity, err := RecheckSeatCapacity(inner, orgID, snapshotSeatCapacityCause(previous, result.Plan))
			if err != nil {
				return err
			}
			alerts = SeatCapacityAlerts{capacity}
			return nil
		})
		if !errors.Is(err, needsSnapshot) {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_, lookupErr := models.FindOrganizationByIDInTransaction(db.WithContext(ctx), orgID.String())
				if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
					return nil, subscriptionOrganizationNotFound(db.WithContext(ctx), orgID, data)
				}
				if lookupErr != nil {
					return nil, lookupErr
				}
			}
			if errors.Is(err, billingmodels.ErrInvalidPurchasedSeats) && !errors.Is(err, billingmodels.ErrPolarSubscriptionRefreshFailed) {
				return nil, permanentApplyError(err.Error())
			}
			if err != nil {
				return nil, err
			}
			return alerts, nil
		}
		// Roll back every tentative write before HTTP. A new transaction
		// rechecks identity and ordering under the lock after the response.
		// An outer transaction belongs to the caller and cannot be released here.
		if inTransaction(db) {
			return nil, fmt.Errorf("%w: %s Polar subscription %q outside the caller transaction", billingmodels.ErrPolarSubscriptionRefreshFailed, readReason, readID)
		}
		refreshed, err := readOrganizationSubscription(ctx, orgID, readID, readReason)
		if err != nil {
			return nil, err
		}
		if readReason == "verify" {
			verifiedIncoming = refreshed
			continue
		}
		refreshedStored = refreshed
	}
}

func readOrganizationSubscription(ctx context.Context, orgID uuid.UUID, subscriptionID, reason string) (*SubscriptionData, error) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	snapshot, err := NewClientFromEnv().GetSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("%s Polar subscription %q: %w", reason, subscriptionID, err)
	}
	if snapshot.ID != subscriptionID {
		return nil, fmt.Errorf("%s Polar subscription does not match requested identity", reason)
	}
	if externalID := snapshot.organizationExternalID(); externalID != "" && externalID != orgID.String() {
		return nil, fmt.Errorf("%s Polar subscription customer does not match organization", reason)
	}
	return snapshot, nil
}

func subscriptionSnapshot(data SubscriptionData) billingmodels.PolarSubscriptionApply {
	return billingmodels.PolarSubscriptionApply{
		ID:                data.ID,
		ProductID:         data.ProductID,
		Status:            strings.ToLower(strings.TrimSpace(data.Status)),
		PeriodStart:       data.CurrentPeriodStart.TimePtr(),
		PeriodEnd:         data.CurrentPeriodEnd.TimePtr(),
		CancelAtPeriodEnd: data.CancelAtPeriodEnd,
		ModifiedAt:        data.ModifiedAt.TimePtr(),
		PurchasedSeats:    data.Seats,
		PastDueAt:         data.PastDueAt.TimePtr(),
	}
}

func syncSubscriptionSideEffects(tx *gorm.DB, orgID uuid.UUID, data SubscriptionData, result *billingmodels.PolarSubscriptionApplyResult) error {
	if result.Disposition != billingmodels.PolarSubscriptionApplied {
		return nil
	}
	if customerID := firstNonEmpty(data.Customer.ID, data.CustomerID); customerID != "" {
		if err := billingmodels.SetOrganizationPolarCustomerID(tx, orgID, customerID); err != nil {
			return err
		}
	}
	// A live subscription ends the checkout that created it. Polar names
	// that checkout, so the product or the seat count alone never resolves
	// an intent. An intent without a matching Polar checkout ID stays
	// unresolved until checkout reconciliation reads Polar.
	if billingmodels.PolarSubscriptionIsLive(data.Status, data.CancelAtPeriodEnd, data.CurrentPeriodEnd.TimePtr(), time.Now()) {
		if err := billingmodels.ResolveOrganizationBillingCheckoutForSubscription(tx, orgID, data.CheckoutID); err != nil {
			return err
		}
	}
	return billingmodels.SyncIncludedLLMCreditGrant(tx, billingmodels.IncludedUsageSync{
		OrganizationID:   orgID,
		SubscriptionID:   data.ID,
		PeriodEnd:        data.CurrentPeriodEnd.TimePtr(),
		GrantIncluded:    result.GrantIncluded,
		IsActiveBusiness: result.Plan != nil && result.Plan.IsActiveBusiness(),
	})
}

func subscriptionOrganizationNotFound(tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) error {
	logged, err := logSubscriptionForDeletedOrganization(tx, orgID, data)
	if err != nil {
		return err
	}
	if logged {
		return fmt.Errorf("%w: %w", ErrPermanentApply, ErrSubscriptionForDeletedOrganization)
	}
	return permanentApplyError("organization not found")
}

// subscriptionForDeletedOrganizationEvent is the stable log key for a
// subscription webhook that targets a soft-deleted organization. Polar can
// still charge for that subscription, so an operator must cancel it in Polar.
const subscriptionForDeletedOrganizationEvent = "billing.subscription_for_deleted_organization"

// logSubscriptionForDeletedOrganization records one error log when the webhook
// targets a soft-deleted organization and reports whether it did.
func logSubscriptionForDeletedOrganization(tx *gorm.DB, orgID uuid.UUID, data SubscriptionData) (bool, error) {
	organization, err := models.FindOrganizationByIDIncludingDeleted(tx, orgID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !organization.DeletedAt.Valid {
		return false, nil
	}

	log.WithFields(log.Fields{
		"event":                     subscriptionForDeletedOrganizationEvent,
		"organization_id":           orgID.String(),
		"polar_subscription_id":     data.ID,
		"polar_subscription_status": strings.ToLower(strings.TrimSpace(data.Status)),
	}).Error(subscriptionForDeletedOrganizationEvent)
	return true, nil
}

func SyncOrganizationSubscription(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) error {
	if !Configured() {
		return nil
	}

	if err := billingmodels.BindLegacyBillingProduct(tx, BusinessProductID()); err != nil {
		return err
	}
	items, err := NewClientFromEnv().ListSubscriptions(ctx, orgID.String(), "")
	if err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("failed to list Polar subscriptions")
		return nil
	}

	known := make([]SubscriptionData, 0, len(items))
	for _, item := range items {
		_, err := billingmodels.FindBillingPlanByProduct(tx, item.ProductID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		known = append(known, item)
	}
	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	now := time.Now()
	selected := selectBusinessSubscription(known, polarSubscriptionID(plan), now)
	if selected == nil {
		return nil
	}
	reportConflictingLiveSubscriptions(orgID, known, selected, now)
	return ApplyOrganizationSubscription(ctx, tx, orgID, *selected)
}

// Sync keeps one subscription, but Polar can hold more than one live
// subscription for the same organization. Report every conflicting live
// subscription so operators can cancel the duplicate charges.
func reportConflictingLiveSubscriptions(orgID uuid.UUID, items []SubscriptionData, selected *SubscriptionData, now time.Time) {
	for i := range items {
		item := &items[i]
		if item.ID == selected.ID {
			continue
		}
		if !billingmodels.PolarSubscriptionIsLive(item.Status, item.CancelAtPeriodEnd, item.CurrentPeriodEnd.TimePtr(), now) {
			continue
		}
		log.WithFields(log.Fields{
			"organization_id":              orgID.String(),
			"stored_subscription_id":       selected.ID,
			"incoming_subscription_id":     item.ID,
			"stored_subscription_status":   strings.ToLower(strings.TrimSpace(selected.Status)),
			"incoming_subscription_status": strings.ToLower(strings.TrimSpace(item.Status)),
		}).Error("billing.multiple_live_subscriptions")
	}
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
		live := billingmodels.PolarSubscriptionIsLive(item.Status, item.CancelAtPeriodEnd, item.CurrentPeriodEnd.TimePtr(), now)
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

// OrganizationID identifies the organization in a subscription snapshot.
func (data SubscriptionData) OrganizationID() (uuid.UUID, error) {
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

	plan, err := billingmodels.FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if plan == nil {
		return ErrSubscriptionNotCancelable
	}
	if plan.PlanSource == billingmodels.BillingPlanSourceAdmin {
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

func polarSubscriptionID(plan *billingmodels.OrganizationBillingPlan) string {
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
		if !billingmodels.PolarSubscriptionStatusIsKnown(status) {
			return "failed", fmt.Errorf("unexpected Polar subscription status %q", item.Status)
		}
		if !item.EndedAt.Time.IsZero() && !item.EndedAt.Time.After(now) {
			continue
		}
		if billingmodels.PolarSubscriptionIsLive(status, item.CancelAtPeriodEnd, item.CurrentPeriodEnd.TimePtr(), now) {
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
