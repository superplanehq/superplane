package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners/control"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestSuspensionCancelsRunnerTasksAndRecoversNotifications(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	fleets := models.DefaultInstallationRunnerFleets("test")
	require.NoError(t, fleets[0].Create(db))
	var tasks []models.RunnerTask
	for _, state := range []string{models.RunnerTaskStatePending, models.RunnerTaskStateQueued, models.RunnerTaskStateReserved, models.RunnerTaskStateRunning} {
		task := models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleets[0].ID, State: state, Backend: models.RunnerTaskBackendIntegrated, QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted")}
		if state == models.RunnerTaskStateReserved || state == models.RunnerTaskStateRunning {
			runner := models.Runner{ID: uuid.New(), FleetID: fleets[0].ID, State: models.RunnerStateBusy, RunnerVersion: "test"}
			require.NoError(t, db.Create(&runner).Error)
			task.RunnerID = &runner.ID
		}
		require.NoError(t, db.Create(&task).Error)
		tasks = append(tasks, task)
	}
	w := NewOrganizationSuspensionWorker(nil)
	notified := 0
	w.publishRunner = func(control.Notification) error { notified++; return errors.New("publication unavailable") }
	require.NoError(t, w.Process(t.Context()))
	require.Zero(t, notified)
	require.NoError(t, r.Organization.Suspend(db, uuid.New(), "Stop work"))
	events, err := models.ListOrganizationSuspensionEvents(db, r.Organization.ID)
	require.NoError(t, err)
	failing := db.Begin()
	require.NoError(t, failing.Exec("ALTER TABLE runner_tasks ADD CONSTRAINT reject_suspension_cancellation CHECK (state <> 'canceled') NOT VALID").Error)
	require.Error(t, events[0].WithCurrentSuspension(failing, func(tx *gorm.DB) error {
		_, err := events[0].CancelRunnerTasks(tx, uuid.Nil)
		return err
	}))
	require.NoError(t, failing.Rollback().Error)
	unchanged, err := models.FindRunnerTask(db, tasks[0].ID)
	require.NoError(t, err)
	require.Equal(t, models.RunnerTaskStatePending, unchanged.State)
	require.Error(t, w.Process(t.Context()))
	require.Equal(t, 2, notified)
	for i, task := range tasks {
		stored, err := models.FindRunnerTask(db, task.ID)
		require.NoError(t, err)
		if i < 2 {
			require.Equal(t, models.RunnerTaskStateCanceled, stored.State)
		} else {
			require.Equal(t, task.State, stored.State)
			require.NotNil(t, stored.CancelRequestedAt)
		}
	}
	require.ErrorIs(t, r.Organization.Reactivate(db, uuid.New()), models.ErrSuspensionCancellationPending)
	// A new replica reconstructs notification delivery from committed requests.
	w = NewOrganizationSuspensionWorker(nil)
	notified = 0
	w.publishRunner = func(control.Notification) error { notified++; return nil }
	require.NoError(t, w.Process(t.Context()))
	require.Equal(t, 2, notified)
	for _, task := range tasks[2:] {
		require.NoError(t, db.Model(&task).Update("state", models.RunnerTaskStateCanceled).Error)
	}
	require.NoError(t, w.Process(t.Context()))
	require.NoError(t, r.Organization.Reactivate(db, uuid.New()))
	// Captured stale candidates must not cancel a later generation of work.
	events, err = models.ListOrganizationSuspensionEvents(db, r.Organization.ID)
	require.NoError(t, err)
	task := tasks[0]
	task.ID = uuid.New()
	task.State = models.RunnerTaskStatePending
	require.NoError(t, db.Create(&task).Error)
	require.NoError(t, w.process(t.Context(), events[1]))
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	require.Equal(t, models.RunnerTaskStatePending, stored.State)
}

func TestSuspensionReplicaOwnershipAndOrganizationFirstLocks(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, r.Organization.Suspend(db, uuid.New(), "Stop work"))
	events, err := models.ListOrganizationSuspensionEvents(db, r.Organization.ID)
	require.NoError(t, err)
	unlock, acquired, err := tryOrganizationSuspensionLock(t.Context(), r.Organization.ID)
	require.NoError(t, err)
	require.True(t, acquired)
	released := false
	defer func() {
		if !released {
			unlock()
		}
	}()
	w := NewOrganizationSuspensionWorker(nil)
	require.NoError(t, w.Process(t.Context()))
	history, err := models.ListOrganizationSuspensionEvents(db, r.Organization.ID)
	require.NoError(t, err)
	require.Nil(t, history[0].CancellationCompletedAt)
	unlock()
	released = true
	tx := db.Begin()
	defer tx.Rollback()
	_, err = models.LockOrganization(tx, r.Organization.ID)
	require.NoError(t, err)
	// The event row remains available while cancellation waits for organization.
	blockedCtx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- events[0].WithCurrentSuspension(database.DB(blockedCtx), events[0].CompleteCancellation)
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%organizations%')").Scan(&waiting).Error
		return err == nil && waiting
	}, 200*time.Millisecond, 10*time.Millisecond)
	probe := db.Begin()
	defer probe.Rollback()
	require.NoError(t, probe.Exec("SET LOCAL lock_timeout = '100ms'").Error)
	require.NoError(t, probe.Exec("SELECT id FROM organization_suspension_events WHERE id = ? FOR UPDATE", events[0].ID).Error)
	require.Error(t, <-done)
}

