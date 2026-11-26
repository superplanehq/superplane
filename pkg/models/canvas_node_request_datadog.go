package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DatadogMonitorRequest extends the existing hook request; queued hooks without
// this state start with zero attempts and no lease.
type DatadogMonitorRequest struct {
	LastError      string                 `json:"last_error,omitempty"`
	Target         *RetiredDatadogMonitor `json:"target,omitempty"`
	Attempts       int                    `json:"attempts"`
	LeaseID        string                 `json:"lease_id,omitempty"`
	LeaseUntil     *time.Time             `json:"lease_until,omitempty"`
	RetiredMonitor *RetiredDatadogMonitor `json:"retired_monitor,omitempty"`
}
type RetiredDatadogMonitor struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	IntegrationID  uuid.UUID `json:"integration_id"`
	MonitorID      string    `json:"monitor_id"`
	Service        string    `json:"service"`
}

// LockCanvasNodeRequest also permits completion of a lease whose request was
// superseded by a concurrent publish, and cleanup after a canvas was deleted.
func LockCanvasNodeRequest(tx *gorm.DB, id uuid.UUID) (*CanvasNodeRequest, error) {
	var request CanvasNodeRequest
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).First(&request, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *CanvasNodeRequest) ActiveDatadogMonitorLease(tx *gorm.DB) (*time.Time, error) {
	var active CanvasNodeRequest
	err := tx.Where("workflow_id = ? AND node_id = ? AND id <> ?", r.WorkflowID, r.NodeID, r.ID).
		Where("spec->'datadog_monitor'->>'lease_until' IS NOT NULL").
		Where("(spec->'datadog_monitor'->>'lease_until')::timestamptz > ?", time.Now()).
		Order("run_at DESC").First(&active).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return active.Spec.Data().DatadogMonitor.LeaseUntil, nil
}

// Cleanup requests must also run for soft-deleted canvases.
func ListDueRetiredDatadogMonitorRequests(tx *gorm.DB) ([]CanvasNodeRequest, error) {
	var requests []CanvasNodeRequest
	err := tx.Table("workflow_node_requests").Select("workflow_node_requests.*").
		Joins("JOIN workflows ON workflows.id = workflow_node_requests.workflow_id").
		Joins("JOIN workflow_nodes ON workflow_nodes.workflow_id = workflow_node_requests.workflow_id AND workflow_nodes.node_id = workflow_node_requests.node_id").
		Where("workflow_node_requests.state = ? AND workflow_node_requests.run_at <= ?", NodeExecutionRequestStatePending, time.Now()).
		Where("workflow_node_requests.type = ? AND workflow_node_requests.spec->'invoke_action'->>'action_name' = ?", NodeRequestTypeInvokeAction, "reconcileIntakeMonitor").
		Where("workflows.deleted_at IS NOT NULL OR workflow_nodes.deleted_at IS NOT NULL").Find(&requests).Error
	return requests, err
}

func (c *CanvasNode) FindPendingDatadogMonitorRequest(tx *gorm.DB) (*CanvasNodeRequest, error) {
	var request CanvasNodeRequest
	err := tx.Where("workflow_id = ? AND node_id = ? AND state = ? AND execution_id IS NULL", c.WorkflowID, c.NodeID, NodeExecutionRequestStatePending).
		Where("type = ? AND spec->'invoke_action'->>'action_name' = ?", NodeRequestTypeInvokeAction, "reconcileIntakeMonitor").
		Where("spec->'datadog_monitor'->'retired_monitor' IS NULL").First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}
