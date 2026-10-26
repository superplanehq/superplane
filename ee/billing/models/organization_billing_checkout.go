package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Lifecycle states of a subscription checkout intent.
//
// The states creating, ambiguous, and open are unresolved. An organization has
// at most one unresolved intent. A resolved intent never changes again.
const (
	// BillingCheckoutStateCreating means the intent exists and the Polar create
	// call has not returned yet.
	BillingCheckoutStateCreating = "creating"
	// BillingCheckoutStateAmbiguous means the Polar create call may have
	// succeeded. The intent ID in the checkout metadata finds the checkout.
	BillingCheckoutStateAmbiguous = "ambiguous"
	// BillingCheckoutStateOpen means Polar holds a payable checkout.
	BillingCheckoutStateOpen = "open"
	// BillingCheckoutStateSucceeded means the checkout succeeded or the
	// subscription of the intent product became live.
	BillingCheckoutStateSucceeded = "succeeded"
	// BillingCheckoutStateExpired means Polar reported the checkout as expired.
	BillingCheckoutStateExpired = "expired"
	// BillingCheckoutStateFailed means Polar reported the checkout as failed.
	BillingCheckoutStateFailed = "failed"
	// BillingCheckoutStateReleased means no payable checkout exists for the
	// intent. Polar rejected the create call, or no checkout appeared.
	BillingCheckoutStateReleased = "released"
)

// ErrBillingCheckoutStateConflict reports that the intent is not in a state
// that allows the transition. Another writer changed it first.
var ErrBillingCheckoutStateConflict = errors.New("billing checkout intent changed state")

var ErrBillingCheckoutLiveSubscription = errors.New("organization has a live subscription")

// ErrBillingCheckoutSeatsBelowMembers rejects a checkout whose seat count does
// not cover the current members. Payment of such a checkout would leave more
// members than paid seats.
var ErrBillingCheckoutSeatsBelowMembers = errors.New("the selected seats do not cover the current members")

var unresolvedBillingCheckoutStates = []string{
	BillingCheckoutStateCreating,
	BillingCheckoutStateAmbiguous,
	BillingCheckoutStateOpen,
}

var resolvedBillingCheckoutStates = []string{
	BillingCheckoutStateSucceeded,
	BillingCheckoutStateExpired,
	BillingCheckoutStateFailed,
	BillingCheckoutStateReleased,
}

// OrganizationBillingCheckout is the local record of one subscription checkout
// request. It keeps the selected plan and seat count, and the Polar checkout
// that belongs to them. A row never grants capacity. Capacity changes only from
// a Polar subscription snapshot.
type OrganizationBillingCheckout struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	PlanKey        string
	PolarProductID string
	Seats          *int
	State          string
	// Polar fields stay empty until Polar returns a checkout. They are never
	// invented locally.
	PolarCheckoutID *string
	CheckoutURL     *string
	PolarStatus     *string
	ExpiresAt       *time.Time
	LastError       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ResolvedAt      *time.Time
}

// PolarCheckoutSnapshot is what Polar reported about a checkout.
type PolarCheckoutSnapshot struct {
	CheckoutID string
	URL        string
	Status     string
	ExpiresAt  *time.Time
}

func (OrganizationBillingCheckout) TableName() string {
	return "organization_billing_checkouts"
}

// Unresolved reports that the intent can still lead to a payable checkout.
func (c *OrganizationBillingCheckout) Unresolved() bool {
	return slices.Contains(unresolvedBillingCheckoutStates, c.State)
}

// MatchesSelection reports whether the intent was created for the same plan,
// Polar product, and seat count. The legacy Business key is an alias of business.
func (c *OrganizationBillingCheckout) MatchesSelection(planKey, productID string, seats *int) bool {
	if checkoutPlanKey(c.PlanKey) != checkoutPlanKey(planKey) || c.PolarProductID != productID {
		return false
	}
	if c.Seats == nil || seats == nil {
		return c.Seats == nil && seats == nil
	}
	return *c.Seats == *seats
}

// AdmitsMembers reports that the intent leaves room for the given number of
// members. A checkout without a seat count is not per seat and limits no
// membership.
func (c *OrganizationBillingCheckout) AdmitsMembers(members int) bool {
	return c.Seats == nil || members <= *c.Seats
}

// PolarCheckoutIDValue returns the Polar checkout ID, or an empty string.
func (c *OrganizationBillingCheckout) PolarCheckoutIDValue() string {
	if c.PolarCheckoutID == nil {
		return ""
	}
	return *c.PolarCheckoutID
}

