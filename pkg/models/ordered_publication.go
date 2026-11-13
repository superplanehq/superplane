package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderedPublication struct {
	IntegrationID uuid.UUID `gorm:"primaryKey"`
	Key           string    `gorm:"primaryKey"`
	Record        datatypes.JSONType[core.OrderedPublication]
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (p *OrderedPublication) TableName() string { return "review_request_publications" }

func FindOrderedPublication(tx *gorm.DB, integrationID uuid.UUID, key string) (*core.OrderedPublication, error) {
	var publication OrderedPublication
	err := tx.Where("integration_id = ? AND key = ?", integrationID, key).First(&publication).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	record := publication.Record.Data()
	return &record, nil
}

func ListOrderedPublications(tx *gorm.DB, integrationID uuid.UUID, prefix string) ([]core.OrderedPublication, error) {
	var publications []OrderedPublication
	err := tx.Where("integration_id = ? AND starts_with(key, ?)", integrationID, prefix).Find(&publications).Error
	if err != nil {
		return nil, err
	}
	records := make([]core.OrderedPublication, 0, len(publications))
	for _, publication := range publications {
		records = append(records, publication.Record.Data())
	}
	return records, nil
}

func UpdateOrderedPublication(tx *gorm.DB, integrationID uuid.UUID, key string, apply func(*core.OrderedPublication) error) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("publication key is required")
	}
	return tx.Transaction(func(tx *gorm.DB) error {
		// This also serializes the first insert, before a row exists to lock.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "publication:"+integrationID.String()+":"+key).Error; err != nil {
			return err
		}
		record, err := FindOrderedPublication(tx, integrationID, key)
		if err != nil {
			return err
		}
		if record == nil {
			record = &core.OrderedPublication{Key: key}
		}
		if err := apply(record); err != nil {
			if errors.Is(err, core.ErrPublicationUnchanged) {
				return nil
			}
			return err
		}
		record.Revision++
		publication := OrderedPublication{IntegrationID: integrationID, Key: key, Record: datatypes.NewJSONType(*record)}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "integration_id"}, {Name: "key"}}, DoUpdates: clause.Assignments(map[string]any{"record": publication.Record, "updated_at": time.Now()})}).Create(&publication).Error; err != nil {
			return err
		}
		if record.Hook == "" {
			return nil
		}
		parameters := map[string]any{"key": key}
		if err := CompletePendingActionRequestsInTransaction(tx, integrationID, record.Hook, parameters); err != nil {
			return err
		}
		if record.WakeAt == nil {
			return nil
		}
		integration := Integration{ID: integrationID}
		return integration.CreateActionRequest(tx, record.Hook, parameters, record.WakeAt)
	})
}

// ResolvePublicationOrder authorizes both identities against the execution's
// canvas and integration. Cross-canvas sources require persisted broadcast provenance.
func ResolvePublicationOrder(tx *gorm.DB, organizationID, integrationID uuid.UUID, execution *CanvasNodeExecution, sourceRunID, attemptRootEventID uuid.UUID) (core.PublicationOrder, error) {
	var order core.PublicationOrder
	if execution != nil && attemptRootEventID == uuid.Nil {
		attemptRootEventID = execution.RootEventID
	}
	if execution == nil || attemptRootEventID != execution.RootEventID {
		return order, fmt.Errorf("attempt must be the current execution root event")
	}
	var node CanvasNode
	if err := tx.Joins("JOIN workflows ON workflows.id = workflow_nodes.workflow_id").
		Joins("JOIN app_installations ON app_installations.id = workflow_nodes.app_installation_id").
		Joins("JOIN organizations ON organizations.id = workflows.organization_id").
		Where("workflows.deleted_at IS NULL AND workflow_nodes.deleted_at IS NULL AND app_installations.deleted_at IS NULL AND organizations.deleted_at IS NULL").
		Where("workflow_nodes.workflow_id = ? AND workflow_nodes.node_id = ? AND workflows.organization_id = ? AND workflow_nodes.app_installation_id = ? AND app_installations.organization_id = ?", execution.WorkflowID, execution.NodeID, organizationID, integrationID, organizationID).
		First(&node).Error; err != nil {
		return order, fmt.Errorf("publication scope is not authorized: %w", err)
	}
	attempt, err := FindCanvasEventInTransaction(tx, attemptRootEventID)
	if err != nil {
		return order, err
	}
	if attempt.WorkflowID != execution.WorkflowID || attempt.RunID != execution.RunID || attempt.ExecutionID != nil {
		return order, fmt.Errorf("attempt is not the current run root")
	}
	if sourceRunID == uuid.Nil {
		sourceRunID = execution.RunID
		if attempt.SourceRunID != nil {
			sourceRunID = *attempt.SourceRunID
		}
	}
	trustedSource := execution.RunID
	if attempt.SourceRunID != nil {
		trustedSource = *attempt.SourceRunID
	}
	if sourceRunID != trustedSource {
		return order, fmt.Errorf("source run has no authorized broadcast relationship")
	}
	var source CanvasRun
	if err := tx.Joins("JOIN workflows ON workflows.id = workflow_runs.workflow_id").
		Where("workflow_runs.id = ? AND workflows.organization_id = ?", sourceRunID, organizationID).First(&source).Error; err != nil {
		return order, fmt.Errorf("source run is not authorized: %w", err)
	}
	if source.CreatedAt == nil || attempt.CreatedAt == nil {
		return order, fmt.Errorf("publication ordering timestamps are missing")
	}
	return core.PublicationOrder{SourceRunID: source.ID, SourceCreatedAt: *source.CreatedAt, AttemptRootEventID: attempt.ID, AttemptCreatedAt: *attempt.CreatedAt}, nil
}

