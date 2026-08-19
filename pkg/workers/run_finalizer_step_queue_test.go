package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type stepQueueFixture struct {
	factory *models.Factory
	line    *models.FactoryLine
}

func stepMaxParallelism(limit int) *int {
	return &limit
}

// setupStepQueueLine creates a factory line whose steps use the given
// maxParallelism values, one app per step.
func setupStepQueueLine(t *testing.T, r *support.ResourceRegistry, stepMaxParallelisms []*int) *stepQueueFixture {
	t.Helper()

	f, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := f.CreateLine(database.Conn(), support.RandomName("line"), nil)
	require.NoError(t, err)

	steps := make([]models.FactoryLineStep, len(stepMaxParallelisms))
	for i, limit := range stepMaxParallelisms {
		name := support.RandomName("step")
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, f.ID, name, "start-"+name)
		steps[i] = models.FactoryLineStep{
			Type:           models.FactoryLineStepTypeRunApp,
			AppID:          app.ID,
			Entrypoint:     entrypoint,
			MaxParallelism: limit,
		}
	}
	require.NoError(t, line.Update(database.Conn(), nil, steps))

	return &stepQueueFixture{factory: f, line: line}
}

func (f *stepQueueFixture) createOpenWorkOrder(t *testing.T, r *support.ResourceRegistry, title string) *models.FactoryWorkOrder {
	t.Helper()

	order, err := f.factory.CreateWorkOrder(database.Conn(), title, "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)
	return order
}

// dispatchLine runs the dispatch flow (create the traversal, then start or
// queue step 0) in one transaction, like the DispatchWorkOrder API does.
func (f *stepQueueFixture) dispatchLine(
	t *testing.T,
	order *models.FactoryWorkOrder,
) (*models.FactoryWorkOrderLineDispatch, *models.FactoryLineStepResult) {
	t.Helper()

	var dispatch *models.FactoryWorkOrderLineDispatch
	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		dispatch, result, err = f.line.Dispatch(tx, order)
		return err
	}))
	return dispatch, result
}

func finishRun(t *testing.T, run *models.CanvasRun, result string) {
	t.Helper()

	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      result,
		"updated_at":  &now,
		"finished_at": &now,
	}).Error)
}

func advanceFactoryLine(t *testing.T, r *support.ResourceRegistry, runID uuid.UUID) []factoryLinePendingRun {
	t.Helper()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	var pending []factoryLinePendingRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		pending, err = finalizer.executeNextFactoryLineStep(tx, runID)
		return err
	}))
	return pending
}

func queueItemForDispatch(t *testing.T, dispatchID uuid.UUID) *models.FactoryWorkOrderQueueItemRecord {
	t.Helper()

	byDispatch, err := models.ListFactoryWorkOrderQueueItemsByLineDispatchIDs(database.Conn(), []uuid.UUID{dispatchID})
	require.NoError(t, err)
	if record, ok := byDispatch[dispatchID]; ok {
		return &record
	}
	return nil
}

func executionsForOrder(t *testing.T, orderID uuid.UUID) []models.FactoryWorkOrderExecution {
	t.Helper()

	var executions []models.FactoryWorkOrderExecution
	require.NoError(t, database.Conn().
		Where("work_order_id = ?", orderID).
		Order("created_at ASC").
		Find(&executions).Error)
	return executions
}

func reloadDispatch(t *testing.T, dispatchID uuid.UUID) *models.FactoryWorkOrderLineDispatch {
	t.Helper()

	dispatch, err := models.FindWorkOrderLineDispatch(database.Conn(), dispatchID)
	require.NoError(t, err)
	return dispatch
}

