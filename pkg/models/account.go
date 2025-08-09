package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type Account struct {
	ID    uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Email string
}

func FindAccount(email string) (*Account, error) {
	var account Account
	err := database.Conn().
		Where("email = ?", email).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func CreateAccount(email string) (*Account, error) {
	account := &Account{Email: email}
	err := database.Conn().Create(account).Error
	if err != nil {
		return nil, err
	}

	return account, nil
}
