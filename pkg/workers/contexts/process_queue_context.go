package contexts

import (
	"fmt"
	"net/http"
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

func BuildProcessQueueContext(httpClient *http.Client, tx *gorm.DB, node *models.WorkflowNode, queueItem *models.WorkflowNodeQueueItem, configFields []configuration.Field) (*core.ProcessQueueContext, error) {
	event, err := models.FindWorkflowEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, err
	}

	configBuilder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithInput(map[string]any{event.NodeID: event.Data.Data()})
	if len(configFields) > 0 {
		configBuilder = configBuilder.WithConfigurationFields(configFields)
	}

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
	ctx.ExpressionEnv = func(expression string) (map[string]any, error) {
		builder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
			WithRootEvent(&queueItem.RootEventID).
			WithInput(map[string]any{event.NodeID: event.Data.Data()})
		if event.ExecutionID != nil {
			builder = builder.WithPreviousExecution(event.ExecutionID)
		}
		chain, err := builder.BuildMessageChainForExpression(expression)
		if err != nil {
			return nil, err
		}
		return map[string]any{"$": chain}, nil
	}

	ctx.CreateExecution = func() (*core.ExecutionContext, error) {
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

		err := tx.Create(&execution).Error
		if err != nil {
			return nil, err
		}

		return &core.ExecutionContext{
			ID:             execution.ID,
			WorkflowID:     execution.WorkflowID.String(),
			NodeID:         execution.NodeID,
			RootEventID:    execution.RootEventID,
			EventID:        execution.EventID,
			SourceNodeID:   event.NodeID,
			Configuration:  execution.Configuration.Data(),
			HTTP:           NewHTTPContext(httpClient),
			Metadata:       NewExecutionMetadataContext(tx, &execution),
			NodeMetadata:   NewNodeMetadataContext(tx, node),
			ExecutionState: NewExecutionStateContext(tx, &execution),
			Requests:       NewExecutionRequestContext(tx, &execution),
			Logger:         logging.WithExecution(logging.ForNode(*node), &execution, nil),
			Notifications:  NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
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

		if err := ctx.UpdateNodeState(models.WorkflowNodeStateProcessing); err != nil {
			return nil, err
		}

		return &executionCtx.ID, nil
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
			RootEventID:    execution.RootEventID,
			EventID:        execution.EventID,
			Configuration:  execution.Configuration.Data(),
			HTTP:           NewHTTPContext(httpClient),
			Metadata:       NewExecutionMetadataContext(tx, execution),
			NodeMetadata:   NewNodeMetadataContext(tx, node),
			ExecutionState: NewExecutionStateContext(tx, execution),
			Requests:       NewExecutionRequestContext(tx, execution),
			Logger:         logging.WithExecution(logging.ForNode(*node), execution, nil),
			Notifications:  NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
		}, nil
	}

	return ctx, nil
}
