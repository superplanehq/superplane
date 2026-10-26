package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationInviteLink is a shared invite link from older releases.
// SuperPlane no longer creates or manages these links. Stored links remain
// so that existing members can still open their organization through them.
type OrganizationInviteLink struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `gorm:"uniqueIndex"`
	Token          uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
