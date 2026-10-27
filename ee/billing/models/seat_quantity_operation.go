package models

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	SeatQuantityStatusSubmitting     = "submitting"
	SeatQuantityStatusUnknown        = "unknown"
	SeatQuantityStatusScheduled      = "scheduled"
	SeatQuantityStatusRejected       = "rejected"
	SeatQuantityStatusApplied        = "applied"
	SeatQuantityStatusCanceled       = "canceled"
	SeatQuantityStatusSuperseded     = "superseded"
	SeatQuantityStatusRepairRequired = "repair_required"

	SeatQuantityOriginAdmin         = "admin"
	SeatQuantityOriginPurchase      = "purchase"
	SeatQuantityOriginProtective    = "protective"
	SeatQuantityOriginLegacyCleanup = "legacy_cleanup"

	SeatQuantityActionSchedule         = "schedule"
	SeatQuantityActionChange           = "change"
	SeatQuantityActionCancel           = "cancel"
	SeatQuantityActionSupersede        = "supersede"
	SeatQuantityActionProtectiveCancel = "protective_cancel"
	SeatQuantityActionLegacyCleanup    = "legacy_cleanup"
)

const (
	seatQuantityOperationKeyIndex = "uq_seat_quantity_operations_operation_key"
	seatQuantityInFlightIndex     = "uq_seat_quantity_operations_inflight"
)

// SeatQuantityOperation is one explicit next-renewal seat change.
// Renewal does not create these rows. A verified schedule stays admission-active
// and does not hold the mutation lock.
type SeatQuantityOperation struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrganizationID     uuid.UUID
	ActorUserID        *uuid.UUID
	ActorAPIKeyID      *uuid.UUID
	Origin             string
	Action             string
	SubscriptionID     string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	SourcePaidQuantity int
	TargetQuantity     int
	EffectiveAt        time.Time
	OperationKey       string
	RequestFingerprint string
	Revision           int
	PredecessorID      *uuid.UUID
	SupersederID       *uuid.UUID
	SeatPurchaseID     *uuid.UUID
	ProviderEvidence   datatypes.JSON
	Status             string
	Cause              string
	SeatChangeVersion  int64
	DispatchedAt       *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (SeatQuantityOperation) TableName() string {
	return "seat_quantity_operations"
}

// Open reports that the operation can still change admission or block a mutation.
func (op *SeatQuantityOperation) Open() bool {
	switch op.Status {
	case SeatQuantityStatusSubmitting, SeatQuantityStatusUnknown, SeatQuantityStatusScheduled, SeatQuantityStatusRepairRequired:
		return true
	default:
		return false
	}
}

// InFlight reports that the provider outcome is not known, so another mutation
// must wait. Age does not clear this state.
func (op *SeatQuantityOperation) InFlight() bool {
	return op.Status == SeatQuantityStatusSubmitting || op.Status == SeatQuantityStatusUnknown
}

// SeatQuantityFingerprint identifies one removal request. The seat-change
// version is the version the caller previewed, before this operation advanced it.
func SeatQuantityFingerprint(action string, target int, subscriptionID string, periodEnd time.Time, seatChangeVersion int64, sourcePaid int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf(
		"seat_quantity/v1;action=%s;target=%d;subscription_id=%q;period_end=%s;seat_change_version=%d;source_paid=%d",
		action, target, subscriptionID, periodEnd.UTC().Format(time.RFC3339Nano), seatChangeVersion, sourcePaid,
	)))
	return hex.EncodeToString(sum[:])
}

func (op *SeatQuantityOperation) Create(tx *gorm.DB) error {
	if op.ID == uuid.Nil {
		op.ID = uuid.New()
	}
	if op.Revision < 1 {
		op.Revision = 1
	}
	return tx.Create(op).Error
}

// Transition moves an open operation to status and stores the provider evidence.
// A terminal operation is left unchanged, so a duplicate or reordered event
// cannot revive it. The revision advances on every accepted change.
func (op *SeatQuantityOperation) Transition(tx *gorm.DB, status, cause string, evidence datatypes.JSON) error {
	now := time.Now()
	revision := op.Revision + 1
	updates := map[string]any{
		"status":     status,
		"cause":      cause,
		"revision":   revision,
		"updated_at": now,
	}
	if len(evidence) > 0 {
		updates["provider_evidence"] = evidence
	}
	result := tx.Model(&SeatQuantityOperation{}).
		Where("id = ? AND status IN ?", op.ID, []string{
			SeatQuantityStatusSubmitting, SeatQuantityStatusUnknown,
			SeatQuantityStatusScheduled, SeatQuantityStatusRepairRequired,
		}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return tx.First(op, "id = ?", op.ID).Error
	}
	op.Status = status
	op.Cause = cause
	op.Revision = revision
	op.UpdatedAt = now
	if len(evidence) > 0 {
		op.ProviderEvidence = evidence
	}
	return nil
}