func Test__StepQueue_DispatchQueuesWhenStepAtCapacity(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")

	_, firstResult := fixture.dispatchLine(t, first)
	require.NotNil(t, firstResult.Run)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, firstResult.Execution.Status)

	secondDispatch, secondResult := fixture.dispatchLine(t, second)
	assert.Nil(t, secondResult.Run)
	assert.Nil(t, secondResult.Execution)
	require.NotNil(t, secondResult.QueueItem)
	assert.Equal(t, second.ID, secondResult.QueueItem.WorkOrderID)

	// A queued traversal stays active, holds a queue item, and has no
	// execution yet.
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, secondDispatch.State)
	item := queueItemForDispatch(t, secondDispatch.ID)
	require.NotNil(t, item)
	assert.Equal(t, 1, item.Position)
	assert.Empty(t, executionsForOrder(t, second.ID))

	// The queued decision is recorded as a work order event.
	var queuedEvents int64
	require.NoError(t, database.Conn().
		Model(&models.FactoryWorkOrderEvent{}).
		Where("work_order_id = ? AND type = ?", second.ID, factory.EventTypeLineStepExecutionQueued).
		Count(&queuedEvents).Error)
	assert.Equal(t, int64(1), queuedEvents)
}

func Test__StepQueue_TerminalRunAdmitsOldestQueuedWorkOrder(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")
	third := fixture.createOpenWorkOrder(t, r, "Third")

	firstDispatch, firstResult := fixture.dispatchLine(t, first)
	require.NotNil(t, firstResult.Run)

	secondDispatch, secondResult := fixture.dispatchLine(t, second)
	require.NotNil(t, secondResult.QueueItem)

	thirdDispatch, thirdResult := fixture.dispatchLine(t, third)
	require.NotNil(t, thirdResult.QueueItem)

	// Queue positions follow enqueue order.
	secondItem := queueItemForDispatch(t, secondDispatch.ID)
	require.NotNil(t, secondItem)
	assert.Equal(t, 1, secondItem.Position)
	thirdItem := queueItemForDispatch(t, thirdDispatch.ID)
	require.NotNil(t, thirdItem)
	assert.Equal(t, 2, thirdItem.Position)

	// A failed run frees its slot the same way a passed run does.
	finishRun(t, firstResult.Run, models.CanvasRunResultFailed)
	pending := advanceFactoryLine(t, r, firstResult.Run.ID)
	require.Len(t, pending, 1)

	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloadDispatch(t, firstDispatch.ID).State)

	// Second was admitted: its queue item is gone, its own traversal now
	// has an execution.
	assert.Nil(t, queueItemForDispatch(t, secondDispatch.ID))
	admitted := executionsForOrder(t, second.ID)
	require.Len(t, admitted, 1)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, admitted[0].Status)
	assert.Equal(t, secondDispatch.ID, admitted[0].LineDispatchID)
	assert.Equal(t, pending[0].runID, admitted[0].RunID)

	// Third is still queued, now first in line: only one slot was freed.
	thirdItem = queueItemForDispatch(t, thirdDispatch.ID)
	require.NotNil(t, thirdItem)
	assert.Equal(t, 1, thirdItem.Position)
	assert.Empty(t, executionsForOrder(t, third.ID))
}

// Closing a work order that waits in a step's queue must abandon the
// traversal right away: no run exists that could ever finish it later, and
// a zombie active dispatch would block re-dispatch after a reopen.
func Test__StepQueue_CloseDropsQueuedWorkAndCancelsDispatch(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")

	firstDispatch, firstResult := fixture.dispatchLine(t, first)
	require.NotNil(t, firstResult.Run)

	secondDispatch, secondResult := fixture.dispatchLine(t, second)
	require.NotNil(t, secondResult.QueueItem)

	_, err := second.Close(database.Conn(), models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)

	assert.Nil(t, queueItemForDispatch(t, secondDispatch.ID))
	reloaded := reloadDispatch(t, secondDispatch.ID)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
	assert.Equal(t, models.CanvasRunResultCancelled, reloaded.Result)

	// The freed queue entry does not resurrect: when the first order's run
	// finishes, there is nothing left to admit.
	finishRun(t, firstResult.Run, models.CanvasRunResultPassed)
	pending := advanceFactoryLine(t, r, firstResult.Run.ID)
	assert.Empty(t, pending)
	assert.Empty(t, executionsForOrder(t, second.ID))
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloadDispatch(t, firstDispatch.ID).State)
}

