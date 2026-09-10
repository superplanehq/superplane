package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFactoryWorkOrderDispatchKeyConflict = errors.New(
	"idempotency key was already used with different dispatch parameters",
)

// FactoryWorkOrderDispatchRequest records a committed dispatch request. The
// row is created in the dispatch transaction, so failed attempts do not
// consume their idempotency keys.
type FactoryWorkOrderDispatchRequest struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	FactoryID          uuid.UUID
	WorkOrderID        uuid.UUID
	IdempotencyKey     string
	RequestFingerprint string
	CreatedAt          time.Time
}

func (FactoryWorkOrderDispatchRequest) TableName() string {
	return "factory_work_order_dispatch_requests"
}

// ReserveFactoryWorkOrderDispatchRequest records request when its key is new.
// It returns false when a completed request already owns the key.
func ReserveFactoryWorkOrderDispatchRequest(tx *gorm.DB, request *FactoryWorkOrderDispatchRequest) (bool, error) {
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "work_order_id"}, {Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(request)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	var existing FactoryWorkOrderDispatchRequest
	err := tx.
		Where("work_order_id = ? AND idempotency_key = ?", request.WorkOrderID, request.IdempotencyKey).
		First(&existing).
		Error
	if err != nil {
		return false, err
	}
	if existing.RequestFingerprint != request.RequestFingerprint {
		return false, ErrFactoryWorkOrderDispatchKeyConflict
	}

	return false, nil
}
