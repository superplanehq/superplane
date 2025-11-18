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

type ValueDefinitionFrom struct {
	Secret *ValueDefinitionFromSecret `json:"secret,omitempty"`
}

type ValueDefinitionFromSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
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

func (i *Integration) Update() error {
	now := time.Now()
	i.UpdatedAt = &now

	err := database.Conn().Save(i).Error
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return ErrNameAlreadyUsed
	}

	return err
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
	return FindIntegrationByIDInTransaction(database.Conn(), id)
}

func FindIntegrationByIDInTransaction(tx *gorm.DB, id uuid.UUID) (*Integration, error) {
	integration := Integration{}

	err := tx.
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
	Type            string
	IntegrationName string
	DomainType      string
}
