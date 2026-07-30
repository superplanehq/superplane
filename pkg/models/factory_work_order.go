package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderStateOpen   = "open"
	FactoryWorkOrderStateClosed = "closed"

	FactoryWorkOrderResultCompleted = "completed"
	FactoryWorkOrderResultRejected  = "rejected"
)

const factoryWorkOrderSourceDedupIndex = "idx_factory_work_orders_source_dedup"

var ErrFactoryWorkOrderNotFound = errors.New("factory work order not found")
var ErrFactoryWorkOrderSourceAlreadyExists = errors.New("factory work order for source already exists")

type FactoryWorkOrderAttribute struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type FactoryWorkOrderSourceRef struct {
	ID   *uuid.UUID
	Name string
	Key  string
}

type FactoryWorkOrder struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Title          string
	Description    string
	State          string
	Result         string
	SourceID       *uuid.UUID
	SourceName     string
	SourceKey      string
	Attributes     datatypes.JSONSlice[FactoryWorkOrderAttribute]
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Assignees []FactoryWorkOrderAssignee `gorm:"foreignKey:WorkOrderID"`
}

func (FactoryWorkOrder) TableName() string {
	return "factory_work_orders"
}

type FactoryWorkOrderAssignee struct {
	WorkOrderID uuid.UUID
	UserID      uuid.UUID
	CreatedAt   time.Time

	User *User `gorm:"foreignKey:UserID"`
}

func (FactoryWorkOrderAssignee) TableName() string {
	return "factory_work_order_assignees"
}

type ListFactoryWorkOrdersFilters struct {
	AssigneeIDs []uuid.UUID
	States      []string
	Results     []string
	Unassigned  *bool
}

type CreateFactoryWorkOrderParams struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Title          string
	Description    string
	AssigneeIDs    []uuid.UUID
	Attributes     []FactoryWorkOrderAttribute
	Source         *FactoryWorkOrderSourceRef
}

func CreateFactoryWorkOrder(tx *gorm.DB, params CreateFactoryWorkOrderParams) (*FactoryWorkOrder, error) {
	now := time.Now()
	attributes := params.Attributes
	if attributes == nil {
		attributes = []FactoryWorkOrderAttribute{}
	}

	order := &FactoryWorkOrder{
		OrganizationID: params.OrganizationID,
		FactoryID:      params.FactoryID,
		Title:          params.Title,
		Description:    params.Description,
		State:          FactoryWorkOrderStateOpen,
		Result:         "",
		Attributes:     datatypes.NewJSONSlice(attributes),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if params.Source != nil {
		order.SourceID = params.Source.ID
		order.SourceName = params.Source.Name
		order.SourceKey = params.Source.Key
	}

	if err := tx.Clauses(clause.Returning{}).Create(order).Error; err != nil {
		return nil, mapFactoryWorkOrderSourceDedupError(err)
	}

	if len(params.AssigneeIDs) > 0 {
		if err := replaceFactoryWorkOrderAssignees(tx, order.ID, params.AssigneeIDs, now); err != nil {
			return nil, err
		}
	}

	return FindFactoryWorkOrder(tx, params.OrganizationID, params.FactoryID, order.ID)
}

func mapFactoryWorkOrderSourceDedupError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryWorkOrderSourceDedupIndex {
		return ErrFactoryWorkOrderSourceAlreadyExists
	}

	return err
}

func FindFactoryWorkOrder(tx *gorm.DB, organizationID, factoryID, orderID uuid.UUID) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := tx.
		Preload("Assignees").
		Preload("Assignees.User").
		Where("organization_id = ? AND factory_id = ? AND id = ?", organizationID, factoryID, orderID).
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

func ListFactoryWorkOrders(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	filters ListFactoryWorkOrdersFilters,
) ([]FactoryWorkOrder, error) {
	query := tx.
		Model(&FactoryWorkOrder{}).
		Preload("Assignees").
		Preload("Assignees.User").
		Where("factory_work_orders.organization_id = ?", organizationID).
		Where("factory_work_orders.factory_id = ?", factoryID)

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

func AssignFactoryWorkOrder(
	tx *gorm.DB,
	organizationID, factoryID, orderID uuid.UUID,
	assigneeIDs []uuid.UUID,
) (*FactoryWorkOrder, error) {
	order, err := FindFactoryWorkOrder(tx, organizationID, factoryID, orderID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := replaceFactoryWorkOrderAssignees(tx, order.ID, assigneeIDs, now); err != nil {
		return nil, err
	}

	order.UpdatedAt = now
	if err := tx.Model(order).Update("updated_at", now).Error; err != nil {
		return nil, err
	}

	return FindFactoryWorkOrder(tx, organizationID, factoryID, orderID)
}

func CloseFactoryWorkOrder(
	tx *gorm.DB,
	organizationID, factoryID, orderID uuid.UUID,
	result string,
) (*FactoryWorkOrder, error) {
	order, err := FindFactoryWorkOrder(tx, organizationID, factoryID, orderID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	order.State = FactoryWorkOrderStateClosed
	order.Result = result
	order.UpdatedAt = now

	if err := tx.Save(order).Error; err != nil {
		return nil, err
	}

	return order, nil
}

func replaceFactoryWorkOrderAssignees(tx *gorm.DB, workOrderID uuid.UUID, assigneeIDs []uuid.UUID, now time.Time) error {
	if err := tx.Where("work_order_id = ?", workOrderID).Delete(&FactoryWorkOrderAssignee{}).Error; err != nil {
		return err
	}

	if len(assigneeIDs) == 0 {
		return nil
	}

	assignees := make([]FactoryWorkOrderAssignee, 0, len(assigneeIDs))
	for _, assigneeID := range assigneeIDs {
		assignees = append(assignees, FactoryWorkOrderAssignee{
			WorkOrderID: workOrderID,
			UserID:      assigneeID,
			CreatedAt:   now,
		})
	}

	return tx.Create(&assignees).Error
}

func FindFactoryWorkOrderBySourceKey(
	tx *gorm.DB,
	factoryID, sourceID uuid.UUID,
	sourceKey string,
) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := tx.
		Preload("Assignees").
		Preload("Assignees.User").
		Where("factory_id = ? AND source_id = ? AND source_key = ?", factoryID, sourceID, sourceKey).
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
