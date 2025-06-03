package workers

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/apis/semaphore"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const ExecutionStartedRoutingKey = "execution-started"

func Test__PendingExecutionsWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source: true, Stage: true, SemaphoreAPI: true, Approvals: 1,
	})

	defer r.Close()

	w := PendingExecutionsWorker{
		JwtSigner: jwt.NewSigner("test"),
		Encryptor: &crypto.NoOpEncryptor{},
	}

	amqpURL, _ := config.RabbitMQURL()

	t.Run("semaphore task is triggered with simple parameters", func(t *testing.T) {
		//
		// Create stage that trigger Semaphore task.
		//
		spec := support.ExecutorSpecWithURL(r.SemaphoreAPIMock.Server.URL)
		require.NoError(t, r.Canvas.CreateStage("stage-task", r.User.String(), []models.StageCondition{}, spec, []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}, []models.ValueDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-task")

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
		assert.Equal(t, models.StageExecutionStarted, execution.State)
		assert.NotEmpty(t, execution.ReferenceID)
		assert.NotEmpty(t, execution.StartedAt)
		assert.True(t, testconsumer.HasReceivedMessage())

		req := r.SemaphoreAPIMock.LastTaskTrigger
		require.NotNil(t, req)
		assert.Equal(t, "main", req.Spec.Branch)
		assert.Equal(t, ".semaphore/run.yml", req.Spec.PipelineFile)
		assertParameters(t, req, execution, map[string]string{
			"PARAM_1": "VALUE_1",
			"PARAM_2": "VALUE_2",
		})
	})

	t.Run("semaphore task with resolved parameters is triggered", func(t *testing.T) {
		//
		// Create stage that trigger Semaphore task.
		//
		spec := support.ExecutorSpecWithURL(r.SemaphoreAPIMock.Server.URL)
		spec.Semaphore.Parameters = map[string]string{
			"REF":      "${{ inputs.REF }}",
			"REF_TYPE": "${{ inputs.REF_TYPE }}",
		}

		require.NoError(t, r.Canvas.CreateStage("stage-task-2", r.User.String(), []models.StageCondition{}, spec, []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceName: r.Source.Name,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{
			{Name: "REF"},
			{Name: "REF_TYPE"},
		}, []models.InputMapping{
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
		}, []models.OutputDefinition{}, []models.ValueDefinition{}))

		stage, err := r.Canvas.FindStageByName("stage-task-2")
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
		assert.Equal(t, models.StageExecutionStarted, execution.State)
		assert.NotEmpty(t, execution.ReferenceID)
		assert.NotEmpty(t, execution.StartedAt)
		assert.True(t, testconsumer.HasReceivedMessage())

		req := r.SemaphoreAPIMock.LastTaskTrigger
		require.NotNil(t, req)
		assert.Equal(t, "main", req.Spec.Branch)
		assert.Equal(t, ".semaphore/run.yml", req.Spec.PipelineFile)
		assertParameters(t, req, execution, map[string]string{
			"REF":      "refs/heads/test",
			"REF_TYPE": "branch",
		})
	})
}

func assertParameters(t *testing.T, trigger *semaphore.TaskTrigger, execution *models.StageExecution, parameters map[string]string) {
	all := map[string]string{
		"SEMAPHORE_STAGE_ID":           execution.StageID.String(),
		"SEMAPHORE_STAGE_EXECUTION_ID": execution.ID.String(),
	}

	for k, v := range parameters {
		all[k] = v
	}

	assert.Len(t, trigger.Spec.Parameters, len(all)+1)
	for name, value := range all {
		assert.True(t, slices.ContainsFunc(trigger.Spec.Parameters, func(p semaphore.TaskTriggerParameter) bool {
			return p.Name == name && p.Value == value
		}))
	}

	assert.True(t, slices.ContainsFunc(trigger.Spec.Parameters, func(p semaphore.TaskTriggerParameter) bool {
		return p.Name == "SEMAPHORE_STAGE_EXECUTION_TOKEN" && p.Value != ""
	}))
}
