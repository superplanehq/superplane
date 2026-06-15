package workers

import (
	"context"
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
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(event models.CanvasEvent) {
					attemptStart := time.Now()
					logger := logging.ForEvent(w.logger, event)
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessEvent(logger, event, attemptStart); err != nil {
						logger.Errorf("Error processing event: %v", err)
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
		RemoteExchange: messages.CanvasExchange,
		Service:        messages.CanvasExchange + "." + messages.CanvasEventCreatedRoutingKey + "." + w.Name(),
		RoutingKey:     messages.CanvasEventCreatedRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))
	w.consumer = consumer

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.CanvasEventCreatedRoutingKey)

		err := w.consumer.Start(&options, w.Consume)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.CanvasEventCreatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.CanvasEventCreatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *EventRouter) Consume(delivery tackle.Delivery) error {
	start := time.Now()

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
		telemetry.RecordEventWorkerEventProcessing(
			context.Background(),
			time.Since(start),
			executorOutcomeSkipped,
			executorReasonNone,
		)
		return nil
	}

	logger := logging.ForEvent(w.logger, *event)
	err = w.LockAndProcessEvent(logger, *event, start)
	if err != nil {
		logger.Errorf("Error processing event: %v", err)
		return err
	}

	return nil
}

func (w *EventRouter) LockAndProcessEvent(logger *log.Entry, event models.CanvasEvent, attemptStart time.Time) error {
	//
	// For every event we process, we track the following metrics:
	// - outcome: success, failed, skipped
	// - reason: none, locked, deadlock, not_found, internal
	//
	metricOutcome := executorOutcomeSuccess
	metricReason := executorReasonNone
	defer func() {
		telemetry.RecordEventWorkerEventProcessing(
			context.Background(),
			time.Since(attemptStart),
			metricOutcome,
			metricReason,
		)
	}()

	var createdQueueItems []models.CanvasNodeQueueItem
	var runID uuid.UUID
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedEvent, err := models.LockCanvasEvent(tx, event.ID)
		if err != nil {
			logger.Info("Event already being processed - skipping")
			metricOutcome = executorOutcomeSkipped
			metricReason = executorReasonLocked
			return nil
		}

		createdQueueItems, runID, err = w.processEvent(tx, logger, lockedEvent)
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

	if len(createdQueueItems) > 0 {
		for _, queueItem := range createdQueueItems {
			messages.NewCanvasQueueItemMessage(
				event.WorkflowID.String(),
				queueItem.ID.String(),
				queueItem.NodeID,
			).Publish(false)
		}
	}

	if runID != uuid.Nil {
		err := messages.NewCanvasRunMessage(event.WorkflowID.String(), runID.String()).Publish()
		if err != nil {
			logger.WithError(err).Warnf(
				"Failed to publish run state message for run %s in workflow %s",
				runID,
				event.WorkflowID,
			)
		}
	}

	return nil
}

func (w *EventRouter) processEvent(tx *gorm.DB, logger *log.Entry, event *models.CanvasEvent) ([]models.CanvasNodeQueueItem, uuid.UUID, error) {
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, event.WorkflowID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	_, liveEdges, err := models.FindLiveCanvasSpecInTransaction(tx, canvas.ID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	if event.ExecutionID == nil {
		return w.processRootEvent(tx, canvas, liveEdges, event)
	}

	execution, err := models.FindNodeExecutionInTransaction(tx, event.WorkflowID, *event.ExecutionID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	queueItems, runID, err := w.processExecutionEvent(tx, logger, canvas, liveEdges, execution, event)
	return queueItems, runID, err
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

func (w *EventRouter) processRootEvent(tx *gorm.DB, canvas *models.Canvas, edges []models.Edge, event *models.CanvasEvent) ([]models.CanvasNodeQueueItem, uuid.UUID, error) {
	now := time.Now()

	w.logger.Infof("Processing root event %s", event.ID)

	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(tx, event)
	if err != nil {
		return nil, uuid.Nil, err
	}

	outgoingEdges := findOutgoingEdges(edges, event.NodeID, event.Channel)
	var queueItems []models.CanvasNodeQueueItem
	for _, edge := range outgoingEdges {
		targetNode, err := models.FindCanvasNode(tx, canvas.ID, edge.TargetID)
		if err != nil {
			return nil, uuid.Nil, err
		}

		if targetNode.State == models.CanvasNodeStateError {
			continue
		}

		queueItem := models.CanvasNodeQueueItem{
			WorkflowID:  canvas.ID,
			NodeID:      targetNode.NodeID,
			RootEventID: event.ID,
			RunID:       run.ID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return nil, uuid.Nil, err
		}

		queueItems = append(queueItems, queueItem)
	}

	err = event.RoutedInTransaction(tx)
	if err != nil {
		return nil, uuid.Nil, err
	}

	//
	// If we created any queue items, we know for sure that the run is not finished yet,
	// so there is no need to lock the run record to check it.
	//
	if len(queueItems) > 0 {
		return queueItems, run.ID, nil
	}

	_, err = models.MaybeFinalizeRunInTransaction(tx, run.ID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	return queueItems, run.ID, nil
}

func (w *EventRouter) processExecutionEvent(
	tx *gorm.DB,
	logger *log.Entry,
	canvas *models.Canvas,
	edges []models.Edge,
	execution *models.CanvasNodeExecution,
	event *models.CanvasEvent,
) ([]models.CanvasNodeQueueItem, uuid.UUID, error) {
	now := time.Now()

	logger = logging.WithExecution(logger, execution)
	w.logger.Infof("Processing event")

	var createdQueueItems []models.CanvasNodeQueueItem
	outgoingEdges := findOutgoingEdges(edges, execution.NodeID, event.Channel)
	for _, edge := range outgoingEdges {
		targetNode, err := models.FindCanvasNode(tx, canvas.ID, edge.TargetID)
		if err != nil {
			return nil, uuid.Nil, err
		}

		if targetNode.State == models.CanvasNodeStateError {
			continue
		}

		queueItem := models.CanvasNodeQueueItem{
			WorkflowID:  canvas.ID,
			NodeID:      targetNode.NodeID,
			RootEventID: execution.RootEventID,
			RunID:       execution.RunID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(&queueItem).Error; err != nil {
			return nil, uuid.Nil, err
		}

		createdQueueItems = append(createdQueueItems, queueItem)
	}

	if err := event.RoutedInTransaction(tx); err != nil {
		return nil, uuid.Nil, err
	}

	finalized, err := models.MaybeFinalizeRunInTransaction(tx, execution.RunID)
	if err != nil {
		return nil, uuid.Nil, err
	}

	if finalized {
		return createdQueueItems, execution.RunID, nil
	}

	return createdQueueItems, uuid.Nil, nil
}
