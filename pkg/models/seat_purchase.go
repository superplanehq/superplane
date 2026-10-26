package models

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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
	SeatPurchaseErrorRenewalBoundary       = "renewal_boundary"
)

// seatPurchaseOperationKeyIndex is the unique index on the caller operation
// key of an organization.
const seatPurchaseOperationKeyIndex = "uq_organization_seat_purchases_operation_key"

// SeatPurchase records one attempt to buy seats on a Polar subscription.
// Capacity never changes from this row. It changes only from a Polar snapshot.
// At most one purchase per organization is in flight (pending or unknown).
//
// A purchase that a caller started has an operation key, the fingerprint of
// the request, and the billing period it was made for. Purchases recorded
// before operation keys existed have none of them.
type SeatPurchase struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrganizationID      uuid.UUID
	UserID              *uuid.UUID
	PolarSubscriptionID string
	FromSeats           int
	ToSeats             int
	Status              string
	ErrorCode           string
	OperationKey        *string
	RequestFingerprint  *string
	PeriodStart         *time.Time
	PeriodEnd           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SeatPurchaseOperation is the caller request that starts a purchase and the
// billing period that the purchase is made for.
type SeatPurchaseOperation struct {
	Key         string
	Fingerprint string
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// NewSeatPurchase builds a pending purchase. A nil operation builds a
// purchase without an operation key or a period, like the rows recorded
// before operation keys existed.
func NewSeatPurchase(orgID uuid.UUID, userID *uuid.UUID, subscriptionID string, fromSeats, toSeats int, operation *SeatPurchaseOperation) *SeatPurchase {
	purchase := &SeatPurchase{
		OrganizationID:      orgID,
		UserID:              userID,
		PolarSubscriptionID: subscriptionID,
		FromSeats:           fromSeats,
		ToSeats:             toSeats,
		Status:              SeatPurchaseStatusPending,
	}
	if operation != nil {
		key, fingerprint := operation.Key, operation.Fingerprint
		start, end := operation.PeriodStart.UTC(), operation.PeriodEnd.UTC()
		purchase.OperationKey = &key
		purchase.RequestFingerprint = &fingerprint
		purchase.PeriodStart = &start
		purchase.PeriodEnd = &end
	}
	return purchase
}

// SeatPurchaseFingerprint identifies the charge that a purchase request asks
// for. Two requests with the same fingerprint buy the same seats from the
// same seat count. The user is not part of the charge, so it is not included.
func SeatPurchaseFingerprint(additionalSeats, expectedPurchasedSeats int) string {
	canonical := fmt.Sprintf("seat_purchase/v1;additional_seats=%d;expected_purchased_seats=%d", additionalSeats, expectedPurchasedSeats)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func (SeatPurchase) TableName() string {
	return "organization_seat_purchases"
}

// InFlight reports that the outcome is not settled yet.
func (p *SeatPurchase) InFlight() bool {
	return p.Status == SeatPurchaseStatusPending || p.Status == SeatPurchaseStatusUnknown
}

// MatchesRequest reports that the purchase was started by a request with
// this fingerprint.
func (p *SeatPurchase) MatchesRequest(fingerprint string) bool {
	return p.RequestFingerprint != nil && *p.RequestFingerprint == fingerprint
}

// StartedInPeriod reports that a Polar snapshot with this period start
// describes the billing period in which the purchase started. Only such a
// snapshot is evidence for the purchase: after a renewal, Polar can report
// the original seat count again, and that count proves nothing about the
// purchase.
//
// A purchase without a stored period predates operation keys. Its snapshot
// is in the purchase period only when that period started before the
// purchase was recorded.
func (p *SeatPurchase) StartedInPeriod(periodStart *time.Time) bool {
	if periodStart == nil {
		return false
	}
	if p.PeriodStart != nil {
		return p.PeriodStart.Equal(*periodStart)
	}
	return !periodStart.After(p.CreatedAt)
}

// SettlementFor returns the final status that a Polar snapshot proves for an
// in-flight purchase. It reports false when the snapshot proves nothing.
//
//   - A snapshot from another billing period proves nothing.
//   - Seats at or above the target prove the purchase succeeded.
//   - The original seats with a modification time after the purchase prove
//     that Polar did not apply an unknown purchase. A pending purchase can
//     still be on its way to Polar, so only its own result can fail it.
func (p *SeatPurchase) SettlementFor(periodStart *time.Time, seats *int, modifiedAt *time.Time) (string, bool) {
	if !p.InFlight() || seats == nil || !p.StartedInPeriod(periodStart) {
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

// BlocksRenewalMaintenance reports that the purchase can still race a
// next-period seat update on a subscription that the caller verified in
// Polar. The caller passes the subscription and the period start of a Polar
// snapshot that it read in this run, the current time, and how long a
// purchase request can take to reach Polar.
//
// An in-flight purchase stops blocking only when all of these are true:
//
//   - The purchase request can no longer reach Polar: the submission window
//     after the purchase passed.
//   - A verified Polar snapshot with a period start exists.
//   - The snapshot is on another subscription, or on a later period of the
//     purchase subscription. The period of a keyed purchase ends at or
//     before the verified period start. A purchase without a stored period
//     closed its submission window before the verified period started.
//
// Elapsed time alone never releases the guard. The purchase keeps its status:
// the release proves nothing about the charge.
func (p *SeatPurchase) BlocksRenewalMaintenance(subscriptionID string, periodStart *time.Time, now time.Time, submissionWindow time.Duration) bool {
	if !p.InFlight() {
		return false
	}
	submissionClosedAt := p.CreatedAt.Add(submissionWindow)
	if !now.After(submissionClosedAt) || periodStart == nil || subscriptionID == "" {
		return true
	}
	if subscriptionID != p.PolarSubscriptionID {
		return false
	}
	return !p.periodEndedBy(*periodStart, submissionClosedAt)
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

// IsSeatPurchaseOperationKeyConflict reports that a purchase with the same
// operation key already exists for the organization.
func IsSeatPurchaseOperationKeyConflict(err error) bool {
	var conflict *pgconn.PgError
	return errors.As(err, &conflict) && conflict.Code == pgUniqueViolation && conflict.ConstraintName == seatPurchaseOperationKeyIndex
}

// FindSeatPurchaseByOperationKey returns the purchase that an organization
// started with an operation key, or nil when none exists.
func FindSeatPurchaseByOperationKey(tx *gorm.DB, orgID uuid.UUID, key string) (*SeatPurchase, error) {
	var purchase SeatPurchase
	err := tx.Where("organization_id = ? AND operation_key = ?", orgID, key).First(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchase, nil
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

// ListUnresolvedSeatPurchasesCreatedBefore returns the pending and unknown
// purchases of all organizations that are not deleted and that started before
// the cutoff, the oldest first. An organization has at most one such purchase.
func ListUnresolvedSeatPurchasesCreatedBefore(tx *gorm.DB, cutoff time.Time) ([]SeatPurchase, error) {
	var purchases []SeatPurchase
	err := tx.
		Where("status IN ?", []string{SeatPurchaseStatusPending, SeatPurchaseStatusUnknown}).
		Where("created_at < ?", cutoff).
		Where("organization_id IN (SELECT id FROM organizations WHERE deleted_at IS NULL)").
		Order("created_at, id").
		Find(&purchases).Error
	if err != nil {
		return nil, err
	}
	return purchases, nil
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
// Polar snapshot proves its outcome. Only a snapshot from the period in which
// the purchase started can prove it. The caller holds the organization lock.
func SettleSeatPurchase(tx *gorm.DB, orgID uuid.UUID, subscriptionID string, periodStart *time.Time, seats *int, modifiedAt *time.Time) error {
	purchase, err := FindInFlightSeatPurchase(tx, orgID)
	if err != nil || purchase == nil || purchase.PolarSubscriptionID != subscriptionID {
		return err
	}
	status, settled := purchase.SettlementFor(periodStart, seats, modifiedAt)
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
// Polar to set on a subscription in the billing period that starts at
// periodStart: the target of the latest pending, unknown, or succeeded
// purchase of that period. It returns nil when no purchase expects a count.
// A snapshot with this seat count is not an outside change.
//
// A purchase from an earlier period expects nothing: a renewal can reduce
// the seats back to the count before that purchase.
func FindExpectedSeatCount(tx *gorm.DB, orgID uuid.UUID, subscriptionID string, periodStart time.Time) (*int, error) {
	var purchase SeatPurchase
	err := tx.Where(
		"organization_id = ? AND polar_subscription_id = ? AND status IN ?",
		orgID, subscriptionID,
		[]string{SeatPurchaseStatusPending, SeatPurchaseStatusUnknown, SeatPurchaseStatusSucceeded},
	).Where(
		"period_start = ? OR (period_start IS NULL AND created_at >= ?)", periodStart, periodStart,
	).Order("created_at DESC, id DESC").First(&purchase).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchase.ToSeats, nil
}

// periodEndedBy reports that the purchase period ended at or before a period
// that starts at periodStart. A purchase without a stored period has only its
// creation time: its period ended by then when the submission window closed
// before that period started.
func (p *SeatPurchase) periodEndedBy(periodStart, submissionClosedAt time.Time) bool {
	if p.PeriodEnd != nil {
		return !p.PeriodEnd.After(periodStart)
	}
	return periodStart.After(submissionClosedAt)
}
