package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	startedRunsSweepLimit = 100

	runFinalizerTriggerSweep             = "sweep"
	runFinalizerTriggerExecutionFinished = "execution_finished"
	runFinalizerTriggerEventTerminal     = "event_terminal"
	runFinalizerTriggerQueueItemDeleted  = "queue_item_deleted"

	runFinalizerReasonAlreadyFinished = "already_finished"
	runFinalizerReasonOpenWork        = "open_work"

	loopComponentName = "loop"
)

type RunFinalizer struct {
	logger      *log.Entry
	rabbitMQURL string
}

func NewRunFinalizer(rabbitMQURL string) *RunFinalizer {
	return &RunFinalizer{
		logger:      log.WithFields(log.Fields{"worker": "RunFinalizer"}),
		rabbitMQURL: rabbitMQURL,
	}
}

func (w *RunFinalizer) Name() string {
	return "RunFinalizer"
}

func (w *RunFinalizer) Start(ctx context.Context) {
	go w.startExecutionFinishedConsumer(ctx)
	go w.startEventTerminalConsumer(ctx)
	go w.startQueueItemDeletedConsumer(ctx)

	//
	// The database poller is supposed to catch runs that weren't finalized properly,
	// due to some issue in the RabbitMQ event processing plumbing.
	// Also, runs can be open for quite some time - for example,
	// a run waiting for an approval that never comes.
	// So, using the database poller every 5 minutes is a good compromise.
	//
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			runs, err := models.ListStartedCanvasRuns(startedRunsSweepLimit)
			if err != nil {
				w.logger.Errorf("Error listing started runs: %v", err)
				continue
			}

			telemetry.RecordRunFinalizerRunsCount(context.Background(), len(runs))

			for _, run := range runs {
				if err := w.finalizeRun(run.WorkflowID, run.ID, runFinalizerTriggerSweep); err != nil {
					w.logger.WithFields(log.Fields{
						"workflow_id": run.WorkflowID,
						"run_id":      run.ID,
					}).Errorf("Error finalizing run from sweep: %v", err)
				}
			}

			telemetry.RecordRunFinalizerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *RunFinalizer) startQueueItemDeletedConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name() + ".queue-item-deleted",
		RemoteExchange: messages.CanvasExchange,
		Service:        messages.CanvasExchange + "." + messages.CanvasQueueItemDeletedRoutingKey + "." + w.Name(),
		RoutingKey:     messages.CanvasQueueItemDeletedRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", messages.CanvasQueueItemDeletedRoutingKey)

		err := consumer.Start(&options, w.consumeQueueItemDeleted)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.CanvasQueueItemDeletedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.CanvasQueueItemDeletedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *RunFinalizer) startExecutionFinishedConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name() + ".execution-finished",
		RemoteExchange: messages.ExecutionsExchange,
		Service:        messages.ExecutionsExchange + "." + messages.ExecutionFinishedRoutingKey + "." + w.Name(),
		RoutingKey:     messages.ExecutionFinishedRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", messages.ExecutionFinishedRoutingKey)

		err := consumer.Start(&options, w.consumeExecutionFinished)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.ExecutionFinishedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.ExecutionFinishedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *RunFinalizer) startEventTerminalConsumer(ctx context.Context) {
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name() + ".event-terminal",
		RemoteExchange: messages.EventsExchange,
		Service:        messages.EventsExchange + "." + messages.EventTerminalRoutingKey + "." + w.Name(),
		RoutingKey:     messages.EventTerminalRoutingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", messages.EventTerminalRoutingKey)

		err := consumer.Start(&options, w.consumeEventTerminal)
		if err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", messages.EventTerminalRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.EventTerminalRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *RunFinalizer) consumeExecutionFinished(delivery tackle.Delivery) error {
	data := &pb.CanvasNodeExecutionMessage{}
	if err := proto.Unmarshal(delivery.Body(), data); err != nil {
		w.logger.Errorf("Error unmarshaling execution finished message: %v", err)
		return err
	}

	workflowID, err := uuid.Parse(data.CanvasId)
	if err != nil {
		w.logger.Errorf("Error parsing canvas id: %v", err)
		return err
	}

	executionID, err := uuid.Parse(data.Id)
	if err != nil {
		w.logger.Errorf("Error parsing execution id: %v", err)
		return err
	}

	execution, err := models.FindNodeExecution(workflowID, executionID)
	if err != nil {
		w.logger.Errorf("Error finding execution %s: %v", executionID, err)
		return err
	}

	return w.finalizeRun(workflowID, execution.RunID, runFinalizerTriggerExecutionFinished)
}

