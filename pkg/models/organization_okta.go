package models

import (
	"crypto/sha256"
	"encoding/base64"
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

func SaveOrganizationOktaConfig(config *OrganizationOktaConfig) error {
	if config.ID == uuid.Nil {
		return database.Conn().Create(config).Error
	}
	return database.Conn().Save(config).Error
}

func HashSCIMToken(token string) string {
	// Placeholder: actual hashing is implemented in pkg/public/okta_saml.go hashSCIMToken.
	// This function exists so that gRPC actions don't depend on public package.
	return hashString(token)
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.StdEncoding.EncodeToString(sum[:])
}
