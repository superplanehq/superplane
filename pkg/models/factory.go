package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const factoryNameUniqueConstraint = "factories_organization_id_name_key"

var ErrFactoryNameAlreadyExists = errors.New("factory name already exists")
var ErrFactoryNameRequired = errors.New("factory name is required")
var ErrFactoryNotFound = errors.New("factory not found")
var ErrFactoryWorkOrderTitleRequired = errors.New("title is required")

type Factory struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func MapFactoryNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryNameUniqueConstraint {
		return ErrFactoryNameAlreadyExists
	}

	return err
}

func CreateFactory(tx *gorm.DB, organizationID uuid.UUID, name, description string) (*Factory, error) {
	now := time.Now()
	factory := &Factory{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           name,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(factory).Error; err != nil {
		return nil, MapFactoryNameUniqueConstraintError(err)
	}

	return factory, nil
}

func FindFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*Factory, error) {
	var factory Factory
	err := tx.
		Where("organization_id = ? AND id = ?", organizationID, factoryID).
		First(&factory).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryNotFound
		}
		return nil, err
	}

	return &factory, nil
}

func ListFactories(tx *gorm.DB, organizationID uuid.UUID) ([]Factory, error) {
	var factories []Factory
	err := tx.
		Where("organization_id = ?", organizationID).
		Order("name ASC").
		Order("id ASC").
		Find(&factories).
		Error
	if err != nil {
		return nil, err
	}

	return factories, nil
}

func (f *Factory) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	newName := fmt.Sprintf("%s (deleted-%d)", f.Name, now.Unix())

	err := tx.Model(f).Updates(map[string]any{
		"name":       newName,
		"deleted_at": now,
		"updated_at": now,
	}).Error
	if err != nil {
		return err
	}

	f.Name = newName
	f.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	f.UpdatedAt = now
	return nil
}

func (f *Factory) Update(tx *gorm.DB, name, description *string) error {
	updates := map[string]any{}

	if name != nil {
		nextName := strings.TrimSpace(*name)
		if nextName == "" {
			return ErrFactoryNameRequired
		}
		if f.Name != nextName {
			updates["name"] = nextName
		}
	}

	if description != nil && f.Description != *description {
		updates["description"] = *description
	}

	if len(updates) == 0 {
		return nil
	}

	now := time.Now()
	updates["updated_at"] = now

	err := MapFactoryNameUniqueConstraintError(
		tx.Model(f).
			Where("organization_id = ? AND id = ?", f.OrganizationID, f.ID).
			Updates(updates).
			Error,
	)
	if err != nil {
		return err
	}

	if nextName, ok := updates["name"].(string); ok {
		f.Name = nextName
	}
	if nextDescription, ok := updates["description"].(string); ok {
		f.Description = nextDescription
	}
	f.UpdatedAt = now
	return nil
}

func (f *Factory) ListCanvases(tx *gorm.DB) ([]Canvas, error) {
	var canvases []Canvas
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Order("name ASC").
		Order("id ASC").
		Find(&canvases).
		Error
	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListDeletedFactories(tx *gorm.DB) ([]Factory, error) {
	var factories []Factory
	err := tx.
		Model(&Factory{}).
		Unscoped().
		Joins("JOIN organizations ON organizations.id = factories.organization_id").
		Select(
			"factories.id",
			"factories.organization_id",
			"factories.name",
			"factories.description",
			"factories.created_at",
			"factories.updated_at",
			// Earliest deletion wins so neither factory nor org soft-delete resets grace.
			"LEAST(factories.deleted_at, organizations.deleted_at) AS deleted_at",
		).
		Where("factories.deleted_at IS NOT NULL OR organizations.deleted_at IS NOT NULL").
		Find(&factories).
		Error
	if err != nil {
		return nil, err
	}

	return factories, nil
}

func LockDeletedFactory(tx *gorm.DB, id uuid.UUID) (*Factory, error) {
	var factory Factory
	err := tx.
		Unscoped().
		Model(&Factory{}).
		Joins("JOIN organizations ON organizations.id = factories.organization_id").
		Select(
			"factories.id",
			"factories.organization_id",
			"factories.name",
			"factories.description",
			"factories.created_at",
			"factories.updated_at",
			"LEAST(factories.deleted_at, organizations.deleted_at) AS deleted_at",
		).
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "factories"},
			Options:  "SKIP LOCKED",
		}).
		Where("factories.id = ?", id).
		Where("factories.deleted_at IS NOT NULL OR organizations.deleted_at IS NOT NULL").
		First(&factory).
		Error
	if err != nil {
		return nil, err
	}

	return &factory, nil
}

func (f *Factory) CountCanvases(tx *gorm.DB) (int64, error) {
	var count int64
	err := tx.Unscoped().
		Model(&Canvas{}).
		Where("factory_id = ?", f.ID).
		Count(&count).
		Error
	return count, err
}

