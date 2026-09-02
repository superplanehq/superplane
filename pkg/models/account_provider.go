package models

import (
	"time"

	"github.com/google/uuid"
)

type AccountProvider struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AccountID      uuid.UUID
	Provider       string
	ProviderID     string
	Username       string
	Email          string
	Name           string
	AvatarURL      string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
