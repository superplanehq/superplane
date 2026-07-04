package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrganizationInviteLink struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `gorm:"uniqueIndex"`
	Token          uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func FindInviteLinkByOrganizationID(tx *gorm.DB, organizationID string) (*OrganizationInviteLink, error) {
	var inviteLink OrganizationInviteLink

	err := tx.
		Where("organization_id = ?", organizationID).
		First(&inviteLink).
		Error

	return &inviteLink, err
}

func FindInviteLinkByToken(tx *gorm.DB, token string) (*OrganizationInviteLink, error) {
	var inviteLink OrganizationInviteLink

	tokenUUID, err := uuid.Parse(token)
	if err != nil {
		return &inviteLink, err
	}

	err = tx.
		Where("token = ?", tokenUUID).
		First(&inviteLink).
		Error

	return &inviteLink, err
}

func CreateInviteLink(tx *gorm.DB, organizationID uuid.UUID) (*OrganizationInviteLink, error) {
	inviteLink := &OrganizationInviteLink{
		OrganizationID: organizationID,
		Token:          uuid.New(),
		Enabled:        true,
	}

	err := tx.Create(inviteLink).Error
	if err != nil {
		return nil, err
	}

	return inviteLink, nil
}

func FindOrCreateInviteLink(tx *gorm.DB, organizationID uuid.UUID) (*OrganizationInviteLink, error) {
	seed := &OrganizationInviteLink{
		OrganizationID: organizationID,
		Token:          uuid.New(),
		Enabled:        true,
	}

	// This avoids the "read then insert" race under concurrent first-time requests.
	// Postgres will serialize conflicts on the unique index, so the losing insert blocks and then no-ops.
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}},
		DoNothing: true,
	}).Create(seed).Error; err != nil {
		return nil, err
	}

	return FindInviteLinkByOrganizationID(tx, organizationID.String())
}

func SaveInviteLink(tx *gorm.DB, inviteLink *OrganizationInviteLink) error {
	return tx.Save(inviteLink).Error
}
