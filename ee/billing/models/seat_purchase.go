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
	// SeatPurchaseErrorNotApplied comes only from an operator repair with
	// Polar evidence. No subscription snapshot can prove it.
	SeatPurchaseErrorNotApplied      = "not_applied"
	SeatPurchaseErrorRenewalBoundary = "renewal_boundary"
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

// SeatPurchaseEvidence is a Polar subscription state that can describe the
// outcome of a seat purchase.
type SeatPurchaseEvidence struct {
	SubscriptionID string
	PeriodStart    *time.Time
	Seats          *int
	ModifiedAt     *time.Time
	// Response marks the state that Polar returned for the update request of
	// the purchase itself. Webhooks, syncs, and reads are not responses.
	Response bool
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

// SeatPurchaseFingerprints are the fingerprints of one purchase request.
// Counts is the v1 fingerprint of the seat counts. Bound is the v2
// fingerprint that also binds the confirmed subscription and billing period.
// Bound is empty when the request has no complete binding.
type SeatPurchaseFingerprints struct {
	Counts string
	Bound  string
}

// SeatPurchaseFingerprint is the v1 fingerprint: it identifies the charge
// from the seat counts only. Two requests with the same fingerprint buy the
// same seats from the same seat count. The user is not part of the charge,
// so it is not included.
//
// New purchases store SeatPurchaseBoundFingerprint. Rows recorded before
// the binding existed keep this fingerprint and stay replayable.
func SeatPurchaseFingerprint(additionalSeats, expectedPurchasedSeats int) string {
	return seatPurchaseHash(fmt.Sprintf("seat_purchase/v1;additional_seats=%d;expected_purchased_seats=%d", additionalSeats, expectedPurchasedSeats))
}

// SeatPurchaseBoundFingerprint is the v2 fingerprint: the seat counts plus
// the subscription and the billing period that the caller confirmed. The
// period times are normalized to UTC, so equal instants in other time zones
// give the same fingerprint.
func SeatPurchaseBoundFingerprint(additionalSeats, expectedPurchasedSeats int, subscriptionID string, periodStart, periodEnd time.Time) string {
	return seatPurchaseHash(fmt.Sprintf(
		"seat_purchase/v2;additional_seats=%d;expected_purchased_seats=%d;subscription_id=%q;period_start=%s;period_end=%s",
		additionalSeats, expectedPurchasedSeats, subscriptionID,
		periodStart.UTC().Format(time.RFC3339Nano), periodEnd.UTC().Format(time.RFC3339Nano),
	))
}

func (SeatPurchase) TableName() string {
	return "organization_seat_purchases"
}

// InFlight reports that the outcome is not settled yet.
func (p *SeatPurchase) InFlight() bool {
	return p.Status == SeatPurchaseStatusPending || p.Status == SeatPurchaseStatusUnknown
}

// MatchesRequest reports that the purchase was started by this request.
//
// A purchase with a v2 fingerprint matches only a request with the same
// counts and the same binding. A legacy purchase with a v1 fingerprint
// matches a request with the same counts, with or without a binding. A v2
// fingerprint never equals a v1 fingerprint, so a request without a binding
// never matches a v2 purchase.
func (p *SeatPurchase) MatchesRequest(request SeatPurchaseFingerprints) bool {
	if p.RequestFingerprint == nil {
		return false
	}
	stored := *p.RequestFingerprint
	return stored == request.Counts || (request.Bound != "" && stored == request.Bound)
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

// SucceededBy reports that the evidence proves that this in-flight purchase
// succeeded. Only a successful result settles a purchase from Polar state.
// The seat count, the modification time, or the time since the purchase
// alone proves nothing: an outside change or a delayed event about another
// update can report any of them. The evidence must describe this purchase:
//
//   - It is from the subscription of the purchase.
//   - It is from the billing period in which the purchase started.
//   - It reports exactly the target seats. A purchase sends an absolute
//     seat count, so a different count describes another change.
//   - It is the response to the update request of this purchase, or Polar
//     modified the subscription after the purchase was recorded. A state
//     from before the purchase cannot describe it.
//
// No snapshot proves that a purchase failed. A snapshot with fewer seats can
// come from an update that Polar processed before the purchase, delivered
// late. Only the rejection in the response of the purchase records a failure.
func (p *SeatPurchase) SucceededBy(evidence SeatPurchaseEvidence) bool {
	if !p.InFlight() || !p.describedBy(evidence) {
		return false
	}
	if evidence.Response {
		return true
	}
	return evidence.ModifiedAt != nil && evidence.ModifiedAt.After(p.CreatedAt)
}

func (p *SeatPurchase) Create(tx *gorm.DB) error {
	return tx.Create(p).Error
}

// UpdateStatus moves an in-flight purchase to a new status.
// It leaves a settled purchase unchanged, so a late result, a duplicate
// event, or a reordered event cannot overwrite a recorded outcome.
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

// SettleSeatPurchase records the success of the in-flight purchase of an
// organization when the evidence proves it. Other evidence leaves the
// purchase unchanged, so an unknown outcome stays unknown. The caller holds
// the organization lock.
func SettleSeatPurchase(tx *gorm.DB, orgID uuid.UUID, evidence SeatPurchaseEvidence) error {
	purchase, err := FindInFlightSeatPurchase(tx, orgID)
	if err != nil || purchase == nil || !purchase.SucceededBy(evidence) {
		return err
	}
	return purchase.UpdateStatus(tx, SeatPurchaseStatusSucceeded, "")
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

// describedBy reports that the evidence is about the subscription, the
// billing period, and the seat count of the purchase.
func (p *SeatPurchase) describedBy(evidence SeatPurchaseEvidence) bool {
	return evidence.SubscriptionID == p.PolarSubscriptionID &&
		p.StartedInPeriod(evidence.PeriodStart) &&
		evidence.Seats != nil && *evidence.Seats == p.ToSeats
}

func seatPurchaseHash(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}
