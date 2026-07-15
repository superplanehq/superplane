package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (

	//
	// The invocation is pending for a run to be started in the target app/entrypoint.
	//
	AppInvocationStatePending = "pending"

	//
	// Run has been started in the target app/entrypoint.
	//
	AppInvocationStateWaiting = "waiting"

	//
	// The invocation has been completed.
	//
	AppInvocationStateCompleted = "completed"
)

type AppInvocation struct {
	ID        uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	State     string
	Payload   datatypes.JSON
	CreatedAt time.Time
	UpdatedAt time.Time

	//
	// The reference to the target canvas/node to invoke / invoked.
	//
	TargetCanvasID *uuid.UUID
	TargetNodeID   string

	//
	// The references to the caller app/execution.
	//
	CallerAppID       uuid.UUID
	CallerExecutionID *uuid.UUID

	//
	// The run ID that this invocation generated.
	//
	RunID *uuid.UUID
}

func (i *AppInvocation) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func FindInvocationForRun(tx *gorm.DB, runID uuid.UUID) (*AppInvocation, error) {
	var invocation AppInvocation
	if err := tx.Where("run_id = ?", runID).First(&invocation).Error; err != nil {
		return nil, err
	}
	return &invocation, nil
}

func ListInvocations(tx *gorm.DB) ([]AppInvocation, error) {
	var invocations []AppInvocation
	err := tx.
		Where("state = ?", AppInvocationStatePending).
		Find(&invocations).Error
	if err != nil {
		return nil, err
	}

	return invocations, err
}

func LockInvocation(tx *gorm.DB, invocationID uuid.UUID) (*AppInvocation, error) {
	var invocation AppInvocation

	err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", invocationID).
		Where("state = ?", WebhookStatePending).
		First(&invocation).
		Error

	if err != nil {
		return nil, err
	}

	return &invocation, nil
}

func (i *AppInvocation) FindTargetNode(tx *gorm.DB) (*CanvasNode, error) {
	node, err := FindCanvasNode(tx, *i.TargetCanvasID, i.TargetNodeID)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (i *AppInvocation) AttachRun(tx *gorm.DB, runID uuid.UUID) error {
	return tx.Model(i).Updates(map[string]any{
		"state":      AppInvocationStateWaiting,
		"run_id":     runID,
		"updated_at": time.Now(),
	}).Error
}

func (i *AppInvocation) Delete(tx *gorm.DB) error {
	return tx.Delete(i).Error
}
