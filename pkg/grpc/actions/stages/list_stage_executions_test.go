package stages

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestListStageExecutions(t *testing.T) {
	t.Run("return empty list when no executions exist", func(t *testing.T) {
		r := support.Setup(t)
		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
	})

	t.Run("return empty list for non-existent stage", func(t *testing.T) {
		r := support.Setup(t)
		_, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), uuid.NewString(), nil, nil, 0, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stage not found")
	})

	t.Run("return list of stage executions", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
		require.NoError(t, err)

		execution, err := models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
		require.NoError(t, err)

		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 1)

		exec := res.Executions[0]
		assert.Equal(t, execution.ID.String(), exec.Id)
		assert.Equal(t, protos.Execution_STATE_PENDING, exec.State)
		assert.Equal(t, protos.Execution_RESULT_UNKNOWN, exec.Result)
		assert.NotNil(t, exec.CreatedAt)
		assert.Nil(t, exec.StartedAt)
		assert.Nil(t, exec.FinishedAt)
		assert.Empty(t, exec.Outputs)
		assert.Empty(t, exec.Resources)

		require.NotNil(t, exec.StageEvent)
		assert.Equal(t, stageEvent.ID.String(), exec.StageEvent.Id)
		assert.Equal(t, "test-stage-event", exec.StageEvent.Name)

		require.NotNil(t, exec.StageEvent.TriggerEvent)
		assert.Equal(t, event.ID.String(), exec.StageEvent.TriggerEvent.Id)
		assert.Equal(t, "test.event", exec.StageEvent.TriggerEvent.Type)
	})

	t.Run("filter by execution states", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
		require.NoError(t, err)

		execution, err := models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
		require.NoError(t, err)

		err = execution.Start()
		require.NoError(t, err)

		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.Execution_State{protos.Execution_STATE_STARTED}, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 1)
		assert.Equal(t, protos.Execution_STATE_STARTED, res.Executions[0].State)

		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.Execution_State{protos.Execution_STATE_PENDING}, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
	})

	t.Run("filter by execution results", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
		require.NoError(t, err)

		execution, err := models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
		require.NoError(t, err)

		err = execution.Start()
		require.NoError(t, err)

		emittedEvent, err := execution.Finish(r.Stage, models.ResultPassed)
		require.NoError(t, err)
		require.NotNil(t, emittedEvent)

		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, []protos.Execution_Result{protos.Execution_RESULT_PASSED}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 1)

		exec := res.Executions[0]
		assert.Equal(t, protos.Execution_RESULT_PASSED, exec.Result)
		assert.Equal(t, protos.Execution_STATE_FINISHED, exec.State)
		assert.NotNil(t, exec.StartedAt)
		assert.NotNil(t, exec.FinishedAt)

		require.NotNil(t, exec.EmmitedEvent)
		assert.Equal(t, emittedEvent.ID.String(), exec.EmmitedEvent.Id)
		assert.Equal(t, "execution_finished", exec.EmmitedEvent.Type)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_STAGE, exec.EmmitedEvent.SourceType)
		assert.Equal(t, r.Stage.Name, exec.EmmitedEvent.SourceName)

		require.NotNil(t, exec.StageEvent)
		assert.Equal(t, stageEvent.ID.String(), exec.StageEvent.Id)
		assert.Equal(t, protos.StageEvent_STATE_PROCESSED, exec.StageEvent.State)

		require.NotNil(t, exec.StageEvent.TriggerEvent)
		assert.Equal(t, event.ID.String(), exec.StageEvent.TriggerEvent.Id)

		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, []protos.Execution_Result{protos.Execution_RESULT_FAILED}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
	})

	t.Run("limit results", func(t *testing.T) {
		r := support.Setup(t)

		for i := 0; i < 5; i++ {
			event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
			require.NoError(t, err)

			stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
			require.NoError(t, err)

			_, err = models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
			require.NoError(t, err)
		}

		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 3, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 3)
	})

	t.Run("filter by before timestamp", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
		require.NoError(t, err)

		execution, err := models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
		require.NoError(t, err)

		futureTime := execution.CreatedAt.Add(1 * time.Hour)
		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 0, timestamppb.New(futureTime))
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 1)

		pastTime := execution.CreatedAt.Add(-1 * time.Hour)
		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 0, timestamppb.New(pastTime))
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
	})

	t.Run("find stage by name", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
		require.NoError(t, err)

		_, err = models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
		require.NoError(t, err)

		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.Name, nil, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 1)
	})

	t.Run("test emitted events and complete relationships", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "webhook.received", []byte(`{"branch": "main", "commit": "abc123"}`), []byte(`{"x-github-event": "push"}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"branch": "main", "commit": "abc123"}, "deploy-main")
		require.NoError(t, err)

		execution, err := models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
		require.NoError(t, err)

		err = execution.Start()
		require.NoError(t, err)

		outputs := map[string]any{"deployment_id": "deploy-456", "url": "https://app.example.com"}
		err = execution.UpdateOutputs(outputs)
		require.NoError(t, err)

		emittedEvent, err := execution.Finish(r.Stage, models.ResultPassed)
		require.NoError(t, err)

		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 1)

		exec := res.Executions[0]

		assert.Equal(t, execution.ID.String(), exec.Id)
		assert.Equal(t, protos.Execution_STATE_FINISHED, exec.State)
		assert.Equal(t, protos.Execution_RESULT_PASSED, exec.Result)
		assert.NotNil(t, exec.CreatedAt)
		assert.NotNil(t, exec.StartedAt)
		assert.NotNil(t, exec.FinishedAt)

		require.Len(t, exec.Outputs, 2)
		outputMap := make(map[string]string)
		for _, output := range exec.Outputs {
			outputMap[output.Name] = output.Value
		}
		assert.Equal(t, "deploy-456", outputMap["deployment_id"])
		assert.Equal(t, "https://app.example.com", outputMap["url"])

		require.NotNil(t, exec.EmmitedEvent)
		assert.Equal(t, emittedEvent.ID.String(), exec.EmmitedEvent.Id)
		assert.Equal(t, "execution_finished", exec.EmmitedEvent.Type)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_STAGE, exec.EmmitedEvent.SourceType)
		assert.Equal(t, r.Stage.ID.String(), exec.EmmitedEvent.SourceId)
		assert.Equal(t, r.Stage.Name, exec.EmmitedEvent.SourceName)
		assert.Equal(t, protos.Event_STATE_PENDING, exec.EmmitedEvent.State)
		assert.NotNil(t, exec.EmmitedEvent.ReceivedAt)
		assert.NotNil(t, exec.EmmitedEvent.Raw)

		require.NotNil(t, exec.EmmitedEvent.Raw)
		require.NotNil(t, exec.EmmitedEvent.Raw.Fields)
		assert.NotNil(t, exec.EmmitedEvent.Raw.Fields["execution"])
		executionData := exec.EmmitedEvent.Raw.Fields["execution"].GetStructValue()
		require.NotNil(t, executionData)
		assert.Equal(t, execution.ID.String(), executionData.Fields["id"].GetStringValue())
		assert.Equal(t, "passed", executionData.Fields["result"].GetStringValue())

		require.NotNil(t, exec.StageEvent)
		assert.Equal(t, stageEvent.ID.String(), exec.StageEvent.Id)
		assert.Equal(t, "deploy-main", exec.StageEvent.Name)
		assert.Equal(t, protos.StageEvent_STATE_PROCESSED, exec.StageEvent.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_UNKNOWN, exec.StageEvent.StateReason)
		assert.Equal(t, r.Source.ID.String(), exec.StageEvent.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, exec.StageEvent.SourceType)
		assert.NotNil(t, exec.StageEvent.CreatedAt)

		require.Len(t, exec.StageEvent.Inputs, 2)
		inputMap := make(map[string]string)
		for _, input := range exec.StageEvent.Inputs {
			inputMap[input.Name] = input.Value
		}
		assert.Equal(t, "main", inputMap["branch"])
		assert.Equal(t, "abc123", inputMap["commit"])

		require.NotNil(t, exec.StageEvent.TriggerEvent)
		assert.Equal(t, event.ID.String(), exec.StageEvent.TriggerEvent.Id)
		assert.Equal(t, "webhook.received", exec.StageEvent.TriggerEvent.Type)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, exec.StageEvent.TriggerEvent.SourceType)
		assert.Equal(t, r.Source.ID.String(), exec.StageEvent.TriggerEvent.SourceId)
		assert.Equal(t, r.Source.Name, exec.StageEvent.TriggerEvent.SourceName)
		assert.Equal(t, protos.Event_STATE_PENDING, exec.StageEvent.TriggerEvent.State)
		assert.NotNil(t, exec.StageEvent.TriggerEvent.ReceivedAt)

		assert.NotNil(t, exec.StageEvent.TriggerEvent.Raw)
		assert.Equal(t, "main", exec.StageEvent.TriggerEvent.Raw.Fields["branch"].GetStringValue())
		assert.Equal(t, "abc123", exec.StageEvent.TriggerEvent.Raw.Fields["commit"].GetStringValue())

		assert.NotNil(t, exec.StageEvent.TriggerEvent.Headers)
		assert.Equal(t, "push", exec.StageEvent.TriggerEvent.Headers.Fields["x-github-event"].GetStringValue())
	})
}
