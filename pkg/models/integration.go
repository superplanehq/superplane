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

	IntegrationStatePending = "pending"
	IntegrationStateActive  = "active"
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
	State      string
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
	integration.State = IntegrationStatePending

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

func (i *Integration) UpdateStateInTransaction(tx *gorm.DB, state string) error {
	now := time.Now()
	i.State = state
	i.UpdatedAt = &now

	err := database.Conn().
		Clauses(clause.Returning{}).
		Save(&i).
		Error

	if err != nil {
		return err
	}

	return nil
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

func FindDomainIntegrationByID(domainType string, domainID, id uuid.UUID) (*Integration, error) {
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

func ListPendingIntegrations() ([]Integration, error) {
	integrations := []Integration{}

	err := database.Conn().
		Where("state = ?", IntegrationStatePending).
		Find(&integrations).
		Error

	if err != nil {
		return nil, err
	}

	return integrations, nil
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
