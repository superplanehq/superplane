package models

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const OrganizationSortRemainingCreditCents = "remaining_credit_cents"

type Organization struct {
	ID uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	// Name is a display label and is not required to be unique: two
	// organizations may share a name (for example, onboarding keeps an
	// organization's name equal to its GitHub owner across slug-collision
	// retries). Only Slug is unique among active organizations.
	Name string
	// Slug is the URL-friendly identifier used to route to this organization
	// in the frontend. Uniqueness among non-deleted organizations is enforced
	// by a partial unique index added in the add-organization-slug migration,
	// not by a gorm tag.
	Slug                        string
	Description                 string
	CreatedByAccountID          *uuid.UUID
	AllowedProviders            datatypes.JSONSlice[string]
	EnabledExperimentalFeatures datatypes.JSONSlice[string]
	UsageRetentionWindowDays    *int32
	CreatedAt                   *time.Time
	UpdatedAt                   *time.Time
	DeletedAt                   gorm.DeletedAt `gorm:"index"`
}

func (o *Organization) IsProviderAllowed(provider string) bool {
	return slices.Contains(o.AllowedProviders, provider)
}

// HasExperimentalFeature reports whether the given feature id is active for
// this organization. Released features (per the features registry) are always
// considered active regardless of per-organization state.
func (o *Organization) HasExperimentalFeature(id string) bool {
	if features.IsReleased(id) {
		return true
	}
	return slices.Contains(o.EnabledExperimentalFeatures, id)
}

type OrganizationWithCounts struct {
	Organization
	CanvasCount           int64  `gorm:"column:canvas_count"`
	TaskCount             int64  `gorm:"column:task_count"`
	DoneTaskCount         int64  `gorm:"column:done_task_count"`
	MemberCount           int64  `gorm:"column:member_count"`
	WorkspaceCount        int64  `gorm:"column:workspace_count"`
	RepositoryCount       int64  `gorm:"column:repository_count"`
	RemainingCreditMicros *int64 `gorm:"-"`
}

func ListAllOrganizations(tx *gorm.DB, search string, limit, offset int, sortBy, sortDirection string) ([]OrganizationWithCounts, int64, error) {
	organizations, total, _, err := ListUnpinnedOrganizations(tx, search, limit, offset, sortBy, sortDirection, nil)
	return organizations, total, err
}

func ListUnpinnedOrganizations(tx *gorm.DB, search string, limit, offset int, sortBy, sortDirection string, excludeIDs []uuid.UUID) (organizations []OrganizationWithCounts, unpinnedTotal int64, matchTotal int64, err error) {
	if err = organizationsMatchingSearch(tx, search).Count(&matchTotal).Error; err != nil {
		return nil, 0, 0, err
	}

	if sortBy == OrganizationSortRemainingCreditCents {
		organizations, unpinnedTotal, err = listOrganizationsByRemainingCredit(tx, search, limit, offset, sortDirection, excludeIDs)
		return organizations, unpinnedTotal, matchTotal, err
	}

	organizations, unpinnedTotal, err = listOrganizations(tx, search, limit, offset, sortBy, sortDirection, excludeIDs)
	return organizations, unpinnedTotal, matchTotal, err
}

func listOrganizations(tx *gorm.DB, search string, limit, offset int, sortBy, sortDirection string, excludeIDs []uuid.UUID) ([]OrganizationWithCounts, int64, error) {
	query := excludeOrganizationIDs(organizationsMatchingSearch(tx, search), excludeIDs)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = withOrganizationCounts(tx, query)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var organizations []OrganizationWithCounts
	if err := query.Order(resolveOrganizationOrderClause(sortBy, sortDirection)).Find(&organizations).Error; err != nil {
		return nil, 0, err
	}

	return organizations, total, nil
}

func listOrganizationsByRemainingCredit(tx *gorm.DB, search string, limit, offset int, sortDirection string, excludeIDs []uuid.UUID) ([]OrganizationWithCounts, int64, error) {
	organizations, total, err := listOrganizations(tx, search, 0, 0, "name", "asc", excludeIDs)
	if err != nil {
		return nil, 0, err
	}
	if len(organizations) == 0 {
		return []OrganizationWithCounts{}, total, nil
	}

	ids := make([]uuid.UUID, len(organizations))
	for i, organization := range organizations {
		ids[i] = organization.ID
	}
	remaining, err := billing.RemainingCreditMicros(tx, ids)
	if err != nil {
		return nil, 0, err
	}

	descending := sortDirection != "asc"
	sort.SliceStable(organizations, func(i, j int) bool {
		left := billing.MicrosToCents(remaining[organizations[i].ID])
		right := billing.MicrosToCents(remaining[organizations[j].ID])
		if left != right {
			if descending {
				return left > right
			}
			return left < right
		}
		return organizations[i].Name < organizations[j].Name
	})

	page := sliceOrganizations(organizations, limit, offset)
	for i := range page {
		micros := remaining[page[i].ID]
		page[i].RemainingCreditMicros = &micros
	}
	return page, total, nil
}

