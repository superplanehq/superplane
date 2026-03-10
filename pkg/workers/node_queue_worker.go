package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type NodeQueueWorker struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
	logger    *log.Entry

	rabbitMQURL string
	consumer    *tackle.Consumer
}

func NewNodeQueueWorker(registry *registry.Registry, rabbitMQURL string) *NodeQueueWorker {
	return &NodeQueueWorker{
		registry:    registry,
		rabbitMQURL: rabbitMQURL,
		semaphore:   semaphore.NewWeighted(25),
		logger:      log.WithFields(log.Fields{"worker": "NodeQueueWorker"}),
	}
}

func (w *NodeQueueWorker) Name() string {
	return "NodeQueueWorker"
}

func (w *NodeQueueWorker) Start(ctx context.Context) {
	go w.StartRabbitMQConsumer(ctx)

	//
	// Differently from the other workers, the NodeQueueWorker needs to be
	// aware of two things: queue items being created and nodes becoming ready.
	//
	// Since we don't have events for nodes becoming ready, we need to poll still,
	// so we cannot decrease this interval yet.
	//
	// Once we have events for nodes becoming ready,
	// we can make this worker react to both events.
	//
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			nodes, err := models.ListCanvasNodesReady()
			if err != nil {
				w.logger.Errorf("Error finding canvas nodes ready to be processed: %v", err)
			}

			telemetry.RecordQueueWorkerNodesCount(context.Background(), len(nodes))

			for _, node := range nodes {
				logger := logging.WithNode(w.logger, node)
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(node models.CanvasNode) {
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

func (w *NodeQueueWorker) StartRabbitMQConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name(),
		RemoteExchange: messages.WorkflowExchange,
		Service:        messages.WorkflowExchange + "." + messages.WorkflowQueueItemCreatedRoutingKey + "." + w.Name(),
		RoutingKey:     messages.WorkflowQueueItemCreatedRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))
	w.consumer = consumer

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.WorkflowQueueItemCreatedRoutingKey)

		err := w.consumer.Start(&options, w.Consume)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.WorkflowQueueItemCreatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.WorkflowQueueItemCreatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *NodeQueueWorker) Consume(delivery tackle.Delivery) error {
	data := &pb.CanvasNodeQueueItemMessage{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		w.logger.Errorf("Error unmarshaling canvas queue item created message: %v", err)
		return err
	}

	canvasID, err := uuid.Parse(data.CanvasId)
	if err != nil {
		w.logger.Errorf("Error parsing canvas id: %v", err)
		return err
	}

	node, err := models.FindCanvasNode(database.Conn(), canvasID, data.NodeId)
	if err != nil {
		w.logger.Errorf("Error finding canvas node: %v", err)
		return err
	}

	//
	// New queue item created for a node that is not ready, we should skip it.
	//
	if node.State != models.CanvasNodeStateReady {
		w.logger.Infof("Node %s is not ready, skipping", node.NodeID)
		return nil
	}

	//
	// Node is ready for processing, let's lock it and process it.
	//
	logger := logging.WithNode(w.logger, *node)
	return w.LockAndProcessNode(logger, *node)
}

func (w *NodeQueueWorker) LockAndProcessNode(logger *log.Entry, node models.CanvasNode) error {
	var executionIDs []*uuid.UUID
	var queueItem *models.CanvasNodeQueueItem

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			logger.Info("Node already being processed - skipping")
			return nil
		}

		executionIDs, queueItem, err = w.processNode(tx, logger, n, onNewEvents)
		return err
	})

	if err == nil {
		if len(executionIDs) > 0 {
			for _, executionID := range executionIDs {
				if executionID == nil {
					continue
				}

				messages.NewCanvasExecutionMessage(
					node.WorkflowID.String(),
					executionID.String(),
					node.NodeID,
				).Publish()
			}
		}

		if queueItem != nil {
			messages.NewCanvasQueueItemMessage(
				queueItem.WorkflowID.String(),
				queueItem.ID.String(),
				queueItem.NodeID,
			).Publish(true)
		}

		for _, event := range newEvents {
			messages.NewCanvasEventCreatedMessage(event.WorkflowID.String(), &event).Publish()
		}
	}

	return err
}

func (w *NodeQueueWorker) processNode(tx *gorm.DB, logger *log.Entry, node *models.CanvasNode, onNewEvents func([]models.CanvasEvent)) ([]*uuid.UUID, *models.CanvasNodeQueueItem, error) {
	queueItem, err := node.FirstQueueItem(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	logger = logging.WithQueueItem(logger, *queueItem)
	logger.Info("Processing queue item")

	configFields, err := w.configurationFieldsForNode(tx, node)
	if err != nil {
		return nil, nil, err
	}

	ctx, err := contexts.BuildProcessQueueContext(w.registry.HTTPContext(), tx, node, queueItem, configFields, onNewEvents)
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

func (w *NodeQueueWorker) configurationFieldsForNode(tx *gorm.DB, node *models.CanvasNode) ([]configuration.Field, error) {
	ref := node.Ref.Data()
	switch node.Type {
	case models.NodeTypeComponent:
		if ref.Component == nil || ref.Component.Name == "" {
			return nil, fmt.Errorf("node %s has no component reference", node.NodeID)
		}

		comp, err := w.registry.GetComponent(ref.Component.Name)
		if err != nil {
			return nil, fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
		}

		return comp.Configuration(), nil
	case models.NodeTypeBlueprint:
		if ref.Blueprint == nil || ref.Blueprint.ID == "" {
			return nil, fmt.Errorf("node %s has no blueprint reference", node.NodeID)
		}

		blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, ref.Blueprint.ID)
		if err != nil {
			return nil, fmt.Errorf("blueprint %s not found: %w", ref.Blueprint.ID, err)
		}

		return blueprint.Configuration, nil
	default:
		return nil, nil
	}
}

func (w *NodeQueueWorker) processComponentNode(ctx *core.ProcessQueueContext, node *models.CanvasNode) (*uuid.UUID, error) {
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

func (w *NodeQueueWorker) handleNodeConfigurationError(tx *gorm.DB, logger *log.Entry, configErr *contexts.ConfigurationBuildError) ([]*uuid.UUID, error) {
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
	execution := models.CanvasNodeExecution{
		WorkflowID:          configErr.QueueItem.WorkflowID,
		NodeID:              configErr.Node.NodeID,
		RootEventID:         configErr.RootEventID,
		EventID:             configErr.Event.ID,
		PreviousExecutionID: configErr.Event.ExecutionID,
		ParentExecutionID:   parentExecutionID,
		State:               models.CanvasNodeExecutionStateFinished,
		Configuration:       configErr.Node.Configuration,
		Result:              models.CanvasNodeExecutionResultFailed,
		ResultReason:        models.CanvasNodeExecutionResultReasonError,
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

	err = parent.FailInTransaction(tx, models.CanvasNodeExecutionResultReasonError, configErr.Err.Error())
	if err != nil {
		return nil, err
	}

	return []*uuid.UUID{&execution.ID, &parent.ID}, nil
}

func (w *NodeQueueWorker) getParentExecutionID(tx *gorm.DB, logger *log.Entry, configErr *contexts.ConfigurationBuildError) (*uuid.UUID, error) {
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
