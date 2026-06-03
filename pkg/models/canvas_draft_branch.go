package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type CanvasDraftBranch struct {
	ID             uuid.UUID
	CanvasID       uuid.UUID
	OrganizationID uuid.UUID
	BranchName     string
	DisplayName    string
	OwnerID        *uuid.UUID
	CreatedBy      *uuid.UUID
	TipSHA         string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func (b *CanvasDraftBranch) TableName() string {
	return "canvas_draft_branches"
}

func FindDraftBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) (*CanvasDraftBranch, error) {
	var branch CanvasDraftBranch
	err := tx.
		Where("canvas_id = ?", canvasID).
		Where("branch_name = ?", branchName).
		First(&branch).
		Error
	if err != nil {
		return nil, err
	}

	return &branch, nil
}

func FindDraftBranch(canvasID uuid.UUID, branchName string) (*CanvasDraftBranch, error) {
	return FindDraftBranchInTransaction(database.Conn(), canvasID, branchName)
}

func ListDraftBranchesForCanvasInTransaction(tx *gorm.DB, canvasID uuid.UUID) ([]CanvasDraftBranch, error) {
	var branches []CanvasDraftBranch
	err := tx.
		Where("canvas_id = ?", canvasID).
		Order("updated_at DESC").
		Find(&branches).
		Error
	if err != nil {
		return nil, err
	}

	return branches, nil
}

func ListDraftBranchesForCanvas(canvasID uuid.UUID) ([]CanvasDraftBranch, error) {
	return ListDraftBranchesForCanvasInTransaction(database.Conn(), canvasID)
}

func CreateDraftBranchInTransaction(tx *gorm.DB, branch *CanvasDraftBranch) error {
	now := time.Now()
	if branch.CreatedAt == nil {
		branch.CreatedAt = &now
	}
	if branch.UpdatedAt == nil {
		branch.UpdatedAt = &now
	}
	if branch.ID == uuid.Nil {
		branch.ID = uuid.New()
	}

	return tx.Create(branch).Error
}

func CreateDraftBranch(branch *CanvasDraftBranch) error {
	return CreateDraftBranchInTransaction(database.Conn(), branch)
}

func UpdateDraftBranchTipInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName, tipSHA string) error {
	now := time.Now()
	return tx.
		Model(&CanvasDraftBranch{}).
		Where("canvas_id = ?", canvasID).
		Where("branch_name = ?", branchName).
		Updates(map[string]any{
			"tip_sha":    tipSHA,
			"updated_at": now,
		}).
		Error
}

func UpdateDraftBranchTip(canvasID uuid.UUID, branchName, tipSHA string) error {
	return UpdateDraftBranchTipInTransaction(database.Conn(), canvasID, branchName, tipSHA)
}

func DeleteDraftBranchInTransaction(tx *gorm.DB, canvasID uuid.UUID, branchName string) error {
	return tx.
		Where("canvas_id = ?", canvasID).
		Where("branch_name = ?", branchName).
		Delete(&CanvasDraftBranch{}).
		Error
}

func DeleteDraftBranch(canvasID uuid.UUID, branchName string) error {
	return DeleteDraftBranchInTransaction(database.Conn(), canvasID, branchName)
}
