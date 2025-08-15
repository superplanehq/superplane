package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	AccountID      uuid.UUID
	Email          string
	Name           string
	TokenHash      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (u *User) UpdateTokenHash(tokenHash string) error {
	u.UpdatedAt = time.Now()
	u.TokenHash = tokenHash
	return database.Conn().Save(u).Error
}

func CreateUser(orgID, accountID uuid.UUID, email, name string) (*User, error) {
	return CreateUserInTransaction(database.Conn(), orgID, accountID, email, name)
}

func CreateUserInTransaction(tx *gorm.DB, orgID, accountID uuid.UUID, email, name string) (*User, error) {
	user := &User{
		OrganizationID: orgID,
		AccountID:      accountID,
		Email:          email,
		Name:           name,
	}

	err := tx.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

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

func FindUserByEmail(orgID, email string) (*User, error) {
	var user User

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("email = ?", email).
		First(&user).
		Error

	return &user, err
}

func FindUserByTokenHash(tokenHash string) (*User, error) {
	var user User

	err := database.Conn().
		Where("token_hash = ?", tokenHash).
		First(&user).
		Error

	return &user, err
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