func sliceOrganizations(organizations []OrganizationWithCounts, limit, offset int) []OrganizationWithCounts {
	if offset >= len(organizations) {
		return []OrganizationWithCounts{}
	}
	if offset > 0 {
		organizations = organizations[offset:]
	}
	if limit > 0 && len(organizations) > limit {
		organizations = organizations[:limit]
	}
	return organizations
}

func FindOrganizationWithCounts(tx *gorm.DB, id uuid.UUID) (*OrganizationWithCounts, error) {
	query := withOrganizationCounts(
		tx,
		tx.Model(&Organization{}).Where("organizations.deleted_at IS NULL"),
	)

	var organization OrganizationWithCounts
	err := query.Where("organizations.id = ?", id).First(&organization).Error
	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func withOrganizationCounts(tx *gorm.DB, query *gorm.DB) *gorm.DB {
	canvasCountsQuery := tx.
		Table("workflows").
		Select("organization_id, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Group("organization_id")

	taskCountsQuery := tx.
		Table("factory_work_orders").
		Select("factory_work_orders.organization_id, COUNT(*) AS count").
		Joins("JOIN factories ON factories.id = factory_work_orders.factory_id AND factories.deleted_at IS NULL").
		Group("factory_work_orders.organization_id")

	doneTaskCountsQuery := tx.
		Table("factory_work_orders").
		Select("factory_work_orders.organization_id, COUNT(*) AS count").
		Joins("JOIN factories ON factories.id = factory_work_orders.factory_id AND factories.deleted_at IS NULL").
		Where("factory_work_orders.state = ?", FactoryWorkOrderStateClosed).
		Where("factory_work_orders.result IN ?", []string{FactoryWorkOrderResultCompleted, FactoryWorkOrderResultFailed}).
		Group("factory_work_orders.organization_id")

	memberCountsQuery := tx.
		Table("users").
		Select("organization_id, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Where("type = ?", UserTypeHuman).
		Group("organization_id")

	workspaceCountsQuery := tx.
		Table("factories").
		Select("organization_id, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Group("organization_id")

	repositoryCountsQuery := tx.
		Table("factories").
		Select("organization_id, COUNT(*) AS count").
		Where("deleted_at IS NULL").
		Where("BTRIM(COALESCE(onboarding_config->>'app_repository', '')) <> ''").
		Group("organization_id")

	return query.
		Select(`
			organizations.*,
			COALESCE(canvas_counts.count, 0) AS canvas_count,
			COALESCE(task_counts.count, 0) AS task_count,
			COALESCE(done_task_counts.count, 0) AS done_task_count,
			COALESCE(member_counts.count, 0) AS member_count,
			COALESCE(workspace_counts.count, 0) AS workspace_count,
			COALESCE(repository_counts.count, 0) AS repository_count
		`).
		Joins("LEFT JOIN (?) AS canvas_counts ON canvas_counts.organization_id = organizations.id", canvasCountsQuery).
		Joins("LEFT JOIN (?) AS task_counts ON task_counts.organization_id = organizations.id", taskCountsQuery).
		Joins("LEFT JOIN (?) AS done_task_counts ON done_task_counts.organization_id = organizations.id", doneTaskCountsQuery).
		Joins("LEFT JOIN (?) AS member_counts ON member_counts.organization_id = organizations.id", memberCountsQuery).
		Joins("LEFT JOIN (?) AS workspace_counts ON workspace_counts.organization_id = organizations.id", workspaceCountsQuery).
		Joins("LEFT JOIN (?) AS repository_counts ON repository_counts.organization_id = organizations.id", repositoryCountsQuery)
}

func resolveOrganizationOrderClause(sortBy, sortDirection string) string {
	direction := "DESC"
	if sortDirection == "asc" {
		direction = "ASC"
	}

	switch sortBy {
	case "name":
		return "organizations.name " + direction
	case "created_at":
		return "organizations.created_at " + direction
	case "canvas_count":
		return "COALESCE(canvas_counts.count, 0) " + direction + ", organizations.name ASC"
	case "task_count":
		return "COALESCE(task_counts.count, 0) " + direction + ", organizations.name ASC"
	case "done_task_count":
		return "COALESCE(done_task_counts.count, 0) " + direction + ", organizations.name ASC"
	case "member_count":
		return "COALESCE(member_counts.count, 0) " + direction + ", organizations.name ASC"
	case "workspace_count":
		return "COALESCE(workspace_counts.count, 0) " + direction + ", organizations.name ASC"
	case "repository_count":
		return "COALESCE(repository_counts.count, 0) " + direction + ", organizations.name ASC"
	case OrganizationSortRemainingCreditCents:
		return "organizations.name ASC"
	default:
		return "organizations.created_at DESC"
	}
}

// ListOrganizations returns every organization that is not deleted.
func ListOrganizations(tx *gorm.DB) ([]Organization, error) {
	var organizations []Organization
	err := tx.Order("name ASC").Order("id ASC").Find(&organizations).Error
	if err != nil {
		return nil, err
	}
	return organizations, nil
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

// FindOrganizationByIDIncludingDeleted finds an organization that can be soft-deleted.
func FindOrganizationByIDIncludingDeleted(tx *gorm.DB, id string) (*Organization, error) {
	organization := Organization{}

	err := tx.
		Unscoped().
		Where("id = ?", id).
		First(&organization).
		Error

	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func LockOrganization(tx *gorm.DB, orgID uuid.UUID) (*Organization, error) {
	return FindOrganizationByIDInTransaction(
		tx.Clauses(clause.Locking{Strength: "UPDATE"}),
		orgID.String(),
	)
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
	if _, err := billing.Current(); err != nil {
		return nil, err
	}
	slug, err := GenerateUniqueOrganizationSlug(tx, name, uuid.Nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	organization := Organization{
		Name:             name,
		Slug:             slug,
		Description:      description,
		AllowedProviders: datatypes.JSONSlice[string]{ProviderGitHub},
		EnabledExperimentalFeatures: datatypes.JSONSlice[string]{
			features.FeatureFactories,
		},
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	err = tx.
		Clauses(clause.Returning{}).
		Create(&organization).
		Error

	if err == nil {
		if err := billing.InitializeOrganization(tx, organization.ID); err != nil {
			return nil, err
		}

		return &organization, nil
	}

	if strings.Contains(err.Error(), "organizations_slug_active_key") {
		return nil, ErrSlugAlreadyUsed
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

func SetOrganizationCreatedByAccount(tx *gorm.DB, organizationID, accountID uuid.UUID) error {
	return tx.Model(&Organization{}).
		Where("id = ?", organizationID).
		Update("created_by_account_id", accountID).
		Error
}

func ListOrganizationsCreatedByAccount(tx *gorm.DB, accountID uuid.UUID) ([]Organization, error) {
	var organizations []Organization
	err := tx.
		Where("created_by_account_id = ?", accountID).
		Order("created_at DESC").
		Find(&organizations).
		Error
	return organizations, err
}

// SoftDeleteOrganization deletes an organization whose stored billing row
// cannot bill. It refuses an organization that needs Polar verification.
func SoftDeleteOrganization(id string) error {
	return SoftDeleteOrganizationInTransaction(database.Conn(), id)
}

func SoftDeleteOrganizationInTransaction(tx *gorm.DB, id string) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		organization, err := FindOrganizationByIDInTransaction(inner, id)
		if err != nil {
			return err
		}
		guard, err := billing.LocalDeletionGuard(inner, organization.ID)
		if err != nil {
			return err
		}
		return organization.SoftDelete(inner, guard)
	})
}

// SoftDelete validates the prepared billing snapshot under the organization lock.
// The caller prepares provider verification before it opens the transaction.
func (o *Organization) SoftDelete(tx *gorm.DB, guard billing.DeletionGuard) error {
	if guard == nil {
		return billing.ErrOrganizationSubscriptionUnverified
	}
	return tx.Transaction(func(inner *gorm.DB) error {
		if _, err := LockOrganization(inner, o.ID); err != nil {
			return err
		}
		if err := guard.Validate(inner, o.ID); err != nil {
			return err
		}
		return inner.Delete(o).Error
	})
}

func HardDeleteOrganization(id string) error {
	return database.Conn().
		Unscoped().
		Where("id = ?", id).
		Delete(&Organization{}).
		Error
}

func ListDeletedOrganizations() ([]Organization, error) {
	return ListDeletedOrganizationsInTransaction(database.Conn())
}

func ListDeletedOrganizationsInTransaction(tx *gorm.DB) ([]Organization, error) {
	var organizations []Organization

	err := tx.
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&organizations).
		Error
	if err != nil {
		return nil, err
	}

	return organizations, nil
}

func LockDeletedOrganization(tx *gorm.DB, id uuid.UUID) (*Organization, error) {
	var organization Organization

	err := tx.
		Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("deleted_at IS NOT NULL").
		First(&organization).
		Error
	if err != nil {
		return nil, err
	}

	return &organization, nil
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

// EnableExperimentalFeature adds the given feature id to the organization's
// enabled set, deduping. The caller is responsible for validating that the
// feature id exists in the registry.
func EnableExperimentalFeature(orgID uuid.UUID, featureID string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return EnableExperimentalFeatureInTransaction(tx, orgID, featureID)
	})
}

func EnableExperimentalFeatureInTransaction(tx *gorm.DB, orgID uuid.UUID, featureID string) error {
	organization, err := FindOrganizationByIDInTransaction(
		tx.Clauses(clause.Locking{Strength: "UPDATE"}),
		orgID.String(),
	)
	if err != nil {
		return err
	}

	if slices.Contains(organization.EnabledExperimentalFeatures, featureID) {
		return nil
	}

	updated := append(append(datatypes.JSONSlice[string]{}, organization.EnabledExperimentalFeatures...), featureID)
	now := time.Now()
	return tx.
		Model(&Organization{}).
		Where("id = ?", orgID).
		Updates(map[string]any{
			"enabled_experimental_features": updated,
			"updated_at":                    &now,
		}).
		Error
}

// DisableExperimentalFeature removes the given feature id from the
// organization's enabled set. Disabling an id that is not enabled is a no-op.
func DisableExperimentalFeature(orgID uuid.UUID, featureID string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return DisableExperimentalFeatureInTransaction(tx, orgID, featureID)
	})
}

func DisableExperimentalFeatureInTransaction(tx *gorm.DB, orgID uuid.UUID, featureID string) error {
	organization, err := FindOrganizationByIDInTransaction(
		tx.Clauses(clause.Locking{Strength: "UPDATE"}),
		orgID.String(),
	)
	if err != nil {
		return err
	}

	if !slices.Contains(organization.EnabledExperimentalFeatures, featureID) {
		return nil
	}

	updated := datatypes.JSONSlice[string]{}
	for _, id := range organization.EnabledExperimentalFeatures {
		if id != featureID {
			updated = append(updated, id)
		}
	}

	now := time.Now()
	return tx.
		Model(&Organization{}).
		Where("id = ?", orgID).
		Updates(map[string]any{
			"enabled_experimental_features": updated,
			"updated_at":                    &now,
		}).
		Error
}

// OrganizationHasExperimentalFeatures reports whether every feature is active
// for the organization. Released features count as active.
func OrganizationHasExperimentalFeatures(tx *gorm.DB, orgID uuid.UUID, featureIDs ...string) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("transaction is required")
	}
	organization, err := FindOrganizationByIDInTransaction(tx, orgID.String())
	if err != nil {
		return false, err
	}
	for _, featureID := range featureIDs {
		if !organization.HasExperimentalFeature(featureID) {
			return false, nil
		}
	}
	return true, nil
}

// HasExperimentalFeature reports whether the given feature id is active for
// the organization with the given id. Released features are reported true
// without loading the organization from the database.
func HasExperimentalFeature(orgID uuid.UUID, featureID string) (bool, error) {
	if features.IsReleased(featureID) {
		return true, nil
	}

	organization, err := FindOrganizationByID(orgID.String())
	if err != nil {
		return false, err
	}
	return organization.HasExperimentalFeature(featureID), nil
}

func organizationsMatchingSearch(tx *gorm.DB, search string) *gorm.DB {
	query := tx.
		Model(&Organization{}).
		Where("organizations.deleted_at IS NULL")

	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	pattern := "%" + search + "%"
	return query.Where(
		"organizations.name ILIKE ? OR organizations.id::text ILIKE ?",
		pattern,
		pattern,
	)
}

func excludeOrganizationIDs(query *gorm.DB, ids []uuid.UUID) *gorm.DB {
	if len(ids) == 0 {
		return query
	}

	return query.Where("organizations.id NOT IN ?", ids)
}
