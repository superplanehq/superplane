package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

const (
	IntegrationTypeSemaphore = "semaphore"
	IntegrationTypeGithub    = "github"

	IntegrationAuthTypeToken = "token"
	IntegrationAuthTypeOIDC  = "oidc"
)

type Integration struct {
	ID         uuid.UUID
	Name       string
	DomainType string
	DomainID   uuid.UUID
	Type       string
	URL        string
	AuthType   string
	Auth       datatypes.JSONType[IntegrationAuth]
	OIDC       datatypes.JSONType[IntegrationOIDC]
	CreatedAt  *time.Time
	CreatedBy  uuid.UUID
	UpdatedAt  *time.Time
}

type IntegrationAuth struct {
	Token IntegrationAuthToken `json:"token"`
}

type IntegrationAuthToken struct {
	ValueFrom ValueDefinitionFrom `json:"value_from"`
}

type IntegrationOIDC struct {
	Enabled bool `json:"enabled"`
}

func CreateIntegration(integration *Integration) (*Integration, error) {
	now := time.Now()
	integration.CreatedAt = &now
	integration.UpdatedAt = &now

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

func FindIntegrationByID(domainType string, domainID, id uuid.UUID) (*Integration, error) {
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
