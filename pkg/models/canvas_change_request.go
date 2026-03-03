package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	CanvasChangeRequestStatusOpen       = "open"
	CanvasChangeRequestStatusPublished  = "published"
	CanvasChangeRequestStatusConflicted = "conflicted"
	CanvasChangeRequestStatusClosed     = "closed"
)

type CanvasChangeRequest struct {
	ID                 uuid.UUID
	WorkflowID         uuid.UUID
	VersionID          uuid.UUID
	OwnerID            *uuid.UUID
	BasedOnVersionID   *uuid.UUID
	Status             string
	ChangedNodeIDs     datatypes.JSONSlice[string]
	ConflictingNodeIDs datatypes.JSONSlice[string]
	PublishedAt        *time.Time
	CreatedAt          *time.Time
	UpdatedAt          *time.Time
}

func (c *CanvasChangeRequest) TableName() string {
	return "workflow_change_requests"
}

func FindCanvasChangeRequest(workflowID, changeRequestID uuid.UUID) (*CanvasChangeRequest, error) {
	return FindCanvasChangeRequestInTransaction(database.Conn(), workflowID, changeRequestID)
}

func FindCanvasChangeRequestInTransaction(tx *gorm.DB, workflowID, changeRequestID uuid.UUID) (*CanvasChangeRequest, error) {
	var request CanvasChangeRequest
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("id = ?", changeRequestID).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func FindCanvasChangeRequestByVersionInTransaction(tx *gorm.DB, workflowID, versionID uuid.UUID) (*CanvasChangeRequest, error) {
	var request CanvasChangeRequest
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("version_id = ?", versionID).
		First(&request).
		Error
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func ListCanvasChangeRequests(workflowID uuid.UUID) ([]CanvasChangeRequest, error) {
	return ListCanvasChangeRequestsInTransaction(database.Conn(), workflowID)
}

func ListCanvasChangeRequestsInTransaction(tx *gorm.DB, workflowID uuid.UUID) ([]CanvasChangeRequest, error) {
	var requests []CanvasChangeRequest
	err := tx.
		Where("workflow_id = ?", workflowID).
		Order("created_at DESC").
		Find(&requests).
		Error
	if err != nil {
		return nil, err
	}

	return requests, nil
}
