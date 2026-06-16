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
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const startedRunsSweepLimit = 100

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

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runs, err := models.ListStartedCanvasRuns(startedRunsSweepLimit)
			if err != nil {
				w.logger.Errorf("Error listing started runs: %v", err)
				continue
			}

			for _, run := range runs {
				if err := w.finalizeRun(run.WorkflowID, run.ID, true); err != nil {
					w.logger.WithFields(log.Fields{
						"workflow_id": run.WorkflowID,
						"run_id":      run.ID,
					}).Errorf("Error finalizing run from sweep: %v", err)
				}
			}
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

	return w.finalizeRun(workflowID, execution.RunID, false)
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

	return w.finalizeRun(workflowID, runID, false)
}

func (w *RunFinalizer) finalizeRun(workflowID, runID uuid.UUID, fromSweep bool) error {
	var finalized bool
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		finalized, err = w.maybeFinalizeRun(tx, runID, fromSweep)
		return err
	})

	if err != nil {
		return err
	}

	if !finalized {
		return nil
	}

	w.logger.WithFields(log.Fields{
		"workflow_id": workflowID,
		"run_id":      runID,
	}).Info("Run finalized")

	if err := messages.NewCanvasRunMessage(workflowID.String(), runID.String()).Publish(); err != nil {
		w.logger.WithError(err).Warnf("Failed to publish run state message for run %s", runID)
	}

	return nil
}

func (w *RunFinalizer) maybeFinalizeRun(tx *gorm.DB, runID uuid.UUID, fromSweep bool) (bool, error) {
	run, err := models.LockCanvasRunInTransaction(tx, runID)
	if err != nil {
		return false, err
	}

	if run.State == models.CanvasRunStateFinished {
		return false, nil
	}

	openWork, err := models.FindOpenCanvasRunWorkInTransaction(tx, runID)
	if err != nil {
		return false, err
	}

	if openWork.HasActiveExecutions || openWork.HasQueueItems || openWork.HasPendingEvents {
		if fromSweep {
			// The started-run sweep loads candidates with ORDER BY updated_at ASC
			// LIMIT N. Bump updated_at here so a run that is still open is pushed to
			// the back of the queue instead of being retried on every tick.
			now := time.Now()
			return false, tx.Model(run).Update("updated_at", &now).Error
		}

		return false, nil
	}

	result, err := models.CalculateCanvasRunResultInTransaction(tx, runID)
	if err != nil {
		return false, err
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
		return false, err
	}

	return true, nil
}
