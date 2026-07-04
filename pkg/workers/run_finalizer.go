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
	"gorm.io/gorm"
)

const (
	startedRunsSweepLimit = 100

	runFinalizerTriggerSweep             = "sweep"
	runFinalizerTriggerExecutionFinished = "execution_finished"
	runFinalizerTriggerEventTerminal     = "event_terminal"

	runFinalizerReasonAlreadyFinished = "already_finished"
	runFinalizerReasonOpenWork        = "open_work"
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

func (w *RunFinalizer) finalizeRun(workflowID, runID uuid.UUID, trigger string) error {
	//
	// For every run we process, we track the following metrics:
	// - trigger: sweep, execution_finished, event_terminal
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
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var skipReason string
		var err error
		finalized, skipReason, err = w.maybeFinalizeRun(tx, runID, trigger)
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

	if !finalized {
		return nil
	}

	logger.Info("Run finalized")

	if err := messages.NewCanvasRunMessage(workflowID.String(), runID.String()).Publish(); err != nil {
		w.logger.WithError(err).Warnf("Failed to publish run state message for run %s", runID)
	}

	return nil
}

func (w *RunFinalizer) maybeFinalizeRun(tx *gorm.DB, runID uuid.UUID, trigger string) (bool, string, error) {
	run, err := models.LockCanvasRunInTransaction(tx, runID)
	if err != nil {
		return false, "", err
	}

	if run.State == models.CanvasRunStateFinished {
		return false, runFinalizerReasonAlreadyFinished, nil
	}

	openWork, err := models.FindOpenCanvasRunWorkInTransaction(tx, runID)
	if err != nil {
		return false, "", err
	}

	if openWork.HasActiveExecutions || openWork.HasQueueItems || openWork.HasPendingEvents {
		if trigger == runFinalizerTriggerSweep {
			// The started-run sweep loads candidates with ORDER BY updated_at ASC
			// LIMIT N. Bump updated_at here so a run that is still open is pushed to
			// the back of the queue instead of being retried on every tick.
			now := time.Now()
			return false, runFinalizerReasonOpenWork, tx.Model(run).Update("updated_at", &now).Error
		}

		return false, runFinalizerReasonOpenWork, nil
	}

	result, err := models.CalculateCanvasRunResultInTransaction(tx, runID)
	if err != nil {
		return false, "", err
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
		return false, "", err
	}

	return true, "", nil
}
