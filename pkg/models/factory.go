package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const factoryNameUniqueConstraint = "factories_organization_id_name_key"

var ErrFactoryNameAlreadyExists = errors.New("factory name already exists")
var ErrFactoryNotFound = errors.New("factory not found")

type Factory struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

func (f *Factory) CreateWorkOrder(tx *gorm.DB, title, description string, createdBy uuid.UUID, assignees []uuid.UUID) (*FactoryWorkOrder, error) {
	now := time.Now()
	order := &FactoryWorkOrder{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Title:          title,
		Description:    description,
		State:          FactoryWorkOrderStateOpen,
		Result:         "",
		CreatedByID:    &createdBy,
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

	return f.FindWorkOrder(tx, order.ID)
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
