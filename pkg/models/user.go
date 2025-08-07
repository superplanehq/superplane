package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;not null;index"`
	Email          string    `json:"email" gorm:"index:idx_users_email_org,unique"`
	Name           string    `json:"name"`
	IsActive       bool      `json:"is_active" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u *User) Create() error {
	return database.Conn().Create(u).Error
}

func (u *User) Update() error {
	return database.Conn().Save(u).Error
}

// TODO: check this function usage and remove if possible
func FindUserByIDOnly(id string) (*User, error) {
	var user User
	userUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Where("id = ?", userUUID).First(&user).Error
	return &user, err
}

func FindUserByID(id, organizationID uuid.UUID) (*User, error) {
	var user User

	err := database.Conn().
		Where("id = ?", id).
		Where("organization_id = ?", organizationID).
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

func FindUserByEmail(email string, organizationID uuid.UUID) (*User, error) {
	var user User

	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Where("email = ?", email).
		First(&user).
		Error

	return &user, err
}

func FindInactiveUserByEmail(email string, organizationID uuid.UUID) (*User, error) {
	var user User

	err := database.Conn().
		Where("email = ?", email).
		Where("organization_id = ?", organizationID).
		Where("is_active = false").
		First(&user).
		Error

	return &user, err
}

func ListUsersInOrganization(organizationID uuid.UUID) ([]User, error) {
	var users []User

	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Order("name ASC").
		Find(&users).
		Error

	return users, err
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

func FindUserOrganizationsByEmail(email string) ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Table("organizations").
		Joins("JOIN users ON organizations.id = users.organization_id").
		Where("users.email = ? AND users.is_active = true", email).
		Find(&organizations).
		Error

	return organizations, err
}
