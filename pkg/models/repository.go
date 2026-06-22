package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RepositoryStatusPending = "pending"
	RepositoryStatusReady   = "ready"
	RepositoryStatusError   = "error"
)

type Repository struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CanvasID       uuid.UUID
	OrganizationID uuid.UUID
	Provider       string
	RepoID         string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func FindRepository(organizationID, canvasID uuid.UUID) (*Repository, error) {
	var repository Repository
	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("organization_id = ?", organizationID).
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

func FindRepositoryUnscoped(canvasID uuid.UUID) (*Repository, error) {
	return FindRepositoryInTransaction(database.Conn(), canvasID)
}

func FindRepositoryInTransaction(tx *gorm.DB, canvasID uuid.UUID) (*Repository, error) {
	var repository Repository
	err := tx.
		Where("canvas_id = ?", canvasID).
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

func (c *Canvas) CreatePendingRepository(provider, providerRepoID string) (*Repository, error) {
	return c.CreatePendingRepositoryInTransaction(database.Conn(), provider, providerRepoID)
}

func (c *Canvas) CreatePendingRepositoryInTransaction(tx *gorm.DB, provider, providerRepoID string) (*Repository, error) {
	r := &Repository{
		CanvasID:       c.ID,
		OrganizationID: c.OrganizationID,
		Provider:       provider,
		RepoID:         providerRepoID,
		Status:         RepositoryStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := tx.Clauses(clause.Returning{}).Create(r).Error; err != nil {
		return nil, err
	}

	return r, nil
}

func ListPendingRepositories(limit int) ([]Repository, error) {
	if limit <= 0 {
		limit = 100
	}

	var repositories []Repository
	err := database.Conn().
		Where("status = ?", RepositoryStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&repositories).
		Error
	if err != nil {
		return nil, err
	}

	return repositories, nil
}

func LockPendingRepository(tx *gorm.DB, id uuid.UUID) (*Repository, error) {
	var repository Repository

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("status = ?", RepositoryStatusPending).
		First(&repository).
		Error
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

func (r *Repository) MarkReady(tx *gorm.DB) error {
	return tx.Model(r).Updates(map[string]any{
		"status":     RepositoryStatusReady,
		"updated_at": time.Now(),
	}).Error
}

func (r *Repository) MarkError(tx *gorm.DB) error {
	return tx.Model(r).Updates(map[string]any{
		"status":     RepositoryStatusError,
		"updated_at": time.Now(),
	}).Error
}