func (f *Factory) SoftDeleteCanvases(tx *gorm.DB) error {
	canvases, err := f.ListCanvases(tx)
	if err != nil {
		return err
	}

	for i := range canvases {
		if err := canvases[i].SoftDeleteInTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

func CountFactoriesByOrganization(tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Unscoped().
		Model(&Factory{}).
		Where("organization_id = ?", organizationID).
		Count(&count).
		Error
	return count, err
}

func SoftDeleteOrganizationFactories(tx *gorm.DB, organizationID uuid.UUID) error {
	var factories []Factory
	err := tx.
		Where("organization_id = ?", organizationID).
		Find(&factories).
		Error
	if err != nil {
		return err
	}

	for i := range factories {
		if factories[i].DeletedAt.Valid {
			continue
		}
		if err := factories[i].SoftDelete(tx); err != nil {
			return err
		}
	}

	return nil
}

func (f *Factory) CreateWorkOrder(tx *gorm.DB, title, description string, createdBy *uuid.UUID, assignees []uuid.UUID, sourceRunID *uuid.UUID) (*FactoryWorkOrder, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrFactoryWorkOrderTitleRequired
	}

	now := time.Now()
	order := &FactoryWorkOrder{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Title:          title,
		Description:    description,
		State:          FactoryWorkOrderStateDraft,
		Result:         "",
		CreatedByID:    createdBy,
		SourceRunID:    sourceRunID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(order).Error; err != nil {
		return nil, err
	}

	if len(assignees) > 0 {
		if err := order.ReplaceAssignees(tx, assignees); err != nil {
			return nil, err
		}
	}

	// Creation is a status transition into `draft` (fromState == "").
	// For orders spawned by a canvas run we snapshot the originating
	// run + app here so the very first timeline entry links back to
	// the run that created the order — matching the enrichment
	// UpdateStatus performs on the draft → open promotion.
	initialStatus := statusUpdatedRecord{
		Actor:   createdBy,
		ToState: FactoryWorkOrderStateDraft,
	}
	if sourceRunID != nil {
		sourceRun, sourceApp, err := order.loadSourceRunRefs(tx)
		if err != nil {
			return nil, err
		}
		initialStatus.Run = sourceRun
		initialStatus.App = sourceApp
	}
	if err := order.RecordStatusUpdated(tx, initialStatus); err != nil {
		return nil, err
	}

	return f.FindWorkOrder(tx, order.ID)
}

// FindWorkOrderByArtifactKey resolves a work order from one of its
// artifacts' `key` values, then delegates to FindWorkOrder so the result
// gets the same preloads/scoping as every other lookup path.
func (f *Factory) FindWorkOrderByArtifactKey(tx *gorm.DB, key string) (*FactoryWorkOrder, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, ErrFactoryWorkOrderNotFound
	}

	var artifact FactoryWorkOrderArtifact
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND key = ?", f.OrganizationID, f.ID, trimmedKey).
		First(&artifact).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderNotFound
		}
		return nil, err
	}

	return f.FindWorkOrder(tx, artifact.WorkOrderID)
}

func (f *Factory) FindWorkOrder(tx *gorm.DB, orderID uuid.UUID) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := tx.
		Preload("CreatedBy").
		Preload("Assignees").
		Preload("Assignees.User").
		Where("organization_id = ? AND factory_id = ? AND id = ?", f.OrganizationID, f.ID, orderID).
		First(&order).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

type ListFactoryWorkOrdersFilters struct {
	AssigneeIDs []uuid.UUID
	States      []string
	Results     []string
	Unassigned  *bool
}

func (f *Factory) ListWorkOrders(tx *gorm.DB, filters ListFactoryWorkOrdersFilters) ([]FactoryWorkOrder, error) {
	query := tx.
		Model(&FactoryWorkOrder{}).
		Preload("CreatedBy").
		Preload("Assignees").
		Preload("Assignees.User").
		Where("factory_work_orders.organization_id = ?", f.OrganizationID).
		Where("factory_work_orders.factory_id = ?", f.ID)

	if len(filters.States) > 0 {
		query = query.Where("factory_work_orders.state IN ?", filters.States)
	}

	if len(filters.Results) > 0 {
		query = query.Where("factory_work_orders.result IN ?", filters.Results)
	}

	if filters.Unassigned != nil && *filters.Unassigned {
		query = query.Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_assignees
				WHERE factory_work_order_assignees.work_order_id = factory_work_orders.id
			)`)
	}

	if len(filters.AssigneeIDs) > 0 {
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM factory_work_order_assignees
				WHERE factory_work_order_assignees.work_order_id = factory_work_orders.id
				AND factory_work_order_assignees.user_id IN ?
			)`, filters.AssigneeIDs)
	}

	var orders []FactoryWorkOrder
	err := query.
		Order("factory_work_orders.created_at DESC").
		Order("factory_work_orders.id DESC").
		Find(&orders).
		Error
	if err != nil {
		return nil, err
	}

	return orders, nil
}
