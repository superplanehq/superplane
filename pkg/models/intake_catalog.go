package models

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	IntakeCategoryIssueTracking      = "issue_tracking"
	IntakeCategoryErrorTracking      = "error_tracking"
	IntakeCategoryIncidentManagement = "incident_management"
	IntakeCategorySecurityAlerts     = "security_alerts"
	IntakeCategoryRepositoryProvider = "repository_provider"

	IntakeStatusPlanned    = "planned"
	IntakeStatusAlpha      = "alpha"
	IntakeStatusBeta       = "beta"
	IntakeStatusGA         = "ga"
	IntakeStatusDeprecated = "deprecated"

	intakeCatalogKeyMaxLength  = 64
	intakeCatalogNameMaxLength = 255
	postgresUniqueViolation    = "23505"
)

var (
	ErrIntakeCatalogEntryNotFound      = errors.New("intake catalog entry not found")
	ErrIntakeCatalogEntryExists        = errors.New("an intake with this key already exists")
	ErrIntakeCatalogKeyInvalid         = errors.New("key must use lowercase letters, numbers, and dashes")
	ErrIntakeCatalogNameRequired       = errors.New("name is required")
	ErrIntakeCatalogCategoryInvalid    = errors.New("category is not valid")
	ErrIntakeCatalogStatusInvalid      = errors.New("status is not valid")
	ErrIntakeCatalogNotImplemented     = errors.New("the intake is not implemented, so its status stays planned")
	ErrIntakeCatalogInternalForAll     = errors.New("an internal intake cannot be available to all companies")
	ErrIntakeCatalogDeleteImplemented  = errors.New("only planned intakes that are not implemented can be deleted")
	ErrIntakeCatalogAccessNotSupported = errors.New("this intake is not implemented, so companies cannot get access")
)

var intakeCatalogKeyPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var intakeCategories = []string{
	IntakeCategoryIssueTracking,
	IntakeCategoryErrorTracking,
	IntakeCategoryIncidentManagement,
	IntakeCategorySecurityAlerts,
	IntakeCategoryRepositoryProvider,
}

var intakeStatuses = []string{
	IntakeStatusPlanned,
	IntakeStatusAlpha,
	IntakeStatusBeta,
	IntakeStatusGA,
	IntakeStatusDeprecated,
}

// implementedRepositoryProviders lists the VCS providers that factory
// onboarding can connect. Keep it in sync with SelectFactoryVCSProviderRepository.
var implementedRepositoryProviders = []string{ProviderGitHub}

// IntakeCatalogEntry records the maturity of one intake source or repository
// provider and which organizations can use it. The code decides whether the
// intake exists; the entry decides who gets it.
type IntakeCatalogEntry struct {
	Key           string `gorm:"primaryKey"`
	Name          string
	Category      string
	Status        string
	StatusNote    string
	EnabledForAll bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UpdatedBy     *uuid.UUID

	UpdatedByName string `gorm:"->;-:migration"`
}

type IntakeCatalogOrganization struct {
	EntryKey       string    `gorm:"primaryKey"`
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	CreatedAt      time.Time
}

// IntakeCatalogOrganizationAccess is an organization that has access to an
// entry, with the organization name for display.
type IntakeCatalogOrganizationAccess struct {
	EntryKey         string
	OrganizationID   uuid.UUID
	OrganizationName string
	CreatedAt        time.Time
}

// IntakeCatalogPatch holds the fields an admin can change. Nil fields stay
// as they are.
type IntakeCatalogPatch struct {
	Name          *string
	Category      *string
	Status        *string
	StatusNote    *string
	EnabledForAll *bool
}

// IntakeAvailability is one catalog entry as seen by one organization.
type IntakeAvailability struct {
	Entry     IntakeCatalogEntry
	Available bool
}

func NewIntakeCatalogEntry(key, name, category, statusNote string) (*IntakeCatalogEntry, error) {
	entry := &IntakeCatalogEntry{
		Key:        strings.TrimSpace(key),
		Name:       strings.TrimSpace(name),
		Category:   strings.TrimSpace(category),
		Status:     IntakeStatusPlanned,
		StatusNote: strings.TrimSpace(statusNote),
	}
	if err := entry.validate(); err != nil {
		return nil, err
	}
	return entry, nil
}

func ValidIntakeCategory(category string) bool {
	return slices.Contains(intakeCategories, category)
}

func ValidIntakeStatus(status string) bool {
	return slices.Contains(intakeStatuses, status)
}

