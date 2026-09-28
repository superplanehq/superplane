package models

import (
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkflowStagedFile struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null"`
	WorkflowID     uuid.UUID `gorm:"type:uuid;not null"`
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	BaseVersionID  uuid.UUID `gorm:"type:uuid;not null"`
	Path           string    `gorm:"type:text;not null"`
	Content        string    `gorm:"type:text;not null;default:''"`
	UpdatedAt      time.Time `gorm:"not null"`

	//
	// Deleted marks a leftover staged file removal (row kept). Effective read
	// returns empty content when true. DiscardWorkflowStaging hard-deletes rows
	// to revert staging. New staging updates cannot mark paths deleted.
	//
	Deleted bool `gorm:"not null;default:false"`
}

func (WorkflowStagedFile) TableName() string {
	return "workflow_staged_files"
}

func UpsertStagedFile(
	db *gorm.DB,
	workflowID, userID, baseVersionID, organizationID uuid.UUID,
	path, content string,
) (*WorkflowStagedFile, error) {
	row := WorkflowStagedFile{
		WorkflowID:     workflowID,
		UserID:         userID,
		BaseVersionID:  baseVersionID,
		OrganizationID: organizationID,
		Path:           path,
		Content:        content,
		Deleted:        false,
		UpdatedAt:      time.Now(),
	}

	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "workflow_id"},
			{Name: "user_id"},
			{Name: "path"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"content":    content,
			"deleted":    false,
			"updated_at": time.Now(),
		}),
	}).Create(&row).Error

	if err != nil {
		return nil, err
	}

	return FindStagedFileForUser(db, workflowID, userID, path)
}

func ListStagedFilesForUser(db *gorm.DB, workflowID, userID uuid.UUID) ([]WorkflowStagedFile, error) {
	var rows []WorkflowStagedFile
	err := db.
		Where("workflow_id = ? AND user_id = ?", workflowID, userID).
		Order("path ASC").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func LockStagedFilesForUser(tx *gorm.DB, workflowID, userID uuid.UUID) error {
	sum := sha256.Sum256(append(append([]byte("canvas-staging:"), workflowID[:]...), userID[:]...))
	key := int64(binary.BigEndian.Uint64(sum[:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error
}

func DiscardStagedFilesForUser(db *gorm.DB, workflowID, userID uuid.UUID, paths []string) error {
	query := db.Where("workflow_id = ? AND user_id = ?", workflowID, userID)
	if len(paths) > 0 {
		query = query.Where("path IN ?", paths)
	}

	return query.Delete(&WorkflowStagedFile{}).Error
}

func HasStagedFilesForUser(db *gorm.DB, workflowID, userID uuid.UUID) (bool, error) {
	var count int64
	err := db.Model(&WorkflowStagedFile{}).
		Where("workflow_id = ? AND user_id = ?", workflowID, userID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func FindStagedFileForUser(db *gorm.DB, workflowID, userID uuid.UUID, path string) (*WorkflowStagedFile, error) {
	var row WorkflowStagedFile
	err := db.
		Where("workflow_id = ? AND user_id = ? AND path = ?", workflowID, userID, path).
		First(&row).
		Error
	if err != nil {
		return nil, err
	}

	return &row, nil
}
