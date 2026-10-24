package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationInvitationSend records one invitation email that an
// organization sent. Rate limits count these rows.
type OrganizationInvitationSend struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	Email          string
	SentAt         time.Time
}

func RecordOrganizationInvitationSend(tx *gorm.DB, orgID uuid.UUID, email string, sentAt time.Time) error {
	send := OrganizationInvitationSend{OrganizationID: orgID, Email: email, SentAt: sentAt}
	return tx.Create(&send).Error
}

func CountOrganizationInvitationSendsSince(tx *gorm.DB, orgID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	err := tx.Model(&OrganizationInvitationSend{}).Where("organization_id = ? AND sent_at > ?", orgID, since).Count(&count).Error
	return count, err
}

func CountRecipientInvitationSendsSince(tx *gorm.DB, orgID uuid.UUID, email string, since time.Time) (int64, error) {
	var count int64
	err := tx.Model(&OrganizationInvitationSend{}).Where("organization_id = ? AND email = ? AND sent_at > ?", orgID, email, since).Count(&count).Error
	return count, err
}
