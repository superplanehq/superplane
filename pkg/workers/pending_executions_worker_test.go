package workers

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/events"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	schedulepb "github.com/superplanehq/superplane/pkg/protos/periodic_scheduler"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const ExecutionStartedRoutingKey = "execution-started"

func Test__PendingExecutionsWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source: true, Stage: true, Grpc: true, Approvals: 1,
	})

	w := PendingExecutionsWorker{
		RepoProxyURL: "0.0.0.0:50052",
		SchedulerURL: "0.0.0.0:50052",
		JwtSigner:    jwt.NewSigner("test"),
	}
	amqpURL, _ := config.RabbitMQURL()

	t.Run("semaphore workflow is created", func(t *testing.T) {
		//
		// Create stage that creates Semaphore workflows.
		//
		require.NoError(t, r.Canvas.CreateStage("stage-wf", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name)))

		stage, err := r.Canvas.FindStageByName("stage-wf")
		require.NoError(t, err)

		testconsumer := testconsumer.New(amqpURL, ExecutionStartedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Create pending execution.
		//
		execution := support.CreateExecution(t, r.Source, stage)

		//
		// Trigger the worker, and verify that request to repo proxy was sent,
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
		repoProxyReq := r.Grpc.RepoProxyService.GetLastCreateRequest()
		require.NotNil(t, repoProxyReq)
		assert.Equal(t, "demo-project", repoProxyReq.ProjectId)
		assert.Equal(t, ".semaphore/semaphore.yml", repoProxyReq.DefinitionFile)
		assert.Equal(t, stage.CreatedBy.String(), repoProxyReq.RequesterId)
		assert.Equal(t, "refs/heads/main", repoProxyReq.Git.Reference)
	})

	t.Run("semaphore task is triggered with simple parameters", func(t *testing.T) {
		//
		// Create stage that trigger Semaphore task.
		//
		require.NoError(t, r.Canvas.CreateStage("stage-task", r.User.String(), []models.StageCondition{}, support.TaskRunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name)))

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

		req := r.Grpc.SchedulerService.GetLastRunNowRequest()
		require.NotNil(t, req)
		assert.Equal(t, "demo-task", req.Id)
		assert.Equal(t, "main", req.Branch)
		assert.Equal(t, ".semaphore/run.yml", req.PipelineFile)
		assert.Equal(t, stage.CreatedBy.String(), req.Requester)
		assertParameters(t, req, execution, map[string]string{
			"PARAM_1": "VALUE_1",
			"PARAM_2": "VALUE_2",
		})
	})

	t.Run("semaphore task with resolved parameters is triggered", func(t *testing.T) {
		//
		// Create stage that trigger Semaphore task.
		//
		template := support.TaskRunTemplate()
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
		}, support.TagUsageDef(r.Source.Name)))

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

		req := r.Grpc.SchedulerService.GetLastRunNowRequest()
		require.NotNil(t, req)
		assert.Equal(t, "demo-task", req.Id)
		assert.Equal(t, "main", req.Branch)
		assert.Equal(t, ".semaphore/run.yml", req.PipelineFile)
		assert.Equal(t, stage.CreatedBy.String(), req.Requester)
		assertParameters(t, req, execution, map[string]string{
			"REF":             "refs/heads/test",
			"REF_TYPE":        "branch",
			"STAGE_1_VERSION": "1.0.0",
		})
	})
}

func assertParameters(t *testing.T, req *schedulepb.RunNowRequest, execution *models.StageExecution, parameters map[string]string) {
	all := map[string]string{
		"SEMAPHORE_STAGE_ID":           execution.StageID.String(),
		"SEMAPHORE_STAGE_EXECUTION_ID": execution.ID.String(),
	}

	for k, v := range parameters {
		all[k] = v
	}

	assert.Len(t, req.ParameterValues, len(all)+1)
	for name, value := range all {
		assert.True(t, slices.ContainsFunc(req.ParameterValues, func(v *schedulepb.ParameterValue) bool {
			return v.Name == name && v.Value == value
		}))
	}

	assert.True(t, slices.ContainsFunc(req.ParameterValues, func(v *schedulepb.ParameterValue) bool {
		return v.Name == "SEMAPHORE_STAGE_EXECUTION_TOKEN" && v.Value != ""
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
