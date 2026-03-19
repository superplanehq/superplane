package models

import (
	"slices"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Organization struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name              string    `gorm:"uniqueIndex"`
	Description       string
	AllowedProviders  datatypes.JSONSlice[string]
	VersioningEnabled bool
	UsageSyncedAt     *time.Time
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

func (o *Organization) IsProviderAllowed(provider string) bool {
	return slices.Contains(o.AllowedProviders, provider)
}

func ListOrganizationsByIDs(ids []string) ([]Organization, error) {
	var organizations []Organization

	err := database.Conn().
		Where("id IN (?)", ids).
		Order("name ASC").
		Find(&organizations).
		Error

	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func FindOrganizationByID(id string) (*Organization, error) {
	return FindOrganizationByIDInTransaction(database.Conn(), id)
}

func FindOrganizationByIDInTransaction(tx *gorm.DB, id string) (*Organization, error) {
	organization := Organization{}

	err := tx.
		Where("id = ?", id).
		First(&organization).
		Error

	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func FindOrganizationByName(name string) (*Organization, error) {
	organization := Organization{}

	err := database.Conn().
		Where("name = ?", name).
		First(&organization).
		Error

	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func CreateOrganization(name, description string) (*Organization, error) {
	return CreateOrganizationInTransaction(database.Conn(), name, description)
}

func CreateOrganizationInTransaction(tx *gorm.DB, name, description string) (*Organization, error) {
	now := time.Now()
	organization := Organization{
		Name:              name,
		Description:       description,
		AllowedProviders:  datatypes.JSONSlice[string]{ProviderGitHub},
		VersioningEnabled: false,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	err := tx.
		Clauses(clause.Returning{}).
		Create(&organization).
		Error

	if err == nil {
		_, inviteErr := CreateInviteLinkInTransaction(tx, organization.ID)
		if inviteErr != nil {
			return nil, inviteErr
		}

		return &organization, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

func SoftDeleteOrganization(id string) error {
	return SoftDeleteOrganizationInTransaction(database.Conn(), id)
}

func SoftDeleteOrganizationInTransaction(tx *gorm.DB, id string) error {
	return tx.
		Where("id = ?", id).
		Delete(&Organization{}).
		Error
}

func HardDeleteOrganization(id string) error {
	return database.Conn().
		Unscoped().
		Where("id = ?", id).
		Delete(&Organization{}).
		Error
}

func GetActiveOrganizationIDs() ([]string, error) {
	var orgIDs []string
	err := database.Conn().Model(&Organization{}).
		Select("id").
		Where("deleted_at IS NULL").
		Pluck("id", &orgIDs).Error

	if err != nil {
		return nil, err
	}

	return orgIDs, nil
}

func ListOrganizationsPendingUsageSync(limit int) ([]Organization, error) {
	return ListOrganizationsPendingUsageSyncInTransaction(database.Conn(), limit)
}

func ListOrganizationsPendingUsageSyncInTransaction(tx *gorm.DB, limit int) ([]Organization, error) {
	var organizations []Organization

	query := tx.
		Where("deleted_at IS NULL").
		Where("usage_synced_at IS NULL").
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&organizations).Error
	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func MarkOrganizationUsageSynced(orgID string, syncedAt time.Time) error {
	return MarkOrganizationUsageSyncedInTransaction(database.Conn(), orgID, syncedAt)
}

func MarkOrganizationUsageSyncedInTransaction(tx *gorm.DB, orgID string, syncedAt time.Time) error {
	return tx.
		Model(&Organization{}).
		Where("id = ?", orgID).
		Update("usage_synced_at", syncedAt.UTC()).
		Error
}

func MarkOrganizationUsageSyncedIfUnset(orgID string, syncedAt time.Time) error {
	return MarkOrganizationUsageSyncedIfUnsetInTransaction(database.Conn(), orgID, syncedAt)
}

func MarkOrganizationUsageSyncedIfUnsetInTransaction(tx *gorm.DB, orgID string, syncedAt time.Time) error {
	return tx.
		Model(&Organization{}).
		Where("id = ?", orgID).
		Where("usage_synced_at IS NULL").
		Update("usage_synced_at", syncedAt.UTC()).
		Error
}

func IsCanvasVersioningEnabled(organizationID uuid.UUID) (bool, error) {
	return IsCanvasVersioningEnabledInTransaction(database.Conn(), organizationID)
}

func IsCanvasVersioningEnabledInTransaction(tx *gorm.DB, organizationID uuid.UUID) (bool, error) {
	var organization Organization
	err := tx.
		Select("versioning_enabled").
		Where("id = ?", organizationID).
		First(&organization).
		Error
	if err != nil {
		return false, err
	}

	return organization.VersioningEnabled, nil
}
