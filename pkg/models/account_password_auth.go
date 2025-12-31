package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type AccountPasswordAuth struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AccountID    uuid.UUID
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func CreateAccountPasswordAuth(accountID uuid.UUID, passwordHash string) (*AccountPasswordAuth, error) {
	return CreateAccountPasswordAuthInTransaction(database.Conn(), accountID, passwordHash)
}

func CreateAccountPasswordAuthInTransaction(tx *gorm.DB, accountID uuid.UUID, passwordHash string) (*AccountPasswordAuth, error) {
	auth := &AccountPasswordAuth{
		AccountID:    accountID,
		PasswordHash: passwordHash,
	}

	err := tx.Create(auth).Error
	if err != nil {
		return nil, err
	}

	return auth, nil
}

func FindAccountPasswordAuthByAccountID(accountID uuid.UUID) (*AccountPasswordAuth, error) {
	return FindAccountPasswordAuthByAccountIDInTransaction(database.Conn(), accountID)
}

func FindAccountPasswordAuthByAccountIDInTransaction(tx *gorm.DB, accountID uuid.UUID) (*AccountPasswordAuth, error) {
	var auth AccountPasswordAuth

	err := tx.
		Where("account_id = ?", accountID).
		First(&auth).
		Error

	if err != nil {
		return nil, err
	}

	return &auth, nil
}

func (a *AccountPasswordAuth) UpdatePasswordHash(passwordHash string) error {
	return a.UpdatePasswordHashInTransaction(database.Conn(), passwordHash)
}

func (a *AccountPasswordAuth) UpdatePasswordHashInTransaction(tx *gorm.DB, passwordHash string) error {
	a.PasswordHash = passwordHash
	a.UpdatedAt = time.Now()
	return tx.Save(a).Error
}