func (w *RunFinalizer) consumeEventTerminal(delivery tackle.Delivery) error {
	data := &pb.CanvasEventTerminalMessage{}
	if err := proto.Unmarshal(delivery.Body(), data); err != nil {
		w.logger.Errorf("Error unmarshaling event terminal message: %v", err)
		return err
	}

	workflowID, err := uuid.Parse(data.CanvasId)
	if err != nil {
		w.logger.Errorf("Error parsing canvas id: %v", err)
		return err
	}

	runID, err := uuid.Parse(data.RunId)
	if err != nil {
		w.logger.Errorf("Error parsing run id: %v", err)
		return err
	}

	return w.finalizeRun(workflowID, runID, runFinalizerTriggerEventTerminal)
}

func (w *RunFinalizer) consumeQueueItemDeleted(delivery tackle.Delivery) error {
	data := &pb.CanvasNodeQueueItemMessage{}
	if err := proto.Unmarshal(delivery.Body(), data); err != nil {
		w.logger.Errorf("Error unmarshaling queue item deleted message: %v", err)
		return err
	}

	workflowID, err := uuid.Parse(data.CanvasId)
	if err != nil {
		w.logger.Errorf("Error parsing canvas id: %v", err)
		return err
	}

	runID, err := uuid.Parse(data.RunId)
	if err != nil {
		w.logger.Errorf("Error parsing run id: %v", err)
		return err
	}

	return w.finalizeRun(workflowID, runID, runFinalizerTriggerQueueItemDeleted)
}

func (w *RunFinalizer) finalizeRun(workflowID, runID uuid.UUID, trigger string) error {
	//
	// For every run we process, we track the following metrics:
	// - trigger: sweep, execution_finished, event_terminal, queue_item_deleted
	// - outcome: success, failed, skipped
	// - reason: none, already_finished, open_work, locked, deadlock, not_found, internal
	//
	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	defer func() {
		telemetry.RecordRunFinalizerRunProcessing(
			context.Background(),
			time.Since(start),
			trigger,
			outcome,
			reason,
		)
	}()

	logger := w.logger.WithFields(log.Fields{
		"workflow_id": workflowID,
		"run_id":      runID,
	})

	var finalized bool
	var updatedExecutionIDs []uuid.UUID
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var skipReason string
		var err error
		finalized, updatedExecutionIDs, skipReason, err = w.maybeFinalizeRun(tx, runID, trigger)
		if skipReason != "" {
			outcome = executorOutcomeSkipped
			reason = skipReason
		}
		return err
	})

	if err != nil {
		logger.WithError(err).Errorf("Error finalizing run: %v", err)
		outcome = executorOutcomeFailed
		reason = classifyProcessError(err)
		return err
	}

	for _, executionID := range updatedExecutionIDs {
		if err := messages.PublishCanvasExecutionByID(workflowID, executionID); err != nil {
			w.logger.WithError(err).Warnf("Failed to publish execution state message for execution %s", executionID)
		}
	}

	if !finalized {
		return nil
	}

	logger.Info("Run finalized")

	if err := messages.NewCanvasRunMessage(workflowID.String(), runID.String()).Publish(); err != nil {
		w.logger.WithError(err).Warnf("Failed to publish run state message for run %s", runID)
	}

	return nil
}

