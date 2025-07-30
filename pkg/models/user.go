package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AccountProviders []AccountProvider `json:"account_providers,omitempty" gorm:"foreignKey:UserID"`
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

func FindUserByID(id string) (*User, error) {
	var user User
	userUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Where("id = ?", userUUID).First(&user).Error
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

func FindUserByEmail(email string) (*User, error) {
	var user User
	err := database.Conn().
		Joins("JOIN account_providers ON users.id = account_providers.user_id").
		Where("account_providers.email = ? AND account_providers.email != '' AND account_providers.email IS NOT NULL", email).
		First(&user).Error
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
