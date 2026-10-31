package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners/control"
	"gorm.io/gorm"
)

type suspensionCursor struct{ runs, tasks, sessions, executions uuid.UUID }

type OrganizationSuspensionWorker struct {
	agents        *agents.Service
	after         uuid.UUID
	cursors       map[uuid.UUID]suspensionCursor
	publishRunner func(control.Notification) error
}

func NewOrganizationSuspensionWorker(service *agents.Service) *OrganizationSuspensionWorker {
	return &OrganizationSuspensionWorker{agents: service, cursors: make(map[uuid.UUID]suspensionCursor), publishRunner: control.Publish}
}

func (w *OrganizationSuspensionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := w.Process(ctx); err != nil {
			log.WithError(err).Error("Failed to cancel suspended organization work")
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
			log.WithError(err).WithField("organization_id", event.OrganizationID).Error("Suspension cancellation remains pending")
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
	unlock, acquired, err := tryOrganizationSuspensionLock(ctx, event.OrganizationID)
	if err != nil || !acquired {
		return err
	}
	defer unlock()
	db := database.DB(ctx)
	cursor := w.cursors[event.ID]
	defer func() {
		if event.CancellationCompletedAt != nil {
			delete(w.cursors, event.ID)
			return
		}
		w.cursors[event.ID] = cursor
	}()
	if err := event.WithCurrentSuspension(db, event.DiscardQueuedWork); err != nil {
		return err
	}

	var runs []models.CanvasRun
	if err := event.WithCurrentSuspension(db, func(tx *gorm.DB) error {
		var err error
		runs, err = event.ListRuns(tx, cursor.runs)
		return err
	}); err != nil {
		return err
	}
	notifier := &RunCancellationNotifier{}
	for _, run := range runs {
		if err := event.WithCurrentSuspension(db, func(tx *gorm.DB) error {
			result, err := run.RequestCancellationBatch(tx)
			if err != nil {
				return err
			}
			notifier.RecordRuns([]*models.RunCancellationResult{result})
			return nil
		}); err != nil {
			return err
		}
		cursor.runs = run.ID
	}
	if len(runs) < models.SuspensionWorkBatchSize {
		cursor.runs = uuid.Nil
	}
	notifier.Publish() // Database sweeps and reconciliation below recover delivery.

	var tasks []models.RunnerTask
	if err := event.WithCurrentSuspension(db, func(tx *gorm.DB) error {
		var err error
		tasks, err = event.CancelRunnerTasks(tx, cursor.tasks)
		return err
	}); err != nil {
		return err
	}
	var failures []error
	for _, task := range tasks {
		cursor.tasks = task.ID
		if task.RunnerID != nil && task.CancelRequestedAt != nil && !task.IsTerminal() {
			if err := w.publishRunner(control.Notification{RunnerID: task.RunnerID.String()}); err != nil {
				failures = append(failures, err)
			}
		}
	}
	if len(tasks) < models.SuspensionWorkBatchSize {
		cursor.tasks = uuid.Nil
	}

	var executions []models.CanvasNodeExecution
	var sessions []models.AgentSession
	if err := event.WithCurrentSuspension(db, func(tx *gorm.DB) error {
		var err error
		executions, err = event.ListCancellingExecutions(tx, cursor.executions)
		if err != nil {
			return err
		}
		sessions, err = event.PrepareAgentStops(tx, cursor.sessions)
		return err
	}); err != nil {
		return err
	}
	for _, execution := range executions {
		cursor.executions = execution.ID
		if err := messages.PublishCanvasExecutionByID(execution.WorkflowID, execution.ID); err != nil {
			failures = append(failures, err)
		}
	}
	if len(executions) < models.SuspensionWorkBatchSize {
		cursor.executions = uuid.Nil
	}
	for _, session := range sessions {
		cursor.sessions = session.ID
		if err := w.stopAgent(ctx, &event, &session); err != nil {
			failures = append(failures, err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if len(sessions) < models.SuspensionWorkBatchSize {
		cursor.sessions = uuid.Nil
	}
	if err := errors.Join(failures...); err != nil {
		return err
	}
	return event.WithCurrentSuspension(db, event.CompleteCancellation)
}

func (w *OrganizationSuspensionWorker) stopAgent(ctx context.Context, event *models.OrganizationSuspensionEvent, session *models.AgentSession) error {
	if w.agents == nil {
		return errors.New("managed agent cancellation requires ANTHROPIC_API_KEY, ANTHROPIC_AGENT_ID, and ANTHROPIC_ENVIRONMENT_ID on regular workers")
	}
	if session.Provider != w.agents.ProviderName() {
		return errors.New("managed agent cancellation provider does not match the stored session")
	}
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	stopped, err := w.agents.StopSuspendedSession(callCtx, session.ProviderSessionID)
	cancel()
	if err != nil || !stopped {
		return err
	}
	// Owning this lock proves custom tool execution and local SSE handling have
	// ended. Do not wait for the stream while holding any database row lock.
	unlock, acquired, err := tryAgentStreamLock(ctx, session.ID)
	if err != nil || !acquired {
		return err
	}
	defer unlock()
	var toolsClosed bool
	if err := event.WithCurrentSuspension(database.DB(ctx), func(tx *gorm.DB) error {
		var err error
		toolsClosed, err = session.CloseSuspensionTools(tx)
		return err
	}); err != nil {
		return err
	}
	if !toolsClosed {
		return nil
	}
	// Publish before clearing durable stopping intent: a failed publication is
	// reconstructed on the next pass, including after a process restart.
	if err := messages.PublishAgentSessionEvent(messages.AgentSessionEventMessage{SessionID: session.ID.String(), Event: "turn_completed", Status: models.AgentSessionStatusIdle}); err != nil {
		return err
	}
	return event.WithCurrentSuspension(database.DB(ctx), session.CompleteSuspensionStop)
}

func tryOrganizationSuspensionLock(ctx context.Context, organizationID uuid.UUID) (func(), bool, error) {
	db, err := database.OpenDedicatedSQLDB("organization-suspension-lock", 1)
	if err != nil {
		return nil, false, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		return nil, false, err
	}
	// Two-key locks use a separate namespace from single-key agent stream locks.
	const namespace = 73421
	var acquired bool
	err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1, hashtext($2))", namespace, organizationID.String()).Scan(&acquired)
	if err != nil || !acquired {
		_ = conn.Close()
		_ = db.Close()
		return nil, false, err
	}
	return func() {
		defer db.Close()
		defer conn.Close()
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := conn.ExecContext(releaseCtx, "SELECT pg_advisory_unlock($1, hashtext($2))", namespace, organizationID.String()); err != nil {
			log.WithError(err).Warn("Failed to release organization suspension lock")
		}
	}, true, nil
}
