package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	AccountID      uuid.UUID
	Email          string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func CreateUser(orgID, accountID uuid.UUID, email, name string) (*User, error) {
	user := &User{
		OrganizationID: orgID,
		AccountID:      accountID,
		Email:          email,
		Name:           name,
	}

	err := database.Conn().Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

// TODO: check this function usage and remove if possible
func FindUnscopedUserByID(id string) (*User, error) {
	var user User
	userUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Where("id = ?", userUUID).First(&user).Error
	return &user, err
}

func FindUserByID(orgID, id string) (*User, error) {
	var user User

	err := database.Conn().
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		First(&user).
		Error

	return &user, err
}

func FindUserByProviderId(providerId, provider string) (*User, error) {
	var user User

	err := database.Conn().
		Joins("JOIN account_providers ON users.id = account_providers.user_id").
		Where("account_providers.provider_id = ?", providerId).
		Where("account_providers.provider = ?", provider).
		First(&user).Error

	return &user, err
}

func FindUserByEmail(orgID, email string) (*User, error) {
	var user User

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("email = ?", email).
		First(&user).
		Error

	return &user, err
}

func ListUsersInOrganization(orgID uuid.UUID) ([]User, error) {
	var users []User

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Order("name ASC").
		Find(&users).
		Error

	return users, err
}

func FindUserOrganizationsByEmail(email string) ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Table("organizations").
		Joins("JOIN users ON organizations.id = users.organization_id").
		Where("users.email = ?", email).
		Find(&organizations).
		Error

	return organizations, err
}
