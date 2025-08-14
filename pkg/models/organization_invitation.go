package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
)

type OrganizationInvitation struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	Email          string
	InvitedBy      uuid.UUID
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func FindPendingInvitation(email, organizationID string) (*OrganizationInvitation, error) {
	return FindPendingInvitationInTransaction(database.Conn(), email, organizationID)
}

func FindPendingInvitationInTransaction(tx *gorm.DB, email, organizationID string) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation

	err := tx.
		Where("email = ?", email).
		Where("organization_id = ?", organizationID).
		Where("status = ?", InvitationStatusPending).
		First(&invitation).
		Error

	return &invitation, err
}

func ListPendingInvitations(organizationID string) ([]OrganizationInvitation, error) {
	var invitations []OrganizationInvitation

	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Where("status = ?", InvitationStatusPending).
		Order("created_at DESC").
		Find(&invitations).
		Error

	return invitations, err
}

func CreateInvitation(organizationID, invitedBy uuid.UUID, email, status string) (*OrganizationInvitation, error) {
	return CreateInvitationInTransaction(database.Conn(), organizationID, invitedBy, email, status)
}

func CreateInvitationInTransaction(tx *gorm.DB, organizationID, invitedBy uuid.UUID, email, status string) (*OrganizationInvitation, error) {
	_, err := FindPendingInvitationInTransaction(tx, email, organizationID.String())
	if err == nil {
		return nil, fmt.Errorf("invitation already exists for %s", email)
	}

	invitation := &OrganizationInvitation{
		OrganizationID: organizationID,
		Email:          email,
		InvitedBy:      invitedBy,
		Status:         status,
	}

	err = tx.Create(invitation).Error
	if err != nil {
		return nil, err
	}

	return invitation, err
}
