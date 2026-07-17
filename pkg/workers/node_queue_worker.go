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
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type NodeQueueWorker struct {
	registry    *registry.Registry
	gitProvider gitprovider.Provider
	semaphore   *semaphore.Weighted
	logger      *log.Entry

	rabbitMQURL               string
	queueItemConsumer         *tackle.Consumer
	executionFinishedConsumer *tackle.Consumer
}

func NewNodeQueueWorker(registry *registry.Registry, gitProvider gitprovider.Provider, rabbitMQURL string) *NodeQueueWorker {
	logger := log.WithFields(log.Fields{"worker": "NodeQueueWorker"})

	queueItemConsumer := tackle.NewConsumer()
	queueItemConsumer.SetLogger(logging.NewTackleLogger(logger))

	executionFinishedConsumer := tackle.NewConsumer()
	executionFinishedConsumer.SetLogger(logging.NewTackleLogger(logger))

	return &NodeQueueWorker{
		registry:                  registry,
		gitProvider:               gitProvider,
		rabbitMQURL:               rabbitMQURL,
		semaphore:                 semaphore.NewWeighted(25),
		logger:                    logger,
		queueItemConsumer:         queueItemConsumer,
		executionFinishedConsumer: executionFinishedConsumer,
	}
}

func (w *NodeQueueWorker) Name() string {
	return "NodeQueueWorker"
}

func (w *NodeQueueWorker) Start(ctx context.Context) {
	go w.startConsumerLoop(
		ctx,
		w.queueItemConsumer,
		messages.CanvasExchange+"."+messages.CanvasQueueItemCreatedRoutingKey+"."+w.Name(),
		messages.CanvasExchange,
		messages.CanvasQueueItemCreatedRoutingKey,
		w.ConsumeQueueItemCreated,
	)

	go w.startConsumerLoop(
		ctx,
		w.executionFinishedConsumer,
		messages.ExecutionsExchange+"."+messages.ExecutionFinishedRoutingKey+"."+w.Name(),
		messages.ExecutionsExchange,
		messages.ExecutionFinishedRoutingKey,
		w.ConsumeExecutionFinished,
	)

	//
	// Slow safety-net poll in case RabbitMQ is not working.
	//
	ticker := time.NewTicker(time.Minute)
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
					attemptStart := time.Now()
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNode(logger, node, attemptStart); err != nil {
						logger.Errorf("Error processing: %v", err)
					}
				}(node)
			}

			telemetry.RecordQueueWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *NodeQueueWorker) startConsumerLoop(
	ctx context.Context,
	consumer *tackle.Consumer,
	serviceName string,
	exchangeName string,
	routingKey string,
	handler func(tackle.Delivery) error,
) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name(),
		RemoteExchange: exchangeName,
		Service:        serviceName,
		RoutingKey:     routingKey,
	}

	for {
		if ctx.Err() != nil {
			return
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", routingKey)

		err := consumer.Start(&options, handler)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", routingKey, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", routingKey)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (w *NodeQueueWorker) ConsumeQueueItemCreated(delivery tackle.Delivery) error {
	start := time.Now()

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

	return w.tryProcessReadyNode(canvasID, data.NodeId, start)
}

func (w *NodeQueueWorker) ConsumeExecutionFinished(delivery tackle.Delivery) error {
	start := time.Now()

	data := &pb.CanvasNodeExecutionMessage{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		w.logger.Errorf("Error unmarshaling canvas execution finished message: %v", err)
		return err
	}

	canvasID, err := uuid.Parse(data.CanvasId)
	if err != nil {
		w.logger.Errorf("Error parsing canvas id: %v", err)
		return err
	}

	return w.tryProcessReadyNode(canvasID, data.NodeId, start)
}

func (w *NodeQueueWorker) tryProcessReadyNode(canvasID uuid.UUID, nodeID string, attemptStart time.Time) error {
	node, err := models.FindCanvasNode(database.Conn(), canvasID, nodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.logger.Infof("Node %s not found, skipping", nodeID)
			telemetry.RecordQueueWorkerNodeProcessing(
				context.Background(),
				time.Since(attemptStart),
				executorOutcomeSkipped,
				executorReasonNotFound,
			)
			return nil
		}

		w.logger.Errorf("Error finding canvas node: %v", err)
		return err
	}

	//
	// Node is not ready yet, skip it. For queue-item-created messages this happens
	// when a new item arrives while the node is still executing. For
	// execution-finished messages this can happen if another worker has already
	// moved the node into a non-ready state.
	//
	if node.State != models.CanvasNodeStateReady {
		w.logger.Infof("Node %s is not ready, skipping", node.NodeID)
		telemetry.RecordQueueWorkerNodeProcessing(
			context.Background(),
			time.Since(attemptStart),
			executorOutcomeSkipped,
			executorReasonNone,
		)
		return nil
	}

	logger := logging.WithNode(w.logger, *node)
	return w.LockAndProcessNode(logger, *node, attemptStart)
}