// Safety net: a queue item whose order closed without going through the
// close path (which drops queued work) is skipped at admission — its
// traversal finishes as cancelled and the next open order is admitted.
func Test__StepQueue_ClosedWorkOrderIsSkippedAtAdmission(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")
	third := fixture.createOpenWorkOrder(t, r, "Third")

	_, firstResult := fixture.dispatchLine(t, first)
	require.NotNil(t, firstResult.Run)

	secondDispatch, secondResult := fixture.dispatchLine(t, second)
	require.NotNil(t, secondResult.QueueItem)

	thirdDispatch, thirdResult := fixture.dispatchLine(t, third)
	require.NotNil(t, thirdResult.QueueItem)

	// Close the second order behind the FSM's back, so its queue item
	// survives until admission looks at it.
	require.NoError(t, database.Conn().
		Model(&models.FactoryWorkOrder{}).
		Where("id = ?", second.ID).
		Updates(map[string]any{"state": models.FactoryWorkOrderStateClosed, "result": models.FactoryWorkOrderResultRejected}).
		Error)

	finishRun(t, firstResult.Run, models.CanvasRunResultPassed)
	pending := advanceFactoryLine(t, r, firstResult.Run.ID)
	require.Len(t, pending, 1)

	// The closed order's queue item is dropped without an execution and
	// its traversal is abandoned...
	assert.Nil(t, queueItemForDispatch(t, secondDispatch.ID))
	assert.Empty(t, executionsForOrder(t, second.ID))
	reloaded := reloadDispatch(t, secondDispatch.ID)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
	assert.Equal(t, models.CanvasRunResultCancelled, reloaded.Result)

	// ...and the next open order is admitted instead.
	assert.Nil(t, queueItemForDispatch(t, thirdDispatch.ID))
	admitted := executionsForOrder(t, third.ID)
	require.Len(t, admitted, 1)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, admitted[0].Status)
	assert.Equal(t, pending[0].runID, admitted[0].RunID)
}

func Test__StepQueue_AdvancementQueuesWhenNextStepAtCapacity(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(2), stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")

	// Both orders start step one (capacity 2).
	firstDispatch, firstStepOne := fixture.dispatchLine(t, first)
	require.NotNil(t, firstStepOne.Run)
	secondDispatch, secondStepOne := fixture.dispatchLine(t, second)
	require.NotNil(t, secondStepOne.Run)

	// First order passes step one and takes the single step-two slot.
	finishRun(t, firstStepOne.Run, models.CanvasRunResultPassed)
	firstPending := advanceFactoryLine(t, r, firstStepOne.Run.ID)
	require.Len(t, firstPending, 1)

	// Second order passes step one; step two is full, so its traversal
	// queues there and stays active.
	finishRun(t, secondStepOne.Run, models.CanvasRunResultPassed)
	secondPending := advanceFactoryLine(t, r, secondStepOne.Run.ID)
	assert.Empty(t, secondPending)

	queued := queueItemForDispatch(t, secondDispatch.ID)
	require.NotNil(t, queued)
	assert.Equal(t, 1, queued.StepIndex)
	assert.Equal(t, 1, queued.Position)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, reloadDispatch(t, secondDispatch.ID).State)

	// When the first order's step-two run finishes, the second is admitted
	// into its own traversal at the queued step.
	var firstStepTwoRun models.CanvasRun
	require.NoError(t, database.Conn().Where("id = ?", firstPending[0].runID).First(&firstStepTwoRun).Error)
	finishRun(t, &firstStepTwoRun, models.CanvasRunResultPassed)
	thirdPending := advanceFactoryLine(t, r, firstStepTwoRun.ID)
	require.Len(t, thirdPending, 1)

	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloadDispatch(t, firstDispatch.ID).State)
	assert.Nil(t, queueItemForDispatch(t, secondDispatch.ID))

	var admitted models.FactoryWorkOrderExecution
	require.NoError(t, database.Conn().
		Where("work_order_id = ? AND step_index = 1", second.ID).
		First(&admitted).Error)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, admitted.Status)
	assert.Equal(t, secondDispatch.ID, admitted.LineDispatchID)
	assert.Equal(t, thirdPending[0].runID, admitted.RunID)
}