// IntakeImplemented reports whether SuperPlane has code for the intake or
// repository provider with this key.
func IntakeImplemented(key string) bool {
	return ValidFactoryIntakeSource(key) || slices.Contains(implementedRepositoryProviders, key)
}

func (IntakeCatalogEntry) TableName() string {
	return "intake_catalog_entries"
}

func (IntakeCatalogOrganization) TableName() string {
	return "intake_catalog_organizations"
}

func (e *IntakeCatalogEntry) Implemented() bool {
	return IntakeImplemented(e.Key)
}

// AvailableTo applies the availability rule for an organization.
// organizationAdded tells whether an admin added the organization to the entry.
func (e *IntakeCatalogEntry) AvailableTo(organizationAdded bool) bool {
	if !e.Implemented() {
		return false
	}

	switch e.Status {
	case IntakeStatusGA:
		return true
	case IntakeStatusBeta:
		return e.EnabledForAll || organizationAdded
	case IntakeStatusAlpha:
		return organizationAdded
	default:
		return false
	}
}

// AcceptsOrganizations reports whether adding an organization can change
// what the organization sees.
func (e *IntakeCatalogEntry) AcceptsOrganizations() bool {
	return e.Implemented()
}

func (e *IntakeCatalogEntry) Deletable() bool {
	return !e.Implemented() && e.Status == IntakeStatusPlanned
}

