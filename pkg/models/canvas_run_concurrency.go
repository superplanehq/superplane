package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HasActiveCanvasRun serializes workflow-wide run checks until tx completes.
// A caller that starts a run after a false result must use the same explicit transaction.
func HasActiveCanvasRun(tx *gorm.DB, workflowID uuid.UUID) (bool, error) {
	if err := lockCanvasRunGate(tx, workflowID); err != nil {
		return false, err
	}

	var run CanvasRun
	result := tx.
		Select("id").
		Where("workflow_id = ?", workflowID).
		Where("state IN ?", []string{
			CanvasRunStatePending,
			CanvasRunStateStarted,
			CanvasRunStateCancelling,
		}).
		Limit(1).
		Find(&run)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func lockCanvasRunGate(tx *gorm.DB, workflowID uuid.UUID) error {
	var canvas Canvas
	return tx.
		Select("id").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", workflowID).
		Take(&canvas).
		Error
}
