package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners/control"
	"gorm.io/gorm"
)

type suspensionCursor struct{ runs, tasks, executions uuid.UUID }

type OrganizationSuspensionWorker struct {
	logger        *log.Entry
	after         uuid.UUID
	cursors       map[uuid.UUID]suspensionCursor
	publishRunner func(control.Notification) error
}

func NewOrganizationSuspensionWorker() *OrganizationSuspensionWorker {
	return &OrganizationSuspensionWorker{
		logger:        log.WithField("worker", "OrganizationSuspensionWorker"),
		cursors:       make(map[uuid.UUID]suspensionCursor),
		publishRunner: control.Publish,
	}
}

func (w *OrganizationSuspensionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := w.Process(ctx); err != nil {
			w.logger.WithError(err).Error("Failed to cancel suspended organization work")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *OrganizationSuspensionWorker) Process(ctx context.Context) error {
	events, err := models.ListPendingOrganizationSuspensions(database.DB(ctx), w.after)
	if err != nil {
		return err
	}
	var failures []error
	for _, event := range events {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.after = event.ID
		if err := w.process(ctx, event); err != nil {
			w.logger.WithError(err).WithFields(log.Fields{
				"organization_id":     event.OrganizationID,
				"suspension_event_id": event.ID,
			}).Error("Suspension cancellation remains pending")
			failures = append(failures, err)
		}
	}
	if len(events) < models.SuspensionWorkBatchSize {
		w.after = uuid.Nil
	}
	return errors.Join(failures...)
}

func (w *OrganizationSuspensionWorker) process(parent context.Context, event models.OrganizationSuspensionEvent) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	logger := w.logger.WithFields(log.Fields{
		"organization_id":     event.OrganizationID,
		"suspension_event_id": event.ID,
	})
	// Commit queue discard before taking run locks. Run finalization takes
	// run locks before line locks, so retaining both would invert that order.
	if err := event.WithCurrentSuspension(database.DB(ctx), func(tx *gorm.DB) error {
		logger.Info("Processing organization suspension cancellation")
		return event.DiscardQueuedWork(tx)
	}); err != nil {
		return err
	}
	cursor := w.cursors[event.ID]
	notifier := &RunCancellationNotifier{}
	var tasks []models.RunnerTask
	var executions []models.CanvasNodeExecution
	var runCount int
	claimed := false
	err := event.WithCurrentSuspension(database.DB(ctx), func(tx *gorm.DB) error {
		claimed = true
		runs, err := event.ListRuns(tx, cursor.runs)
		if err != nil {
			return err
		}
		runCount = len(runs)
		for _, run := range runs {
			result, err := run.RequestCancellationBatch(tx)
			if err != nil {
				return err
			}
			notifier.RecordRuns([]*models.RunCancellationResult{result})
			cursor.runs = run.ID
		}
		if len(runs) < models.SuspensionWorkBatchSize {
			cursor.runs = uuid.Nil
		}
		tasks, err = event.CancelRunnerTasks(tx, cursor.tasks)
		if err != nil {
			return err
		}
		for _, task := range tasks {
			cursor.tasks = task.ID
		}
		if len(tasks) < models.SuspensionWorkBatchSize {
			cursor.tasks = uuid.Nil
		}
		executions, err = event.ListCancellingExecutions(tx, cursor.executions)
		if err != nil {
			return err
		}
		for _, execution := range executions {
			cursor.executions = execution.ID
		}
		if len(executions) < models.SuspensionWorkBatchSize {
			cursor.executions = uuid.Nil
		}
		return event.CompleteCancellation(tx)
	})
	if err != nil {
		return err
	}
	if !claimed || event.CancellationCompletedAt != nil {
		delete(w.cursors, event.ID)
	} else {
		w.cursors[event.ID] = cursor
	}
	if claimed {
		logger.WithFields(log.Fields{
			"run_count":                  runCount,
			"runner_task_count":          len(tasks),
			"cancelling_execution_count": len(executions),
		}).Info("Processed suspension cancellation batch")
		if event.CancellationCompletedAt != nil {
			logger.Info("Organization suspension cancellation completed")
		}
	}
	// Notifications only follow committed changes. Pending work is selected
	// again on later passes, so publication failures survive process restarts.
	notifier.Publish()
	var failures []error
	for _, task := range tasks {
		if task.RunnerID != nil && task.CancelRequestedAt != nil && !task.IsTerminal() {
			if err := w.publishRunner(control.Notification{RunnerID: task.RunnerID.String()}); err != nil {
				failures = append(failures, err)
			}
		}
	}
	for _, execution := range executions {
		if err := messages.PublishCanvasExecutionByID(execution.WorkflowID, execution.ID); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
