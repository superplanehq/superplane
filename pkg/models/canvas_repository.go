package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasRepositoryProviderCodeStorage = "code_storage"
	CanvasRepositoryProviderLocalGit    = "local_git"

	CanvasRepositoryStatusProvisioning = "provisioning"
	CanvasRepositoryStatusReady        = "ready"
	CanvasRepositoryStatusError        = "error"
)

type CanvasRepository struct {
	CanvasID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrganizationID uuid.UUID
	Provider       string
	RepoID         string
	DefaultBranch  string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (CanvasRepository) TableName() string {
	return "canvas_repositories"
}

func FindCanvasRepositoryInTransaction(tx *gorm.DB, canvasID uuid.UUID) (*CanvasRepository, error) {
	var repository CanvasRepository
	err := tx.
		Where("canvas_id = ?", canvasID).
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

func FindCanvasRepository(canvasID uuid.UUID) (*CanvasRepository, error) {
	return FindCanvasRepositoryInTransaction(database.Conn(), canvasID)
}

func UpsertCanvasRepositoryInTransaction(tx *gorm.DB, repository *CanvasRepository) error {
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "canvas_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"organization_id",
			"provider",
			"repo_id",
			"default_branch",
			"status",
			"updated_at",
		}),
	}).Create(repository).Error
}

func UpsertCanvasRepository(repository *CanvasRepository) error {
	return UpsertCanvasRepositoryInTransaction(database.Conn(), repository)
}
