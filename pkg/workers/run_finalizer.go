package workers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/proto"
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
)

type RunFinalizer struct {
	logger      *log.Entry
	registry    *registry.Registry
	rabbitMQURL string
}

func NewRunFinalizer(rabbitMQURL string, registry *registry.Registry) *RunFinalizer {
	return &RunFinalizer{
		logger:      log.WithFields(log.Fields{"worker": "RunFinalizer"}),
		registry:    registry,
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

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				w.sweepStartedRuns()
			}()
			go func() {
				defer wg.Done()
				w.sweepCancellingRuns()
			}()
			wg.Wait()

			telemetry.RecordRunFinalizerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *RunFinalizer) sweepStartedRuns() error {
	runs, err := models.ListStartedCanvasRuns(database.Conn(), startedRunsSweepLimit)
	if err != nil {
		w.logger.Errorf("Error listing started runs: %v", err)
		return err
	}

	telemetry.RecordRunFinalizerRunsCount(context.Background(), len(runs))

	for _, run := range runs {
		if err := w.finalizeRun(run.WorkflowID, run.ID, runFinalizerTriggerSweep); err != nil {
			logger := logging.WithRun(w.logger, run)
			logger.WithError(err).Errorf("Error finalizing run from sweep: %v", err)
		}
	}

	return nil
}

func (w *RunFinalizer) sweepCancellingRuns() error {
	cancellingRuns, err := models.ListCancellingCanvasRuns(database.Conn(), startedRunsSweepLimit)
	if err != nil {
		w.logger.Errorf("Error listing cancelling runs: %v", err)
		return err
	}

	for _, run := range cancellingRuns {
		logger := logging.WithRun(w.logger, run)

		var cancellationResult *models.RunCancellationDrainResult
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			result, err := run.DrainForCancellation(tx, run.CancelledBy)
			if err != nil {
				return err
			}

			cancellationResult = result
			return nil
		})

		if err != nil {
			logger.WithError(err).Errorf("Error draining cancelling run from sweep: %v", err)
			continue
		}

		if cancellationResult != nil {
			messages.PublishRunCancellationDrain(run.WorkflowID, cancellationResult)
		}

		if err := w.finalizeRun(run.WorkflowID, run.ID, runFinalizerTriggerSweep); err != nil {
			logger.WithError(err).Errorf("Error finalizing cancelling run from sweep: %v", err)
		}
	}

	return nil
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

	var newEvents []models.CanvasEvent
	eventCollector := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	var executionUpdates []models.CanvasNodeExecution
	executionCollector := func(executions []models.CanvasNodeExecution) {
		executionUpdates = append(executionUpdates, executions...)
	}

	var finalized bool
	var nextFactoryLineRuns []factoryLinePendingRun
	var factoryOrderUpdates []factoryWorkOrderUpdate
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var skipReason string
		var err error
		finalized, skipReason, err = w.maybeFinalizeRun(tx, runID, trigger, eventCollector, executionCollector)
		if skipReason != "" {
			outcome = executorOutcomeSkipped
			reason = skipReason
		}
		if err != nil {
			return err
		}
		if !finalized {
			return nil
		}

		nextFactoryLineRuns, factoryOrderUpdates, err = w.executeNextFactoryLineStep(tx, runID)
		return err
	})

	if err != nil {
		logger.WithError(err).Errorf("Error finalizing run: %v", err)
		outcome = executorOutcomeFailed
		reason = classifyProcessError(err)
		return err
	}

	if !finalized {
		return nil
	}

	logger.Info("Run finalized")

	if err := messages.NewCanvasRunMessage(workflowID.String(), runID.String()).Publish(); err != nil {
		w.logger.WithError(err).Warnf("Failed to publish run state message for run %s", runID)
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

	for _, nextFactoryLineRun := range nextFactoryLineRuns {
		if err := messages.NewCanvasRunMessage(nextFactoryLineRun.workflowID.String(), nextFactoryLineRun.runID.String()).PublishPending(); err != nil {
			w.logger.WithError(err).Warnf("Failed to publish pending run message for run %s", nextFactoryLineRun.runID)
			return err
		}
	}

	// The finished run's own work order is refreshed by the run-state
	// fan-out above; admitted orders are other work orders whose queued →
	// started transition the UI would otherwise miss until their run starts.
	for _, update := range factoryOrderUpdates {
		if err := messages.PublishFactoryWorkOrderUpdated(
			update.factoryID.String(),
			update.orderID.String(),
			factoryevents.EventTypeLineStepExecutionCreated,
		); err != nil {
			w.logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", update.orderID)
		}
	}

	return nil
}

