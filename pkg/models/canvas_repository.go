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
	CanvasRepositoryProviderSupergit    = "supergit"

	CanvasRepositoryStatusPending      = "pending"
	CanvasRepositoryStatusProvisioning = "provisioning"
	CanvasRepositoryStatusReady        = "ready"
	CanvasRepositoryStatusError        = "error"
)

type CanvasRepository struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CanvasID       uuid.UUID
	OrganizationID uuid.UUID
	Provider       string
	RepoID         string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

func CreateCanvasRepositoryInTransaction(tx *gorm.DB, repository *CanvasRepository) error {
	return tx.Create(repository).Error
}

func ListPendingCanvasRepositories(limit int) ([]CanvasRepository, error) {
	if limit <= 0 {
		limit = 100
	}

	var repositories []CanvasRepository
	err := database.Conn().
		Where("status = ?", CanvasRepositoryStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&repositories).
		Error
	if err != nil {
		return nil, err
	}

	return repositories, nil
}

func LockPendingCanvasRepository(tx *gorm.DB, canvasID uuid.UUID) (*CanvasRepository, error) {
	var repository CanvasRepository

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("canvas_id = ?", canvasID).
		Where("status = ?", CanvasRepositoryStatusPending).
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

func ResetStuckProvisioningCanvasRepositories() (int64, error) {
	result := database.Conn().
		Model(&CanvasRepository{}).
		Where("status = ?", CanvasRepositoryStatusProvisioning).
		Updates(map[string]any{
			"status":     CanvasRepositoryStatusPending,
			"updated_at": time.Now(),
		})

	return result.RowsAffected, result.Error
}

func (r *CanvasRepository) MarkProvisioning(tx *gorm.DB) error {
	return tx.Model(r).Updates(map[string]any{
		"status":     CanvasRepositoryStatusProvisioning,
		"updated_at": time.Now(),
	}).Error
}

func (r *CanvasRepository) MarkReady(tx *gorm.DB) error {
	return tx.Model(r).Updates(map[string]any{
		"status":     CanvasRepositoryStatusReady,
		"updated_at": time.Now(),
	}).Error
}

func (r *CanvasRepository) MarkError(tx *gorm.DB) error {
	return tx.Model(r).Updates(map[string]any{
		"status":     CanvasRepositoryStatusError,
		"updated_at": time.Now(),
	}).Error
}
