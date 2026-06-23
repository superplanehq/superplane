package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
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

func FindInviteLinkByOrganizationID(organizationID string) (*OrganizationInviteLink, error) {
	return FindInviteLinkByOrganizationIDInTransaction(database.Conn(), organizationID)
}

func FindInviteLinkByOrganizationIDInTransaction(tx *gorm.DB, organizationID string) (*OrganizationInviteLink, error) {
	var inviteLink OrganizationInviteLink

	err := tx.
		Where("organization_id = ?", organizationID).
		First(&inviteLink).
		Error

	return &inviteLink, err
}

func FindInviteLinkByToken(token string) (*OrganizationInviteLink, error) {
	return FindInviteLinkByTokenInTransaction(database.Conn(), token)
}

func FindInviteLinkByTokenInTransaction(tx *gorm.DB, token string) (*OrganizationInviteLink, error) {
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

func CreateInviteLink(organizationID uuid.UUID) (*OrganizationInviteLink, error) {
	return CreateInviteLinkInTransaction(database.Conn(), organizationID)
}

// CreateInviteLinkInTransaction inserts an invite link for the organization.
// The insert uses ON CONFLICT DO NOTHING on the organization_id unique key so
// that two concurrent callers racing to seed the invite link for the same
// organization both succeed and end up returning the same row instead of one
// of them failing with a unique-violation error.
func CreateInviteLinkInTransaction(tx *gorm.DB, organizationID uuid.UUID) (*OrganizationInviteLink, error) {
	inviteLink := &OrganizationInviteLink{
		OrganizationID: organizationID,
		Token:          uuid.New(),
		Enabled:        true,
	}

	err := tx.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}},
			DoNothing: true,
		}).
		Create(inviteLink).
		Error
	if err != nil {
		return nil, err
	}

	return FindInviteLinkByOrganizationIDInTransaction(tx, organizationID.String())
}

func SaveInviteLink(inviteLink *OrganizationInviteLink) error {
	return database.Conn().Save(inviteLink).Error
}
