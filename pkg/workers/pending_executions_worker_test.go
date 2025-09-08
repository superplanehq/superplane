package workers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
)

const ExecutionStartedRoutingKey = "execution-started"

func Test__PendingExecutionsWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
		Approvals:   1,
	})

	defer r.Close()

	w := PendingExecutionsWorker{
		JwtSigner:   jwt.NewSigner("test"),
		Encryptor:   &crypto.NoOpEncryptor{},
		SpecBuilder: executors.SpecBuilder{},
		Registry:    r.Registry,
	}

	amqpURL, _ := config.RabbitMQURL()

	t.Run("semaphore workflow is triggered with simple parameters", func(t *testing.T) {
		//
		// Create stage that runs Semaphore workflow.
		//
		executorType, executorSpec, resource := support.Executor(t, r)
		stage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("stage-1").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		//
		// Create pending execution.
		//
		execution := support.CreateExecution(t, r.Source, stage)
		testconsumer := testconsumer.New(amqpURL, ExecutionStartedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Trigger the worker, and verify that request to scheduler was sent,
		// and that execution was moved to 'started' state.
		//
		err = w.Tick()
		require.NoError(t, err)
		execution, err = stage.FindExecutionByID(execution.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStarted, execution.State)
		assert.NotEmpty(t, execution.StartedAt)
		resources, err := execution.Resources()
		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, models.ExecutionResourcePending, resources[0].State)
		assert.True(t, testconsumer.HasReceivedMessage())

		req := r.SemaphoreAPIMock.LastRunWorkflow
		require.NotNil(t, req)
		assert.Equal(t, "refs/heads/main", req.Reference)
		assert.Equal(t, ".semaphore/run.yml", req.PipelineFile)
		assertParameters(t, req, execution, map[string]string{
			"PARAM_1": "VALUE_1",
			"PARAM_2": "VALUE_2",
		})
	})

	t.Run("semaphore workflow with resolved parameters is triggered", func(t *testing.T) {
		//
		// Create stage that runs Semaphore workflow.
		//
		executorType, _, resource := support.Executor(t, r)
		executorSpec, err := json.Marshal(map[string]any{
			"ref":          "refs/heads/main",
			"pipelineFile": ".semaphore/run.yml",
			"parameters": map[string]string{
				"REF":      "${{ inputs.REF }}",
				"REF_TYPE": "${{ inputs.REF_TYPE }}",
			},
		})
		require.NoError(t, err)

		stage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("stage-2").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceName: r.Source.Name,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithInputs([]models.InputDefinition{
				{Name: "REF"},
				{Name: "REF_TYPE"},
			}).
			WithInputMappings([]models.InputMapping{
				{
					Values: []models.ValueDefinition{
						{
							Name: "REF",
							ValueFrom: &models.ValueDefinitionFrom{
								EventData: &models.ValueDefinitionFromEventData{
									Connection: r.Source.Name,
									Expression: "ref",
								},
							},
						},
						{
							Name: "REF_TYPE",
							ValueFrom: &models.ValueDefinitionFrom{
								EventData: &models.ValueDefinitionFromEventData{
									Connection: r.Source.Name,
									Expression: "ref_type",
								},
							},
						},
					},
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		//
		// Create pending execution for a new event source event.
		//
		execution := support.CreateExecutionWithData(
			t, r.Source, stage,
			[]byte(`{"ref_type":"branch","ref":"refs/heads/test"}`),
			[]byte(`{}`),
			map[string]any{"REF": "refs/heads/test", "REF_TYPE": "branch"},
		)

		testconsumer := testconsumer.New(amqpURL, ExecutionStartedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Trigger the worker, and verify that request to scheduler was sent,
		// and that execution was moved to 'started' state.
		//
		err = w.Tick()
		require.NoError(t, err)
		execution, err = stage.FindExecutionByID(execution.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStarted, execution.State)
		assert.NotEmpty(t, execution.StartedAt)
		resources, err := execution.Resources()
		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, models.ExecutionResourcePending, resources[0].State)
		assert.True(t, testconsumer.HasReceivedMessage())

		req := r.SemaphoreAPIMock.LastRunWorkflow
		require.NotNil(t, req)
		assert.Equal(t, "refs/heads/main", req.Reference)
		assert.Equal(t, ".semaphore/run.yml", req.PipelineFile)
		assertParameters(t, req, execution, map[string]string{
			"REF":      "refs/heads/test",
			"REF_TYPE": "branch",
		})
	})

	t.Run("executions for soft-deleted stages are filtered out", func(t *testing.T) {
		executorType, executorSpec, resource := support.Executor(t, r)

		invalidStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("failing-stage-test1").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		validStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("valid-stage-test1").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		validExecution := support.CreateExecution(t, r.Source, validStage)

		err = database.Conn().Delete(&invalidStage).Error
		require.NoError(t, err)

		err = w.Tick()
		assert.NoError(t, err)

		//
		// Verify that the valid execution was processed (moved to started state).
		// The execution for the soft-deleted stage should be filtered out by ListExecutionsInState.
		//
		validExecution, err = validStage.FindExecutionByID(validExecution.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStarted, validExecution.State, "Valid execution should be processed because executions for soft-deleted stages are filtered out")
	})

	t.Run("executions for soft-deleted stages are filtered out and other executions continue", func(t *testing.T) {
		executorType, executorSpec, resource := support.Executor(t, r)

		softDeletedStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("soft-deleted-stage-test2").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)
		require.NoError(t, softDeletedStage.Delete())

		validStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("valid-stage-test2").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		//
		// Create pending executions for both stages.
		//
		support.CreateExecution(t, r.Source, softDeletedStage)
		validExecution := support.CreateExecution(t, r.Source, validStage)

		require.NoError(t, err)

		testconsumer := testconsumer.New(amqpURL, ExecutionStartedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Trigger the worker - it should skip the execution for the soft-deleted stage and process the valid one.
		//
		err = w.Tick()
		assert.NoError(t, err)

		//
		// Verify that the valid execution was processed successfully (moved to started state).
		//
		validExecution, err = validStage.FindExecutionByID(validExecution.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStarted, validExecution.State, "Valid execution should be processed even when another execution fails")
		assert.NotEmpty(t, validExecution.StartedAt)

		resources, err := validExecution.Resources()
		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, models.ExecutionResourcePending, resources[0].State)
		assert.True(t, testconsumer.HasReceivedMessage())
	})
}

func assertParameters(t *testing.T, req *semaphore.CreateWorkflowRequest, execution *models.StageExecution, parameters map[string]string) {
	all := map[string]string{
		"SUPERPLANE_STAGE_ID":           execution.StageID.String(),
		"SUPERPLANE_STAGE_EXECUTION_ID": execution.ID.String(),
	}

	for k, v := range parameters {
		all[k] = v
	}

	assert.Len(t, req.Parameters, len(all))
	for name, value := range all {
		v, ok := req.Parameters[name]
		assert.True(t, ok)
		assert.Equal(t, value, v)
	}
}
