package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BuiltInAutomationReset is the latest installation reset of one built-in automation.
type BuiltInAutomationReset struct {
	TemplateID string `gorm:"primaryKey"`
	AccountID  *uuid.UUID
	ResetAt    time.Time
}

func (BuiltInAutomationReset) TableName() string {
	return "built_in_automation_resets"
}

// SaveBuiltInAutomationReset stores who reset one built-in automation and when.
// A later reset replaces the previous record.
func SaveBuiltInAutomationReset(tx *gorm.DB, templateID string, accountID uuid.UUID, resetAt time.Time) error {
	reset := BuiltInAutomationReset{
		TemplateID: templateID,
		AccountID:  &accountID,
		ResetAt:    resetAt,
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "template_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"account_id", "reset_at"}),
	}).Create(&reset).Error
}

// ListBuiltInAutomationResets returns the latest reset for each built-in automation.
func ListBuiltInAutomationResets(tx *gorm.DB) ([]BuiltInAutomationReset, error) {
	var resets []BuiltInAutomationReset
	err := tx.Find(&resets).Error
	if err != nil {
		return nil, err
	}
	return resets, nil
}