// MarkDispatched records the time of the provider call. It does not change
// the revision, so a preview bound to the submitting operation stays valid
// until the outcome does.
func (op *SeatQuantityOperation) MarkDispatched(tx *gorm.DB, at time.Time) error {
	dispatched := at.UTC()
	result := tx.Model(&SeatQuantityOperation{}).Where("id = ?", op.ID).
		Updates(map[string]any{"dispatched_at": dispatched, "updated_at": time.Now()})
	if result.Error != nil {
		return result.Error
	}
	op.DispatchedAt = &dispatched
	return nil
}

// SetSuperseder records the operation that replaced this one.
func (op *SeatQuantityOperation) SetSuperseder(tx *gorm.DB, superseder uuid.UUID) error {
	if err := tx.Model(&SeatQuantityOperation{}).Where("id = ?", op.ID).
		Update("superseder_id", superseder).Error; err != nil {
		return err
	}
	op.SupersederID = &superseder
	return nil
}

func IsSeatQuantityOperationKeyConflict(err error) bool {
	var conflict *pgconn.PgError
	return errors.As(err, &conflict) && conflict.Code == pgUniqueViolation && conflict.ConstraintName == seatQuantityOperationKeyIndex
}

func IsSeatQuantityInFlightConflict(err error) bool {
	var conflict *pgconn.PgError
	return errors.As(err, &conflict) && conflict.Code == pgUniqueViolation && conflict.ConstraintName == seatQuantityInFlightIndex
}

func FindSeatQuantityOperation(tx *gorm.DB, orgID, id uuid.UUID) (*SeatQuantityOperation, error) {
	var op SeatQuantityOperation
	err := tx.Where("organization_id = ? AND id = ?", orgID, id).First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func FindSeatQuantityOperationByKey(tx *gorm.DB, orgID uuid.UUID, key string) (*SeatQuantityOperation, error) {
	var op SeatQuantityOperation
	err := tx.Where("organization_id = ? AND operation_key = ?", orgID, key).First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// FindInFlightSeatQuantityOperation returns the submitting or unknown
// operation, or nil. An organization has at most one.
func FindInFlightSeatQuantityOperation(tx *gorm.DB, orgID uuid.UUID) (*SeatQuantityOperation, error) {
	var op SeatQuantityOperation
	err := tx.Where("organization_id = ? AND status IN ?", orgID, []string{SeatQuantityStatusSubmitting, SeatQuantityStatusUnknown}).
		First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// FindScheduledSeatQuantityOperation returns the verified schedule, or nil.
func FindScheduledSeatQuantityOperation(tx *gorm.DB, orgID uuid.UUID) (*SeatQuantityOperation, error) {
	var op SeatQuantityOperation
	err := tx.Where("organization_id = ? AND status = ?", orgID, SeatQuantityStatusScheduled).First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// ListOpenSeatQuantityOperations returns the operations that reconciliation
// must still examine, oldest first.
func ListOpenSeatQuantityOperations(tx *gorm.DB, orgID uuid.UUID) ([]SeatQuantityOperation, error) {
	var ops []SeatQuantityOperation
	err := tx.Where("organization_id = ? AND status IN ?", orgID, []string{
		SeatQuantityStatusSubmitting, SeatQuantityStatusUnknown,
		SeatQuantityStatusScheduled, SeatQuantityStatusRepairRequired,
	}).Order("created_at, id").Find(&ops).Error
	return ops, err
}

// ListSeatQuantityReconcileOrganizations returns organizations with an open
// removal, including organizations that the ordinary subscription list skips.
func ListSeatQuantityReconcileOrganizations(tx *gorm.DB) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Raw(`
		SELECT DISTINCT organization_id
		FROM seat_quantity_operations
		WHERE status IN ?
		  AND organization_id IN (SELECT id FROM organizations WHERE deleted_at IS NULL)
		ORDER BY organization_id`,
		[]string{
			SeatQuantityStatusSubmitting, SeatQuantityStatusUnknown,
			SeatQuantityStatusScheduled, SeatQuantityStatusRepairRequired,
		},
	).Scan(&ids).Error
	return ids, err
}

// SaveSeatChange writes the removal columns of a plan. A subscription snapshot
// does not write these columns.
func SaveSeatChange(tx *gorm.DB, plan *OrganizationBillingPlan) error {
	return tx.Model(&OrganizationBillingPlan{}).Where("organization_id = ?", plan.OrganizationID).Updates(map[string]any{
		"seat_change_version":        plan.SeatChangeVersion,
		"seat_admission_limit":       nullableInt(plan.SeatAdmissionLimit),
		"seat_proved_target":         nullableInt(plan.SeatProvedTarget),
		"seat_quantity_operation_id": nullableUUID(plan.SeatQuantityOperationID),
		"seat_bound_period_end":      nullableTime(plan.SeatBoundPeriodEnd),
		"updated_at":                 time.Now(),
	}).Error
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

// ClearSeatRestriction removes the proved removal and the conservative limit.
// The seat-change version is left as it is.
func ClearSeatRestriction(plan *OrganizationBillingPlan) {
	plan.SeatAdmissionLimit = nil
	plan.SeatProvedTarget = nil
	plan.SeatQuantityOperationID = nil
	plan.SeatBoundPeriodEnd = nil
}