func (w *RunFinalizer) maybeFinalizeRun(tx *gorm.DB, runID uuid.UUID, trigger string) (bool, []uuid.UUID, string, error) {
	run, err := models.LockCanvasRunInTransaction(tx, runID)
	if err != nil {
		return false, nil, "", err
	}

	if run.State == models.CanvasRunStateFinished {
		return false, nil, runFinalizerReasonAlreadyFinished, nil
	}

	openWork, err := models.FindOpenCanvasRunWorkInTransaction(tx, runID)
	if err != nil {
		return false, nil, "", err
	}

	var updatedExecutionIDs []uuid.UUID
	if openWork.HasActiveExecutions && !openWork.HasQueueItems && !openWork.HasPendingEvents {
		updatedExecutionIDs, err = w.failStalledLoopExecutions(tx, runID)
		if err != nil {
			return false, nil, "", err
		}

		if len(updatedExecutionIDs) > 0 {
			openWork, err = models.FindOpenCanvasRunWorkInTransaction(tx, runID)
			if err != nil {
				return false, nil, "", err
			}
		}
	}

	if openWork.HasActiveExecutions || openWork.HasQueueItems || openWork.HasPendingEvents {
		if trigger == runFinalizerTriggerSweep {
			// The started-run sweep loads candidates with ORDER BY updated_at ASC
			// LIMIT N. Bump updated_at here so a run that is still open is pushed to
			// the back of the queue instead of being retried on every tick.
			now := time.Now()
			return false, updatedExecutionIDs, runFinalizerReasonOpenWork, tx.Model(run).Update("updated_at", &now).Error
		}

		return false, updatedExecutionIDs, runFinalizerReasonOpenWork, nil
	}

	result, err := models.CalculateCanvasRunResultInTransaction(tx, runID)
	if err != nil {
		return false, nil, "", err
	}

	now := time.Now()
	err = tx.Model(run).
		Updates(map[string]any{
			"state":       models.CanvasRunStateFinished,
			"result":      result,
			"updated_at":  &now,
			"finished_at": &now,
		}).
		Error

	if err != nil {
		return false, nil, "", err
	}

	return true, updatedExecutionIDs, "", nil
}

func (w *RunFinalizer) failStalledLoopExecutions(tx *gorm.DB, runID uuid.UUID) ([]uuid.UUID, error) {
	activeExecutions, err := findActiveRunExecutions(tx, runID)
	if err != nil {
		return nil, err
	}

	if len(activeExecutions) == 0 {
		return nil, nil
	}

	loopExecutions := make([]models.CanvasNodeExecution, 0, len(activeExecutions))
	for _, execution := range activeExecutions {
		isLoop, err := isLoopExecution(tx, &execution)
		if err != nil {
			return nil, err
		}

		if !isLoop {
			continue
		}

		if loopExecutionIsWaitingBetweenIterations(execution) {
			continue
		}

		loopExecutions = append(loopExecutions, execution)
	}

	failedExecutionIDs := make([]uuid.UUID, 0, len(loopExecutions))
	for i := range loopExecutions {
		if err := failStalledLoopExecution(tx, &loopExecutions[i]); err != nil {
			return nil, err
		}
		failedExecutionIDs = append(failedExecutionIDs, loopExecutions[i].ID)
	}

	return failedExecutionIDs, nil
}

func findActiveRunExecutions(tx *gorm.DB, runID uuid.UUID) ([]models.CanvasNodeExecution, error) {
	var executions []models.CanvasNodeExecution
	err := tx.
		Where("run_id = ?", runID).
		Where("state IN ?", []string{
			models.CanvasNodeExecutionStatePending,
			models.CanvasNodeExecutionStateStarted,
		}).
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func isLoopExecution(tx *gorm.DB, execution *models.CanvasNodeExecution) (bool, error) {
	node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return false, err
	}

	ref := node.Ref.Data()
	return ref.Component != nil && ref.Component.Name == loopComponentName, nil
}

func loopExecutionIsWaitingBetweenIterations(execution models.CanvasNodeExecution) bool {
	waiting, _ := execution.Metadata.Data()["waitingBetweenIterations"].(bool)
	return waiting
}

func failStalledLoopExecution(tx *gorm.DB, execution *models.CanvasNodeExecution) error {
	metadata := execution.Metadata.Data()
	if metadata == nil {
		metadata = map[string]any{}
	}

	metadata["active"] = false
	metadata["waitingBetweenIterations"] = false
	execution.Metadata = datatypes.NewJSONType(metadata)
	if err := tx.Model(execution).Update("metadata", execution.Metadata).Error; err != nil {
		return err
	}

	_, err := execution.FailInTransaction(
		tx,
		models.CanvasNodeExecutionResultReasonError,
		"loop cannot reach the loop conclusion because all downstream work has reached a terminal state",
	)
	return err
}
