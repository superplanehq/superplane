package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CanvasData is a versioned key-value entry scoped to a canvas.
// Each write creates a new row; history is ordered by created_at DESC
// (current = first row, previous = second, etc.).
type CanvasData struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CanvasID  uuid.UUID  `gorm:"type:uuid;not null"`
	Key       string     `gorm:"type:varchar(256);not null"`
	Value     string     `gorm:"type:text;not null"`
	CreatedAt *time.Time `gorm:"not null"`
}

func (c *CanvasData) TableName() string {
	return "canvas_data"
}

// SetCanvasDataInTransaction writes a new value for the given canvas and key.
// Each call creates a new history entry. Returns the created record.
func SetCanvasDataInTransaction(tx *gorm.DB, canvasID uuid.UUID, key, value string) (*CanvasData, error) {
	now := time.Now()
	rec := CanvasData{
		CanvasID:  canvasID,
		Key:      key,
		Value:    value,
		CreatedAt: &now,
	}
	err := tx.Clauses(clause.Returning{}).Create(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// SetCanvasData writes a new value for the given canvas and key.
func SetCanvasData(canvasID uuid.UUID, key, value string) (*CanvasData, error) {
	return SetCanvasDataInTransaction(database.Conn(), canvasID, key, value)
}

// GetCanvasDataInTransaction returns the value at the given version index for the canvas and key.
// versionOffset 0 = current (latest), 1 = previous, 2 = two steps back, etc.
// Returns nil, nil when no entry exists or offset is beyond history.
func GetCanvasDataInTransaction(tx *gorm.DB, canvasID uuid.UUID, key string, versionOffset int) (*CanvasData, error) {
	var rec CanvasData
	q := tx.Where("canvas_id = ?", canvasID).Where("key = ?", key).Order("created_at DESC")
	if versionOffset > 0 {
		q = q.Offset(versionOffset)
	}
	err := q.Limit(1).First(&rec).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// GetCanvasData returns the value at the given version index for the canvas and key.
func GetCanvasData(canvasID uuid.UUID, key string, versionOffset int) (*CanvasData, error) {
	return GetCanvasDataInTransaction(database.Conn(), canvasID, key, versionOffset)
}

// ListCanvasDataHistoryInTransaction returns up to limit history entries for the canvas and key,
// newest first.
func ListCanvasDataHistoryInTransaction(tx *gorm.DB, canvasID uuid.UUID, key string, limit int) ([]CanvasData, error) {
	if limit <= 0 {
		limit = 100
	}
	var recs []CanvasData
	err := tx.Where("canvas_id = ?", canvasID).
		Where("key = ?", key).
		Order("created_at DESC").
		Limit(limit).
		Find(&recs).Error
	if err != nil {
		return nil, err
	}
	return recs, nil
}

// ListCanvasDataHistory returns up to limit history entries for the canvas and key.
func ListCanvasDataHistory(canvasID uuid.UUID, key string, limit int) ([]CanvasData, error) {
	return ListCanvasDataHistoryInTransaction(database.Conn(), canvasID, key, limit)
}
