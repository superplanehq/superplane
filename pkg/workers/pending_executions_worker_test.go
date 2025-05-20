package workers

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/apis/semaphore"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/events"
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
		Encryptor: &encryptor.NoOpEncryptor{},
	}

	amqpURL, _ := config.RabbitMQURL()

	t.Run("semaphore task is triggered with simple parameters", func(t *testing.T) {
		//
		// Create stage that trigger Semaphore task.
		//
		template := support.RunTemplateWithURL(r.SemaphoreAPIMock.Server.URL)
		require.NoError(t, r.Canvas.CreateStage("stage-task", r.User.String(), []models.StageCondition{}, template, []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}))

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
		template := support.RunTemplateWithURL(r.SemaphoreAPIMock.Server.URL)
		template.Semaphore.Parameters = map[string]string{
			"REF":             "${{ self.Conn('gh').ref }}",
			"REF_TYPE":        "${{ self.Conn('gh').ref_type }}",
			"STAGE_1_VERSION": "${{ self.Conn('stage-1').tags.version }}",
		}

		require.NoError(t, r.Canvas.CreateStage("stage-task-2", r.User.String(), []models.StageCondition{}, template, []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceName: r.Source.Name,
				SourceType: models.SourceTypeEventSource,
			},
			{
				SourceID:   r.Stage.ID,
				SourceName: r.Stage.Name,
				SourceType: models.SourceTypeStage,
			},
		}))

		stage, err := r.Canvas.FindStageByName("stage-task-2")
		require.NoError(t, err)

		//
		// Since we use the tags of a stage in the template for the execution,
		// we need a previous event for that stage to be available, so we create it here.
		//
		data := createStageCompletionEvent(t, r, map[string]string{"version": "1.0.0"})
		_, err = models.CreateEvent(r.Stage.ID, r.Stage.Name, models.SourceTypeStage, data, []byte(`{}`))
		require.NoError(t, err)

		//
		// Create pending execution for a new event source event.
		//
		execution := support.CreateExecutionWithData(t, r.Source, stage, []byte(`{"ref_type":"branch","ref":"refs/heads/test"}`), []byte(`{}`))
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
			"REF":             "refs/heads/test",
			"REF_TYPE":        "branch",
			"STAGE_1_VERSION": "1.0.0",
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

func createStageCompletionEvent(t *testing.T, r *support.ResourceRegistry, tags map[string]string) []byte {
	e, err := events.NewStageExecutionCompletion(&models.StageExecution{
		ID:      uuid.New(),
		StageID: r.Stage.ID,
		Result:  models.StageExecutionResultPassed,
	}, tags)

	require.NoError(t, err)
	data, err := json.Marshal(e)
	require.NoError(t, err)

	return data
}