// CheckoutURLValue returns the stored checkout URL, or an empty string.
func (c *OrganizationBillingCheckout) CheckoutURLValue() string {
	if c.CheckoutURL == nil {
		return ""
	}
	return *c.CheckoutURL
}

// MarkOpen records the Polar checkout of an intent that Polar still holds open.
// It also refreshes an open intent with a newer Polar status.
func (c *OrganizationBillingCheckout) MarkOpen(tx *gorm.DB, snapshot PolarCheckoutSnapshot) error {
	if strings.TrimSpace(snapshot.CheckoutID) == "" {
		return errors.New("a polar checkout id is required to open a billing checkout intent")
	}
	return c.transition(tx, BillingCheckoutStateOpen, polarSnapshotColumns(snapshot))
}

// MarkAmbiguous records that the Polar create call may have succeeded. Only a
// creating intent can become ambiguous.
func (c *OrganizationBillingCheckout) MarkAmbiguous(tx *gorm.DB, cause string) error {
	return c.transitionFrom(tx, []string{BillingCheckoutStateCreating}, BillingCheckoutStateAmbiguous, map[string]any{
		"last_error": cause,
	})
}

// Resolve ends the intent in a resolved state. The snapshot, when present,
// records the last Polar checkout data.
func (c *OrganizationBillingCheckout) Resolve(tx *gorm.DB, state string, snapshot *PolarCheckoutSnapshot) error {
	if !slices.Contains(resolvedBillingCheckoutStates, state) {
		return fmt.Errorf("%q is not a resolved billing checkout state", state)
	}
	columns := map[string]any{}
	if snapshot != nil {
		columns = polarSnapshotColumns(*snapshot)
	}
	columns["resolved_at"] = time.Now()
	return c.transition(tx, state, columns)
}

// Release ends an intent that has no payable Polar checkout.
func (c *OrganizationBillingCheckout) Release(tx *gorm.DB, reason string) error {
	return c.transition(tx, BillingCheckoutStateReleased, map[string]any{
		"last_error":  reason,
		"resolved_at": time.Now(),
	})
}

// ClaimOrganizationBillingCheckout creates the creating intent of an
// organization, unless an unresolved intent exists. The organization lock
// serializes concurrent claims and invitation acceptance, so the selected
// seats cover the members at the claim, and acceptance keeps the members
// within them while the checkout can activate. It returns the intent and
// whether this call created it. The caller makes the Polar call after this
// function returns.
func ClaimOrganizationBillingCheckout(tx *gorm.DB, orgID uuid.UUID, planKey, productID string, seats *int) (*OrganizationBillingCheckout, bool, error) {
	var checkout *OrganizationBillingCheckout
	claimed := false
	err := tx.Transaction(func(inner *gorm.DB) error {
		if _, err := coremodels.LockOrganization(inner, orgID); err != nil {
			return err
		}
		current, err := FindOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if current.HasLivePolarSubscription(time.Now()) {
			return ErrBillingCheckoutLiveSubscription
		}
		existing, err := FindUnresolvedOrganizationBillingCheckout(inner, orgID)
		if err != nil {
			return err
		}
		if existing != nil {
			checkout = existing
			return nil
		}
		candidate := &OrganizationBillingCheckout{
			OrganizationID: orgID,
			PlanKey:        planKey,
			PolarProductID: productID,
			Seats:          seats,
			State:          BillingCheckoutStateCreating,
		}
		members, err := coremodels.CountActiveHumanUsersByOrganization(inner, orgID.String())
		if err != nil {
			return err
		}
		if !candidate.AdmitsMembers(max(int(members), 1)) {
			return ErrBillingCheckoutSeatsBelowMembers
		}
		if err := inner.Create(candidate).Error; err != nil {
			return err
		}
		checkout, claimed = candidate, true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return checkout, claimed, nil
}

// AdoptPreexistingOrganizationBillingCheckout records a payable Polar checkout
// that SuperPlane opened before intents existed. It adopts the checkout only
// for an organization without intent rows, and returns nil otherwise.
func AdoptPreexistingOrganizationBillingCheckout(tx *gorm.DB, orgID uuid.UUID, planKey, productID string, seats *int, snapshot PolarCheckoutSnapshot) (*OrganizationBillingCheckout, error) {
	if strings.TrimSpace(snapshot.CheckoutID) == "" {
		return nil, errors.New("a polar checkout id is required to adopt a checkout")
	}
	var adopted *OrganizationBillingCheckout
	err := tx.Transaction(func(inner *gorm.DB) error {
		if _, err := coremodels.LockOrganization(inner, orgID); err != nil {
			return err
		}
		exists, err := OrganizationHasBillingCheckouts(inner, orgID)
		if err != nil || exists {
			return err
		}
		adopted = &OrganizationBillingCheckout{
			OrganizationID:  orgID,
			PlanKey:         planKey,
			PolarProductID:  productID,
			Seats:           seats,
			State:           BillingCheckoutStateOpen,
			PolarCheckoutID: optionalString(snapshot.CheckoutID),
			CheckoutURL:     optionalString(snapshot.URL),
			PolarStatus:     optionalString(snapshot.Status),
			ExpiresAt:       snapshot.ExpiresAt,
		}
		return inner.Create(adopted).Error
	})
	if err != nil {
		return nil, err
	}
	return adopted, nil
}

// FindUnresolvedOrganizationBillingCheckout returns the creating, ambiguous, or
// open intent of an organization, or nil when none exists.
func FindUnresolvedOrganizationBillingCheckout(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingCheckout, error) {
	var checkout OrganizationBillingCheckout
	err := tx.Where("organization_id = ? AND state IN ?", orgID, unresolvedBillingCheckoutStates).First(&checkout).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &checkout, nil
}

// OrganizationHasPolarCustomer reports that the organization has a Polar
// customer. Only such an organization can hold a Polar checkout.
func OrganizationHasPolarCustomer(tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&OrganizationLLMSettings{}).
		Where("organization_id = ? AND polar_customer_id IS NOT NULL AND polar_customer_id <> ''", orgID).
		Limit(1).Count(&count).Error
	return count > 0, err
}

// OrganizationHasBillingCheckouts reports whether the organization has any
// intent row, resolved or not.
func OrganizationHasBillingCheckouts(tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&OrganizationBillingCheckout{}).Where("organization_id = ?", orgID).Limit(1).Count(&count).Error
	return count > 0, err
}

