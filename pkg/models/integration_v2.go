package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IntegrationV2 struct {
	ID              uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID  uuid.UUID
	Name            string
	IntegrationName string
	Parameters      datatypes.JSONSlice[IntegrationV2Parameter]
	Capabilities    datatypes.JSONSlice[IntegrationV2Capability]
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (i *IntegrationV2) TableName() string {
	return "integrations"
}

type IntegrationV2Parameter struct {
	Name        string
	Label       string
	Description string
	Type        string
	Value       any
	Editable    bool
}

type IntegrationV2Capability struct {
	Type      string
	Component *Component
	Trigger   *Trigger
}

type Component struct {
	Name          string
	Label         string
	Description   string
	Configuration []configuration.Field
}

type Trigger struct {
	Name          string
	Label         string
	Description   string
	Configuration []configuration.Field
}

func ListIntegrationsV2(orgID uuid.UUID) ([]IntegrationV2, error) {
	return ListIntegrationsV2InTransaction(database.Conn(), orgID)
}

func ListIntegrationsV2InTransaction(tx *gorm.DB, orgID uuid.UUID) ([]IntegrationV2, error) {
	var integrations []IntegrationV2
	err := tx.Where("organization_id = ?", orgID).Find(&integrations).Error
	if err != nil {
		return nil, err
	}

	return integrations, nil
}

func FindIntegrationV2(orgID, id uuid.UUID) (*IntegrationV2, error) {
	return FindIntegrationV2InTransaction(database.Conn(), orgID, id)
}

func FindIntegrationV2ByName(orgID uuid.UUID, name string) (*IntegrationV2, error) {
	return FindIntegrationV2ByNameInTransaction(database.Conn(), orgID, name)
}

func FindIntegrationV2ByNameInTransaction(tx *gorm.DB, orgID uuid.UUID, name string) (*IntegrationV2, error) {
	var integration IntegrationV2
	err := tx.Where("organization_id = ?", orgID).Where("name = ?", name).First(&integration).Error
	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func FindIntegrationV2InTransaction(tx *gorm.DB, orgID, id uuid.UUID) (*IntegrationV2, error) {
	var integration IntegrationV2

	err := tx.Where("organization_id = ?", orgID).Where("id = ?", id).First(&integration).Error
	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func CreateIntegrationV2(orgID uuid.UUID, integrationName, name string) (*IntegrationV2, error) {
	return CreateIntegrationV2InTransaction(database.Conn(), orgID, integrationName, name)
}

func CreateIntegrationV2InTransaction(tx *gorm.DB, orgID uuid.UUID, integrationName, name string) (*IntegrationV2, error) {
	now := time.Now()
	integration := IntegrationV2{
		OrganizationID:  orgID,
		Name:            name,
		IntegrationName: integrationName,
		CreatedAt:       &now,
		UpdatedAt:       &now,
	}

	err := tx.Create(&integration).Error
	if err != nil {
		return nil, err
	}

	return &integration, nil
}