func (w *NodeQueueWorker) LockAndProcessNode(logger *log.Entry, node models.CanvasNode, attemptStart time.Time) error {
	//
	// For every node we process, we track the following metrics:
	// - outcome: success, failed, skipped
	// - reason: none, locked, deadlock, not_found, internal
	//
	metricOutcome := executorOutcomeSuccess
	metricReason := executorReasonNone
	defer func() {
		telemetry.RecordQueueWorkerNodeProcessing(
			context.Background(),
			time.Since(attemptStart),
			metricOutcome,
			metricReason,
		)
	}()

	messageCollector := NewMessageCollector(node.WorkflowID, logger)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			logger.Info("Node already being processed - skipping")
			metricOutcome = executorOutcomeSkipped
			metricReason = executorReasonLocked
			return nil
		}

		err = w.processNodeQueueItem(tx, logger, n, messageCollector)
		if err != nil {
			metricOutcome = executorOutcomeFailed
			metricReason = classifyProcessError(err)
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	messageCollector.Publish()
	return nil
}

func (w *NodeQueueWorker) processNodeQueueItem(tx *gorm.DB, logger *log.Entry, node *models.CanvasNode, collector *MessageCollector) error {
	item, err := node.FirstQueueItem(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	logger = logging.WithQueueItem(logger, *item)
	logger.Info("Processing queue item")

	configFields, err := w.configurationFieldsForNode(node)
	if err != nil {
		return err
	}

	ctx, err := contexts.BuildProcessQueueContext(
		w.registry.HTTPContextInTransaction(tx),
		tx,
		node,
		item,
		configFields,
		collector.OnNewEvents,
		contexts.NewRepositoryFilesContext(w.gitProvider, item.WorkflowID),
	)

	if err != nil {

		//
		// If the error returned is not a ConfigurationBuildError,
		// we should retry it, so just return the error as is.
		//
		var configErr *contexts.ConfigurationBuildError
		if !errors.As(err, &configErr) {
			return err
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
		return w.handleNodeConfigurationError(tx, configErr, collector)
	}

	//
	// Check if the run is cancelling.
	// If it is, we should not create new executions,
	// and instead, delete the queue item and return.
	//
	run, err := models.FindCanvasRunInTransaction(tx, item.WorkflowID, item.RunID)
	if err != nil {
		return err
	}

	if run.State == models.CanvasRunStateCancelling {
		if err := tx.Delete(item).Error; err != nil {
			return err
		}

		logger.Infof("Skipping queue item for cancelling run %s", item.RunID)
		collector.AddQueueItemDeleted(item)
		return nil
	}

	switch node.Type {
	case models.NodeTypeComponent:
		/*
		 * For component nodes, delegate to the component's ProcessQueueItem implementation to handle
		 * the processing.
		 */
		err = w.processComponentNode(ctx, node, collector)
	default:
		return fmt.Errorf("unsupported node type: %s", node.Type)
	}

	if errors.Is(err, core.ErrQueueItemDeferred) {
		logger.Info("Queue item deferred")
		return nil
	}

	if err != nil {
		return err
	}

	collector.AddQueueItemConsumed(item)
	return nil
}

func (w *NodeQueueWorker) configurationFieldsForNode(node *models.CanvasNode) ([]configuration.Field, error) {
	ref := node.Ref.Data()
	switch node.Type {
	case models.NodeTypeComponent:
		if ref.Component == nil || ref.Component.Name == "" {
			return nil, fmt.Errorf("node %s has no component reference", node.NodeID)
		}

		action, err := w.registry.GetAction(ref.Component.Name)
		if err != nil {
			return nil, fmt.Errorf("action %s not found: %w", ref.Component.Name, err)
		}

		return action.Configuration(), nil
	default:
		return nil, nil
	}
}

func (w *NodeQueueWorker) processComponentNode(ctx *core.ProcessQueueContext, node *models.CanvasNode, collector *MessageCollector) error {
	ref := node.Ref.Data()

	if ref.Component == nil || ref.Component.Name == "" {
		return fmt.Errorf("node %s has no component reference", node.NodeID)
	}

	action, err := w.registry.GetAction(ref.Component.Name)
	if err != nil {
		return fmt.Errorf("action %s not found: %w", ref.Component.Name, err)
	}

	executionID, err := action.ProcessQueueItem(*ctx)
	if err != nil {
		return err
	}

	collector.AddExecutionID(executionID)
	return nil
}

func (w *NodeQueueWorker) handleNodeConfigurationError(tx *gorm.DB, configErr *contexts.ConfigurationBuildError, collector *MessageCollector) error {
	err := configErr.QueueItem.Delete(tx)
	if err != nil {
		return err
	}

	collector.AddQueueItemConsumed(configErr.QueueItem)

	now := time.Now()
	execution := models.CanvasNodeExecution{
		WorkflowID:          configErr.QueueItem.WorkflowID,
		NodeID:              configErr.Node.NodeID,
		RootEventID:         configErr.RootEventID,
		RunID:               configErr.QueueItem.RunID,
		EventID:             configErr.Event.ID,
		PreviousExecutionID: configErr.Event.ExecutionID,
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
		return err
	}

	//
	// The errored node could not execute, so notify the canvas' On Error nodes.
	//
	contexts.DispatchOnError(tx, &execution, collector.OnNewEvents)
	collector.AddExecutionID(&execution.ID)
	return nil
}

/*
 * To avoid using return values to keep track of which messages need to be published
 * when the transaction is committed, we use a collector.
 */
type MessageCollector struct {
	workflowID         uuid.UUID
	logger             *log.Entry
	executionIDs       []*uuid.UUID
	events             []models.CanvasEvent
	queueItemsConsumed []*models.CanvasNodeQueueItem
	queueItemsDeleted  []*models.CanvasNodeQueueItem
}

func NewMessageCollector(workflowID uuid.UUID, logger *log.Entry) *MessageCollector {
	return &MessageCollector{
		workflowID:         workflowID,
		logger:             logger,
		events:             make([]models.CanvasEvent, 0),
		executionIDs:       make([]*uuid.UUID, 0),
		queueItemsConsumed: make([]*models.CanvasNodeQueueItem, 0),
		queueItemsDeleted:  make([]*models.CanvasNodeQueueItem, 0),
	}
}

func (c *MessageCollector) AddExecutionID(id *uuid.UUID) {
	c.executionIDs = append(c.executionIDs, id)
}

func (c *MessageCollector) AddQueueItemConsumed(item *models.CanvasNodeQueueItem) {
	c.queueItemsConsumed = append(c.queueItemsConsumed, item)
}

func (c *MessageCollector) AddQueueItemDeleted(item *models.CanvasNodeQueueItem) {
	c.queueItemsDeleted = append(c.queueItemsDeleted, item)
}

func (c *MessageCollector) OnNewEvents(events []models.CanvasEvent) {
	c.events = append(c.events, events...)
}

func (c *MessageCollector) Publish() {
	for _, executionID := range c.executionIDs {
		if executionID == nil {
			continue
		}

		if err := messages.PublishCanvasExecutionByID(c.workflowID, *executionID); err != nil {
			c.logger.Errorf("Error publishing execution state: %v", err)
		}
	}

	for _, event := range c.events {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	for _, queueItem := range c.queueItemsConsumed {
		err := messages.NewCanvasQueueItemMessage(*queueItem).PublishConsumed()
		if err != nil {
			c.logger.Errorf("Error publishing queue item consumed message: %v", err)
		}
	}

	for _, queueItem := range c.queueItemsDeleted {
		err := messages.NewCanvasQueueItemMessage(*queueItem).PublishDeleted()
		if err != nil {
			c.logger.Errorf("Error publishing queue item deleted message: %v", err)
		}
	}
}
