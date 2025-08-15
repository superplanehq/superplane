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
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt
}

func (u *User) Delete() error {
	return database.Conn().Delete(u).Error
}

func (u *User) Restore() error {
	return database.Conn().Unscoped().
		Model(u).
		Update("deleted_at", nil).
		Error
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

// NOTE: this method returns soft deleted users too.
// Make sure you really need to use it this one,
// and not FindActiveUserByID instead.
func FindMaybeDeletedUserByID(orgID, id string) (*User, error) {
	var user User

	err := database.Conn().Unscoped().
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		First(&user).
		Error

	return &user, err
}

func FindActiveUserByID(orgID, id string) (*User, error) {
	var user User

	err := database.Conn().
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		First(&user).
		Error

	return &user, err
}

func FindActiveUserByEmail(orgID, email string) (*User, error) {
	var user User

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("email = ?", email).
		First(&user).
		Error

	return &user, err
}

func FindMaybeDeletedUserByEmail(orgID, email string) (*User, error) {
	var user User

	err := database.Conn().Unscoped().
		Where("organization_id = ?", orgID).
		Where("email = ?", email).
		First(&user).
		Error

	return &user, err
}

func FindOrganizationsForAccount(email string) ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Table("organizations").
		Joins("JOIN users ON organizations.id = users.organization_id").
		Where("users.email = ?", email).
		Where("users.deleted_at IS NULL").
		Find(&organizations).
		Error

	return organizations, err
}
