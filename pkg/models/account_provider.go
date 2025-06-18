package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

// AccountProvider represents a user's account on a repository hosting provider
type AccountProvider struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID         uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	Provider       string     `json:"provider" gorm:"not null"`
	ProviderID     string     `json:"provider_id" gorm:"not null"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	AvatarURL      string     `json:"avatar_url"`
	AccessToken    string     `json:"-" gorm:"column:access_token"`
	RefreshToken   string     `json:"-" gorm:"column:refresh_token"`
	TokenExpiresAt *time.Time `json:"token_expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (rha *AccountProvider) BeforeCreate(tx *gorm.DB) error {
	if rha.ID == uuid.Nil {
		rha.ID = uuid.New()
	}
	return nil
}

func (rha *AccountProvider) Create() error {
	return database.Conn().Create(rha).Error
}

func (rha *AccountProvider) Update() error {
	return database.Conn().Save(rha).Error
}

func (rha *AccountProvider) Delete() error {
	return database.Conn().Delete(rha).Error
}

func FindAccountProviderByID(id string) (*AccountProvider, error) {
	var account AccountProvider
	accountUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Where("id = ?", accountUUID).First(&account).Error
	return &account, err
}

func FindAccountProviderByProviderID(provider, providerID string) (*AccountProvider, error) {
	var account AccountProvider
	err := database.Conn().Where("provider = ? AND provider_id = ?", provider, providerID).First(&account).Error
	return &account, err
}

func FindAccountProvidersByUserID(userID uuid.UUID) ([]AccountProvider, error) {
	var accounts []AccountProvider
	err := database.Conn().Where("user_id = ?", userID).Find(&accounts).Error
	return accounts, err
}

func FindAccountProviderByUserAndProvider(userID uuid.UUID, provider string) (*AccountProvider, error) {
	var account AccountProvider
	err := database.Conn().Where("user_id = ? AND provider = ?", userID, provider).First(&account).Error
	return &account, err
}

func (rha *AccountProvider) IsTokenExpired() bool {
	if rha.TokenExpiresAt == nil {
		return false
	}
	return time.Now().After(*rha.TokenExpiresAt)
}

func (rha *AccountProvider) NeedsRefresh() bool {
	if rha.TokenExpiresAt == nil {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(*rha.TokenExpiresAt)
}
