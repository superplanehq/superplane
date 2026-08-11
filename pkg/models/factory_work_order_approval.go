package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderApprovalStatusPending  = "pending"
	FactoryWorkOrderApprovalStatusApproved = "approved"
	FactoryWorkOrderApprovalStatusRejected = "rejected"
)

var (
	ErrFactoryWorkOrderApprovalNotFound      = errors.New("factory work order approval not found")
	ErrFactoryWorkOrderApprovalAlreadyClosed = errors.New("work order approval is already resolved")
	ErrFactoryWorkOrderApprovalInvalidStatus = errors.New("invalid work order approval status")
)

type FactoryWorkOrderApproval struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	ExecutionID    *uuid.UUID
	Title          string
	Message        string
	Status         string
	ApproverID     *uuid.UUID
	Comment        string
	ResolvedByID   *uuid.UUID
	ResolvedAt     *time.Time
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time

	CreatedBy  *User `gorm:"foreignKey:CreatedByID"`
	Approver   *User `gorm:"foreignKey:ApproverID"`
	ResolvedBy *User `gorm:"foreignKey:ResolvedByID"`
}

func (FactoryWorkOrderApproval) TableName() string {
	return "factory_work_order_approvals"
}

// NewFactoryWorkOrderApproval builds a pending approval attached to the given
// work order. Callers still need to persist the returned value with tx.Create.
func NewFactoryWorkOrderApproval(
	order *FactoryWorkOrder,
	executionID *uuid.UUID,
	title, message string,
	approverID *uuid.UUID,
	createdBy *uuid.UUID,
) *FactoryWorkOrderApproval {
	now := time.Now()
	return &FactoryWorkOrderApproval{
		ID:             uuid.New(),
		OrganizationID: order.OrganizationID,
		FactoryID:      order.FactoryID,
		WorkOrderID:    order.ID,
		ExecutionID:    executionID,
		Title:          strings.TrimSpace(title),
		Message:        strings.TrimSpace(message),
		Status:         FactoryWorkOrderApprovalStatusPending,
		ApproverID:     approverID,
		CreatedByID:    createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (a *FactoryWorkOrderApproval) IsResolved() bool {
	return a.Status == FactoryWorkOrderApprovalStatusApproved ||
		a.Status == FactoryWorkOrderApprovalStatusRejected
}

// Ref returns the payload snapshot embedded in approval events.
func (a *FactoryWorkOrderApproval) Ref() factory.ApprovalRef {
	ref := factory.ApprovalRef{
		ID:      a.ID,
		Title:   a.Title,
		Message: a.Message,
		Status:  a.Status,
	}
	if a.ExecutionID != nil {
		ref.ExecutionID = a.ExecutionID
	}
	if a.ApproverID != nil {
		ref.ApproverID = a.ApproverID
	}
	return ref
}

// Resolve records the approver decision and writes both the approval row and
// the `order.approval.resolved` event in a single transaction. `resolver` is
// the user who clicked Approve/Reject; comment is optional.
func (a *FactoryWorkOrderApproval) Resolve(
	tx *gorm.DB,
	status string,
	resolver uuid.UUID,
	comment string,
) error {
	if status != FactoryWorkOrderApprovalStatusApproved &&
		status != FactoryWorkOrderApprovalStatusRejected {
		return ErrFactoryWorkOrderApprovalInvalidStatus
	}
	if a.IsResolved() {
		return ErrFactoryWorkOrderApprovalAlreadyClosed
	}

	order, err := FindUnscopedWorkOrder(tx, a.WorkOrderID)
	if err != nil {
		return err
	}

	now := time.Now()
	a.Status = status
	a.Comment = strings.TrimSpace(comment)
	a.ResolvedByID = &resolver
	a.ResolvedAt = &now
	a.UpdatedAt = now

	err = tx.Model(a).Updates(map[string]any{
		"status":         a.Status,
		"comment":        a.Comment,
		"resolved_by_id": a.ResolvedByID,
		"resolved_at":    a.ResolvedAt,
		"updated_at":     a.UpdatedAt,
	}).Error
	if err != nil {
		return err
	}

	return order.RecordApprovalResolved(tx, a, resolver)
}

func FindFactoryWorkOrderApproval(tx *gorm.DB, organizationID, approvalID uuid.UUID) (*FactoryWorkOrderApproval, error) {
	var approval FactoryWorkOrderApproval
	err := tx.
		Preload("CreatedBy").
		Preload("Approver").
		Preload("ResolvedBy").
		Where("organization_id = ? AND id = ?", organizationID, approvalID).
		First(&approval).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderApprovalNotFound
		}
		return nil, err
	}
	return &approval, nil
}

// ListFactoryWorkOrderApprovals returns approvals for a work order ordered by
// creation time. Unresolved approvals surface first in the UI, but the API
// returns them in a stable chronological order to keep timeline rendering
// deterministic.
func ListFactoryWorkOrderApprovals(tx *gorm.DB, workOrderID uuid.UUID) ([]FactoryWorkOrderApproval, error) {
	var approvals []FactoryWorkOrderApproval
	err := tx.
		Preload("CreatedBy").
		Preload("Approver").
		Preload("ResolvedBy").
		Where("work_order_id = ?", workOrderID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&approvals).
		Error
	if err != nil {
		return nil, err
	}
	return approvals, nil
}

// ListFactoryWorkOrderApprovalsByWorkOrderIDs bulk-loads approvals for a set
// of work orders, so list handlers can serialize them without an N+1.
func ListFactoryWorkOrderApprovalsByWorkOrderIDs(
	tx *gorm.DB,
	workOrderIDs []uuid.UUID,
) (map[uuid.UUID][]FactoryWorkOrderApproval, error) {
	result := make(map[uuid.UUID][]FactoryWorkOrderApproval, len(workOrderIDs))
	if len(workOrderIDs) == 0 {
		return result, nil
	}

	var approvals []FactoryWorkOrderApproval
	err := tx.
		Preload("CreatedBy").
		Preload("Approver").
		Preload("ResolvedBy").
		Where("work_order_id IN ?", workOrderIDs).
		Order("created_at ASC").
		Order("id ASC").
		Find(&approvals).
		Error
	if err != nil {
		return nil, err
	}

	for i := range approvals {
		id := approvals[i].WorkOrderID
		result[id] = append(result[id], approvals[i])
	}
	return result, nil
}

// LockFactoryWorkOrderApproval takes a FOR UPDATE lock on the approval row so
// concurrent Resolve calls don't race on the pending -> resolved transition.
func LockFactoryWorkOrderApproval(tx *gorm.DB, organizationID, approvalID uuid.UUID) (*FactoryWorkOrderApproval, error) {
	var approval FactoryWorkOrderApproval
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ? AND id = ?", organizationID, approvalID).
		First(&approval).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderApprovalNotFound
		}
		return nil, err
	}
	return &approval, nil
}