type suspensionProvider struct {
	agents.Provider
	started chan struct{}
	release chan struct{}
	stopped bool
	failure error
}

func (p *suspensionProvider) Name() string { return "test" }

func (p *suspensionProvider) InterruptSession(ctx context.Context, _ string) error {
	if p.started != nil {
		close(p.started)
		select {
		case <-p.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return p.failure
}
func (p *suspensionProvider) SessionStopped(context.Context, string) (bool, error) {
	return p.stopped, p.failure
}

func TestSuspensionAgentStopSurvivesFailureAndBlocksReactivationDuringProviderCall(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	session := models.AgentSession{OrganizationID: r.Organization.ID, UserID: r.User, CanvasID: canvas.ID, Provider: "test", ProviderSessionID: "session-1", Status: models.AgentSessionStatusStreaming}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, r.Organization.Suspend(db, uuid.New(), "Stop agent"))
	p := &suspensionProvider{started: make(chan struct{}), release: make(chan struct{})}
	w := NewOrganizationSuspensionWorker(agents.NewService(p, nil))
	done := make(chan error, 1)
	go func() { done <- w.Process(t.Context()) }()
	<-p.started
	require.NoError(t, NewOrganizationSuspensionWorker(agents.NewService(p, nil)).Process(t.Context()))
	require.ErrorIs(t, r.Organization.Reactivate(db, uuid.New()), models.ErrSuspensionCancellationPending)
	require.ErrorIs(t, models.UpdateAgentSessionStatus(session.ID, models.AgentSessionStatusIdle), models.ErrAgentSessionStopping)
	close(p.release)
	require.NoError(t, <-done)
	stored, err := models.FindAgentSession(session.ID)
	require.NoError(t, err)
	require.Equal(t, models.AgentSessionStatusStopping, stored.Status)
	p.started = nil
	p.failure = errors.New("temporary provider failure")
	w = NewOrganizationSuspensionWorker(agents.NewService(p, nil))
	require.Error(t, w.Process(t.Context()))
	require.ErrorIs(t, r.Organization.Reactivate(db, uuid.New()), models.ErrSuspensionCancellationPending)
	p.failure = nil
	p.stopped = true
	streamUnlock, acquired, err := tryAgentStreamLock(t.Context(), session.ID)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, w.Process(t.Context()))
	require.ErrorIs(t, r.Organization.Reactivate(db, uuid.New()), models.ErrSuspensionCancellationPending)
	streamUnlock()
	require.NoError(t, w.Process(t.Context()))
	require.NoError(t, r.Organization.Reactivate(db, uuid.New()))
}

func TestSuspensionDoesNotFinishDispatchConcurrentlyAdmittedFromQueue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "Admission race", "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	limit := 1
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &limit}})
	require.NoError(t, err)
	first, err := factory.CreateWorkOrder(db, "Running", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = first.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	_, running, err := line.Dispatch(db, first)
	require.NoError(t, err)
	second, err := factory.CreateWorkOrder(db, "Waiting", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = second.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	dispatch, queued, err := line.Dispatch(db, second)
	require.NoError(t, err)
	require.NotNil(t, queued.QueueItem)
	require.NoError(t, r.Organization.Suspend(db, uuid.New(), "Admission race"))
	require.NoError(t, db.Model(running.Execution).Update("status", models.FactoryWorkOrderExecutionStatusFinished).Error)
	admission := db.Begin()
	defer admission.Rollback()
	admitted, err := models.AdmitQueuedForStep(admission, line.ID, 0)
	require.NoError(t, err)
	require.Len(t, admitted, 1)
	done := make(chan error, 1)
	go func() { done <- NewOrganizationSuspensionWorker(nil).Process(t.Context()) }()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%factory_lines%')").Scan(&waiting).Error
		return err == nil && waiting
	}, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, admission.Commit().Error)
	require.NoError(t, <-done)
	stored, err := models.FindWorkOrderLineDispatch(db, dispatch.ID)
	require.NoError(t, err)
	require.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, stored.State)
	run, err := models.FindUnscopedCanvasRun(db, admitted[0].Run.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasRunStateCancelling, run.State)
}

