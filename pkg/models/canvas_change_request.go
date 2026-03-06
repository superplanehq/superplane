package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	CanvasChangeRequestStatusOpen      = "open"
	CanvasChangeRequestStatusPublished = "published"
)

type CanvasChangeRequest struct {
	ID             uuid.UUID
	WorkflowID     uuid.UUID
	VersionID      uuid.UUID
	OwnerID        *uuid.UUID
	Title          string
	Description    string
	Status         string
	ChangedNodeIDs datatypes.JSONSlice[string]
	PublishedAt    *time.Time
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

type CanvasChangeRequestListOptions struct {
	Limit    int
	Before   *time.Time
	OwnerID  *uuid.UUID
	Statuses []string
	Query    string
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
	return ListCanvasChangeRequestsFilteredInTransaction(tx, workflowID, CanvasChangeRequestListOptions{})
}

func ListCanvasChangeRequestsFilteredInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	options CanvasChangeRequestListOptions,
) ([]CanvasChangeRequest, error) {
	var requests []CanvasChangeRequest
	query := applyCanvasChangeRequestListFilters(tx, workflowID, options, true).
		Order("workflow_change_requests.created_at DESC, workflow_change_requests.id DESC")

	if options.Limit > 0 {
		query = query.Limit(options.Limit)
	}

	err := query.
		Find(&requests).
		Error
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func CountCanvasChangeRequestsFilteredInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	options CanvasChangeRequestListOptions,
) (int64, error) {
	var count int64
	err := applyCanvasChangeRequestListFilters(tx, workflowID, options, false).
		Model(&CanvasChangeRequest{}).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func applyCanvasChangeRequestListFilters(
	tx *gorm.DB,
	workflowID uuid.UUID,
	options CanvasChangeRequestListOptions,
	includeBefore bool,
) *gorm.DB {
	query := tx.Where("workflow_change_requests.workflow_id = ?", workflowID)

	if options.OwnerID != nil {
		query = query.Where("workflow_change_requests.owner_id = ?", *options.OwnerID)
	}

	if len(options.Statuses) > 0 {
		query = query.Where("workflow_change_requests.status IN ?", options.Statuses)
	}

	if includeBefore && options.Before != nil {
		query = query.Where("workflow_change_requests.created_at < ?", *options.Before)
	}

	trimmedQuery := strings.TrimSpace(options.Query)
	if trimmedQuery == "" {
		return query
	}

	like := "%" + strings.ToLower(trimmedQuery) + "%"
	query = query.Where(
		`LOWER(COALESCE(workflow_change_requests.title, '')) LIKE ?
		OR LOWER(COALESCE(workflow_change_requests.description, '')) LIKE ?
		OR LOWER(COALESCE(workflow_change_requests.status, '')) LIKE ?
		OR COALESCE(CAST(workflow_change_requests.owner_id AS TEXT), '') LIKE ?`,
		like,
		like,
		like,
		like,
	)
	return query
}
