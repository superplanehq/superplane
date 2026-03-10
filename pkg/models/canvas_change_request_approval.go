package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type CanvasChangeRequestApproval struct {
	ID                      uuid.UUID
	WorkflowID              uuid.UUID
	WorkflowChangeRequestID uuid.UUID `gorm:"column:workflow_change_request_id"`
	ApproverIndex           int
	ApproverType            string
	ApproverUserID          *uuid.UUID
	ApproverRole            *string
	ActorUserID             *uuid.UUID
	State                   string
	InvalidatedAt           *time.Time
	CreatedAt               *time.Time
	UpdatedAt               *time.Time
}

func (c *CanvasChangeRequestApproval) TableName() string {
	return "workflow_change_request_approvals"
}

func CreateCanvasChangeRequestApprovalInTransaction(tx *gorm.DB, approval *CanvasChangeRequestApproval) error {
	return tx.Create(approval).Error
}

func ListCanvasChangeRequestApprovals(workflowID, changeRequestID uuid.UUID) ([]CanvasChangeRequestApproval, error) {
	return ListCanvasChangeRequestApprovalsInTransaction(database.Conn(), workflowID, changeRequestID)
}

func ListCanvasChangeRequestApprovalsInTransaction(
	tx *gorm.DB,
	workflowID, changeRequestID uuid.UUID,
) ([]CanvasChangeRequestApproval, error) {
	var approvals []CanvasChangeRequestApproval
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("workflow_change_request_id = ?", changeRequestID).
		Order("created_at ASC, id ASC").
		Find(&approvals).
		Error
	if err != nil {
		return nil, err
	}

	return approvals, nil
}

func ListCanvasChangeRequestApprovalsByRequestIDsInTransaction(
	tx *gorm.DB,
	workflowID uuid.UUID,
	changeRequestIDs []uuid.UUID,
) (map[uuid.UUID][]CanvasChangeRequestApproval, error) {
	result := make(map[uuid.UUID][]CanvasChangeRequestApproval)
	if len(changeRequestIDs) == 0 {
		return result, nil
	}

	var approvals []CanvasChangeRequestApproval
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("workflow_change_request_id IN ?", changeRequestIDs).
		Order("created_at ASC, id ASC").
		Find(&approvals).
		Error
	if err != nil {
		return nil, err
	}

	for i := range approvals {
		changeRequestID := approvals[i].WorkflowChangeRequestID
		result[changeRequestID] = append(result[changeRequestID], approvals[i])
	}

	return result, nil
}

func InvalidateCanvasChangeRequestApprovalsInTransaction(
	tx *gorm.DB,
	workflowID, changeRequestID uuid.UUID,
	invalidatedAt time.Time,
) error {
	return tx.
		Model(&CanvasChangeRequestApproval{}).
		Where("workflow_id = ?", workflowID).
		Where("workflow_change_request_id = ?", changeRequestID).
		Where("invalidated_at IS NULL").
		Updates(map[string]any{
			"invalidated_at": invalidatedAt,
			"updated_at":     invalidatedAt,
		}).
		Error
}
