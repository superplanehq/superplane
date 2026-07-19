package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	pendingRunsSweepLimit = 100

	runInitializerTriggerSweep   = "sweep"
	runInitializerTriggerPending = "pending"
)

type RunInitializer struct {
	semaphore   *semaphore.Weighted
	registry    *registry.Registry
	rabbitMQURL string
	logger      *log.Entry
}

func NewRunInitializer(rabbitMQURL string, registry *registry.Registry) *RunInitializer {
	return &RunInitializer{
		registry:    registry,
		rabbitMQURL: rabbitMQURL,
		semaphore:   semaphore.NewWeighted(25),
		logger:      log.WithFields(log.Fields{"worker": "RunInitializer"}),
	}
}

func (w *RunInitializer) Name() string {
	return "RunInitializer"
}

func (w *RunInitializer) Start(ctx context.Context) {
	go w.startPendingRunConsumer(ctx)

	//
	// The database poller catches pending runs that were not initialized due to
	// a RabbitMQ delivery issue.
	//
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweepPendingRuns()
		}
	}
}

func (w *RunInitializer) sweepPendingRuns() {
	runs, err := models.ListPendingRuns(database.Conn())
	if err != nil {
		w.logger.Errorf("Error listing pending runs: %v", err)
		return
	}

	if len(runs) > pendingRunsSweepLimit {
		runs = runs[:pendingRunsSweepLimit]
	}

	for _, run := range runs {
		if err := w.initializeRun(run.WorkflowID, run.ID, runInitializerTriggerSweep); err != nil {
			w.logger.WithFields(log.Fields{
				"workflow_id": run.WorkflowID,
				"run_id":      run.ID,
			}).WithError(err).Errorf("Error initializing run from sweep")
		}
	}
}

func (w *RunInitializer) startPendingRunConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name() + ".pending",
		RemoteExchange: messages.CanvasExchange,
		Service:        messages.CanvasExchange + "." + messages.RunPendingRoutingKey + "." + w.Name(),
		RoutingKey:     messages.RunPendingRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", messages.RunPendingRoutingKey)

		err := consumer.Start(&options, w.consumePendingRun)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.RunPendingRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.RunPendingRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *RunInitializer) consumePendingRun(delivery tackle.Delivery) error {
	data := &pb.CanvasRunMessage{}
	if err := proto.Unmarshal(delivery.Body(), data); err != nil {
		return fmt.Errorf("unmarshal pending run message: %w", err)
	}

	workflowID, err := uuid.Parse(data.CanvasId)
	if err != nil {
		return fmt.Errorf("parse workflow id: %w", err)
	}

	runID, err := uuid.Parse(data.Id)
	if err != nil {
		return fmt.Errorf("parse run id: %w", err)
	}

	if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
		return fmt.Errorf("acquire semaphore: %w", err)
	}
	defer w.semaphore.Release(1)

	return w.initializeRun(workflowID, runID, runInitializerTriggerPending)
}

func (w *RunInitializer) initializeRun(workflowID, runID uuid.UUID, trigger string) error {
	logger := w.logger.WithFields(log.Fields{
		"workflow_id": workflowID,
		"run_id":      runID,
		"trigger":     trigger,
	})
	logger.Infof("Initializing pending run")

	newEvents := []models.CanvasEvent{}
	eventCollector := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	executionUpdates := []models.CanvasNodeExecution{}
	executionCollector := func(executions []models.CanvasNodeExecution) {
		executionUpdates = append(executionUpdates, executions...)
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockCanvasRunInTransaction(tx, runID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Infof("Run not found - skipping")
				return nil
			}

			return err
		}

		if locked.WorkflowID != workflowID {
			return fmt.Errorf("workflow id mismatch: expected %s, got %s", workflowID, locked.WorkflowID)
		}

		if locked.State != models.CanvasRunStatePending {
			logger.Infof("Run already initialized - skipping")
			return nil
		}

		err = NewRunCallbackDispatcher(tx, w.registry, locked).
			WithEventCollector(eventCollector).
			DispatchPending()

		//
		// If there's an error dispatching the pending run callback,
		// the run initialization failed, so we need to fail the run,
		// and run any additional run finished callbacks we might have configured for the run.
		//
		if err != nil {
			logger.WithError(err).Errorf("Error dispatching pending run callback")
			return w.failRun(tx, locked, eventCollector, executionCollector, err.Error())
		}

		//
		// Otherwise, we start the run.
		//
		if err := locked.Start(tx); err != nil {
			return fmt.Errorf("start run: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	for _, event := range newEvents {
		if err := messages.PublishCanvasEventCreatedMessage(&event); err != nil {
			return err
		}
	}

	for _, execution := range executionUpdates {
		if err := messages.NewCanvasExecutionMessage(execution.WorkflowID.String(), execution.ID.String(), execution.NodeID).PublishFinished(); err != nil {
			return err
		}
	}

	return nil
}

func (w *RunInitializer) failRun(
	tx *gorm.DB,
	run *models.CanvasRun,
	eventCollector func([]models.CanvasEvent),
	executionCollector func([]models.CanvasNodeExecution),
	resultMessage string,
) error {
	now := time.Now()
	run.State = models.CanvasRunStateFinished
	run.Result = models.CanvasRunResultFailed
	run.ResultMessage = resultMessage
	run.UpdatedAt = &now
	run.FinishedAt = &now
	err := tx.Model(run).
		Updates(map[string]any{
			"state":          models.CanvasRunStateFinished,
			"result":         models.CanvasRunResultFailed,
			"result_message": &resultMessage,
			"updated_at":     &now,
			"finished_at":    &now,
		}).
		Error

	if err != nil {
		return err
	}

	return NewRunCallbackDispatcher(tx, w.registry, run).
		WithEventCollector(eventCollector).
		WithExecutionCollector(executionCollector).
		DispatchFinished()
}
