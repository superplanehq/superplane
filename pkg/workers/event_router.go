package workers

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type EventRouter struct {
	semaphore   *semaphore.Weighted
	logger      *log.Entry
	rabbitMQURL string
	consumer    *tackle.Consumer
}

func NewEventRouter(rabbitMQURL string) *EventRouter {
	return &EventRouter{
		semaphore:   semaphore.NewWeighted(25),
		logger:      log.WithFields(log.Fields{"worker": "EventRouter"}),
		rabbitMQURL: rabbitMQURL,
	}
}

func (w *EventRouter) Name() string {
	return "EventRouter"
}

func (w *EventRouter) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	go w.StartRabbitMQConsumer(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			events, err := models.ListPendingCanvasEvents()
			if err != nil {
				w.logger.Errorf("Error finding canvas nodes ready to be processed: %v", err)
			}

			telemetry.RecordEventWorkerEventsCount(context.Background(), len(events))

			for _, event := range events {
				logger := logging.ForEvent(w.logger, event)
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(event models.CanvasEvent) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessEvent(logger, event); err != nil {
						w.logger.Errorf("Error processing event %s: %v", event.ID, err)
					}
				}(event)
			}

			telemetry.RecordEventWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *EventRouter) StartRabbitMQConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name(),
		RemoteExchange: messages.WorkflowExchange,
		Service:        messages.WorkflowExchange + "." + messages.WorkflowEventCreatedRoutingKey + "." + w.Name(),
		RoutingKey:     messages.WorkflowEventCreatedRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))
	w.consumer = consumer

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.WorkflowEventCreatedRoutingKey)

		err := w.consumer.Start(&options, w.Consume)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.WorkflowEventCreatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.WorkflowEventCreatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *EventRouter) Consume(delivery tackle.Delivery) error {
	data := &pb.CanvasNodeEventMessage{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		w.logger.Errorf("Error unmarshaling canvas event message: %v", err)
		return err
	}

	eventID, err := uuid.Parse(data.Id)
	if err != nil {
		w.logger.Errorf("Error parsing event id: %v", err)
		return err
	}

	event, err := models.FindCanvasEvent(eventID)
	if err != nil {
		w.logger.Errorf("Error finding canvas event: %v", err)
		return err
	}

	if event.State == models.CanvasEventStateRouted {
		w.logger.Infof("Event %s is already routed - skipping", event.ID)
		return nil
	}

	logger := logging.ForEvent(w.logger, *event)
	return w.LockAndProcessEvent(logger, *event)
}

func (w *EventRouter) LockAndProcessEvent(logger *log.Entry, event models.CanvasEvent) error {
	var createdQueueItems []models.CanvasNodeQueueItem
	var execution *models.CanvasNodeExecution
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		event, err := models.LockCanvasEvent(tx, event.ID)
		if err != nil {
			logger.Info("Event already being processed - skipping")
			return nil
		}

		createdQueueItems, execution, err = w.processEvent(tx, logger, event)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(createdQueueItems) > 0 {
		for _, queueItem := range createdQueueItems {
			messages.NewCanvasQueueItemMessage(
				event.WorkflowID.String(),
				queueItem.ID.String(),
				queueItem.NodeID,
			).Publish(false)
		}
	}

	if execution != nil {
		messages.NewCanvasExecutionMessage(
			event.WorkflowID.String(),
			execution.ID.String(),
			execution.NodeID,
		).Publish()
	}

	return nil
}

func (w *EventRouter) processEvent(tx *gorm.DB, logger *log.Entry, event *models.CanvasEvent) ([]models.CanvasNodeQueueItem, *models.CanvasNodeExecution, error) {
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, event.WorkflowID)
	if err != nil {
		return nil, nil, err
	}

	_, liveEdges, err := models.FindLiveCanvasSpecInTransaction(tx, canvas.ID)
	if err != nil {
		return nil, nil, err
	}

	if event.ExecutionID == nil {
		queueItems, err := w.processRootEvent(tx, canvas, liveEdges, event)
		return queueItems, nil, err
	}

	execution, err := models.FindNodeExecutionInTransaction(tx, event.WorkflowID, *event.ExecutionID)
	if err != nil {
		return nil, nil, err
	}

	if execution.ParentExecutionID != nil {
		return w.processChildExecutionEvent(tx, logger, canvas, execution, event)
	}

	queueItems, err := w.processExecutionEvent(tx, logger, canvas, liveEdges, execution, event)
	return queueItems, execution, err
}

