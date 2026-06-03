package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RepositoryMaterializationState struct {
	ID              uuid.UUID
	CanvasID        uuid.UUID
	Branch          string
	HeadSHA         string
	MaterializedSHA string
	Status          string
	Error           string
	UpdatedAt       time.Time
}

func (s *RepositoryMaterializationState) TableName() string {
	return "repository_materialization_state"
}

func FindRepositoryMaterializationStateInTransaction(tx *gorm.DB, canvasID uuid.UUID, branch string) (*RepositoryMaterializationState, error) {
	var state RepositoryMaterializationState
	err := tx.
		Where("canvas_id = ?", canvasID).
		Where("branch = ?", branch).
		First(&state).
		Error
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func FindRepositoryMaterializationState(canvasID uuid.UUID, branch string) (*RepositoryMaterializationState, error) {
	return FindRepositoryMaterializationStateInTransaction(database.Conn(), canvasID, branch)
}

func ListRepositoryMaterializationStatesForCanvasInTransaction(tx *gorm.DB, canvasID uuid.UUID) ([]RepositoryMaterializationState, error) {
	var states []RepositoryMaterializationState
	err := tx.
		Where("canvas_id = ?", canvasID).
		Order("updated_at DESC").
		Find(&states).
		Error
	if err != nil {
		return nil, err
	}

	return states, nil
}

func ListRepositoryMaterializationStatesForCanvas(canvasID uuid.UUID) ([]RepositoryMaterializationState, error) {
	return ListRepositoryMaterializationStatesForCanvasInTransaction(database.Conn(), canvasID)
}

func UpsertRepositoryMaterializationStateInTransaction(tx *gorm.DB, state *RepositoryMaterializationState) error {
	if state == nil {
		return gorm.ErrInvalidData
	}

	now := time.Now()
	state.UpdatedAt = now
	if state.ID == uuid.Nil {
		state.ID = uuid.New()
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "canvas_id"}, {Name: "branch"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"head_sha",
			"materialized_sha",
			"status",
			"error",
			"updated_at",
		}),
	}).Create(state).Error
}

func UpsertRepositoryMaterializationState(state *RepositoryMaterializationState) error {
	return UpsertRepositoryMaterializationStateInTransaction(database.Conn(), state)
}

func DeleteRepositoryMaterializationStateInTransaction(tx *gorm.DB, canvasID uuid.UUID, branch string) error {
	return tx.
		Where("canvas_id = ?", canvasID).
		Where("branch = ?", branch).
		Delete(&RepositoryMaterializationState{}).
		Error
}

func DeleteRepositoryMaterializationState(canvasID uuid.UUID, branch string) error {
	return DeleteRepositoryMaterializationStateInTransaction(database.Conn(), canvasID, branch)
}
