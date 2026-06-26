package contexts

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
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

func BuildProcessQueueContext(
	httpCtx core.HTTPContext,
	encryptor crypto.Encryptor,
	tx *gorm.DB,
	node *models.CanvasNode,
	queueItem *models.CanvasNodeQueueItem,
	configFields []configuration.Field,
	onNewEvents func([]models.CanvasEvent),
	repoFiles core.RepositoryFilesContext,
) (*core.ProcessQueueContext, error) {
	event, err := models.FindCanvasEventInTransaction(tx, queueItem.EventID)
	if err != nil {
		return nil, err
	}

	workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, node.WorkflowID)
	if err != nil {
		return nil, err
	}

	configBuilder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithNodeID(node.NodeID).
		WithRootEvent(&queueItem.RootEventID).
		WithPreviousExecution(event.ExecutionID).
		WithIncomingEventID(&event.ID).
		WithInput(map[string]any{event.NodeID: event.Data.Data()})
	if len(configFields) > 0 {
		configBuilder = configBuilder.WithConfigurationFields(configFields)
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

	//
	// The Expressions context evaluates bare expressions (e.g. Merge's
	// stopIfExpression, Loop's untilExpression) immediately during queue
	// processing to decide control flow. Those results are never persisted,
	// so we give this builder a runtime secret resolver - canvas validation
	// allows secrets() in these fields, and without a resolver evaluation
	// would fail with "secrets() is not available in this context".
	//
	secretResolver := NewRuntimeSecretResolver(tx, encryptor, models.DomainTypeOrganization, workflow.OrganizationID)
	builder := NewNodeConfigurationBuilder(tx, queueItem.WorkflowID).
		WithNodeID(node.NodeID).
		WithRootEvent(&queueItem.RootEventID).
		WithIncomingEventID(&event.ID).
		WithInput(map[string]any{event.NodeID: event.Data.Data()}).
		WithSecretResolver(secretResolver)
	if event.ExecutionID != nil {
		builder = builder.WithPreviousExecution(event.ExecutionID)
	}

	ctx := &core.ProcessQueueContext{
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: config,
		RootEventID:   queueItem.RootEventID.String(),
		EventID:       event.ID.String(),
		SourceNodeID:  event.NodeID,
		Input:         map[string]any{event.NodeID: normalizeExpressionValue(event.Data.Data())},
		Expressions:   NewExpressionContext(builder),
	}

	ctx.CreateExecution = func() (*core.ExecutionContext, error) {
		now := time.Now()

		execution := models.CanvasNodeExecution{
			WorkflowID:          queueItem.WorkflowID,
			NodeID:              node.NodeID,
			RootEventID:         queueItem.RootEventID,
			RunID:               queueItem.RunID,
			EventID:             event.ID,
			PreviousExecutionID: event.ExecutionID,
			State:               models.CanvasNodeExecutionStatePending,
			Configuration:       datatypes.NewJSONType(config),
			CreatedAt:           &now,
			UpdatedAt:           &now,
		}

		err := tx.Create(&execution).Error
		if err != nil {
			return nil, err
		}

		orgID := ""
		canvasName := ""
		if workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID); err == nil && workflow != nil {
			orgID = workflow.OrganizationID.String()
			canvasName = workflow.Name
		}

		return &core.ExecutionContext{
			ID:             execution.ID,
			WorkflowID:     execution.WorkflowID.String(),
			OrganizationID: orgID,
			CanvasName:     canvasName,
			NodeID:         execution.NodeID,
			NodeName:       node.Name,
			Configuration:  execution.Configuration.Data(),
			HTTP:           httpCtx,
			Metadata:       NewExecutionMetadataContext(tx, &execution),
			NodeMetadata:   NewNodeMetadataContext(tx, node),
			ExecutionState: NewExecutionStateContext(tx, &execution, onNewEvents),
			Requests:       NewExecutionRequestContext(tx, &execution),
			Logger:         logging.WithExecution(logging.ForNode(*node), &execution),
			CanvasMemory:   NewCanvasMemoryContext(tx, execution.WorkflowID),
			Files:          repoFiles,
		}, nil
	}

	ctx.DequeueItem = func() error {
		return queueItem.Delete(tx)
	}

	//
	// The node queue is a FIFO ordered by created_at (see CanvasNode.FirstQueueItem),
	// so deferring an item is simply moving it to the tail by stamping created_at to
	// now. This lets other already-queued items (e.g. feedback for an in-progress run)
	// be processed ahead of it, instead of the worker re-picking this same item forever.
	//
	ctx.DeferQueueItem = func() error {
		now := time.Now()
		return tx.Model(queueItem).Update("created_at", &now).Error
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

	ctx.DistinctIncomingSources = func() ([]core.Node, error) {
		wf, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, node.WorkflowID)
		if err != nil {
			return nil, err
		}
		_, liveEdges, err := models.FindLiveCanvasSpecInTransaction(tx, wf.ID)
		if err != nil {
			return nil, err
		}

		sources := []core.Node{}
		for _, edge := range liveEdges {
			if edge.TargetID == node.NodeID {
				sources = append(sources, core.Node{
					ID: edge.SourceID,
				})
			}
		}
		return uniqueSourceNodes(sources), nil
	}

	ctx.FindExecutionByKV = func(key string, value string) (*core.ExecutionContext, error) {
		execution, err := models.FirstNodeExecutionByKVInTransaction(tx, node.WorkflowID, node.NodeID, key, value)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil
			}

			return nil, err
		}

		orgID := ""
		canvasName := ""
		if workflow, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID); err == nil && workflow != nil {
			orgID = workflow.OrganizationID.String()
			canvasName = workflow.Name
		}

		return &core.ExecutionContext{
			ID:             execution.ID,
			WorkflowID:     execution.WorkflowID.String(),
			OrganizationID: orgID,
			CanvasName:     canvasName,
			NodeID:         execution.NodeID,
			NodeName:       node.Name,
			Configuration:  execution.Configuration.Data(),
			HTTP:           httpCtx,
			Metadata:       NewExecutionMetadataContext(tx, execution),
			NodeMetadata:   NewNodeMetadataContext(tx, node),
			ExecutionState: NewExecutionStateContext(tx, execution, onNewEvents),
			Requests:       NewExecutionRequestContext(tx, execution),
			Logger:         logging.WithExecution(logging.ForNode(*node), execution),
			CanvasMemory:   NewCanvasMemoryContext(tx, execution.WorkflowID),
			Files:          repoFiles,
		}, nil
	}

	ctx.HasRunningExecutions = func() (bool, error) {
		count, err := models.CountRunningExecutionsForNodeInTransaction(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			return false, err
		}
		return count > 0, nil
	}

	return ctx, nil
}

func uniqueSourceNodes(nodes []core.Node) []core.Node {
	unique := []core.Node{}
	for _, node := range nodes {
		if !slices.ContainsFunc(unique, func(a core.Node) bool { return a.ID == node.ID }) {
			unique = append(unique, node)
		}
	}
	return unique
}
