package models

import (
	"time"

	"gorm.io/gorm"
)

// RecordPublication updates only delivery fields. A concurrent agent completion
// retains its immutable Result and decision state.
func (d *FactoryHumanReviewDecision) RecordPublication(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return tx.Model(d).Where("state <> ?", FactoryHumanReviewStateSuperseded).Updates(map[string]any{
		"check_run_id":        d.CheckRunID,
		"check_url":           d.CheckURL,
		"publication_pending": d.PublicationPending,
		"publication_error":   d.PublicationError,
		"retry_at":            d.RetryAt,
		"blocking_reason":     d.BlockingReason,
		"updated_at":          d.UpdatedAt,
	}).Error
}