func TestSuspensionWorkAndEventBatchesEventuallyComplete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	fleets := models.DefaultInstallationRunnerFleets("test")
	require.NoError(t, fleets[0].Create(db))
	for i := 0; i < 101; i++ {
		organization := r.Organization
		if i > 0 {
			var err error
			organization, err = models.CreateOrganization(uuid.NewString(), "")
			require.NoError(t, err)
		}
		require.NoError(t, organization.Suspend(db, uuid.New(), "Batch cancellation"))
	}
	for i := 0; i < 101; i++ {
		task := models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleets[0].ID, State: models.RunnerTaskStatePending, Backend: models.RunnerTaskBackendIntegrated, QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted")}
		require.NoError(t, db.Create(&task).Error)
	}
	w := NewOrganizationSuspensionWorker(nil)
	for i := 0; i < 4; i++ {
		require.NoError(t, w.Process(t.Context()))
	}
	var pending int64
	require.NoError(t, db.Model(&models.OrganizationSuspensionEvent{}).Where("cancellation_completed_at IS NULL").Count(&pending).Error)
	require.Zero(t, pending)
	require.NoError(t, r.Organization.Reactivate(db, uuid.New()))
}

func TestSuspensionDiscardsQueuesPreservesSchedulesAndWaitsForRunFinalization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "start", Type: models.NodeTypeTrigger},
		{NodeID: "schedule", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}})},
	}, nil)
	parent, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	child, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStatePending, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(child).Update("parent_run_id", parent.ID).Error)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	require.NoError(t, db.Model(event).Update("run_id", parent.ID).Error)
	queue := support.CreateQueueItem(t, canvas.ID, "start", event.ID, event.ID)
	require.NoError(t, db.Model(queue).Update("run_id", parent.ID).Error)
	request := models.CanvasNodeRequest{ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "start", State: models.NodeExecutionRequestStatePending, Type: models.NodeRequestTypeInvokeAction, RunAt: time.Now().Add(time.Hour)}
	require.NoError(t, db.Create(&request).Error)
	schedule := request
	schedule.ID = uuid.New()
	schedule.NodeID = "schedule"
	require.NoError(t, db.Create(&schedule).Error)
	factory, err := models.CreateFactory(db, r.Organization.ID, "Suspension queue", "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Queued task", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := models.FactoryWorkOrderLineDispatch{ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID, WorkOrderID: order.ID, LineID: line.ID, LineName: line.Name, State: models.FactoryWorkOrderLineDispatchStateActive}
	require.NoError(t, db.Create(&dispatch).Error)
	item := models.FactoryWorkOrderQueueItem{ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID, WorkOrderID: order.ID, LineID: line.ID, LineDispatchID: dispatch.ID, StepName: "Build", CreatedAt: time.Now()}
	require.NoError(t, db.Create(&item).Error)
	require.NoError(t, r.Organization.Suspend(db, uuid.New(), "Stop queued work"))
	w := NewOrganizationSuspensionWorker(nil)
	require.NoError(t, w.Process(t.Context()))
	for _, run := range []*models.CanvasRun{parent, child} {
		stored, err := models.FindUnscopedCanvasRun(db, run.ID)
		require.NoError(t, err)
		require.Equal(t, models.CanvasRunStateCancelling, stored.State)
	}
	require.ErrorIs(t, r.Organization.Reactivate(db, uuid.New()), models.ErrSuspensionCancellationPending)
	require.NoError(t, db.First(&dispatch, "id = ?", dispatch.ID).Error)
	require.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, dispatch.State)
	require.Equal(t, models.CanvasRunResultCancelled, dispatch.Result)
	require.NoError(t, db.First(&request, "id = ?", request.ID).Error)
	require.Equal(t, models.NodeExecutionRequestStateCompleted, request.State)
	require.NoError(t, db.First(&schedule, "id = ?", schedule.ID).Error)
	require.Equal(t, models.NodeExecutionRequestStatePending, schedule.State)
	finalizer := NewRunFinalizer("", r.Registry)
	require.NoError(t, finalizer.finalizeRun(canvas.ID, child.ID, runFinalizerTriggerSweep))
	require.NoError(t, finalizer.finalizeRun(canvas.ID, parent.ID, runFinalizerTriggerSweep))
	require.NoError(t, finalizer.sweepCancellingRuns())
	require.NoError(t, w.Process(t.Context()))
	require.NoError(t, r.Organization.Reactivate(db, uuid.New()))
	var remaining int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderQueueItem{}).Where("organization_id = ?", r.Organization.ID).Count(&remaining).Error)
	require.Zero(t, remaining)
}
