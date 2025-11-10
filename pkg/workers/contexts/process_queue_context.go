package contexts

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func BuildProcessQueueContext(tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem) (*components.ProcessQueueContext, error) {
	event, err := models.FindWorkflowEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, err
	}

	config, err := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(event.Data.Data()).
		Build(node.Configuration.Data())
	if err != nil {
		return nil, err
	}

	ctx := &components.ProcessQueueContext{
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: config,
		RootEventID:   queueItem.RootEventID.String(),
		EventID:       event.ID.String(),
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

		messages.NewWorkflowExecutionCreatedMessage(execution.WorkflowID.String(), &execution).PublishWithDelay(1 * time.Second)
		return execution.ID, nil
	}

	ctx.DequeueItem = func() error {
		if err := queueItem.Delete(tx); err != nil {
			return err
		}
		// Notify deletion
		messages.NewWorkflowQueueItemDeletedMessage(queueItem.WorkflowID.String(), queueItem).PublishWithDelay(1 * time.Second)
		return nil
	}

	ctx.UpdateNodeState = func(state string) error {
		return node.UpdateState(tx, state)
	}

	ctx.DefaultProcessing = func() error {
		if _, err := ctx.CreateExecution(); err != nil {
			return err
		}
		if err := ctx.DequeueItem(); err != nil {
			return err
		}
		return ctx.UpdateNodeState(models.WorkflowNodeStateProcessing)
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

		wf, err := models.FindUnscopedWorkflowInTransaction(tx, node.WorkflowID)
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

	ctx.FinishExecution = func(execID uuid.UUID, outputs map[string][]any) error {
		exec, err := models.FindNodeExecutionInTransaction(tx, node.WorkflowID, execID)
		if err != nil {
			return err
		}

		exec.PassInTransaction(tx, outputs)
		messages.NewWorkflowExecutionFinishedMessage(exec.WorkflowID.String(), exec).PublishWithDelay(1 * time.Second)

		return nil
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
