package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	IntegrationTypeSemaphore = "semaphore"
	IntegrationTypeGithub    = "github"

	IntegrationAuthTypeToken = "token"
	IntegrationAuthTypeOIDC  = "oidc"
)

type Integration struct {
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name       string
	DomainType string
	DomainID   uuid.UUID
	Type       string
	URL        string
	AuthType   string
	Auth       datatypes.JSONType[IntegrationAuth]
	OIDC       datatypes.JSONType[IntegrationOIDC] `gorm:"column:oidc"`
	CreatedAt  *time.Time
	CreatedBy  uuid.UUID
	UpdatedAt  *time.Time
}

type IntegrationAuth struct {
	Token *IntegrationAuthToken `json:"token"`
}

type IntegrationAuthToken struct {
	ValueFrom ValueDefinitionFrom `json:"value_from"`
}

type IntegrationOIDC struct {
	Supported bool `json:"supported"`
}

func CreateIntegration(integration *Integration) (*Integration, error) {
	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(&integration).
		Error

	if err == nil {
		return integration, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

func (i *Integration) ListResources(resourceType string) ([]*Resource, error) {
	resources := []*Resource{}

	err := database.Conn().
		Where("integration_id = ?", i.ID).
		Where("type = ?", resourceType).
		Find(&resources).
		Error

	if err != nil {
		return nil, err
	}

	return resources, nil
}

func (i *Integration) CreateResource(resourceType, externalID, name string) (*Resource, error) {
	return i.CreateResourceInTransaction(database.Conn(), resourceType, externalID, name)
}

func (i *Integration) CreateResourceInTransaction(tx *gorm.DB, resourceType, externalID, name string) (*Resource, error) {
	now := time.Now()

	resource := Resource{
		ExternalID:    externalID,
		ResourceName:  name,
		CreatedAt:     &now,
		IntegrationID: i.ID,
		ResourceType:  resourceType,
	}

	err := tx.
		Clauses(clause.Returning{}).
		Create(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func FindIntegrationByName(domainType string, domainID uuid.UUID, name string) (*Integration, error) {
	integration := Integration{}

	err := database.Conn().
		Where("domain_type = ?", domainType).
		Where("domain_id = ?", domainID).
		Where("name = ?", name).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func FindIntegrationByID(id uuid.UUID) (*Integration, error) {
	integration := Integration{}

	err := database.Conn().
		Where("id = ?", id).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func FindDomainIntegration(domainType string, domainID uuid.UUID, id uuid.UUID) (*Integration, error) {
	integration := Integration{}

	err := database.Conn().
		Where("domain_type = ?", domainType).
		Where("domain_id = ?", domainID).
		Where("id = ?", id).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func ListIntegrations(domainType string, domainID uuid.UUID) ([]*Integration, error) {
	integrations := []*Integration{}

	err := database.Conn().
		Where("domain_type = ?", domainType).
		Where("domain_id = ?", domainID).
		Find(&integrations).
		Error

	if err != nil {
		return nil, err
	}

	return integrations, nil
}

type IntegrationResource struct {
	Name            string
	IntegrationName string
}
