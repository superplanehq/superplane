package contexts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ConfigurationBuildError struct {
	Err         error
	QueueItem   *models.CanvasNodeQueueItem
	Node        *models.CanvasNode
	Event       *models.CanvasEvent
	RootEventID uuid.UUID
}

func (e *ConfigurationBuildError) Error() string {
	return fmt.Sprintf("configuration build failed: %v", e.Err)
}

func (e *ConfigurationBuildError) Unwrap() error {
	return e.Err
}

func BuildProcessQueueContext(httpCtx core.HTTPContext, tx *gorm.DB, node *models.CanvasNode, queueItem *models.CanvasNodeQueueItem, configFields []configuration.Field) (*core.ProcessQueueContext, error) {
	event, err := models.FindCanvasEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, err
	}

	configBuilder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithNodeID(node.NodeID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(map[string]any{event.NodeID: event.Data.Data()})
	if len(configFields) > 0 {
		configBuilder = configBuilder.WithConfigurationFields(configFields)
	}

	if node.ParentNodeID != nil {
		parent, err := models.FindCanvasNode(tx, node.WorkflowID, *node.ParentNodeID)
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
	ctx.ExpressionEnv = func(expression string) (map[string]any, error) {
		builder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
			WithNodeID(node.NodeID).
			WithRootEvent(&queueItem.RootEventID).
			WithInput(map[string]any{event.NodeID: event.Data.Data()})
		if event.ExecutionID != nil {
			builder = builder.WithPreviousExecution(event.ExecutionID)
		}
		return builder.BuildExpressionEnv(expression)
	}

	ctx.CreateExecution = func() (*core.ExecutionContext, error) {
		now := time.Now()

		execution := models.CanvasNodeExecution{
			WorkflowID:          queueItem.WorkflowID,
			NodeID:              node.NodeID,
			RootEventID:         queueItem.RootEventID,
			EventID:             event.ID,
			PreviousExecutionID: event.ExecutionID,
			State:               models.CanvasNodeExecutionStatePending,
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

		err := tx.Create(&execution).Error
		if err != nil {
			return nil, err
		}

		return &core.ExecutionContext{
			ID:             execution.ID,
			WorkflowID:     execution.WorkflowID.String(),
			NodeID:         execution.NodeID,
			Configuration:  execution.Configuration.Data(),
			HTTP:           httpCtx,
			Metadata:       NewExecutionMetadataContext(tx, &execution),
			NodeMetadata:   NewNodeMetadataContext(tx, node),
			ExecutionState: NewExecutionStateContext(tx, &execution),
			Requests:       NewExecutionRequestContext(tx, &execution),
			Logger:         logging.WithExecution(logging.ForNode(*node), &execution, nil),
			Notifications:  NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
			CanvasMemory:   NewCanvasMemoryContext(tx, execution.WorkflowID),
		}, nil
	}

	ctx.DequeueItem = func() error {
		return queueItem.Delete(tx)
	}

	ctx.UpdateNodeState = func(state string) error {
		return node.UpdateState(tx, state)
	}

	ctx.DefaultProcessing = func() (*uuid.UUID, error) {
		executionCtx, err := ctx.CreateExecution()
		if err != nil {
			return nil, err
		}

		if err := ctx.DequeueItem(); err != nil {
			return nil, err
		}

		if err := ctx.UpdateNodeState(models.CanvasNodeStateProcessing); err != nil {
			return nil, err
		}

		return &executionCtx.ID, nil
	}

	ctx.CountDistinctIncomingSources = func() (int, error) {
		// Similar blueprint-aware logic as CountIncomingEdges, but count
		// distinct source nodes rather than edge count.
		if node.ParentNodeID != nil && *node.ParentNodeID != "" {
			parent, err := models.FindCanvasNode(tx, node.WorkflowID, *node.ParentNodeID)
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

		wf, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, node.WorkflowID)
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

	ctx.FindExecutionByKV = func(key string, value string) (*core.ExecutionContext, error) {
		execution, err := models.FirstNodeExecutionByKVInTransaction(tx, node.WorkflowID, node.NodeID, key, value)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil
			}

			return nil, err
		}

		return &core.ExecutionContext{
			ID:             execution.ID,
			WorkflowID:     execution.WorkflowID.String(),
			NodeID:         execution.NodeID,
			Configuration:  execution.Configuration.Data(),
			HTTP:           httpCtx,
			Metadata:       NewExecutionMetadataContext(tx, execution),
			NodeMetadata:   NewNodeMetadataContext(tx, node),
			ExecutionState: NewExecutionStateContext(tx, execution),
			Requests:       NewExecutionRequestContext(tx, execution),
			Logger:         logging.WithExecution(logging.ForNode(*node), execution, nil),
			Notifications:  NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
			CanvasMemory:   NewCanvasMemoryContext(tx, execution.WorkflowID),
		}, nil
	}

	return ctx, nil
}
