package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

const (
	InvitationStatePending  = "pending"
	InvitationStateAccepted = "accepted"
)

type OrganizationInvitation struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	Email          string
	InvitedBy      uuid.UUID
	State          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func FindPendingInvitation(email, organizationID string) (*OrganizationInvitation, error) {
	return FindPendingInvitationInTransaction(database.Conn(), email, organizationID)
}

func FindPendingInvitationInTransaction(tx *gorm.DB, email, organizationID string) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation

	err := tx.
		Where("email = ?", utils.NormalizeEmail(email)).
		Where("organization_id = ?", organizationID).
		Where("state = ?", InvitationStatePending).
		First(&invitation).
		Error

	return &invitation, err
}

func FindInvitationByID(invitationID string) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation

	err := database.Conn().
		Where("id = ?", invitationID).
		First(&invitation).
		Error

	return &invitation, err
}

func FindInvitationByIDWithState(invitationID string, state string) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation

	err := database.Conn().
		Where("id = ?", invitationID).
		Where("state = ?", state).
		First(&invitation).
		Error

	return &invitation, err
}

func ListInvitationsInState(organizationID string, state string) ([]OrganizationInvitation, error) {
	var invitations []OrganizationInvitation

	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Where("state = ?", state).
		Order("created_at DESC").
		Find(&invitations).
		Error

	return invitations, err
}

func CreateInvitation(organizationID, invitedBy uuid.UUID, email, state string) (*OrganizationInvitation, error) {
	return CreateInvitationInTransaction(database.Conn(), organizationID, invitedBy, email, state)
}

func CreateInvitationInTransaction(tx *gorm.DB, organizationID, invitedBy uuid.UUID, email, state string) (*OrganizationInvitation, error) {
	normalizedEmail := utils.NormalizeEmail(email)
	_, err := FindPendingInvitationInTransaction(tx, normalizedEmail, organizationID.String())
	if err == nil {
		return nil, fmt.Errorf("invitation already exists for %s", normalizedEmail)
	}

	invitation := &OrganizationInvitation{
		OrganizationID: organizationID,
		Email:          normalizedEmail,
		InvitedBy:      invitedBy,
		State:          state,
	}

	err = tx.Create(invitation).Error
	if err != nil {
		return nil, err
	}

	return invitation, err
}

func SaveInvitation(invitation *OrganizationInvitation) error {
	return database.Conn().Save(invitation).Error
}

func (i *OrganizationInvitation) Delete() error {
	return database.Conn().Delete(i).Error
}