func findOutgoingEdges(edges []models.Edge, sourceID string, channel string) []models.Edge {
	matches := make([]models.Edge, 0, len(edges))
	for _, edge := range edges {
		if edge.SourceID == sourceID && edge.Channel == channel {
			matches = append(matches, edge)
		}
	}

	return matches
}

func (w *EventRouter) processRootEvent(tx *gorm.DB, canvas *models.Canvas, edges []models.Edge, event *models.CanvasEvent) ([]models.CanvasNodeQueueItem, error) {
	now := time.Now()

	w.logger.Infof("Processing root event %s", event.ID)

	outgoingEdges := findOutgoingEdges(edges, event.NodeID, event.Channel)
	var queueItems []models.CanvasNodeQueueItem
	for _, edge := range outgoingEdges {
		targetNode, err := models.FindCanvasNode(tx, canvas.ID, edge.TargetID)
		if err != nil {
			return nil, err
		}

		if targetNode.State == models.CanvasNodeStateError {
			continue
		}

		queueItem := models.CanvasNodeQueueItem{
			WorkflowID:  canvas.ID,
			NodeID:      targetNode.NodeID,
			RootEventID: event.ID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return nil, err
		}

		queueItems = append(queueItems, queueItem)
	}

	err := event.RoutedInTransaction(tx)
	if err != nil {
		return nil, err
	}

	return queueItems, nil
}

func (w *EventRouter) processExecutionEvent(
	tx *gorm.DB,
	logger *log.Entry,
	canvas *models.Canvas,
	edges []models.Edge,
	execution *models.CanvasNodeExecution,
	event *models.CanvasEvent,
) ([]models.CanvasNodeQueueItem, error) {
	now := time.Now()

	logger = logging.WithExecution(logger, execution, nil)
	w.logger.Infof("Processing event")

	var createdQueueItems []models.CanvasNodeQueueItem
	outgoingEdges := findOutgoingEdges(edges, execution.NodeID, event.Channel)
	for _, edge := range outgoingEdges {
		targetNode, err := models.FindCanvasNode(tx, canvas.ID, edge.TargetID)
		if err != nil {
			return nil, err
		}

		if targetNode.State == models.CanvasNodeStateError {
			continue
		}

		queueItem := models.CanvasNodeQueueItem{
			WorkflowID:  canvas.ID,
			NodeID:      targetNode.NodeID,
			RootEventID: execution.RootEventID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return nil, err
		}

		createdQueueItems = append(createdQueueItems, queueItem)
	}

	return createdQueueItems, event.RoutedInTransaction(tx)
}

func (w *EventRouter) processChildExecutionEvent(tx *gorm.DB, logger *log.Entry, canvas *models.Canvas, execution *models.CanvasNodeExecution, event *models.CanvasEvent) ([]models.CanvasNodeQueueItem, *models.CanvasNodeExecution, error) {
	parentExecution, err := models.FindNodeExecutionInTransaction(tx, canvas.ID, *execution.ParentExecutionID)
	if err != nil {
		logger.Errorf("Error finding parent execution: %v", err)
		return nil, nil, err
	}

	parentNode, err := models.FindCanvasNode(tx, canvas.ID, parentExecution.NodeID)
	if err != nil {
		logger.Errorf("Error finding parent node: %v", err)
		return nil, nil, err
	}

	logger = logging.WithExecution(logger, execution, parentExecution)
	logger.Info("Processing child execution event")

	blueprintID := parentNode.Ref.Data().Blueprint.ID
	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, blueprintID)
	if err != nil {
		logger.Errorf("Error finding blueprint: %v", err)
		return nil, nil, err
	}

	childNodeID := execution.NodeID[len(parentNode.NodeID)+1:]
	edges := blueprint.FindEdges(childNodeID, event.Channel)

	var createdQueueItems []models.CanvasNodeQueueItem
	//
	// If there are no edges, it means the child node is a terminal node.
	// We should update the parent execution, if needed.
	//
	if len(edges) == 0 {

		//
		// Lock the parent execution to ensure we are not processing it multiple times for terminal nodes.
		//
		parentExecution, err := models.LockCanvasNodeExecution(tx, *execution.ParentExecutionID)
		if err != nil {
			logger.Info("Child node is a terminal node, but parent is locked - skipping")
			return createdQueueItems, nil, nil
		}

		logger.Info("Child node is a terminal node - checking parent execution")
		return createdQueueItems, parentExecution, w.completeParentExecutionIfNeeded(
			tx,
			logger,
			parentNode,
			parentExecution,
			execution,
			event,
			blueprint,
		)
	}

	logger.Infof("Child node %s is not a terminal node - creating next executions: %v", childNodeID, edges)

	//
	// Not a terminal node, create queue items for next internal nodes.
	// The queue worker will create child executions, preserving parent linkage.
	//
	now := time.Now()
	for _, edge := range edges {
		// Ensure target internal node exists as a workflow node
		targetNodeID := parentNode.NodeID + ":" + edge.TargetID
		targetNode, err := models.FindCanvasNode(tx, canvas.ID, targetNodeID)
		if err != nil {
			logger.Errorf("Error finding target node: %v", err)
			return nil, nil, err
		}

		if targetNode.State == models.CanvasNodeStateError {
			continue
		}

		queueItem := models.CanvasNodeQueueItem{
			WorkflowID:  canvas.ID,
			NodeID:      targetNodeID,
			RootEventID: execution.RootEventID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			logger.Errorf("Error creating queue item: %v", err)
			return nil, nil, err
		}

		createdQueueItems = append(createdQueueItems, queueItem)
	}

	return createdQueueItems, nil, event.RoutedInTransaction(tx)
}

