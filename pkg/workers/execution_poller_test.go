package workers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const ExecutionFinishedRoutingKey = "execution-finished"

func Test__ExecutionPoller(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:       true,
		SemaphoreAPI: true,
	})

	defer r.Close()

	connections := []models.Connection{
		{
			SourceID:   r.Source.ID,
			SourceType: models.SourceTypeEventSource,
		},
	}

	spec := support.ExecutorSpecWithURL(r.SemaphoreAPIMock.Server.URL)
	err := r.Canvas.CreateStage("stage-1", r.User.String(), []models.StageCondition{}, spec, connections, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}, []models.ValueDefinition{})
	require.NoError(t, err)
	stage, err := r.Canvas.FindStageByName("stage-1")
	require.NoError(t, err)

	encryptor := &crypto.NoOpEncryptor{}

	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	w := NewExecutionPoller(encryptor)

	t.Run("failed pipeline -> execution fails", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage,
			[]byte(`{"ref":"v1"}`),
			[]byte(`{"ref":"v1"}`),
			map[string]any{},
		)

		require.NoError(t, execution.StartWithReferenceID(workflowID))

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock failed result and tick worker
		//
		pipelineID := uuid.New().String()
		r.SemaphoreAPIMock.AddPipeline(pipelineID, workflowID, integrations.SemaphorePipelineResultFailed)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify execution eventually goes to the finished state, with result = failed.
		//
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionByID(execution.ID)
			if err != nil {
				return false
			}

			return e.State == models.StageExecutionFinished && e.Result == models.StageExecutionResultFailed
		}, 5*time.Second, 200*time.Millisecond)

		//
		// Verify that new pending event for stage completion is created.
		//
		list, err := models.ListEventsBySourceID(stage.ID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, list[0].State, models.StageEventStatePending)
		assert.Equal(t, list[0].SourceID, stage.ID)
		assert.Equal(t, list[0].SourceType, models.SourceTypeStage)
		e, err := unmarshalCompletionEvent(list[0].Raw)
		require.NoError(t, err)
		assert.Equal(t, models.StageExecutionCompletionType, e.Type)
		assert.Equal(t, stage.ID.String(), e.Stage.ID)
		assert.Equal(t, execution.ID.String(), e.Execution.ID)
		assert.Equal(t, models.StageExecutionResultFailed, e.Execution.Result)
		assert.NotEmpty(t, e.Execution.CreatedAt)
		assert.NotEmpty(t, e.Execution.StartedAt)
		assert.NotEmpty(t, e.Execution.FinishedAt)
		require.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("missing required output -> execution fails", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		spec := support.ExecutorSpecWithURL(r.SemaphoreAPIMock.Server.URL)
		err = r.Canvas.CreateStage("stage-with-output", r.User.String(), []models.StageCondition{}, spec, []models.Connection{
			{
				SourceID:   r.Source.ID,
				SourceName: r.Source.Name,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{
			{Name: "MY_OUTPUT", Required: true},
		}, []models.ValueDefinition{})

		require.NoError(t, err)
		stageWithOutput, err := r.Canvas.FindStageByName("stage-with-output")
		require.NoError(t, err)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stageWithOutput,
			[]byte(`{}`),
			[]byte(`{}`),
			map[string]any{},
		)

		require.NoError(t, execution.StartWithReferenceID(workflowID))

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock passed result and tick worker
		//
		pipelineID := uuid.New().String()
		r.SemaphoreAPIMock.AddPipeline(pipelineID, workflowID, integrations.SemaphorePipelineResultPassed)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify execution eventually goes to the finished state, with result = failed,
		// even though the semaphore pipeline passed.
		//
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionByID(execution.ID)
			if err != nil {
				return false
			}

			return e.State == models.StageExecutionFinished && e.Result == models.StageExecutionResultFailed
		}, 5*time.Second, 200*time.Millisecond)
	})

	t.Run("passed pipeline -> execution passes", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage,
			[]byte(`{"ref":"v1"}`),
			[]byte(`{"ref":"v1"}`),
			map[string]any{},
		)

		require.NoError(t, execution.StartWithReferenceID(workflowID))

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock passed result and tick worker
		//
		pipelineID := uuid.New().String()
		r.SemaphoreAPIMock.AddPipeline(pipelineID, workflowID, integrations.SemaphorePipelineResultPassed)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify execution eventually goes to the finished state, with result = failed.
		//
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionByID(execution.ID)
			if err != nil {
				return false
			}

			return e.State == models.StageExecutionFinished && e.Result == models.StageExecutionResultPassed
		}, 5*time.Second, 200*time.Millisecond)

		//
		// Verify that new pending event for stage completion is created with proper result.
		//
		list, err := models.ListEventsBySourceID(stage.ID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, list[0].State, models.StageEventStatePending)
		assert.Equal(t, list[0].SourceID, stage.ID)
		assert.Equal(t, list[0].SourceType, models.SourceTypeStage)
		e, err := unmarshalCompletionEvent(list[0].Raw)
		require.NoError(t, err)
		assert.Equal(t, models.StageExecutionCompletionType, e.Type)
		assert.Equal(t, stage.ID.String(), e.Stage.ID)
		assert.Equal(t, execution.ID.String(), e.Execution.ID)
		assert.Equal(t, models.StageExecutionResultPassed, e.Execution.Result)
		assert.NotEmpty(t, e.Execution.CreatedAt)
		assert.NotEmpty(t, e.Execution.StartedAt)
		assert.NotEmpty(t, e.Execution.FinishedAt)
		require.True(t, testconsumer.HasReceivedMessage())
	})
}

func unmarshalCompletionEvent(raw []byte) (*models.StageExecutionCompletion, error) {
	e := models.StageExecutionCompletion{}
	err := json.Unmarshal(raw, &e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}
