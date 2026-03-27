package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationScimUserMapping links a SCIM-provisioned user to optional IdP externalId.
type OrganizationScimUserMapping struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID `gorm:"type:uuid"`
	UserID         uuid.UUID `gorm:"type:uuid"`
	ExternalID     *string   `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (OrganizationScimUserMapping) TableName() string {
	return "organization_scim_user_mappings"
}

func CreateOrganizationScimUserMappingInTransaction(tx *gorm.DB, orgID, userID uuid.UUID, externalID *string) error {
	now := time.Now()
	m := OrganizationScimUserMapping{
		OrganizationID: orgID,
		UserID:         userID,
		ExternalID:     externalID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return tx.Create(&m).Error
}

func FindScimMappingByOrganizationAndUserID(tx *gorm.DB, orgID, userID string) (*OrganizationScimUserMapping, error) {
	var m OrganizationScimUserMapping
	err := tx.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func DeleteOrganizationScimUserMappingInTransaction(tx *gorm.DB, orgID, userID string) error {
	return tx.Where("organization_id = ? AND user_id = ?", orgID, userID).
		Delete(&OrganizationScimUserMapping{}).Error
}

// OktaOrgForEmail is returned by FindOktaOrgsForEmail.
type OktaOrgForEmail struct {
	OrgID   string
	OrgName string
}

// FindOktaOrgsForEmail returns all organizations where an active user with the
// given email exists, has a SCIM mapping, and the org has Okta OIDC enabled.
// Used by the SSO login-page lookup endpoint.
func FindOktaOrgsForEmail(db *gorm.DB, email string) ([]OktaOrgForEmail, error) {
	var results []OktaOrgForEmail
	err := db.Raw(`
		SELECT DISTINCT o.id AS org_id, o.name AS org_name
		FROM organizations o
		INNER JOIN organization_okta_idp idp ON idp.organization_id = o.id AND idp.saml_enabled = true
		INNER JOIN users u ON u.organization_id = o.id AND u.email = ? AND u.deleted_at IS NULL
		INNER JOIN organization_scim_user_mappings scim ON scim.user_id = u.id AND scim.organization_id = o.id
	`, email).Scan(&results).Error
	return results, err
}

// UserWithScimMapping is a join result used by the SCIM GetAll handler.
type UserWithScimMapping struct {
	User
	ExternalID *string `gorm:"column:external_id"`
}

// ListUsersWithScimMappingInOrganization returns all active, non-service-account
// users in the org that have a SCIM mapping, in a single query.
func ListUsersWithScimMappingInOrganization(db *gorm.DB, orgID string) ([]UserWithScimMapping, error) {
	var results []UserWithScimMapping
	err := db.Table("users u").
		Select("u.*, m.external_id").
		Joins("INNER JOIN organization_scim_user_mappings m ON m.user_id = u.id AND m.organization_id = ?", orgID).
		Where("u.organization_id = ? AND u.deleted_at IS NULL AND u.type != ?", orgID, UserTypeServiceAccount).
		Order("u.created_at ASC").
		Scan(&results).Error
	return results, err
}

// ListAllHumanUsersForScimInOrganization returns all active human users in the org,
// including those without a SCIM mapping (externalId will be nil for those).
// Used by SCIM GetAll so Okta's credential test passes even before any users are provisioned.
func ListAllHumanUsersForScimInOrganization(db *gorm.DB, orgID string) ([]UserWithScimMapping, error) {
	var results []UserWithScimMapping
	err := db.Table("users u").
		Select("u.*, m.external_id").
		Joins("LEFT JOIN organization_scim_user_mappings m ON m.user_id = u.id AND m.organization_id = ?", orgID).
		Where("u.organization_id = ? AND u.deleted_at IS NULL AND u.type != ?", orgID, UserTypeServiceAccount).
		Order("u.created_at ASC").
		Scan(&results).Error
	return results, err
}

func ListUserIDsWithScimMappingInOrganization(tx *gorm.DB, orgID string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Raw(`
		SELECT m.user_id FROM organization_scim_user_mappings m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = ? AND u.deleted_at IS NULL`,
		orgID).Scan(&ids).Error
	return ids, err
}
