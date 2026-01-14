package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WorkflowNodeQueueWorker struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
	logger    *log.Entry
}

func NewWorkflowNodeQueueWorker(registry *registry.Registry) *WorkflowNodeQueueWorker {
	return &WorkflowNodeQueueWorker{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "WorkflowNodeQueueWorker"}),
	}
}

func (w *WorkflowNodeQueueWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			nodes, err := models.ListWorkflowNodesReady()
			if err != nil {
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordQueueWorkerNodesCount(context.Background(), len(nodes))

			for _, node := range nodes {
				logger := logging.WithNode(w.logger, node)
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(node models.WorkflowNode) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNode(logger, node); err != nil {
						logger.Errorf("Error processing: %v", err)
					}
				}(node)
			}

			telemetry.RecordQueueWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *WorkflowNodeQueueWorker) LockAndProcessNode(logger *log.Entry, node models.WorkflowNode) error {
	var executionIDs []*uuid.UUID
	var queueItem *models.WorkflowNodeQueueItem
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockWorkflowNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			logger.Info("Node already being processed - skipping")
			return nil
		}

		executionIDs, queueItem, err = w.processNode(tx, logger, n)
		return err
	})

	if err == nil {
		if len(executionIDs) > 0 {
			for _, executionID := range executionIDs {
				if executionID == nil {
					continue
				}

				messages.NewWorkflowExecutionMessage(
					node.WorkflowID.String(),
					executionID.String(),
					node.NodeID,
				).Publish()
			}
		}

		if queueItem != nil {
			messages.NewWorkflowQueueItemMessage(
				queueItem.WorkflowID.String(),
				queueItem.ID.String(),
				queueItem.NodeID,
			).Publish(true)
		}
	}

	return err
}

func (w *WorkflowNodeQueueWorker) processNode(tx *gorm.DB, logger *log.Entry, node *models.WorkflowNode) ([]*uuid.UUID, *models.WorkflowNodeQueueItem, error) {
	queueItem, err := node.FirstQueueItem(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	logger = logging.WithQueueItem(logger, *queueItem)
	logger.Info("Processing queue item")

	ctx, err := contexts.BuildProcessQueueContext(w.registry.GetHTTPClient(), tx, node, queueItem)
	if err != nil {

		//
		// If the error returned is not a ConfigurationBuildError,
		// we should retry it, so just return the error as is.
		//
		var configErr *contexts.ConfigurationBuildError
		if !errors.As(err, &configErr) {
			return nil, nil, err
		}

		//
		// If we are dealing with a ConfigurationBuildError,
		// it means that the queue context cannot properly build
		// the configuration for the execution.
		//
		// Since this error will always happen until the user fixes the node configuration,
		// we create a failed execution and delete the queue item.
		//
		logger.Errorf("Error building configuration for node execution: %v", configErr.Error())
		executions, err := w.handleNodeConfigurationError(tx, logger, configErr)
		if err != nil {
			return nil, nil, err
		}

		return executions, queueItem, nil
	}

	var executionID *uuid.UUID
	switch node.Type {
	case models.NodeTypeComponent:
		/*
		 * For component nodes, delegate to the component's ProcessQueueItem implementation to handle
		 * the processing.
		 */
		executionID, err = w.processComponentNode(ctx, node)
	case models.NodeTypeBlueprint:
		/*
		 * For blueprint nodes, use the default processing logic.
		 * Blueprint nodes do not have custom processing logic.
		 */
		executionID, err = ctx.DefaultProcessing()
	default:
		return nil, nil, fmt.Errorf("unsupported node type: %s", node.Type)
	}

	return []*uuid.UUID{executionID}, queueItem, err
}

func (w *WorkflowNodeQueueWorker) processComponentNode(ctx *core.ProcessQueueContext, node *models.WorkflowNode) (*uuid.UUID, error) {
	ref := node.Ref.Data()

	if ref.Component == nil || ref.Component.Name == "" {
		return nil, fmt.Errorf("node %s has no component reference", node.NodeID)
	}

	comp, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return nil, fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	return comp.ProcessQueueItem(*ctx)
}

func (w *WorkflowNodeQueueWorker) handleNodeConfigurationError(tx *gorm.DB, logger *log.Entry, configErr *contexts.ConfigurationBuildError) ([]*uuid.UUID, error) {
	err := configErr.QueueItem.Delete(tx)
	if err != nil {
		return nil, err
	}

	//
	// If we are creating a failed execution for a child node execution,
	// we need to include the parent execution ID and fail the parent as well.
	//
	parentExecutionID, err := w.getParentExecutionID(tx, logger, configErr)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	execution := models.WorkflowNodeExecution{
		WorkflowID:          configErr.QueueItem.WorkflowID,
		NodeID:              configErr.Node.NodeID,
		RootEventID:         configErr.RootEventID,
		EventID:             configErr.Event.ID,
		PreviousExecutionID: configErr.Event.ExecutionID,
		ParentExecutionID:   parentExecutionID,
		State:               models.WorkflowNodeExecutionStateFinished,
		Configuration:       configErr.Node.Configuration,
		Result:              models.WorkflowNodeExecutionResultFailed,
		ResultReason:        models.WorkflowNodeExecutionResultReasonError,
		ResultMessage:       configErr.Err.Error(),
		CreatedAt:           &now,
		UpdatedAt:           &now,
	}

	err = tx.Create(&execution).Error
	if err != nil {
		return nil, err
	}

	if parentExecutionID == nil {
		return []*uuid.UUID{&execution.ID}, nil
	}

	//
	// If this execution has a parent, we need to propagate
	// the failure to the parent execution.
	//
	parent, err := models.FindNodeExecutionInTransaction(tx, execution.WorkflowID, *execution.ParentExecutionID)
	if err != nil {
		return nil, err
	}

	err = parent.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, configErr.Err.Error())
	if err != nil {
		return nil, err
	}

	return []*uuid.UUID{&execution.ID, &parent.ID}, nil
}

func (w *WorkflowNodeQueueWorker) getParentExecutionID(tx *gorm.DB, logger *log.Entry, configErr *contexts.ConfigurationBuildError) (*uuid.UUID, error) {
	if configErr.Event.ExecutionID == nil {
		return nil, nil
	}

	previous, err := models.FindNodeExecutionInTransaction(tx, configErr.Node.WorkflowID, *configErr.Event.ExecutionID)
	if err != nil {
		logger.Errorf("Error finding previous execution: %v", err)
		return nil, err
	}

	return previous.ParentExecutionID, nil
}