func (w *RunFinalizer) maybeFinalizeRun(tx *gorm.DB, runID uuid.UUID, trigger string, eventCollector func([]models.CanvasEvent), executionCollector func([]models.CanvasNodeExecution)) (bool, string, error) {
	run, err := models.LockCanvasRunInTransaction(tx, runID)
	if err != nil {
		return false, "", err
	}

	if run.State == models.CanvasRunStateFinished {
		return false, runFinalizerReasonAlreadyFinished, nil
	}

	openWork, err := run.FindOpenWork(tx)
	if err != nil {
		return false, "", err
	}

	if openWork.HasActiveExecutions || openWork.HasQueueItems || openWork.HasPendingEvents {
		if trigger == runFinalizerTriggerSweep && run.State != models.CanvasRunStateCancelling {
			// The started-run sweep loads candidates with ORDER BY updated_at ASC
			// LIMIT N. Bump updated_at here so a run that is still open is pushed to
			// the back of the queue instead of being retried on every tick.
			now := time.Now()
			return false, runFinalizerReasonOpenWork, tx.Model(run).Update("updated_at", &now).Error
		}

		return false, runFinalizerReasonOpenWork, nil
	}

	result, err := run.CalculateResult(tx)
	if err != nil {
		return false, "", err
	}

	now := time.Now()
	run.State = models.CanvasRunStateFinished
	run.Result = result
	run.UpdatedAt = &now
	run.FinishedAt = &now
	err = tx.Model(run).
		Updates(map[string]any{
			"state":       models.CanvasRunStateFinished,
			"result":      result,
			"updated_at":  &now,
			"finished_at": &now,
		}).
		Error

	if err != nil {
		return false, "", err
	}

	err = NewRunCallbackDispatcher(tx, w.registry, run).
		WithEventCollector(eventCollector).
		WithExecutionCollector(executionCollector).
		DispatchFinished()

	if err != nil {
		return false, "", err
	}

	return true, "", nil
}

type factoryLinePendingRun struct {
	workflowID uuid.UUID
	runID      uuid.UUID
}

// factoryWorkOrderUpdate identifies a work order whose factory websocket
// clients must refresh after the finalize transaction commits.
type factoryWorkOrderUpdate struct {
	factoryID uuid.UUID
	orderID   uuid.UUID
}

func (w *RunFinalizer) executeNextFactoryLineStep(tx *gorm.DB, runID uuid.UUID) ([]factoryLinePendingRun, []factoryWorkOrderUpdate, error) {
	//
	// Finish current factory work order execution.
	//
	execution, err := models.FindWorkOrderExecutionByRunID(tx, runID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	run, err := models.LockCanvasRunInTransaction(tx, runID)
	if err != nil {
		return nil, nil, err
	}

	if execution.Status == models.FactoryWorkOrderExecutionStatusFinished {
		return nil, nil, nil
	}

	if err := execution.MarkFinished(tx, run.Result); err != nil {
		return nil, nil, err
	}

	var pendingRuns []factoryLinePendingRun
	var orderUpdates []factoryWorkOrderUpdate

	//
	// The finished run freed a slot at its step: admit queued work orders
	// while slots are free. Failed and cancelled runs free their slot the
	// same way passed runs do.
	//
	admitted, err := models.AdmitQueuedForStep(tx, execution.LineID, execution.StepIndex)
	if err != nil {
		return nil, nil, err
	}

	for _, admission := range admitted {
		if admission.Run != nil {
			pendingRuns = append(pendingRuns, factoryLinePendingRun{
				workflowID: admission.Run.WorkflowID,
				runID:      admission.Run.ID,
			})
		}
		if admission.Execution != nil {
			orderUpdates = append(orderUpdates, factoryWorkOrderUpdate{
				factoryID: admission.Execution.FactoryID,
				orderID:   admission.Execution.WorkOrderID,
			})
		}
	}

	//
	// Advance (or finish) the line dispatch this step run belongs to. The
	// dispatch's steps snapshot — not the live line — is authoritative for
	// what comes next, so a mid-traversal line edit can't change it.
	//
	dispatch, err := models.FindWorkOrderLineDispatch(tx, execution.LineDispatchID)
	if err != nil {
		return nil, nil, err
	}

	if run.Result != models.CanvasRunResultPassed {
		return pendingRuns, orderUpdates, dispatch.Finish(tx, run.Result)
	}

	factory, err := models.FindFactory(tx, execution.OrganizationID, execution.FactoryID)
	if err != nil {
		return nil, nil, err
	}

	workOrder, err := factory.FindWorkOrder(tx, execution.WorkOrderID)
	if err != nil {
		return nil, nil, err
	}

	// The order closed while this step was running. The traversal is
	// abandoned, not just paused: finish it as cancelled so the order
	// doesn't keep a zombie active dispatch (which would block any
	// re-dispatch after a reopen).
	if !workOrder.IsOpen() {
		return pendingRuns, orderUpdates, dispatch.Finish(tx, models.CanvasRunResultCancelled)
	}

	nextIndex := execution.StepIndex + 1
	if nextIndex >= len(dispatch.Steps) {
		return pendingRuns, orderUpdates, dispatch.Finish(tx, models.CanvasRunResultPassed)
	}

	result, err := dispatch.EnqueueOrStartStep(tx, workOrder, nextIndex)
	if err != nil {
		return nil, nil, err
	}

	if result.Run != nil {
		pendingRuns = append(pendingRuns, factoryLinePendingRun{
			workflowID: result.Run.WorkflowID,
			runID:      result.Run.ID,
		})
	}

	return pendingRuns, orderUpdates, nil
}
