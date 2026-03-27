package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

// OrganizationOktaIDP holds per-organization Okta SAML + SCIM configuration (one row per org).
type OrganizationOktaIDP struct {
	ID                     uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	OrganizationID         uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	SamlIdpSSOURL          string    `gorm:"column:saml_idp_sso_url;not null"`
	SamlIdpIssuer          string    `gorm:"column:saml_idp_issuer;not null"`
	SamlIdpCertificatePEM  string    `gorm:"column:saml_idp_certificate_pem;not null"`
	SamlEnabled            bool      `gorm:"column:saml_enabled;not null;default:false"`
	ScimBearerTokenHash    *string   `gorm:"column:scim_bearer_token_hash"`
	ScimEnabled            bool      `gorm:"column:scim_enabled;not null;default:false"`
	CreatedAt              time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt              time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (OrganizationOktaIDP) TableName() string {
	return "organization_okta_idp"
}

func FindOrganizationOktaIDPByOrganizationID(orgID string) (*OrganizationOktaIDP, error) {
	return FindOrganizationOktaIDPByOrganizationIDInTransaction(database.Conn(), orgID)
}

func FindOrganizationOktaIDPByOrganizationIDInTransaction(tx *gorm.DB, orgID string) (*OrganizationOktaIDP, error) {
	var row OrganizationOktaIDP
	err := tx.Where("organization_id = ?", orgID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateOrganizationOktaIDPInTransaction(tx *gorm.DB, row *OrganizationOktaIDP) error {
	now := time.Now()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	return tx.Create(row).Error
}

func SaveOrganizationOktaIDPInTransaction(tx *gorm.DB, row *OrganizationOktaIDP) error {
	row.UpdatedAt = time.Now()
	return tx.Save(row).Error
}
