package models

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TryReserveMerge holds only a task advisory lock until the transaction ends.
// Use a dedicated transaction with no row or organization locks while the
// provider call runs. Admission uses nonblocking shared locks on the same key,
// so other tasks continue and this reservation is released on rollback or
// connection loss. Local merge checks and completion use separate transactions.
func (o *FactoryWorkOrder) TryReserveMerge(tx *gorm.DB) (bool, error) {
	var reserved bool
	err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", workOrderMergeKey(o.ID)).Scan(&reserved).Error
	return reserved, err
}

func checkTaskMergeAdmission(tx *gorm.DB, workOrderID uuid.UUID) error {
	var admitted bool
	err := tx.Raw("SELECT pg_try_advisory_xact_lock_shared(?)", workOrderMergeKey(workOrderID)).Scan(&admitted).Error
	if err != nil {
		return err
	}
	if !admitted {
		return ErrOrganizationTaskCapacity
	}
	return nil
}

func workOrderMergeKey(workOrderID uuid.UUID) int64 {
	sum := sha256.Sum256(append([]byte("factory-work-order-merge:"), workOrderID[:]...))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
