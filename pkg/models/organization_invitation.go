package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
)

type OrganizationInvitation struct {
	ID             uuid.UUID        `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID        `json:"organization_id" gorm:"type:uuid;not null;index:idx_org_email,unique"`
	Email          string           `json:"email" gorm:"not null;index:idx_org_email,unique"`
	InvitedBy      uuid.UUID        `json:"invited_by" gorm:"type:uuid;not null"`
	Status         InvitationStatus `json:"status" gorm:"default:'pending'"`
	ExpiresAt      time.Time        `json:"expires_at" gorm:"not null"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`

	Organization *Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	Inviter      *User         `json:"inviter,omitempty" gorm:"foreignKey:InvitedBy"`
}

func (oi *OrganizationInvitation) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	if oi.ExpiresAt.IsZero() {
		oi.ExpiresAt = time.Now().Add(7 * 24 * time.Hour) // Default 7 days
	}
	return nil
}

func (oi *OrganizationInvitation) Create() error {
	return database.Conn().Create(oi).Error
}

func (oi *OrganizationInvitation) Update() error {
	return database.Conn().Save(oi).Error
}

func FindInvitationByID(id uuid.UUID) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation
	err := database.Conn().Where("id = ?", id).First(&invitation).Error
	return &invitation, err
}

func FindPendingInvitation(email string, organizationID uuid.UUID) (*OrganizationInvitation, error) {
	var invitation OrganizationInvitation

	err := database.Conn().
		Where("email = ?", email).
		Where("organization_id = ?", organizationID).
		Where("status = ?", InvitationStatusPending).
		Where("expires_at > ?", time.Now()).
		First(&invitation).
		Error

	return &invitation, err
}

func ListPendingInvitationsForOrganization(organizationID uuid.UUID) ([]OrganizationInvitation, error) {
	var invitations []OrganizationInvitation
	err := database.Conn().
		Where("organization_id = ? AND status = ? AND expires_at > ?",
			organizationID, InvitationStatusPending, time.Now()).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// IsExpired checks if the invitation has expired
func (oi *OrganizationInvitation) IsExpired() bool {
	return time.Now().After(oi.ExpiresAt)
}

// Accept marks the invitation as accepted
func (oi *OrganizationInvitation) Accept() error {
	oi.Status = InvitationStatusAccepted
	return oi.Update()
}

// Expire marks the invitation as expired
func (oi *OrganizationInvitation) Expire() error {
	oi.Status = InvitationStatusExpired
	return oi.Update()
}

func CreateInvitation(organizationID uuid.UUID, email string, invitedBy uuid.UUID) (*OrganizationInvitation, error) {
	existing, err := FindPendingInvitation(email, organizationID)
	if err == nil {
		return existing, ErrInvitationAlreadyExists
	}

	invitation := &OrganizationInvitation{
		OrganizationID: organizationID,
		Email:          email,
		InvitedBy:      invitedBy,
		Status:         InvitationStatusPending,
	}

	err = invitation.Create()
	return invitation, err
}
