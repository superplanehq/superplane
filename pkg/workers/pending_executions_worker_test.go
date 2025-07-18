package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
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
	}

	amqpURL, _ := config.RabbitMQURL()

	t.Run("semaphore workflow is triggered with simple parameters", func(t *testing.T) {
		//
		// Create stage that runs Semaphore workflow.
		//
		executorType, executorSpec, resource := support.Executor(r)
		stage, err := builders.NewStageBuilder().
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
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
		executorType, executorSpec, resource := support.Executor(r)
		executorSpec.Semaphore.Parameters = map[string]string{
			"REF":      "${{ inputs.REF }}",
			"REF_TYPE": "${{ inputs.REF_TYPE }}",
		}

		stage, err := builders.NewStageBuilder().
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
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
}

func assertParameters(t *testing.T, req *integrations.CreateWorkflowRequest, execution *models.StageExecution, parameters map[string]string) {
	all := map[string]string{
		"SEMAPHORE_STAGE_ID":           execution.StageID.String(),
		"SEMAPHORE_STAGE_EXECUTION_ID": execution.ID.String(),
	}

	for k, v := range parameters {
		all[k] = v
	}

	assert.Len(t, req.Parameters, len(all)+1)
	for name, value := range all {
		v, ok := req.Parameters[name]
		assert.True(t, ok)
		assert.Equal(t, value, v)
	}

	v, ok := req.Parameters["SEMAPHORE_STAGE_EXECUTION_TOKEN"]
	assert.True(t, ok)
	assert.NotEmpty(t, v)
}
