package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	Email          string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (u *User) Create() error {
	return database.Conn().Create(u).Error
}

func (u *User) Update() error {
	return database.Conn().Save(u).Error
}

func FindUserByID(orgID, id string) (*User, error) {
	var user User

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("id = ?", id).
		First(&user).
		Error

	return &user, err
}

func FindUserByProviderId(providerId, provider string) (*User, error) {
	var user User
	err := database.Conn().
		Joins("JOIN account_providers ON users.id = account_providers.user_id").
		Where("account_providers.provider_id = ? AND account_providers.provider = ?", providerId, provider).
		First(&user).Error
	return &user, err
}

func FindUserByEmail(orgID string, email string) (*User, error) {
	var user User

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("email = ?", email).
		First(&user).
		Error

	return &user, err
}

// FindInactiveUserByEmail finds an inactive user by email (used for pre-invited users)
func FindInactiveUserByEmail(email string) (*User, error) {
	var user User
	err := database.Conn().
		Where("name = ? AND is_active = false", email).
		First(&user).Error
	return &user, err
}

func (u *User) GetAccountProviders() ([]AccountProvider, error) {
	return FindAccountProvidersByUserID(u.ID)
}

func (u *User) GetAccountProvider(provider string) (*AccountProvider, error) {
	return FindAccountProviderByUserAndProvider(u.ID, provider)
}

func (u *User) HasAccountProvider(provider string) bool {
	_, err := u.GetAccountProvider(provider)
	return err == nil
}
