package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowStaging struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	BranchID  uuid.UUID  `gorm:"type:uuid;not null"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	Path      string     `gorm:"type:text;not null"`
	Content   string     `gorm:"type:text;not null;default:''"`
	Deleted   bool       `gorm:"not null;default:false"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (WorkflowStaging) TableName() string {
	return "workflow_staged_files"
}

func UpsertWorkflowStagingPath(
	branchID, userID uuid.UUID,
	path, content string,
	updatedBy *uuid.UUID,
) (*WorkflowStaging, error) {
	return UpsertWorkflowStagingPathInTransaction(
		database.Conn(),
		branchID,
		userID,
		path,
		content,
		updatedBy,
	)
}

func MarkWorkflowStagingPathDeleted(
	branchID, userID uuid.UUID,
	path string,
	updatedBy *uuid.UUID,
) error {
	return MarkWorkflowStagingPathDeletedInTransaction(
		database.Conn(),
		branchID,
		userID,
		path,
		updatedBy,
	)
}

func ListWorkflowStaging(branchID, userID uuid.UUID) ([]WorkflowStaging, error) {
	return ListWorkflowStagingInTransaction(database.Conn(), branchID, userID)
}

func DiscardWorkflowStaging(branchID, userID uuid.UUID, paths []string) error {
	return DiscardWorkflowStagingInTransaction(database.Conn(), branchID, userID, paths)
}

func HasWorkflowStaging(branchID, userID uuid.UUID) (bool, error) {
	return HasWorkflowStagingInTransaction(database.Conn(), branchID, userID)
}

func FindWorkflowStagingPath(branchID, userID uuid.UUID, path string) (*WorkflowStaging, error) {
	return FindWorkflowStagingPathInTransaction(database.Conn(), branchID, userID, path)
}

func UpsertWorkflowStagingPathInTransaction(
	tx *gorm.DB,
	branchID, userID uuid.UUID,
	path, content string,
	updatedBy *uuid.UUID,
) (*WorkflowStaging, error) {
	row := WorkflowStaging{
		BranchID:  branchID,
		UserID:    userID,
		Path:      path,
		Content:   content,
		Deleted:   false,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now(),
	}

	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "branch_id"},
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

	return FindWorkflowStagingPathInTransaction(tx, branchID, userID, path)
}

func MarkWorkflowStagingPathDeletedInTransaction(
	tx *gorm.DB,
	branchID, userID uuid.UUID,
	path string,
	updatedBy *uuid.UUID,
) error {
	row := WorkflowStaging{
		BranchID:  branchID,
		UserID:    userID,
		Path:      path,
		Content:   "",
		Deleted:   true,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now(),
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "branch_id"},
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

func ListWorkflowStagingInTransaction(tx *gorm.DB, branchID, userID uuid.UUID) ([]WorkflowStaging, error) {
	var rows []WorkflowStaging
	err := tx.
		Where("branch_id = ? AND user_id = ?", branchID, userID).
		Order("path ASC").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func DiscardWorkflowStagingInTransaction(tx *gorm.DB, branchID, userID uuid.UUID, paths []string) error {
	query := tx.Where("branch_id = ? AND user_id = ?", branchID, userID)
	if len(paths) > 0 {
		query = query.Where("path IN ?", paths)
	}
	return query.Delete(&WorkflowStaging{}).Error
}

func HasWorkflowStagingInTransaction(tx *gorm.DB, branchID, userID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&WorkflowStaging{}).
		Where("branch_id = ? AND user_id = ?", branchID, userID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func FindWorkflowStagingPathInTransaction(tx *gorm.DB, branchID, userID uuid.UUID, path string) (*WorkflowStaging, error) {
	var row WorkflowStaging
	err := tx.
		Where("branch_id = ? AND user_id = ? AND path = ?", branchID, userID, path).
		First(&row).
		Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Legacy adapter: staging was keyed by version_id; map to branch head + user during POC transition.
func ListWorkflowStagingForVersion(versionID uuid.UUID) ([]WorkflowStaging, error) {
	return nil, nil
}

func DiscardWorkflowStagingForVersion(versionID uuid.UUID, paths []string) error {
	_ = versionID
	_ = paths
	return nil
}

func HasWorkflowStagingForVersion(versionID uuid.UUID) (bool, error) {
	_ = versionID
	return false, nil
}
