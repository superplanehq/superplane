package contexts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ConfigurationBuildError struct {
	Err         error
	QueueItem   *models.WorkflowNodeQueueItem
	Node        *models.WorkflowNode
	Event       *models.WorkflowEvent
	RootEventID uuid.UUID
}

func (e *ConfigurationBuildError) Error() string {
	return fmt.Sprintf("configuration build failed: %v", e.Err)
}

func (e *ConfigurationBuildError) Unwrap() error {
	return e.Err
}

func BuildProcessQueueContext(tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem) (*core.ProcessQueueContext, error) {
	event, err := models.FindWorkflowEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, err
	}

	configBuilder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(event.Data.Data())

	if node.ParentNodeID != nil {
		parent, err := models.FindWorkflowNode(tx, node.WorkflowID, *node.ParentNodeID)
		if err != nil {
			return nil, err
		}

		configBuilder = configBuilder.ForBlueprintNode(parent)
	}

	config, err := configBuilder.Build(node.Configuration.Data())
	if err != nil {
		return nil, &ConfigurationBuildError{
			Err:         err,
			QueueItem:   queueItem,
			Node:        node,
			Event:       event,
			RootEventID: queueItem.RootEventID,
		}
	}

	ctx := &core.ProcessQueueContext{
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: config,
		RootEventID:   queueItem.RootEventID.String(),
		EventID:       event.ID.String(),
		SourceNodeID:  event.NodeID,
		Input:         event.Data.Data(),
	}

	ctx.CreateExecution = func() (uuid.UUID, error) {
		now := time.Now()

		execution := models.WorkflowNodeExecution{
			WorkflowID:          queueItem.WorkflowID,
			NodeID:              node.NodeID,
			RootEventID:         queueItem.RootEventID,
			EventID:             event.ID,
			PreviousExecutionID: event.ExecutionID,
			State:               models.WorkflowNodeExecutionStatePending,
			Configuration:       datatypes.NewJSONType(config),
			CreatedAt:           &now,
			UpdatedAt:           &now,
		}

		// If this queue item originated from an internal (blueprint) execution chain,
		// propagate the parent execution id from the previous execution so that
		// child executions are linked to the top-level blueprint execution.
		if event.ExecutionID != nil {
			if prev, err := models.FindNodeExecutionInTransaction(tx, node.WorkflowID, *event.ExecutionID); err == nil {
				if prev.ParentExecutionID != nil {
					execution.ParentExecutionID = prev.ParentExecutionID
				}
			}
		}

		if err := tx.Create(&execution).Error; err != nil {
			return uuid.Nil, err
		}

		return execution.ID, nil
	}

	ctx.DequeueItem = func() error {
		return queueItem.Delete(tx)
	}

	ctx.UpdateNodeState = func(state string) error {
		return node.UpdateState(tx, state)
	}

	ctx.DefaultProcessing = func() (*models.WorkflowNodeExecution, error) {
		execID, err := ctx.CreateExecution()
		if err != nil {
			return nil, err
		}
		if err := ctx.DequeueItem(); err != nil {
			return nil, err
		}
		if err := ctx.UpdateNodeState(models.WorkflowNodeStateProcessing); err != nil {
			return nil, err
		}
		return models.FindNodeExecutionInTransaction(tx, node.WorkflowID, execID)
	}

	ctx.GetExecutionMetadata = func(execID uuid.UUID) (map[string]any, error) {
		exec, err := models.FindNodeExecutionInTransaction(tx, node.WorkflowID, execID)
		if err != nil {
			return nil, err
		}

		return exec.Metadata.Data(), nil
	}

	ctx.SetExecutionMetadata = func(execID uuid.UUID, metadata any) error {
		exec, err := models.FindNodeExecutionInTransaction(tx, node.WorkflowID, execID)
		if err != nil {
			return err
		}

		b, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		var v map[string]any
		err = json.Unmarshal(b, &v)
		if err != nil {
			return err
		}

		exec.Metadata = datatypes.NewJSONType(v)
		return tx.Save(exec).Error
	}

	ctx.CountIncomingEdges = func() (int, error) {
		// If this is an internal (blueprint) node, count incoming edges
		// from the blueprint graph; otherwise count at the workflow level.
		if node.ParentNodeID != nil && *node.ParentNodeID != "" {
			// Parent should be a blueprint node. Load it and its blueprint spec.
			parent, err := models.FindWorkflowNode(tx, node.WorkflowID, *node.ParentNodeID)
			if err != nil {
				return 0, err
			}

			// Defensive: if no blueprint ref, fallback to workflow edges.
			blueprintID := parent.Ref.Data().Blueprint.ID
			if blueprintID != "" {
				bp, err := models.FindUnscopedBlueprintInTransaction(tx, blueprintID)
				if err != nil {
					return 0, err
				}

				// Child node id inside the blueprint (strip the parent prefix + ':')
				prefix := parent.NodeID + ":"
				childID := node.NodeID
				if len(childID) > len(prefix) && childID[:len(prefix)] == prefix {
					childID = childID[len(prefix):]
				}

				count := 0
				for _, e := range bp.Edges {
					if e.TargetID == childID {
						count++
					}
				}
				return count, nil
			}
			// Fallthrough to workflow-level counting if blueprint id missing
		}

		wf, err := models.FindWorkflowWithoutOrgScopeInTransaction(tx, node.WorkflowID)
		if err != nil {
			return 0, err
		}

		count := 0
		for _, edge := range wf.Edges {
			if edge.TargetID == node.NodeID {
				count++
			}
		}
		return count, nil
	}

	ctx.CountDistinctIncomingSources = func() (int, error) {
		// Similar blueprint-aware logic as CountIncomingEdges, but count
		// distinct source nodes rather than edge count.
		if node.ParentNodeID != nil && *node.ParentNodeID != "" {
			parent, err := models.FindWorkflowNode(tx, node.WorkflowID, *node.ParentNodeID)
			if err != nil {
				return 0, err
			}

			blueprintID := parent.Ref.Data().Blueprint.ID
			if blueprintID != "" {
				bp, err := models.FindUnscopedBlueprintInTransaction(tx, blueprintID)
				if err != nil {
					return 0, err
				}

				prefix := parent.NodeID + ":"
				childID := node.NodeID
				if len(childID) > len(prefix) && childID[:len(prefix)] == prefix {
					childID = childID[len(prefix):]
				}

				uniq := map[string]struct{}{}
				for _, e := range bp.Edges {
					if e.TargetID == childID {
						uniq[e.SourceID] = struct{}{}
					}
				}
				return len(uniq), nil
			}
		}

		wf, err := models.FindWorkflowWithoutOrgScopeInTransaction(tx, node.WorkflowID)
		if err != nil {
			return 0, err
		}

		uniq := map[string]struct{}{}
		for _, edge := range wf.Edges {
			if edge.TargetID == node.NodeID {
				uniq[edge.SourceID] = struct{}{}
			}
		}
		return len(uniq), nil
	}

	ctx.PassExecution = func(execID uuid.UUID, outputs map[string][]any) (*models.WorkflowNodeExecution, error) {
		exec, err := models.FindNodeExecutionInTransaction(tx, node.WorkflowID, execID)
		if err != nil {
			return nil, err
		}

		exec.PassInTransaction(tx, outputs)

		return exec, nil
	}

	ctx.FailExecution = func(execID uuid.UUID, reason, message string) (*models.WorkflowNodeExecution, error) {
		exec, err := models.FindNodeExecutionInTransaction(tx, node.WorkflowID, execID)
		if err != nil {
			return nil, err
		}

		if err := exec.FailInTransaction(tx, reason, message); err != nil {
			return nil, err
		}

		return exec, nil
	}

	ctx.FindExecutionIDByKV = func(key string, value string) (uuid.UUID, bool, error) {
		exec, err := models.FirstNodeExecutionByKVInTransaction(tx, node.WorkflowID, node.NodeID, key, value)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return uuid.Nil, false, nil
			}

			return uuid.Nil, false, err
		}

		return exec.ID, true, nil
	}

	ctx.SetExecutionKV = func(execID uuid.UUID, key string, value string) error {
		return models.CreateWorkflowNodeExecutionKVInTransaction(tx, node.WorkflowID, node.NodeID, execID, key, value)
	}

	return ctx, nil
}
