package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowRunSession struct {
	ID              uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID      uuid.UUID
	ContinuationKey string
	RootEventID     uuid.UUID
	LastExecutionID *uuid.UUID
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func (WorkflowRunSession) TableName() string {
	return "workflow_run_sessions"
}

func FindWorkflowRunSessionByKeyInTransaction(tx *gorm.DB, workflowID uuid.UUID, continuationKey string) (*WorkflowRunSession, error) {
	var session WorkflowRunSession
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("continuation_key = ?", continuationKey).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func FindWorkflowRunSessionByKeyForUpdateInTransaction(tx *gorm.DB, workflowID uuid.UUID, continuationKey string) (*WorkflowRunSession, error) {
	var session WorkflowRunSession
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("workflow_id = ?", workflowID).
		Where("continuation_key = ?", continuationKey).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func FindOrCreateWorkflowRunSessionByKeyForUpdateInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	continuationKey string,
	rootEventID uuid.UUID,
) (*WorkflowRunSession, bool, error) {
	session, err := FindWorkflowRunSessionByKeyForUpdateInTransaction(tx, workflowID, continuationKey)
	if err == nil {
		return session, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	now := time.Now()
	newSession := WorkflowRunSession{
		WorkflowID:      workflowID,
		ContinuationKey: continuationKey,
		RootEventID:     rootEventID,
		CreatedAt:       &now,
		UpdatedAt:       &now,
	}

	result := tx.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "workflow_id"},
				{Name: "continuation_key"},
			},
			DoNothing: true,
		}).
		Create(&newSession)
	if result.Error != nil {
		return nil, false, result.Error
	}

	session, err = FindWorkflowRunSessionByKeyForUpdateInTransaction(tx, workflowID, continuationKey)
	if err != nil {
		return nil, false, err
	}

	return session, result.RowsAffected > 0, nil
}

func UpdateWorkflowRunSessionLastExecutionByRootEventInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	rootEventID uuid.UUID,
	executionID uuid.UUID,
) error {
	now := time.Now()

	return tx.
		Model(&WorkflowRunSession{}).
		Where("workflow_id = ?", workflowID).
		Where("root_event_id = ?", rootEventID).
		Updates(map[string]any{
			"last_execution_id": executionID,
			"updated_at":        &now,
		}).
		Error
}

func FindWorkflowRunSessionByRootEventInTransaction(tx *gorm.DB, workflowID uuid.UUID, rootEventID uuid.UUID) (*WorkflowRunSession, error) {
	var session WorkflowRunSession
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("root_event_id = ?", rootEventID).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func FindWorkflowRunSessionByKey(workflowID uuid.UUID, continuationKey string) (*WorkflowRunSession, error) {
	return FindWorkflowRunSessionByKeyInTransaction(database.Conn(), workflowID, continuationKey)
}