// PublicationOwnerStatus does not require source history. Durable delivery can
// finish after retention removes the run, or after its canvas is removed.
func PublicationOwnerStatus(tx *gorm.DB, organizationID, integrationID, canvasID uuid.UUID, nodeID string) (core.PublicationOwnerStatus, error) {
	var status core.PublicationOwnerStatus
	var available int64
	err := tx.Model(&Integration{}).
		Joins("JOIN organizations ON organizations.id = app_installations.organization_id").
		Where("app_installations.id = ? AND organizations.id = ? AND organizations.deleted_at IS NULL", integrationID, organizationID).
		Count(&available).Error
	if err != nil || available == 0 {
		return status, err
	}
	status.Available = true
	var owners int64
	err = tx.Model(&CanvasNode{}).
		Joins("JOIN workflows ON workflows.id = workflow_nodes.workflow_id").
		Where("workflow_nodes.workflow_id = ? AND workflow_nodes.node_id = ? AND workflows.organization_id = ? AND workflows.deleted_at IS NULL AND workflow_nodes.app_installation_id = ?", canvasID, nodeID, organizationID, integrationID).
		Count(&owners).Error
	status.Removed = owners == 0
	return status, err
}

func PublicationAttemptFinished(tx *gorm.DB, organizationID, rootEventID uuid.UUID) (bool, error) {
	run, err := findPublicationAttemptRun(tx, organizationID, rootEventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if run.State == CanvasRunStateFinished || run.State == CanvasRunStateCancelling || len(run.Errors) > 0 {
		return true, nil
	}
	result, err := run.CalculateResult(tx)
	return result != CanvasRunResultPassed, err
}

func PublicationAttemptEnded(tx *gorm.DB, organizationID, rootEventID uuid.UUID) (bool, error) {
	run, err := findPublicationAttemptRun(tx, organizationID, rootEventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	// Errors determine the result, not whether downstream report steps can run.
	return run.State == CanvasRunStateFinished || run.State == CanvasRunStateCancelling, nil
}

func FindPublicationExecutionOutput(tx *gorm.DB, integrationID uuid.UUID, current *CanvasNodeExecution, executionID uuid.UUID) (core.PublicationExecution, error) {
	var result core.PublicationExecution
	if current == nil {
		return result, fmt.Errorf("current execution is required")
	}
	var execution CanvasNodeExecution
	err := tx.Joins("JOIN workflow_nodes ON workflow_nodes.workflow_id = workflow_node_executions.workflow_id AND workflow_nodes.node_id = workflow_node_executions.node_id").
		Where("workflow_node_executions.id = ? AND workflow_node_executions.workflow_id = ? AND workflow_node_executions.root_event_id = ? AND workflow_nodes.app_installation_id = ?", executionID, current.WorkflowID, current.RootEventID, integrationID).First(&execution).Error
	if err != nil {
		return result, err
	}
	if execution.Result != CanvasNodeExecutionResultPassed {
		return result, fmt.Errorf("referenced execution did not pass")
	}
	var event CanvasEvent
	if err := tx.Where("execution_id = ? AND channel = ?", execution.ID, "default").Order("created_at DESC, id DESC").First(&event).Error; err != nil {
		return result, err
	}
	result.Component, err = execution.ComponentName(tx)
	if err != nil {
		return result, err
	}
	result.Configuration, result.Output = execution.Configuration.Data(), event.Data.Data()
	return result, nil
}

// FindPublicationNode exposes configuration, not automation-specific admission
// policy. An attempt always reads its saved run version, never the latest graph.
func FindPublicationNode(tx *gorm.DB, organizationID, integrationID, canvasID uuid.UUID, nodeID string, attempt uuid.UUID) (*core.PublicationNode, error) {
	var canvas Canvas
	err := tx.Where("id = ? AND organization_id = ?", canvasID, organizationID).First(&canvas).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if attempt == uuid.Nil {
		var node CanvasNode
		err := tx.Where("workflow_id = ? AND node_id = ? AND app_installation_id = ?", canvasID, nodeID, integrationID).First(&node).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return &core.PublicationNode{Component: node.ComponentName(), Configuration: node.Configuration.Data()}, nil
	}
	var run CanvasRun
	if err := tx.Joins("JOIN workflow_events ON workflow_events.run_id = workflow_runs.id").
		Where("workflow_events.id = ? AND workflow_runs.workflow_id = ?", attempt, canvasID).First(&run).Error; err != nil {
		return nil, err
	}
	version, err := FindCanvasVersionInTransaction(tx, canvasID, run.VersionID)
	if err != nil {
		return nil, err
	}
	for _, node := range version.Nodes {
		if node.ID == nodeID && node.IntegrationID != nil && *node.IntegrationID == integrationID.String() {
			return &core.PublicationNode{Component: node.ComponentName(), Configuration: node.Configuration}, nil
		}
	}
	return nil, nil
}

// WakeOwnedPublications lets component hooks release removed resources, including
// publications that stopped polling after the pull request closed.
func WakeOwnedPublications(tx *gorm.DB, canvasID uuid.UUID, nodeID string) error {
	query := tx.Where("record->>'ownerCanvasId' = ? AND record->>'hook' <> ''", canvasID.String())
	if nodeID != "" {
		query = query.Where("record->>'ownerNodeId' = ?", nodeID)
	}
	var publications []OrderedPublication
	if err := query.Find(&publications).Error; err != nil {
		return err
	}
	for _, publication := range publications {
		if err := UpdateOrderedPublication(tx, publication.IntegrationID, publication.Key, func(record *core.OrderedPublication) error {
			wake := time.Now()
			record.WakeAt = &wake
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// WakePublications schedules a bounded batch before deleted canvas data is
// removed. The deletion timestamp makes progress durable across worker ticks
// and allows another wakeup if a restored canvas is deleted again.
func (c *Canvas) WakePublications(tx *gorm.DB, limit int) (bool, error) {
	if limit <= 0 {
		return false, fmt.Errorf("publication wake limit must be positive")
	}
	var publications []OrderedPublication
	err := tx.Where("record->>'ownerCanvasId' = ? AND record->>'hook' <> ''", c.ID.String()).
		Where("(record->>'ownerDeletedAt')::timestamptz IS DISTINCT FROM ?", c.DeletedAt.Time).
		Order("integration_id, key").Limit(limit).Find(&publications).Error
	if err != nil {
		return false, err
	}
	for _, publication := range publications {
		if err := UpdateOrderedPublication(tx, publication.IntegrationID, publication.Key, func(record *core.OrderedPublication) error {
			if record.OwnerCanvasID != c.ID || (record.OwnerDeletedAt != nil && record.OwnerDeletedAt.Equal(c.DeletedAt.Time)) {
				return core.ErrPublicationUnchanged
			}
			wake := time.Now()
			record.WakeAt = &wake
			record.OwnerDeletedAt = &c.DeletedAt.Time
			return nil
		}); err != nil {
			return false, err
		}
	}
	return len(publications) < limit, nil
}

func findPublicationAttemptRun(tx *gorm.DB, organizationID, rootEventID uuid.UUID) (*CanvasRun, error) {
	var run CanvasRun
	err := tx.Joins("JOIN workflow_events ON workflow_events.run_id = workflow_runs.id").
		Joins("JOIN workflows ON workflows.id = workflow_runs.workflow_id").
		Where("workflow_events.id = ? AND workflows.organization_id = ?", rootEventID, organizationID).First(&run).Error
	return &run, err
}
