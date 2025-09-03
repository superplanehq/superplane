package workers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ExecutionPoller(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	defer r.Close()

	connections := []models.Connection{
		{
			SourceID:   r.Source.ID,
			SourceName: r.Source.Name,
			SourceType: models.SourceTypeEventSource,
		},
	}

	executorType, executorSpec, integrationResource := support.Executor(t, r)
	stage, err := builders.NewStageBuilder(r.Registry).
		WithEncryptor(r.Encryptor).
		InCanvas(r.Canvas.ID).
		WithName("stage-1").
		WithRequester(r.User).
		WithConnections(connections).
		WithExecutorType(executorType).
		WithExecutorSpec(executorSpec).
		ForResource(integrationResource).
		ForIntegration(r.Integration).
		WithInputs([]models.InputDefinition{{Name: "version"}}).
		WithInputMappings([]models.InputMapping{
			{
				Values: []models.ValueDefinition{
					{
						Name: "version",
						ValueFrom: &models.ValueDefinitionFrom{
							EventData: &models.ValueDefinitionFromEventData{
								Connection: r.Source.Name,
								Expression: "$.ref",
							},
						},
					},
				},
			},
		}).
		Create()

	require.NoError(t, err)
	resource, err := models.FindResource(r.Integration.ID, integrationResource.Type(), integrationResource.Name())
	require.NoError(t, err)

	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	w := NewExecutionPoller(r.Encryptor, r.Registry)

	t.Run("failed resource -> execution fails", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create failed resource
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage,
			[]byte(`{"ref":"v1"}`),
			[]byte(`{"ref":"v1"}`),
			map[string]any{"version": "v1"},
		)

		require.NoError(t, execution.Start())
		executionResource, err := execution.AddResource(workflowID, "workflow", resource.ID)
		require.NoError(t, err)
		require.NoError(t, executionResource.Finish(models.ResultFailed))

		testconsumer := testconsumer.New(amqpURL, messages.ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Trigger worker and verify execution goes to the finished state.
		//
		err = w.Tick()
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionByID(execution.ID)
			if err != nil {
				return false
			}

			return e.State == models.ExecutionFinished && e.Result == models.ResultFailed
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
		assert.Equal(t, models.ExecutionFinishedEventType, e.Type)
		assert.Equal(t, stage.ID.String(), e.Stage.ID)
		assert.Equal(t, execution.ID.String(), e.Execution.ID)
		assert.Equal(t, models.ResultFailed, e.Execution.Result)
		assert.Empty(t, e.Outputs)
		assert.Equal(t, map[string]any{"version": "v1"}, e.Inputs)
		assert.NotEmpty(t, e.Execution.CreatedAt)
		assert.NotEmpty(t, e.Execution.StartedAt)
		assert.NotEmpty(t, e.Execution.FinishedAt)
		require.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("missing required output -> execution fails", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		executorType, executorSpec, integrationResource := support.Executor(t, r)
		stageWithOutput, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("stage-with-output").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceName: r.Source.Name,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithOutputs([]models.OutputDefinition{{Name: "MY_OUTPUT", Required: true}}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)
		resource, err = models.FindResource(r.Integration.ID, resource.Type(), resource.Name())
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

		require.NoError(t, execution.Start())
		executionResource, err := execution.AddResource(workflowID, "workflow", resource.ID)
		require.NoError(t, err)
		require.NoError(t, executionResource.Finish(models.ResultPassed))

		testconsumer := testconsumer.New(amqpURL, messages.ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Trigger worker and verify execution eventually goes to the finished state,
		// with result = failed, even though the resource passed.
		//
		err = w.Tick()
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionByID(execution.ID)
			if err != nil {
				return false
			}

			return e.State == models.ExecutionFinished && e.Result == models.ResultFailed
		}, 5*time.Second, 200*time.Millisecond)
	})

	t.Run("passed resource -> execution passes", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage,
			[]byte(`{"ref":"v1"}`),
			[]byte(`{"ref":"v1"}`),
			map[string]any{"version": "v1"},
		)

		require.NoError(t, execution.Start())
		executionResource, err := execution.AddResource(workflowID, "workflow", resource.ID)
		require.NoError(t, err)
		require.NoError(t, executionResource.Finish(models.ResultPassed))

		testconsumer := testconsumer.New(amqpURL, messages.ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Trigger the worker and erify execution eventually
		// goes to the finished state, with result = failed.
		//
		err = w.Tick()
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionByID(execution.ID)
			if err != nil {
				return false
			}

			return e.State == models.ExecutionFinished && e.Result == models.ResultPassed
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
		assert.Equal(t, models.ExecutionFinishedEventType, e.Type)
		assert.Equal(t, stage.ID.String(), e.Stage.ID)
		assert.Equal(t, execution.ID.String(), e.Execution.ID)
		assert.Equal(t, models.ResultPassed, e.Execution.Result)
		assert.Empty(t, e.Outputs)
		assert.Equal(t, map[string]any{"version": "v1"}, e.Inputs)
		assert.NotEmpty(t, e.Execution.CreatedAt)
		assert.NotEmpty(t, e.Execution.StartedAt)
		assert.NotEmpty(t, e.Execution.FinishedAt)
		require.True(t, testconsumer.HasReceivedMessage())
	})
}

func unmarshalCompletionEvent(raw []byte) (*models.ExecutionFinishedEvent, error) {
	e := models.ExecutionFinishedEvent{}
	err := json.Unmarshal(raw, &e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}
