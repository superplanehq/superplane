package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type AgentMemoryStore struct {
	ID                    uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID        uuid.UUID
	UserID                uuid.UUID
	CanvasID              uuid.UUID
	Provider              string
	ProviderMemoryStoreID string
	Name                  string
	Description           string
	CreatedAt             *time.Time
	UpdatedAt             *time.Time
}

func (AgentMemoryStore) TableName() string { return "agent_memory_stores" }

func CreateAgentMemoryStoreInTransaction(tx *gorm.DB, store *AgentMemoryStore) error {
	return tx.Create(store).Error
}

func FindAgentMemoryStoreByScope(
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
	provider string,
) (*AgentMemoryStore, error) {
	return FindAgentMemoryStoreByScopeInTransaction(database.Conn(), organizationID, userID, canvasID, provider)
}

func FindAgentMemoryStoreByScopeInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
	provider string,
) (*AgentMemoryStore, error) {
	var store AgentMemoryStore
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		Where("canvas_id = ?", canvasID).
		Where("provider = ?", provider).
		First(&store).
		Error
	if err != nil {
		return nil, err
	}
	return &store, nil
}