func (e *IntakeCatalogEntry) applyPatch(patch IntakeCatalogPatch) error {
	if patch.Name != nil {
		e.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Category != nil {
		e.Category = strings.TrimSpace(*patch.Category)
	}
	if patch.StatusNote != nil {
		e.StatusNote = strings.TrimSpace(*patch.StatusNote)
	}
	if patch.Status != nil {
		e.Status = strings.TrimSpace(*patch.Status)
	}

	if patch.EnabledForAll != nil {
		if *patch.EnabledForAll && e.Status == IntakeStatusAlpha {
			return ErrIntakeCatalogInternalForAll
		}
		e.EnabledForAll = *patch.EnabledForAll
	}

	// Moving to Internal closes an open beta, because Internal is only for
	// the companies an admin adds.
	if e.Status == IntakeStatusAlpha {
		e.EnabledForAll = false
	}

	return e.validate()
}

func (e *IntakeCatalogEntry) validate() error {
	if len(e.Key) > intakeCatalogKeyMaxLength || !intakeCatalogKeyPattern.MatchString(e.Key) {
		return ErrIntakeCatalogKeyInvalid
	}
	if e.Name == "" || len(e.Name) > intakeCatalogNameMaxLength {
		return ErrIntakeCatalogNameRequired
	}
	if !ValidIntakeCategory(e.Category) {
		return ErrIntakeCatalogCategoryInvalid
	}
	if !ValidIntakeStatus(e.Status) {
		return ErrIntakeCatalogStatusInvalid
	}
	if e.Status != IntakeStatusPlanned && !e.Implemented() {
		return ErrIntakeCatalogNotImplemented
	}
	if e.Status == IntakeStatusAlpha && e.EnabledForAll {
		return ErrIntakeCatalogInternalForAll
	}
	return nil
}

func CreateIntakeCatalogEntry(tx *gorm.DB, entry *IntakeCatalogEntry, accountID *uuid.UUID) error {
	if err := entry.validate(); err != nil {
		return err
	}

	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	entry.UpdatedBy = accountID

	err := tx.Omit("UpdatedByName").Create(entry).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == postgresUniqueViolation {
		return ErrIntakeCatalogEntryExists
	}
	return err
}

func FindIntakeCatalogEntry(tx *gorm.DB, key string) (*IntakeCatalogEntry, error) {
	var entry IntakeCatalogEntry
	err := intakeCatalogEntriesWithEditor(tx).
		Where("intake_catalog_entries.key = ?", key).
		First(&entry).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrIntakeCatalogEntryNotFound
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func ListIntakeCatalogEntries(tx *gorm.DB) ([]IntakeCatalogEntry, error) {
	var entries []IntakeCatalogEntry
	err := intakeCatalogEntriesWithEditor(tx).
		Order("intake_catalog_entries.category ASC").
		Order("intake_catalog_entries.name ASC").
		Find(&entries).
		Error
	return entries, err
}

func (e *IntakeCatalogEntry) Update(tx *gorm.DB, patch IntakeCatalogPatch, accountID *uuid.UUID) error {
	updated := *e
	if err := updated.applyPatch(patch); err != nil {
		return err
	}

	updated.UpdatedAt = time.Now()
	updated.UpdatedBy = accountID
	err := tx.Model(&IntakeCatalogEntry{}).
		Where("key = ?", e.Key).
		Updates(map[string]any{
			"name":            updated.Name,
			"category":        updated.Category,
			"status":          updated.Status,
			"status_note":     updated.StatusNote,
			"enabled_for_all": updated.EnabledForAll,
			"updated_at":      updated.UpdatedAt,
			"updated_by":      updated.UpdatedBy,
		}).
		Error
	if err != nil {
		return err
	}

	*e = updated
	return nil
}

func (e *IntakeCatalogEntry) Delete(tx *gorm.DB) error {
	if !e.Deletable() {
		return ErrIntakeCatalogDeleteImplemented
	}
	return tx.Where("key = ?", e.Key).Delete(&IntakeCatalogEntry{}).Error
}

func (e *IntakeCatalogEntry) AddOrganization(tx *gorm.DB, organizationID uuid.UUID) error {
	if !e.AcceptsOrganizations() {
		return ErrIntakeCatalogAccessNotSupported
	}
	access := IntakeCatalogOrganization{
		EntryKey:       e.Key,
		OrganizationID: organizationID,
		CreatedAt:      time.Now(),
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&access).Error
}

func (e *IntakeCatalogEntry) RemoveOrganization(tx *gorm.DB, organizationID uuid.UUID) error {
	return tx.
		Where("entry_key = ? AND organization_id = ?", e.Key, organizationID).
		Delete(&IntakeCatalogOrganization{}).
		Error
}

func (e *IntakeCatalogEntry) HasOrganization(tx *gorm.DB, organizationID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&IntakeCatalogOrganization{}).
		Where("entry_key = ? AND organization_id = ?", e.Key, organizationID).
		Count(&count).
		Error
	return count > 0, err
}

// ListIntakeCatalogOrganizations returns every organization access row for
// active organizations, oldest first.
func ListIntakeCatalogOrganizations(tx *gorm.DB) ([]IntakeCatalogOrganizationAccess, error) {
	var rows []IntakeCatalogOrganizationAccess
	err := tx.Table("intake_catalog_organizations").
		Select(
			"intake_catalog_organizations.entry_key",
			"intake_catalog_organizations.organization_id",
			"organizations.name AS organization_name",
			"intake_catalog_organizations.created_at",
		).
		Joins("JOIN organizations ON organizations.id = intake_catalog_organizations.organization_id AND organizations.deleted_at IS NULL").
		Order("intake_catalog_organizations.created_at ASC").
		Scan(&rows).
		Error
	return rows, err
}

// ListIntakeCatalogForOrganization returns every catalog entry with whether
// the organization can create it.
func ListIntakeCatalogForOrganization(tx *gorm.DB, organizationID uuid.UUID) ([]IntakeAvailability, error) {
	entries, err := ListIntakeCatalogEntries(tx)
	if err != nil {
		return nil, err
	}

	var addedKeys []string
	err = tx.Model(&IntakeCatalogOrganization{}).
		Where("organization_id = ?", organizationID).
		Pluck("entry_key", &addedKeys).
		Error
	if err != nil {
		return nil, err
	}

	result := make([]IntakeAvailability, 0, len(entries))
	for _, entry := range entries {
		result = append(result, IntakeAvailability{
			Entry:     entry,
			Available: entry.AvailableTo(slices.Contains(addedKeys, entry.Key)),
		})
	}
	return result, nil
}

// IsIntakeAvailableForOrganization reports whether the organization can
// create a new intake (or connect a repository provider) with this key.
// A key with no catalog entry is not available.
func IsIntakeAvailableForOrganization(tx *gorm.DB, key string, organizationID uuid.UUID) (bool, error) {
	entry, err := FindIntakeCatalogEntry(tx, key)
	if errors.Is(err, ErrIntakeCatalogEntryNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	added, err := entry.HasOrganization(tx, organizationID)
	if err != nil {
		return false, err
	}
	return entry.AvailableTo(added), nil
}

func intakeCatalogEntriesWithEditor(tx *gorm.DB) *gorm.DB {
	return tx.Model(&IntakeCatalogEntry{}).
		Select("intake_catalog_entries.*, COALESCE(accounts.name, '') AS updated_by_name").
		Joins("LEFT JOIN accounts ON accounts.id = intake_catalog_entries.updated_by")
}
