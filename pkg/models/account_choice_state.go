package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountChoiceState struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TokenHash      string    `gorm:"type:varchar(64);not null"`
	Provider       string    `gorm:"type:varchar(64);not null"`
	ProviderID     string    `gorm:"not null"`
	Redirect       string    `gorm:"not null;default:''"`
	Email          string    `gorm:"not null;default:''"`
	Name           string    `gorm:"not null;default:''"`
	Nickname       string    `gorm:"not null;default:''"`
	AvatarURL      string    `gorm:"not null;default:''"`
	AccessToken    []byte
	RefreshToken   []byte
	TokenExpiresAt *time.Time
	ExpiresAt      time.Time  `gorm:"not null"`
	UsedAt         *time.Time `gorm:"default:null"`
	CreatedAt      time.Time
}

func (AccountChoiceState) TableName() string {
	return "account_choice_states"
}

func CreateAccountChoiceState(tx *gorm.DB, state *AccountChoiceState) error {
	if err := DeleteExpiredAccountChoiceStates(tx, time.Now()); err != nil {
		return err
	}
	return tx.Create(state).Error
}

func FindValidAccountChoiceState(tx *gorm.DB, tokenHash string, now time.Time) (*AccountChoiceState, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var state AccountChoiceState
	err := tx.
		Where("token_hash = ?", tokenHash).
		Where("used_at IS NULL").
		Where("expires_at > ?", now).
		First(&state).
		Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func ClaimAccountChoiceState(tx *gorm.DB, tokenHash string, now time.Time) (*AccountChoiceState, error) {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return nil, nil
	}

	states := []AccountChoiceState{}
	err := tx.Raw(
		`UPDATE account_choice_states
		 SET used_at = ?
		 WHERE token_hash = ?
		   AND used_at IS NULL
		   AND expires_at > ?
		 RETURNING *`,
		now,
		tokenHash,
		now,
	).Scan(&states).Error
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, nil
	}
	return &states[0], nil
}

func DeleteExpiredAccountChoiceStates(tx *gorm.DB, now time.Time) error {
	return tx.Where("expires_at <= ?", now).Delete(&AccountChoiceState{}).Error
}
