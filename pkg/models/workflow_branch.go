package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const CanvasGitBranchMain = "main"

var ErrWorkflowBranchNotFound = errors.New("workflow branch not found")

type WorkflowBranch struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	WorkflowID    uuid.UUID  `gorm:"type:uuid;not null"`
	Name          string     `gorm:"type:text;not null"`
	HeadVersionID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
}

func (WorkflowBranch) TableName() string {
	return "workflow_branches"
}

func FindWorkflowBranch(tx *gorm.DB, workflowID uuid.UUID, name string) (*WorkflowBranch, error) {
	var branch WorkflowBranch
	err := tx.
		Where("workflow_id = ? AND name = ?", workflowID, name).
		First(&branch).
		Error
	if err != nil {
		return nil, err
	}
	return &branch, nil
}

func FindWorkflowBranchByID(tx *gorm.DB, branchID uuid.UUID) (*WorkflowBranch, error) {
	var branch WorkflowBranch
	err := tx.Where("id = ?", branchID).First(&branch).Error
	if err != nil {
		return nil, err
	}
	return &branch, nil
}

func ListWorkflowBranches(tx *gorm.DB, workflowID uuid.UUID) ([]WorkflowBranch, error) {
	var branches []WorkflowBranch
	err := tx.
		Where("workflow_id = ?", workflowID).
		Order("CASE WHEN name = 'main' THEN 0 ELSE 1 END, name ASC").
		Find(&branches).
		Error
	if err != nil {
		return nil, err
	}
	return branches, nil
}

func FindMainWorkflowBranch(tx *gorm.DB, workflowID uuid.UUID) (*WorkflowBranch, error) {
	return FindWorkflowBranch(tx, workflowID, CanvasGitBranchMain)
}

func CreateWorkflowBranch(
	tx *gorm.DB,
	workflowID uuid.UUID,
	name string,
	headVersionID *uuid.UUID,
) (*WorkflowBranch, error) {
	now := time.Now()
	branch := WorkflowBranch{
		ID:            uuid.New(),
		WorkflowID:    workflowID,
		Name:          name,
		HeadVersionID: headVersionID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := tx.Create(&branch).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

func UpdateWorkflowBranchHead(tx *gorm.DB, branchID, headVersionID uuid.UUID) error {
	return tx.Model(&WorkflowBranch{}).
		Where("id = ?", branchID).
		Updates(map[string]any{
			"head_version_id": headVersionID,
			"updated_at":      time.Now(),
		}).
		Error
}

func RenameWorkflowBranch(tx *gorm.DB, workflowID uuid.UUID, oldName, newName string) error {
	if oldName == CanvasGitBranchMain {
		return errors.New("cannot rename main branch")
	}
	return tx.Model(&WorkflowBranch{}).
		Where("workflow_id = ? AND name = ?", workflowID, oldName).
		Update("name", newName).
		Error
}

func DeleteWorkflowBranch(tx *gorm.DB, workflowID uuid.UUID, name string) error {
	if name == CanvasGitBranchMain {
		return errors.New("cannot delete main branch")
	}

	result := tx.
		Where("workflow_id = ? AND name = ?", workflowID, name).
		Delete(&WorkflowBranch{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func lockWorkflowBranchForUpdate(tx *gorm.DB, branchID uuid.UUID) (*WorkflowBranch, error) {
	var branch WorkflowBranch
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", branchID).
		First(&branch).
		Error
	if err != nil {
		return nil, err
	}
	return &branch, nil
}

func FindWorkflowBranchByIDConn(branchID uuid.UUID) (*WorkflowBranch, error) {
	return FindWorkflowBranchByID(database.Conn(), branchID)
}

func ListWorkflowBranchesConn(workflowID uuid.UUID) ([]WorkflowBranch, error) {
	return ListWorkflowBranches(database.Conn(), workflowID)
}
