package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowStaging struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	WorkflowID     uuid.UUID `gorm:"type:uuid;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	VersionID      uuid.UUID `gorm:"type:uuid;not null"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null"`
	Path           string    `gorm:"type:text;not null"`
	Content        string    `gorm:"type:text;not null;default:''"`
	// Deleted marks a staged file removal (row kept). Effective read returns empty
	// content when true. DiscardWorkflowStaging hard-deletes rows to revert staging.
	Deleted   bool       `gorm:"not null;default:false"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (WorkflowStaging) TableName() string {
	return "workflow_staged_files"
}

func UpsertWorkflowStagingPath(
	tx *gorm.DB,
	workflowID, userID, baseVersionID, organizationID uuid.UUID,
	path, content string,
	updatedBy *uuid.UUID,
) (*WorkflowStaging, error) {
	if tx == nil {
		tx = database.Conn()
	}
	return UpsertWorkflowStagingPathInTransaction(
		tx,
		workflowID,
		userID,
		baseVersionID,
		organizationID,
		path,
		content,
		updatedBy,
	)
}

func MarkWorkflowStagingPathDeleted(
	tx *gorm.DB,
	workflowID, userID, baseVersionID, organizationID uuid.UUID,
	path string,
	updatedBy *uuid.UUID,
) error {
	if tx == nil {
		tx = database.Conn()
	}
	return MarkWorkflowStagingPathDeletedInTransaction(
		tx,
		workflowID,
		userID,
		baseVersionID,
		organizationID,
		path,
		updatedBy,
	)
}

func ListWorkflowStagingForUser(tx *gorm.DB, workflowID, userID uuid.UUID) ([]WorkflowStaging, error) {
	if tx == nil {
		tx = database.Conn()
	}
	return ListWorkflowStagingForUserInTransaction(tx, workflowID, userID)
}

func DiscardWorkflowStagingForUser(tx *gorm.DB, workflowID, userID uuid.UUID, paths []string) error {
	if tx == nil {
		tx = database.Conn()
	}
	return DiscardWorkflowStagingForUserInTransaction(tx, workflowID, userID, paths)
}

func HasWorkflowStagingForUser(tx *gorm.DB, workflowID, userID uuid.UUID) (bool, error) {
	if tx == nil {
		tx = database.Conn()
	}
	return HasWorkflowStagingForUserInTransaction(tx, workflowID, userID)
}

func FindWorkflowStagingPathForUser(tx *gorm.DB, workflowID, userID uuid.UUID, path string) (*WorkflowStaging, error) {
	if tx == nil {
		tx = database.Conn()
	}
	return FindWorkflowStagingPathForUserInTransaction(tx, workflowID, userID, path)
}

func StagingBaseVersionID(rows []WorkflowStaging) uuid.UUID {
	if len(rows) == 0 {
		return uuid.Nil
	}
	return rows[0].VersionID
}

func UpsertWorkflowStagingPathInTransaction(
	tx *gorm.DB,
	workflowID, userID, baseVersionID, organizationID uuid.UUID,
	path, content string,
	updatedBy *uuid.UUID,
) (*WorkflowStaging, error) {
	row := WorkflowStaging{
		WorkflowID:     workflowID,
		UserID:         userID,
		VersionID:      baseVersionID,
		OrganizationID: organizationID,
		Path:           path,
		Content:        content,
		Deleted:        false,
		UpdatedBy:      updatedBy,
		UpdatedAt:      time.Now(),
	}

	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "workflow_id"},
			{Name: "user_id"},
			{Name: "path"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"content":    content,
			"deleted":    false,
			"updated_by": updatedBy,
			"updated_at": time.Now(),
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}

	return FindWorkflowStagingPathForUserInTransaction(tx, workflowID, userID, path)
}

func MarkWorkflowStagingPathDeletedInTransaction(
	tx *gorm.DB,
	workflowID, userID, baseVersionID, organizationID uuid.UUID,
	path string,
	updatedBy *uuid.UUID,
) error {
	row := WorkflowStaging{
		WorkflowID:     workflowID,
		UserID:         userID,
		VersionID:      baseVersionID,
		OrganizationID: organizationID,
		Path:           path,
		Content:        "",
		Deleted:        true,
		UpdatedBy:      updatedBy,
		UpdatedAt:      time.Now(),
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "workflow_id"},
			{Name: "user_id"},
			{Name: "path"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"content":    "",
			"deleted":    true,
			"updated_by": updatedBy,
			"updated_at": time.Now(),
		}),
	}).Create(&row).Error
}

func ListWorkflowStagingForUserInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID) ([]WorkflowStaging, error) {
	var rows []WorkflowStaging
	err := tx.
		Where("workflow_id = ? AND user_id = ?", workflowID, userID).
		Order("path ASC").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func DiscardWorkflowStagingForUserInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID, paths []string) error {
	query := tx.Where("workflow_id = ? AND user_id = ?", workflowID, userID)
	if len(paths) > 0 {
		query = query.Where("path IN ?", paths)
	}

	return query.Delete(&WorkflowStaging{}).Error
}

func HasWorkflowStagingForUserInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&WorkflowStaging{}).
		Where("workflow_id = ? AND user_id = ?", workflowID, userID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func FindWorkflowStagingPathForUserInTransaction(tx *gorm.DB, workflowID, userID uuid.UUID, path string) (*WorkflowStaging, error) {
	var row WorkflowStaging
	err := tx.
		Where("workflow_id = ? AND user_id = ? AND path = ?", workflowID, userID, path).
		First(&row).
		Error
	if err != nil {
		return nil, err
	}

	return &row, nil
}
