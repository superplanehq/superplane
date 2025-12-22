package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type OrganizationOktaConfig struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `gorm:"uniqueIndex"`

	SamlIssuer      string
	SamlCertificate string
	ScimTokenHash   string
	EnforceSSO      bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

func FindOrganizationOktaConfig(orgID uuid.UUID) (*OrganizationOktaConfig, error) {
	var config OrganizationOktaConfig

	err := database.Conn().
		Where("organization_id = ?", orgID).
		First(&config).
		Error

	if err != nil {
		return nil, err
	}

	return &config, nil
}

