package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"gorm.io/gorm"
)

const (
	InvitationStatePending  = "pending"
	InvitationStateAccepted = "accepted"
	InvitationStateCanceled = "canceled"
	InvitationStateExpired  = "expired"
)

type OrganizationInvitation struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	Email          string
	Role           string
	InvitedBy      uuid.UUID
	TokenHash      *string `json:"-"`
	State          string
	SentAt         time.Time
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	AcceptedBy     *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	InviterName    string `gorm:"->;-:migration"`
}

func (i *OrganizationInvitation) EffectiveState(now time.Time) string {
	if i.State == InvitationStatePending && !i.ExpiresAt.After(now) {
		return InvitationStateExpired
	}
	return i.State
}

func (i *OrganizationInvitation) Save(tx *gorm.DB) error {
	return tx.Omit("InviterName").Save(i).Error
}

func FindOrganizationInvitation(tx *gorm.DB, orgID, id uuid.UUID) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation
	err := invitationQuery(tx, orgID).Where("organization_invitations.id = ?", id).First(&invitation).Error
	return &invitation, err
}

func FindOrganizationInvitationByToken(tx *gorm.DB, token string) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation
	err := tx.Where("token_hash = ?", crypto.HashToken(token)).First(&invitation).Error
	return &invitation, err
}

func ListOrganizationInvitations(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationInvitation, error) {
	var invitations []OrganizationInvitation
	err := invitationQuery(tx, orgID).Order("organization_invitations.sent_at DESC, organization_invitations.id DESC").Find(&invitations).Error
	return invitations, err
}

func HasPendingOrganizationInvitation(tx *gorm.DB, orgID uuid.UUID, email string) (bool, error) {
	var count int64
	err := tx.Model(&OrganizationInvitation{}).Where("organization_id = ? AND lower(email) = lower(?) AND state = ?", orgID, email, InvitationStatePending).Count(&count).Error
	return count > 0, err
}

func CountPendingOrganizationInvitations(tx *gorm.DB, orgID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&OrganizationInvitation{}).Where("organization_id = ? AND state = ? AND expires_at > ?", orgID, InvitationStatePending, time.Now()).Count(&count).Error
	return count, err
}

func ExpireOrganizationInvitations(tx *gorm.DB, orgID uuid.UUID, now time.Time) error {
	return tx.Model(&OrganizationInvitation{}).Where("organization_id = ? AND state = ? AND expires_at <= ?", orgID, InvitationStatePending, now).Updates(map[string]any{"state": InvitationStateExpired, "token_hash": nil}).Error
}

func invitationQuery(tx *gorm.DB, orgID uuid.UUID) *gorm.DB {
	return tx.Model(&OrganizationInvitation{}).Select("organization_invitations.*, users.name AS inviter_name").Joins("LEFT JOIN users ON users.id = organization_invitations.invited_by AND users.organization_id = organization_invitations.organization_id").Where("organization_invitations.organization_id = ?", orgID)
}
