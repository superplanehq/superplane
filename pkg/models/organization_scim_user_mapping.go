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

func ListUserIDsWithScimMappingInOrganization(tx *gorm.DB, orgID string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Raw(`
		SELECT m.user_id FROM organization_scim_user_mappings m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = ? AND u.deleted_at IS NULL`,
		orgID).Scan(&ids).Error
	return ids, err
}
