package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SeatPurchaseStatusPending        = "pending"
	SeatPurchaseStatusSucceeded      = "succeeded"
	SeatPurchaseStatusFailed         = "failed"
	SeatPurchaseStatusActionRequired = "action_required"
	SeatPurchaseStatusUnknown        = "unknown"
)

// Error codes stored with a purchase that did not add seats.
const (
	SeatPurchaseErrorPaymentFailed         = "payment_failed"
	SeatPurchaseErrorPaymentActionRequired = "payment_action_required"
	SeatPurchaseErrorSubscriptionLocked    = "subscription_locked"
	SeatPurchaseErrorSubscriptionInactive  = "subscription_inactive"
	SeatPurchaseErrorRejected              = "rejected"
	SeatPurchaseErrorNotApplied            = "not_applied"
)

// SeatPurchase records one attempt to buy seats on a Polar subscription.
// Capacity never changes from this row. It changes only from a Polar snapshot.
// At most one purchase per organization is in flight (pending or unknown).
type SeatPurchase struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrganizationID      uuid.UUID
	UserID              *uuid.UUID
	PolarSubscriptionID string
	FromSeats           int
	ToSeats             int
	Status              string
	ErrorCode           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (SeatPurchase) TableName() string {
	return "organization_seat_purchases"
}

func NewSeatPurchase(orgID uuid.UUID, userID *uuid.UUID, subscriptionID string, fromSeats, toSeats int) *SeatPurchase {
	return &SeatPurchase{
		OrganizationID:      orgID,
		UserID:              userID,
		PolarSubscriptionID: subscriptionID,
		FromSeats:           fromSeats,
		ToSeats:             toSeats,
		Status:              SeatPurchaseStatusPending,
	}
}

// InFlight reports that the outcome is not settled yet.
func (p *SeatPurchase) InFlight() bool {
	return p.Status == SeatPurchaseStatusPending || p.Status == SeatPurchaseStatusUnknown
}

// SettlementFor returns the final status that a Polar snapshot proves for an
// in-flight purchase. It reports false when the snapshot proves nothing.
//
//   - Seats at or above the target prove the purchase succeeded.
//   - The original seats with a modification time after the purchase prove
//     that Polar did not apply an unknown purchase. A pending purchase can
//     still be on its way to Polar, so only its own result can fail it.
func (p *SeatPurchase) SettlementFor(seats *int, modifiedAt *time.Time) (string, bool) {
	if !p.InFlight() || seats == nil {
		return "", false
	}
	if *seats >= p.ToSeats {
		return SeatPurchaseStatusSucceeded, true
	}
	if p.Status == SeatPurchaseStatusUnknown && *seats == p.FromSeats && modifiedAt != nil && modifiedAt.After(p.CreatedAt) {
		return SeatPurchaseStatusFailed, true
	}
	return "", false
}

func (p *SeatPurchase) Create(tx *gorm.DB) error {
	return tx.Create(p).Error
}

// UpdateStatus moves an in-flight purchase to a new status.
// It leaves a settled purchase unchanged, so a late result cannot overwrite
// the outcome that a Polar snapshot already proved.
func (p *SeatPurchase) UpdateStatus(tx *gorm.DB, status, errorCode string) error {
	now := time.Now()
	result := tx.Model(&SeatPurchase{}).
		Where("id = ? AND status IN ?", p.ID, []string{SeatPurchaseStatusPending, SeatPurchaseStatusUnknown}).
		Updates(map[string]any{"status": status, "error_code": errorCode, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return tx.First(p, "id = ?", p.ID).Error
	}
	p.Status = status
	p.ErrorCode = errorCode
	p.UpdatedAt = now
	return nil
}

// FindInFlightSeatPurchase returns the pending or unknown purchase of an
// organization, or nil when none exists.
func FindInFlightSeatPurchase(tx *gorm.DB, orgID uuid.UUID) (*SeatPurchase, error) {
	var purchase SeatPurchase
	err := tx.Where("organization_id = ? AND status IN ?", orgID, []string{SeatPurchaseStatusPending, SeatPurchaseStatusUnknown}).
		First(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchase, nil
}

// FindLatestSeatPurchase returns the newest purchase of an organization, or nil.
func FindLatestSeatPurchase(tx *gorm.DB, orgID uuid.UUID) (*SeatPurchase, error) {
	var purchase SeatPurchase
	err := tx.Where("organization_id = ?", orgID).Order("created_at DESC, id DESC").First(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchase, nil
}

// SettleSeatPurchase settles the in-flight purchase of a subscription when the
// Polar snapshot proves its outcome. The caller holds the organization lock.
func SettleSeatPurchase(tx *gorm.DB, orgID uuid.UUID, subscriptionID string, seats *int, modifiedAt *time.Time) error {
	purchase, err := FindInFlightSeatPurchase(tx, orgID)
	if err != nil || purchase == nil || purchase.PolarSubscriptionID != subscriptionID {
		return err
	}
	status, settled := purchase.SettlementFor(seats, modifiedAt)
	if !settled {
		return nil
	}
	errorCode := ""
	if status == SeatPurchaseStatusFailed {
		errorCode = SeatPurchaseErrorNotApplied
	}
	return purchase.UpdateStatus(tx, status, errorCode)
}

// FindExpectedSeatCount returns the seat count that SuperPlane itself asked
// Polar to set on a subscription: the target of the latest pending, unknown,
// or succeeded purchase. It returns nil when no purchase expects a count.
// A snapshot with this seat count is not an outside change.
func FindExpectedSeatCount(tx *gorm.DB, orgID uuid.UUID, subscriptionID string) (*int, error) {
	var purchase SeatPurchase
	err := tx.Where(
		"organization_id = ? AND polar_subscription_id = ? AND status IN ?",
		orgID, subscriptionID,
		[]string{SeatPurchaseStatusPending, SeatPurchaseStatusUnknown, SeatPurchaseStatusSucceeded},
	).Order("created_at DESC, id DESC").First(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchase.ToSeats, nil
}
