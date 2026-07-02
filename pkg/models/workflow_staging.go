package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowStaging struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	VersionID      uuid.UUID `gorm:"type:uuid;not null"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null"`
	Path           string    `gorm:"type:text;not null"`
	Content        string    `gorm:"type:text;not null;default:''"`
	BaseHeadSHA    string    `gorm:"column:base_head_sha;type:varchar(40);not null;default:''"`
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
	versionID, organizationID uuid.UUID,
	path, content, baseHeadSHA string,
	updatedBy *uuid.UUID,
) (*WorkflowStaging, error) {
	return UpsertWorkflowStagingPathInTransaction(
		database.Conn(),
		versionID,
		organizationID,
		path,
		content,
		baseHeadSHA,
		updatedBy,
	)
}

func MarkWorkflowStagingPathDeleted(
	versionID, organizationID uuid.UUID,
	path, baseHeadSHA string,
	updatedBy *uuid.UUID,
) error {
	return MarkWorkflowStagingPathDeletedInTransaction(
		database.Conn(),
		versionID,
		organizationID,
		path,
		baseHeadSHA,
		updatedBy,
	)
}

func ListWorkflowStaging(versionID uuid.UUID) ([]WorkflowStaging, error) {
	return ListWorkflowStagingInTransaction(database.Conn(), versionID)
}

func DiscardWorkflowStaging(versionID uuid.UUID, paths []string) error {
	return DiscardWorkflowStagingInTransaction(database.Conn(), versionID, paths)
}

func HasWorkflowStaging(versionID uuid.UUID) (bool, error) {
	return HasWorkflowStagingInTransaction(database.Conn(), versionID)
}

func FindWorkflowStagingPath(versionID uuid.UUID, path string) (*WorkflowStaging, error) {
	return FindWorkflowStagingPathInTransaction(database.Conn(), versionID, path)
}

func UpsertWorkflowStagingPathInTransaction(
	tx *gorm.DB,
	versionID, organizationID uuid.UUID,
	path, content, baseHeadSHA string,
	updatedBy *uuid.UUID,
) (*WorkflowStaging, error) {
	row := WorkflowStaging{
		VersionID:      versionID,
		OrganizationID: organizationID,
		Path:           path,
		Content:        content,
		BaseHeadSHA:    strings.TrimSpace(baseHeadSHA),
		Deleted:        false,
		UpdatedBy:      updatedBy,
		UpdatedAt:      time.Now(),
	}

	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "version_id"},
			{Name: "path"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"content":    content,
			"deleted":    false,
			"updated_by": updatedBy,
			"updated_at": time.Now(),
			"base_head_sha": gorm.Expr(
				"CASE WHEN workflow_staged_files.base_head_sha = '' THEN ? ELSE workflow_staged_files.base_head_sha END",
				strings.TrimSpace(baseHeadSHA),
			),
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}

	return FindWorkflowStagingPathInTransaction(tx, versionID, path)
}

// MarkWorkflowStagingPathDeletedInTransaction stages removal of a path before commit.
// The row is kept with deleted=true so effective read can distinguish "not staged"
// (fall back to the draft version row) from "staged as deleted" (empty content).
func MarkWorkflowStagingPathDeletedInTransaction(
	tx *gorm.DB,
	versionID, organizationID uuid.UUID,
	path, baseHeadSHA string,
	updatedBy *uuid.UUID,
) error {
	row := WorkflowStaging{
		VersionID:      versionID,
		OrganizationID: organizationID,
		Path:           path,
		Content:        "",
		BaseHeadSHA:    strings.TrimSpace(baseHeadSHA),
		Deleted:        true,
		UpdatedBy:      updatedBy,
		UpdatedAt:      time.Now(),
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "version_id"},
			{Name: "path"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"content":    "",
			"deleted":    true,
			"updated_by": updatedBy,
			"updated_at": time.Now(),
			"base_head_sha": gorm.Expr(
				"CASE WHEN workflow_staged_files.base_head_sha = '' THEN ? ELSE workflow_staged_files.base_head_sha END",
				strings.TrimSpace(baseHeadSHA),
			),
		}),
	}).Create(&row).Error
}

func ListWorkflowStagingInTransaction(tx *gorm.DB, versionID uuid.UUID) ([]WorkflowStaging, error) {
	var rows []WorkflowStaging
	err := tx.
		Where("version_id = ?", versionID).
		Order("path ASC").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// DiscardWorkflowStagingInTransaction removes staging rows to revert pending edits.
// This is not the same as MarkWorkflowStagingPathDeleted, which keeps a row to
// record an intentional staged file deletion until commit.
func DiscardWorkflowStagingInTransaction(tx *gorm.DB, versionID uuid.UUID, paths []string) error {
	query := tx.Where("version_id = ?", versionID)
	if len(paths) > 0 {
		query = query.Where("path IN ?", paths)
	}

	return query.Delete(&WorkflowStaging{}).Error
}

func HasWorkflowStagingInTransaction(tx *gorm.DB, versionID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&WorkflowStaging{}).
		Where("version_id = ?", versionID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func FindWorkflowStagingPathInTransaction(tx *gorm.DB, versionID uuid.UUID, path string) (*WorkflowStaging, error) {
	var row WorkflowStaging
	err := tx.
		Where("version_id = ? AND path = ?", versionID, path).
		First(&row).
		Error
	if err != nil {
		return nil, err
	}

	return &row, nil
}
