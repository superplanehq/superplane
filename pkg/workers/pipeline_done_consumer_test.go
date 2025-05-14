package workers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/events"
	"github.com/superplanehq/superplane/pkg/models"
	pplproto "github.com/superplanehq/superplane/pkg/protos/plumber.pipeline"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
	"google.golang.org/protobuf/proto"
)

const ExecutionFinishedRoutingKey = "execution-finished"

func Test__PipelineDoneConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source: true, Grpc: true,
	})

	connections := []models.StageConnection{
		{
			SourceID:   r.Source.ID,
			SourceType: models.SourceTypeEventSource,
		},
	}

	err := r.Canvas.CreateStage("stage-1", r.User.String(), []models.StageCondition{}, support.RunTemplate(), connections, support.TagUsageDef(r.Source.Name))
	require.NoError(t, err)
	stage, err := r.Canvas.FindStageByName("stage-1")
	require.NoError(t, err)

	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	w := NewPipelineDoneConsumer(amqpURL, "0.0.0.0:50052")

	go w.Start()
	defer w.Stop()

	//
	// give the worker a few milliseconds to start before we start running the tests
	//
	time.Sleep(100 * time.Millisecond)

	t.Run("failed pipeline -> execution fails", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`))
		require.NoError(t, execution.Start(workflowID))

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock failed result and publish pipeline done message.
		//
		r.Grpc.PipelineService.MockPipelineResult(pplproto.Pipeline_FAILED)
		r.Grpc.PipelineService.MockWorkflow(workflowID)
		message := pplproto.PipelineEvent{PipelineId: uuid.New().String()}
		body, err := proto.Marshal(&message)
		require.NoError(t, err)
		require.NoError(t, tackle.PublishMessage(&tackle.PublishParams{
			AmqpURL:    amqpURL,
			RoutingKey: "done",
			Exchange:   "pipeline_state_exchange",
			Body:       body,
		}))

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
		assert.Equal(t, events.StageExecutionCompletionType, e.Type)
		assert.Equal(t, stage.ID.String(), e.Stage.ID)
		assert.Equal(t, execution.ID.String(), e.Execution.ID)
		assert.Equal(t, models.StageExecutionResultFailed, e.Execution.Result)
		assert.NotEmpty(t, e.Execution.CreatedAt)
		assert.NotEmpty(t, e.Execution.StartedAt)
		assert.NotEmpty(t, e.Execution.FinishedAt)
		require.True(t, testconsumer.HasReceivedMessage())

		//
		// Verify tags are marked as unhealthy
		//
		tags, err := models.ListStageTags("VERSION", "", []string{}, "", execution.StageEventID.String())
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "VERSION", tags[0].TagName)
		assert.Equal(t, "v1", tags[0].TagValue)
		assert.Equal(t, models.TagStateUnhealthy, tags[0].TagState)
	})

	t.Run("passed pipeline -> execution passes", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`))
		require.NoError(t, execution.Start(workflowID))

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock failed result and publish pipeline done message.
		//
		r.Grpc.PipelineService.MockPipelineResult(pplproto.Pipeline_PASSED)
		r.Grpc.PipelineService.MockWorkflow(workflowID)
		message := pplproto.PipelineEvent{PipelineId: uuid.New().String()}
		body, err := proto.Marshal(&message)
		require.NoError(t, err)
		require.NoError(t, tackle.PublishMessage(&tackle.PublishParams{
			AmqpURL:    amqpURL,
			RoutingKey: "done",
			Exchange:   "pipeline_state_exchange",
			Body:       body,
		}))

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
		assert.Equal(t, events.StageExecutionCompletionType, e.Type)
		assert.Equal(t, stage.ID.String(), e.Stage.ID)
		assert.Equal(t, execution.ID.String(), e.Execution.ID)
		assert.Equal(t, models.StageExecutionResultPassed, e.Execution.Result)
		assert.NotEmpty(t, e.Execution.CreatedAt)
		assert.NotEmpty(t, e.Execution.StartedAt)
		assert.NotEmpty(t, e.Execution.FinishedAt)
		require.True(t, testconsumer.HasReceivedMessage())

		//
		// Verify tags are marked as healthy
		//
		tags, err := models.ListStageTags("VERSION", "", []string{}, "", execution.StageEventID.String())
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "VERSION", tags[0].TagName)
		assert.Equal(t, "v1", tags[0].TagValue)
		assert.Equal(t, models.TagStateHealthy, tags[0].TagState)
	})
}

func unmarshalCompletionEvent(raw []byte) (*events.StageExecutionCompletion, error) {
	e := events.StageExecutionCompletion{}
	err := json.Unmarshal(raw, &e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}
