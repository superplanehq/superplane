package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const ExecutionCreatedRoutingKey = "execution-created"

func Test__PendingStageEventsWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})
	w, _ := NewPendingStageEventsWorker(func() time.Time {
		return time.Now()
	})

	amqpURL, _ := config.RabbitMQURL()

	t.Run("stage does not require approval -> creates execution", func(t *testing.T) {
		//
		// Create stage that does not require approval.
		//
		require.NoError(t, r.Canvas.CreateStage("stage-no-approval-1", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-no-approval-1")
		require.NoError(t, err)
		testconsumer := testconsumer.New(amqpURL, ExecutionCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Create a pending stage event, and trigger the worker.
		//
		event := support.CreateStageEvent(t, r.Source, stage)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify that a new execution record was created and event moves to waiting(execution).
		//
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonExecution, event.StateReason)
		execution, err := models.FindExecutionInState(stage.ID, []string{models.StageExecutionPending})
		require.NoError(t, err)
		assert.NotEmpty(t, execution.ID)
		assert.NotEmpty(t, execution.CreatedAt)
		assert.Equal(t, execution.StageID, stage.ID)
		assert.Equal(t, execution.StageEventID, event.ID)
		assert.Equal(t, execution.State, models.StageExecutionPending)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("stage requires approval and none was given -> waiting-for-approval state", func(t *testing.T) {
		//
		// Create stage that requires approval.
		//
		conditions := []models.StageCondition{
			{Type: models.StageConditionTypeApproval, Approval: &models.ApprovalCondition{Count: 1}},
		}

		require.NoError(t, r.Canvas.CreateStage("stage-with-approval-1", r.User.String(), conditions, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-with-approval-1")
		require.NoError(t, err)

		//
		// Create a pending stage event, and trigger the worker.
		//
		event := support.CreateStageEvent(t, r.Source, stage)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify that event was moved to the waiting(approval) state.
		//
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonApproval, event.StateReason)
	})

	t.Run("stage requires approval and approval was given -> creates execution", func(t *testing.T) {
		//
		// Create stage that requires approval.
		//
		conditions := []models.StageCondition{
			{Type: models.StageConditionTypeApproval, Approval: &models.ApprovalCondition{Count: 1}},
		}
		require.NoError(t, r.Canvas.CreateStage("stage-with-approval-2", r.User.String(), conditions, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-with-approval-2")
		require.NoError(t, err)

		testconsumer := testconsumer.New(amqpURL, ExecutionCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Create a pending stage event, approve it, and trigger the worker.
		//
		event := support.CreateStageEvent(t, r.Source, stage)
		require.NoError(t, event.Approve(uuid.New()))
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify that a new execution record was created and event moves to waiting(execution)
		//
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonExecution, event.StateReason)
		execution, err := models.FindExecutionInState(stage.ID, []string{models.StageExecutionPending})
		require.NoError(t, err)
		assert.NotEmpty(t, execution.ID)
		assert.NotEmpty(t, execution.CreatedAt)
		assert.Equal(t, execution.StageID, stage.ID)
		assert.Equal(t, execution.StageEventID, event.ID)
		assert.Equal(t, execution.State, models.StageExecutionPending)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("stage requires time window and event is outside of it -> moves to waiting", func(t *testing.T) {
		//
		// Create stage that requires time window.
		//
		conditions := []models.StageCondition{
			{
				Type: models.StageConditionTypeTimeWindow,
				TimeWindow: &models.TimeWindowCondition{
					Start:    "08:00",
					End:      "17:00",
					WeekDays: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
				},
			},
		}
		require.NoError(t, r.Canvas.CreateStage("stage-with-time-window", r.User.String(), conditions, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-with-time-window")
		require.NoError(t, err)

		//
		// Create a pending stage event, and trigger the worker.
		//
		event := support.CreateStageEvent(t, r.Source, stage)
		require.NoError(t, event.Approve(uuid.New()))
		w, _ := NewPendingStageEventsWorker(func() time.Time {
			return time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC)
		})

		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify that event was moved to the waiting(time-window) state.
		//
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonTimeWindow, event.StateReason)
	})

	t.Run("stage requires time window and event is inside of it -> creates execution", func(t *testing.T) {
		testconsumer := testconsumer.New(amqpURL, ExecutionCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Create stage that requires time window.
		//
		conditions := []models.StageCondition{
			{
				Type: models.StageConditionTypeTimeWindow,
				TimeWindow: &models.TimeWindowCondition{
					Start:    "08:00",
					End:      "17:00",
					WeekDays: []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
				},
			},
		}
		require.NoError(t, r.Canvas.CreateStage("stage-with-time-window-2", r.User.String(), conditions, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-with-time-window-2")
		require.NoError(t, err)

		//
		// Create a pending stage event, and trigger the worker.
		//
		event := support.CreateStageEvent(t, r.Source, stage)
		require.NoError(t, event.Approve(uuid.New()))
		w, _ := NewPendingStageEventsWorker(func() time.Time {
			return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		})

		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify that a new execution record was created and event moves to waiting(execution)
		//
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonExecution, event.StateReason)
		execution, err := models.FindExecutionInState(stage.ID, []string{models.StageExecutionPending})
		require.NoError(t, err)
		assert.NotEmpty(t, execution.ID)
		assert.NotEmpty(t, execution.CreatedAt)
		assert.Equal(t, execution.StageID, stage.ID)
		assert.Equal(t, execution.StageEventID, event.ID)
		assert.Equal(t, execution.State, models.StageExecutionPending)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("another execution already in progress -> remains in pending state", func(t *testing.T) {
		//
		// Create stage that does not requires approval.
		//
		require.NoError(t, r.Canvas.CreateStage("stage-no-approval-3", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-no-approval-3")
		require.NoError(t, err)

		//
		// Create a pending stage event, trigger the worker,
		// and verify that event is moved to waiting(execution).
		//
		event := support.CreateStageEvent(t, r.Source, stage)
		err = w.Tick()
		require.NoError(t, err)
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStateWaiting, event.State)
		require.Equal(t, models.StageEventStateReasonExecution, event.StateReason)

		//
		// Add another pending event for this stage,
		// trigger the worker, and verify that it remained in the pending state.
		//
		event = support.CreateStageEvent(t, r.Source, stage)
		err = w.Tick()
		require.NoError(t, err)
		event, err = models.FindStageEventByID(event.ID.String(), stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.StageEventStatePending, event.State)
	})
}
