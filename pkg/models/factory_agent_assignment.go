package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFactoryAgentAssignmentNotFound = errors.New("factory agent assignment not found")

type FactoryAgentAssignment struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	AgentID        uuid.UUID
	WorkOrderID    uuid.UUID
	Instructions   string
	State          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (FactoryAgentAssignment) TableName() string {
	return "factory_agent_assignments"
}

type CreateFactoryAgentAssignmentParams struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	AgentID        uuid.UUID
	WorkOrderID    uuid.UUID
	Instructions   string
}

func CreateFactoryAgentAssignments(
	tx *gorm.DB,
	params []CreateFactoryAgentAssignmentParams,
) ([]FactoryAgentAssignment, error) {
	if len(params) == 0 {
		return nil, nil
	}

	now := time.Now()
	assignments := make([]FactoryAgentAssignment, 0, len(params))
	for _, p := range params {
		assignments = append(assignments, FactoryAgentAssignment{
			ID:             uuid.New(),
			OrganizationID: p.OrganizationID,
			FactoryID:      p.FactoryID,
			AgentID:        p.AgentID,
			WorkOrderID:    p.WorkOrderID,
			Instructions:   p.Instructions,
			State:          FactoryAgentAssignmentStatePending,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	if err := tx.Clauses(clause.Returning{}).Create(&assignments).Error; err != nil {
		return nil, err
	}

	return assignments, nil
}

func FindFactoryAgentAssignment(
	tx *gorm.DB,
	organizationID, factoryID, agentID, assignmentID uuid.UUID,
) (*FactoryAgentAssignment, error) {
	var assignment FactoryAgentAssignment
	err := tx.
		Where(
			"organization_id = ? AND factory_id = ? AND agent_id = ? AND id = ?",
			organizationID, factoryID, agentID, assignmentID,
		).
		First(&assignment).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryAgentAssignmentNotFound
		}
		return nil, err
	}

	return &assignment, nil
}

func ListFactoryAgentAssignmentsForAgent(
	tx *gorm.DB,
	organizationID, factoryID, agentID uuid.UUID,
) ([]FactoryAgentAssignment, error) {
	var assignments []FactoryAgentAssignment
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND agent_id = ?", organizationID, factoryID, agentID).
		Order("created_at DESC").
		Order("id DESC").
		Find(&assignments).
		Error
	if err != nil {
		return nil, err
	}

	return assignments, nil
}

func ListFactoryAgentAssignmentsForOrder(
	tx *gorm.DB,
	organizationID, factoryID, workOrderID uuid.UUID,
) ([]FactoryAgentAssignment, error) {
	var assignments []FactoryAgentAssignment
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND work_order_id = ?", organizationID, factoryID, workOrderID).
		Order("created_at DESC").
		Order("id DESC").
		Find(&assignments).
		Error
	if err != nil {
		return nil, err
	}

	return assignments, nil
}

func ListPendingFactoryAgentAssignments(tx *gorm.DB, limit int) ([]FactoryAgentAssignment, error) {
	var assignments []FactoryAgentAssignment
	err := tx.
		Where("state = ?", FactoryAgentAssignmentStatePending).
		Order("created_at ASC").
		Order("id ASC").
		Limit(limit).
		Find(&assignments).
		Error
	if err != nil {
		return nil, err
	}

	return assignments, nil
}

func (a *FactoryAgentAssignment) UpdateState(tx *gorm.DB, state string) error {
	a.State = state
	a.UpdatedAt = time.Now()
	return tx.Model(a).Updates(map[string]any{
		"state":      state,
		"updated_at": a.UpdatedAt,
	}).Error
}
