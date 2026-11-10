package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListRecentCanvasRootEvents includes unfinished runs so callers can select the
// latest event for a subject before deciding whether that subject is eligible.
func ListRecentCanvasRootEvents(db *gorm.DB, canvasID uuid.UUID, payloadType string, since time.Time, limit int) ([]CanvasEvent, error) {
	var events []CanvasEvent
	if limit <= 0 {
		return events, nil
	}
	err := db.Where("workflow_id = ? AND execution_id IS NULL", canvasID).
		Where("data ->> 'type' = ? AND created_at >= ?", payloadType, since).
		Order("created_at DESC, id DESC").Limit(limit).Find(&events).Error
	return events, err
}
