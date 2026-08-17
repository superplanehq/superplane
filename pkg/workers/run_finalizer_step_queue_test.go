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
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type stepQueueFixture struct {
	factory *models.Factory
	line    *models.FactoryLine
	steps   []models.FactoryLineStep
}

func stepMaxParallelism(limit int) *int {
	return &limit
}

// setupStepQueueLine creates a factory line whose steps use the given
// maxParallelism values, one app per step.
func setupStepQueueLine(t *testing.T, r *support.ResourceRegistry, stepMaxParallelisms []*int) *stepQueueFixture {
	t.Helper()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(database.Conn(), support.RandomName("line"), nil)
	require.NoError(t, err)

	steps := make([]models.FactoryLineStep, len(stepMaxParallelisms))
	for i, limit := range stepMaxParallelisms {
		name := support.RandomName("step")
		app, entrypoint := createFactoryAppWithOnRunTrigger(t, r, factory.ID, name, "start-"+name)
		steps[i] = models.FactoryLineStep{
			Name:           name,
			Type:           models.FactoryLineStepTypeRunApp,
			AppID:          app.ID,
			Entrypoint:     entrypoint,
			MaxParallelism: limit,
		}
	}
	require.NoError(t, line.Update(database.Conn(), nil, steps))

	return &stepQueueFixture{factory: factory, line: line, steps: steps}
}

func (f *stepQueueFixture) createOpenWorkOrder(t *testing.T, r *support.ResourceRegistry, title string) *models.FactoryWorkOrder {
	t.Helper()

	order, err := f.factory.CreateWorkOrder(database.Conn(), title, "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)
	return order
}

func (f *stepQueueFixture) enqueueOrStartStep(t *testing.T, order *models.FactoryWorkOrder, stepIndex int) *models.FactoryLineStepResult {
	t.Helper()

	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = f.line.EnqueueOrStartStep(tx, order, stepIndex)
		return err
	}))
	return result
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

func queueItemsForOrder(t *testing.T, orderID uuid.UUID) []models.FactoryWorkOrderQueueItemRecord {
	t.Helper()

	byOrder, err := models.ListFactoryWorkOrderQueueItemsByWorkOrderIDs(database.Conn(), []uuid.UUID{orderID})
	require.NoError(t, err)
	return byOrder[orderID]
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

func Test__StepQueue_WorkOrderWaitsWhenStepAtCapacity(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")

	firstResult := fixture.enqueueOrStartStep(t, first, 0)
	require.NotNil(t, firstResult.Run)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, firstResult.Execution.Status)

	secondResult := fixture.enqueueOrStartStep(t, second, 0)
	assert.Nil(t, secondResult.Run)
	assert.Nil(t, secondResult.Execution)
	require.NotNil(t, secondResult.QueueItem)
	assert.Equal(t, second.ID, secondResult.QueueItem.WorkOrderID)

	// A queued order has a queue item and no execution yet.
	items := queueItemsForOrder(t, second.ID)
	require.Len(t, items, 1)
	assert.Equal(t, 1, items[0].Position)
	assert.Empty(t, executionsForOrder(t, second.ID))

	// The queued decision is recorded as a work order event.
	var queuedEvents int64
	require.NoError(t, database.Conn().
		Model(&models.FactoryWorkOrderEvent{}).
		Where("work_order_id = ? AND type = ?", second.ID, "step.execution.queued").
		Count(&queuedEvents).Error)
	assert.Equal(t, int64(1), queuedEvents)
}

func Test__StepQueue_UnlimitedStepAlwaysStarts(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(0)})

	for _, title := range []string{"One", "Two", "Three"} {
		order := fixture.createOpenWorkOrder(t, r, title)
		result := fixture.enqueueOrStartStep(t, order, 0)
		require.NotNil(t, result.Run, "unlimited step must start immediately")
	}
}

