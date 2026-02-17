package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID uuid.UUID
	AccountID      *uuid.UUID
	Email          *string
	Name           string
	Type           string
	Description    *string
	CreatedBy      *uuid.UUID
	TokenHash      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt
}

func (u *User) IsServiceAccount() bool {
	return u.Type == UserTypeServiceAccount
}

func (u *User) GetEmail() string {
	if u.Email != nil {
		return *u.Email
	}
	return ""
}

func (u *User) Delete() error {
	now := time.Now()
	return database.Conn().Unscoped().
		Model(u).
		Update("deleted_at", now).
		Update("updated_at", now).
		Update("token_hash", nil).
		Error
}

func (u *User) Restore() error {
	return u.RestoreInTransaction(database.Conn())
}

func (u *User) RestoreInTransaction(tx *gorm.DB) error {
	return tx.Unscoped().
		Model(u).
		Update("deleted_at", nil).
		Error
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
	normalizedEmail := utils.NormalizeEmail(email)
	user := &User{
		OrganizationID: orgID,
		AccountID:      &accountID,
		Email:          &normalizedEmail,
		Name:           name,
		Type:           UserTypeHuman,
	}

	err := tx.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func CreateServiceAccount(tx *gorm.DB, orgID uuid.UUID, name string, description *string, createdBy uuid.UUID) (*User, error) {
	user := &User{
		OrganizationID: orgID,
		Name:           name,
		Type:           UserTypeServiceAccount,
		Description:    description,
		CreatedBy:      &createdBy,
	}

	err := tx.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func FindServiceAccountsByOrganization(orgID string) ([]User, error) {
	return FindServiceAccountsByOrganizationInTransaction(database.Conn(), orgID)
}

func FindServiceAccountsByOrganizationInTransaction(tx *gorm.DB, orgID string) ([]User, error) {
	var users []User

	err := tx.
		Where("organization_id = ?", orgID).
		Where("type = ?", UserTypeServiceAccount).
		Find(&users).
		Error

	return users, err
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

func FindUsersByIDs(ids []string) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var users []User
	err := database.Conn().
		Where("id IN ?", ids).
		Find(&users).Error

	return users, err
}

func FindHumanUsersByIDs(ids []string) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var users []User
	err := database.Conn().
		Where("id IN ?", ids).
		Where("type = ?", UserTypeHuman).
		Find(&users).Error

	return users, err
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

func ListActiveUsersByID(orgID string, ids []string) ([]User, error) {
	return ListActiveUsersByIDInTransaction(database.Conn(), orgID, ids)
}

func ListActiveUsersByIDInTransaction(tx *gorm.DB, orgID string, ids []string) ([]User, error) {
	var users []User

	err := tx.
		Where("id IN ?", ids).
		Where("organization_id = ?", orgID).
		Find(&users).
		Error

	return users, err
}

func FindActiveUserByID(orgID, id string) (*User, error) {
	return FindActiveUserByIDInTransaction(database.Conn(), orgID, id)
}

func FindActiveUserByIDInTransaction(tx *gorm.DB, orgID, id string) (*User, error) {
	var user User

	err := tx.
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
		Where("email = ?", utils.NormalizeEmail(email)).
		First(&user).
		Error

	return &user, err
}

func FindMaybeDeletedUserByEmail(orgID, email string) (*User, error) {
	var user User

	err := database.Conn().Unscoped().
		Where("organization_id = ?", orgID).
		Where("email = ?", utils.NormalizeEmail(email)).
		First(&user).
		Error

	return &user, err
}

func FindMaybeDeletedUserByEmailInTransaction(tx *gorm.DB, orgID, email string) (*User, error) {
	var user User

	err := tx.Unscoped().
		Where("organization_id = ?", orgID).
		Where("email = ?", utils.NormalizeEmail(email)).
		First(&user).
		Error

	return &user, err
}

func FindActiveUserByTokenHash(tokenHash string) (*User, error) {
	var user User

	err := database.Conn().
		Where("token_hash = ?", tokenHash).
		First(&user).
		Error

	return &user, err
}

func FindMaybeDeletedUsersByIDs(ids []uuid.UUID) ([]User, error) {
	if len(ids) == 0 {
		return []User{}, nil
	}

	var users []User
	err := database.Conn().Unscoped().
		Where("id IN ?", ids).
		Find(&users).
		Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func FindOrganizationsForAccount(email string) ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Table("organizations").
		Joins("JOIN users ON organizations.id = users.organization_id").
		Where("users.email = ?", utils.NormalizeEmail(email)).
		Where("users.deleted_at IS NULL").
		Find(&organizations).
		Error

	return organizations, err
}

func CountActiveUsersByOrganizationIDs(orgIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64)
	if len(orgIDs) == 0 {
		return counts, nil
	}

	type row struct {
		OrganizationID string
		Count          int64
	}

	var rows []row
	err := database.Conn().
		Table("users").
		Select("organization_id, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Where("organization_id IN ?", orgIDs).
		Group("organization_id").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		counts[r.OrganizationID] = r.Count
	}

	return counts, nil
}

func FindAnyUserByEmail(email string) (*User, error) {
	var user User

	err := database.Conn().
		Where("email = ?", utils.NormalizeEmail(email)).
		First(&user).
		Error

	return &user, err
}
