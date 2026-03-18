package models

import (
	"fmt"
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
	now := time.Now()
	timestamp := now.Unix()

	var org Organization
	if err := tx.Where("id = ?", id).First(&org).Error; err != nil {
		return err
	}

	newName := fmt.Sprintf("%s (deleted-%d)", org.Name, timestamp)
	return tx.Model(&org).Updates(map[string]any{
		"deleted_at": now,
		"name":       newName,
	}).Error
}

func HardDeleteOrganization(id string) error {
	return HardDeleteOrganizationInTransaction(database.Conn(), id)
}

func HardDeleteOrganizationInTransaction(tx *gorm.DB, id string) error {
	return tx.
		Unscoped().
		Where("id = ?", id).
		Delete(&Organization{}).
		Error
}

func ListDeletedOrganizations() ([]Organization, error) {
	var organizations []Organization
	err := database.Conn().
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&organizations).
		Error

	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func LockOrganization(tx *gorm.DB, id uuid.UUID) (*Organization, error) {
	var org Organization

	err := tx.
		Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("deleted_at IS NOT NULL").
		First(&org).
		Error

	if err != nil {
		return nil, err
	}

	return &org, nil
}

func SoftDeleteOrganizationCanvasesInTransaction(tx *gorm.DB, orgID string) error {
	now := time.Now()
	timestamp := now.Unix()

	var canvases []Canvas
	if err := tx.Where("organization_id = ?", orgID).Find(&canvases).Error; err != nil {
		return err
	}

	for _, c := range canvases {
		newName := fmt.Sprintf("%s (deleted-%d)", c.Name, timestamp)
		if err := tx.Model(&c).Updates(map[string]any{
			"deleted_at": now,
			"name":       newName,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

func SoftDeleteOrganizationIntegrationsInTransaction(tx *gorm.DB, orgID string) error {
	now := time.Now()
	timestamp := now.Unix()

	var integrations []Integration
	if err := tx.Where("organization_id = ?", orgID).Find(&integrations).Error; err != nil {
		return err
	}

	for _, i := range integrations {
		newName := fmt.Sprintf("%s (deleted-%d)", i.InstallationName, timestamp)
		if err := tx.Model(&i).Updates(map[string]any{
			"deleted_at":        now,
			"installation_name": newName,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

func SoftDeleteOrganizationUsersInTransaction(tx *gorm.DB, orgID string) error {
	now := time.Now()
	return tx.
		Model(&User{}).
		Where("organization_id = ?", orgID).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"deleted_at": now,
			"updated_at": now,
			"token_hash": nil,
		}).Error
}

func DeleteOrganizationBlueprintsInTransaction(tx *gorm.DB, orgID string) error {
	return tx.Where("organization_id = ?", orgID).Delete(&Blueprint{}).Error
}

func DeleteOrganizationInvitationsInTransaction(tx *gorm.DB, orgID string) error {
	return tx.Where("organization_id = ?", orgID).Delete(&OrganizationInvitation{}).Error
}

func DeleteOrganizationInviteLinksInTransaction(tx *gorm.DB, orgID string) error {
	return tx.Where("organization_id = ?", orgID).Delete(&OrganizationInviteLink{}).Error
}

func DeleteOrganizationAgentSettingsInTransaction(tx *gorm.DB, orgID string) error {
	return tx.Where("organization_id = ?", orgID).Delete(&OrganizationAgentSettings{}).Error
}

func DeleteOrganizationSecretsInTransaction(tx *gorm.DB, orgID uuid.UUID) error {
	return tx.Where("domain_type = ? AND domain_id = ?", DomainTypeOrganization, orgID).Delete(&Secret{}).Error
}

func DeleteOrganizationIntegrationSecretsInTransaction(tx *gorm.DB, orgID string) error {
	return tx.Where("organization_id = ?", orgID).Delete(&IntegrationSecret{}).Error
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