func Test__StepQueue_TerminalRunAdmitsOldestQueuedWorkOrder(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")
	third := fixture.createOpenWorkOrder(t, r, "Third")

	firstResult := fixture.enqueueOrStartStep(t, first, 0)
	require.NotNil(t, firstResult.Run)

	secondResult := fixture.enqueueOrStartStep(t, second, 0)
	require.NotNil(t, secondResult.QueueItem)

	thirdResult := fixture.enqueueOrStartStep(t, third, 0)
	require.NotNil(t, thirdResult.QueueItem)

	// Queue positions follow enqueue order.
	secondItems := queueItemsForOrder(t, second.ID)
	require.Len(t, secondItems, 1)
	assert.Equal(t, 1, secondItems[0].Position)
	thirdItems := queueItemsForOrder(t, third.ID)
	require.Len(t, thirdItems, 1)
	assert.Equal(t, 2, thirdItems[0].Position)

	// A failed run frees its slot the same way a passed run does.
	finishRun(t, firstResult.Run, models.CanvasRunResultFailed)
	pending := advanceFactoryLine(t, r, firstResult.Run.ID)

	require.Len(t, pending, 1)

	// Second was admitted: its queue item is gone, an execution exists.
	assert.Empty(t, queueItemsForOrder(t, second.ID))
	admitted := executionsForOrder(t, second.ID)
	require.Len(t, admitted, 1)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, admitted[0].Status)
	assert.Equal(t, pending[0].runID, admitted[0].RunID)

	// Third is still queued, now first in line: only one slot was freed.
	thirdItems = queueItemsForOrder(t, third.ID)
	require.Len(t, thirdItems, 1)
	assert.Equal(t, 1, thirdItems[0].Position)
	assert.Empty(t, executionsForOrder(t, third.ID))
}

func Test__StepQueue_ClosedWorkOrderIsSkippedAtAdmission(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(1)})

	first := fixture.createOpenWorkOrder(t, r, "First")
	second := fixture.createOpenWorkOrder(t, r, "Second")
	third := fixture.createOpenWorkOrder(t, r, "Third")

	firstResult := fixture.enqueueOrStartStep(t, first, 0)
	require.NotNil(t, firstResult.Run)

	secondResult := fixture.enqueueOrStartStep(t, second, 0)
	require.NotNil(t, secondResult.QueueItem)

	thirdResult := fixture.enqueueOrStartStep(t, third, 0)
	require.NotNil(t, thirdResult.QueueItem)

	// Close the second order while it waits in the step queue.
	_, err := second.Close(database.Conn(), models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)

	finishRun(t, firstResult.Run, models.CanvasRunResultPassed)
	pending := advanceFactoryLine(t, r, firstResult.Run.ID)
	require.Len(t, pending, 1)

	// The closed order's queue item is dropped without an execution...
	assert.Empty(t, queueItemsForOrder(t, second.ID))
	assert.Empty(t, executionsForOrder(t, second.ID))

	// ...and the next open order is admitted instead.
	assert.Empty(t, queueItemsForOrder(t, third.ID))
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
	firstStepOne := fixture.enqueueOrStartStep(t, first, 0)
	require.NotNil(t, firstStepOne.Run)
	secondStepOne := fixture.enqueueOrStartStep(t, second, 0)
	require.NotNil(t, secondStepOne.Run)

	// First order passes step one and takes the single step-two slot.
	finishRun(t, firstStepOne.Run, models.CanvasRunResultPassed)
	firstPending := advanceFactoryLine(t, r, firstStepOne.Run.ID)
	require.Len(t, firstPending, 1)

	// Second order passes step one; step two is full, so it queues.
	finishRun(t, secondStepOne.Run, models.CanvasRunResultPassed)
	secondPending := advanceFactoryLine(t, r, secondStepOne.Run.ID)
	assert.Empty(t, secondPending)

	queued := queueItemsForOrder(t, second.ID)
	require.Len(t, queued, 1)
	assert.Equal(t, 1, queued[0].StepIndex)
	assert.Equal(t, 1, queued[0].Position)

	// When the first order's step-two run finishes, the second is admitted.
	var firstStepTwoRun models.CanvasRun
	require.NoError(t, database.Conn().Where("id = ?", firstPending[0].runID).First(&firstStepTwoRun).Error)
	finishRun(t, &firstStepTwoRun, models.CanvasRunResultPassed)
	thirdPending := advanceFactoryLine(t, r, firstStepTwoRun.ID)
	require.Len(t, thirdPending, 1)

	assert.Empty(t, queueItemsForOrder(t, second.ID))

	var admitted models.FactoryWorkOrderExecution
	require.NoError(t, database.Conn().
		Where("work_order_id = ? AND step_index = 1", second.ID).
		First(&admitted).Error)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, admitted.Status)
	assert.Equal(t, thirdPending[0].runID, admitted.RunID)
}
