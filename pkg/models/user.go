package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// apiKeyNameUniqueConstraint is the partial unique index enforcing one API key
// name per organization (see db/migrations, users(organization_id, name)
// WHERE type = 'api_key').
const apiKeyNameUniqueConstraint = "unique_api_key_in_organization"

// ErrAPIKeyNameAlreadyExists is returned when an API key name collides with an
// existing key in the same organization.
var ErrAPIKeyNameAlreadyExists = errors.New("API key name already exists")

type User struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID  uuid.UUID
	AccountID       *uuid.UUID
	Email           *string
	Name            string
	Type            string
	Description     *string
	CreatedBy       *uuid.UUID
	TokenHash       string
	APIKeyExpiresAt *time.Time                  `gorm:"column:api_key_expires_at"`
	APIKeyCanvasIDs datatypes.JSONSlice[string] `gorm:"column:api_key_canvas_ids"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt
}

func (u *User) IsAPIKey() bool {
	return u.Type == UserTypeAPIKey
}

func (u *User) IsExpiredAPIKey() bool {
	return u.IsAPIKey() && u.APIKeyExpiresAt != nil && !time.Now().Before(*u.APIKeyExpiresAt)
}

func (u *User) HasAPIKeyCanvasScope() bool {
	return u.IsAPIKey() && len(u.APIKeyCanvasIDs) > 0
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

// ClearTokenHashesForAccountInTransaction wipes the API token hash on every
// user that belongs to the given account. Combined with bumping the account's
// password_changed_at, this ensures that no previously issued credential
// (cookie, scoped JWT, or API token) keeps working after a password change.
func ClearTokenHashesForAccountInTransaction(tx *gorm.DB, accountID uuid.UUID) error {
	return tx.Model(&User{}).
		Where("account_id = ?", accountID).
		Updates(map[string]any{
			"token_hash": "",
			"updated_at": time.Now(),
		}).
		Error
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

func CreateAPIKey(tx *gorm.DB, orgID uuid.UUID, name string, description *string, createdBy uuid.UUID, expiresAt *time.Time, canvasIDs []string) (*User, error) {
	user := &User{
		OrganizationID:  orgID,
		Name:            name,
		Type:            UserTypeAPIKey,
		Description:     description,
		CreatedBy:       &createdBy,
		APIKeyExpiresAt: expiresAt,
		APIKeyCanvasIDs: datatypes.NewJSONSlice(canvasIDs),
	}

	err := tx.Create(user).Error
	if err != nil {
		return nil, mapAPIKeyNameUniqueConstraintError(err)
	}

	return user, nil
}

// UpdateAPIKey persists changes to an existing API key, translating a name
// collision into ErrAPIKeyNameAlreadyExists.
func UpdateAPIKey(tx *gorm.DB, apiKey *User) error {
	err := tx.Save(apiKey).Error
	if err != nil {
		return mapAPIKeyNameUniqueConstraintError(err)
	}

	return nil
}

// mapAPIKeyNameUniqueConstraintError converts a Postgres unique-violation on the
// API key name index into the typed ErrAPIKeyNameAlreadyExists; other errors pass
// through unchanged.
func mapAPIKeyNameUniqueConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == apiKeyNameUniqueConstraint {
		return ErrAPIKeyNameAlreadyExists
	}

	return err
}

func FindAPIKeysByOrganization(db *gorm.DB, orgID string) ([]User, error) {
	var users []User

	err := db.
		Where("organization_id = ?", orgID).
		Where("type = ?", UserTypeAPIKey).
		Find(&users).
		Error

	return users, err
}

func FindUsersByIDsInOrganization(db *gorm.DB, orgID string, ids []string) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var users []User
	err := db.Unscoped().
		Where("organization_id = ?", orgID).
		Where("id IN ?", ids).
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
	return FindMaybeDeletedUserByIDInTransaction(database.Conn(), orgID, id)
}

func FindMaybeDeletedUserByIDInTransaction(tx *gorm.DB, orgID, id string) (*User, error) {
	var user User

	err := tx.Unscoped().
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		First(&user).
		Error

	return &user, err
}

func ListAllActiveUsersInOrganization(orgID string) ([]User, error) {
	var users []User
	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("type = ?", UserTypeHuman).
		Find(&users).
		Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

func ListActiveUsersByOrganization(orgID, search string, limit, offset int) ([]User, int64, error) {
	query := database.Conn().
		Where("organization_id = ?", orgID).
		Where("type = ?", UserTypeHuman)

	if search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	if err := query.Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	var users []User
	if err := query.Order("name ASC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
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
	return FindActiveUserByEmailInTransaction(database.Conn(), orgID, email)
}

func FindActiveUserByEmailInTransaction(tx *gorm.DB, orgID, email string) (*User, error) {
	var user User

	err := tx.
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
	return FindActiveUserByTokenHashInTransaction(database.Conn(), tokenHash)
}

func FindActiveUserByTokenHashInTransaction(tx *gorm.DB, tokenHash string) (*User, error) {
	var user User

	err := tx.
		Where("token_hash = ?", tokenHash).
		First(&user).
		Error

	return &user, err
}

func FindMaybeDeletedUsersByIDs(tx *gorm.DB, ids []uuid.UUID) ([]User, error) {
	if len(ids) == 0 {
		return []User{}, nil
	}

	var users []User
	err := tx.Unscoped().
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

func CountActiveHumanUsersByOrganization(orgID string) (int64, error) {
	return CountActiveHumanUsersByOrganizationInTransaction(database.Conn(), orgID)
}

func CountActiveHumanUsersByOrganizationInTransaction(tx *gorm.DB, orgID string) (int64, error) {
	var count int64
	err := tx.
		Model(&User{}).
		Where("organization_id = ?", orgID).
		Where("type = ?", UserTypeHuman).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func CountOrganizationsByBillingAccount(accountID string) (int64, error) {
	return CountOrganizationsByBillingAccountInTransaction(database.Conn(), accountID)
}

func CountOrganizationsByBillingAccountInTransaction(tx *gorm.DB, accountID string) (int64, error) {
	subquery := tx.
		Table("users").
		Select("DISTINCT ON (organization_id) organization_id, account_id").
		Where("account_id IS NOT NULL").
		Where("type = ?", UserTypeHuman).
		Order("organization_id, created_at ASC")

	var count int64
	err := tx.
		Table("(?) AS first_human_users", subquery).
		Where("account_id = ?", accountID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func FindAnyUserByEmail(email string) (*User, error) {
	var user User

	err := database.Conn().
		Where("email = ?", utils.NormalizeEmail(email)).
		First(&user).
		Error

	return &user, err
}

func FindFirstHumanUserByOrganization(orgID string) (*User, error) {
	return FindFirstHumanUserByOrganizationInTransaction(database.Conn(), orgID)
}

func FindFirstHumanUserByOrganizationInTransaction(tx *gorm.DB, orgID string) (*User, error) {
	var user User

	err := tx.
		Where("organization_id = ?", orgID).
		Where("account_id IS NOT NULL").
		Where("type = ?", UserTypeHuman).
		Order("created_at ASC").
		First(&user).
		Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

type UserAccountProvider struct {
	AccountProvider
	UserID string
}

func FindUserAccountProviders(users []User) ([]UserAccountProvider, error) {
	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.ID.String()
	}

	var accountProviders []UserAccountProvider
	err := database.Conn().
		Table("users").
		Select("users.id as user_id, account_providers.*").
		Joins("inner join accounts on accounts.id = users.account_id").
		Joins("inner join account_providers on account_providers.account_id = accounts.id").
		Where("users.id IN (?)", userIDs).
		Find(&accountProviders).
		Error

	if err != nil {
		return nil, err
	}

	return accountProviders, nil
}
