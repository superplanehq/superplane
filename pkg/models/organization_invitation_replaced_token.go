package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"gorm.io/gorm"
)

type OrganizationInvitationReplacedToken struct {
	TokenHash    string `gorm:"primaryKey" json:"-"`
	InvitationID uuid.UUID
}

func (i *OrganizationInvitation) ArchiveToken(tx *gorm.DB) error {
	if i.TokenHash == nil || *i.TokenHash == "" {
		return nil
	}
	return tx.Create(&OrganizationInvitationReplacedToken{TokenHash: *i.TokenHash, InvitationID: i.ID}).Error
}

func HasReplacedOrganizationInvitationToken(tx *gorm.DB, token string) (bool, error) {
	var count int64
	err := tx.Model(&OrganizationInvitationReplacedToken{}).Where("token_hash = ?", crypto.HashToken(token)).Count(&count).Error
	return count > 0, err
}