// ResolveOrganizationBillingCheckoutForSubscription marks the unresolved intent
// of the Polar checkout that created a subscription as succeeded. The
// subscription apply path calls it, under the organization lock, when a live
// subscription is applied. Only the Polar checkout identity matches: the
// product or the seat count can describe another checkout. An empty checkout
// ID resolves nothing.
func ResolveOrganizationBillingCheckoutForSubscription(tx *gorm.DB, orgID uuid.UUID, checkoutID string) error {
	checkoutID = strings.TrimSpace(checkoutID)
	if checkoutID == "" {
		return nil
	}
	now := time.Now()
	return tx.Model(&OrganizationBillingCheckout{}).
		Where("organization_id = ? AND polar_checkout_id = ? AND state IN ?", orgID, checkoutID, unresolvedBillingCheckoutStates).
		Updates(map[string]any{
			"state":       BillingCheckoutStateSucceeded,
			"resolved_at": now,
			"updated_at":  now,
		}).Error
}

// transition moves an unresolved intent to a new state.
func (c *OrganizationBillingCheckout) transition(tx *gorm.DB, state string, columns map[string]any) error {
	return c.transitionFrom(tx, unresolvedBillingCheckoutStates, state, columns)
}

// transitionFrom checks the expected prior state in the UPDATE itself, so a
// stale writer cannot overwrite a newer state.
func (c *OrganizationBillingCheckout) transitionFrom(tx *gorm.DB, from []string, state string, columns map[string]any) error {
	columns["state"] = state
	columns["updated_at"] = time.Now()
	result := tx.Model(&OrganizationBillingCheckout{}).
		Where("id = ? AND state IN ?", c.ID, from).
		Updates(columns)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", ErrBillingCheckoutStateConflict, c.ID)
	}
	return tx.First(c, "id = ?", c.ID).Error
}

// polarSnapshotColumns keeps empty Polar fields out of an update, so a partial
// snapshot never erases data that an earlier call stored.
func polarSnapshotColumns(snapshot PolarCheckoutSnapshot) map[string]any {
	columns := map[string]any{}
	if id := strings.TrimSpace(snapshot.CheckoutID); id != "" {
		columns["polar_checkout_id"] = id
	}
	if url := strings.TrimSpace(snapshot.URL); url != "" {
		columns["checkout_url"] = url
	}
	if status := strings.TrimSpace(snapshot.Status); status != "" {
		columns["polar_status"] = status
	}
	if snapshot.ExpiresAt != nil {
		columns["expires_at"] = *snapshot.ExpiresAt
	}
	return columns
}

func checkoutPlanKey(key string) string {
	if key == "business_legacy" {
		return BillingPlanBusiness
	}
	return key
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