func (w *EventRouter) completeParentExecutionIfNeeded(
	tx *gorm.DB,
	logger *log.Entry,
	parentNode *models.CanvasNode,
	parentExecution *models.CanvasNodeExecution,
	execution *models.CanvasNodeExecution,
	event *models.CanvasEvent,
	blueprint *models.Blueprint,
) error {

	//
	// If the parent already finished, no need to do anything.
	//
	if parentExecution.State == models.CanvasNodeExecutionStateFinished {
		logger.Infof("Parent execution is already finished - skipping")
		return event.RoutedInTransaction(tx)
	}

	//
	// Check if parent execution still has pending/started executions.
	//
	nonFinished, err := models.FindChildExecutionsInTransaction(tx, *execution.ParentExecutionID, []string{
		models.CanvasNodeExecutionStatePending,
		models.CanvasNodeExecutionStateStarted,
	})

	if err != nil {
		logger.Errorf("Error finding child executions: %v", err)
		return err
	}

	//
	// If there are still pending/started executions, we should not complete the parent execution yet.
	//
	if len(nonFinished) > 0 {
		logger.Infof("Parent execution still has %d pending/started executions - skipping", len(nonFinished))
		return event.RoutedInTransaction(tx)
	}

	logger.Infof("Parent execution has no more pending/started executions - completing")

	finishedChildren, err := models.FindChildExecutionsInTransaction(tx, *execution.ParentExecutionID, []string{
		models.CanvasNodeExecutionStateFinished,
	})

	if err != nil {
		logger.Errorf("Error finding child executions: %v", err)
		return err
	}

	//
	// No more pending/started executions, we can complete the parent execution.
	//
	outputs := make(map[string][]any)
	for _, outputChannel := range blueprint.OutputChannels {
		fullNodeID := parentNode.NodeID + ":" + outputChannel.NodeID
		childExecutions := w.findChildrenForNode(finishedChildren, fullNodeID)
		if len(childExecutions) == 0 {
			continue
		}

		for _, childExecution := range childExecutions {
			outputEvents, err := childExecution.GetOutputsInTransaction(tx)
			if err != nil {
				logger.Errorf("Error finding output events for %s: %v", fullNodeID, err)
				return fmt.Errorf("error finding output events for %s: %v", fullNodeID, err)
			}

			for _, outputEvent := range outputEvents {
				if outputEvent.Channel == outputChannel.NodeOutputChannel {
					outputs[outputChannel.Name] = append(outputs[outputChannel.Name], outputEvent.Data.Data())
				}
			}
		}
	}

	_, err = parentExecution.PassInTransaction(tx, outputs)
	if err != nil {
		return err
	}

	logger.Infof("Parent execution completed")
	return event.RoutedInTransaction(tx)
}

func (w *EventRouter) findChildrenForNode(allChildren []models.CanvasNodeExecution, nodeID string) []models.CanvasNodeExecution {
	var childrenForNode []models.CanvasNodeExecution
	for _, child := range allChildren {
		if child.NodeID == nodeID {
			childrenForNode = append(childrenForNode, child)
		}
	}

	return childrenForNode
}
