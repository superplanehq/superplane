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
		assert.Equal(t, uint32(0), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.LastTimestamp)
	})

	t.Run("non-existent stage -> error", func(t *testing.T) {
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
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

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
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
		assert.Equal(t, protos.Execution_STATE_STARTED, res.Executions[0].State)

		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.Execution_State{protos.Execution_STATE_PENDING}, nil, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
		assert.Equal(t, uint32(0), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.LastTimestamp)
	})

	t.Run("filter by execution results", func(t *testing.T) {
		r := support.Setup(t)

		// Create a finished(passed) execution
		execution := support.CreateExecution(t, r.Source, r.Stage)
		require.NoError(t, execution.Start())
		_, err := execution.Finish(r.Stage, models.ResultPassed)
		require.NoError(t, err)

		// Just 1 passed execution is returned
		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, []protos.Execution_Result{protos.Execution_RESULT_PASSED}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, uint32(1), res.TotalCount)
		require.False(t, res.HasNextPage)
		require.NotNil(t, res.LastTimestamp)
		require.Len(t, res.Executions, 1)

		exec := res.Executions[0]
		assert.Equal(t, protos.Execution_RESULT_PASSED, exec.Result)
		assert.Equal(t, protos.Execution_STATE_FINISHED, exec.State)
		assert.NotNil(t, exec.StartedAt)
		assert.NotNil(t, exec.FinishedAt)
		require.NotNil(t, exec.StageEvent)
		require.NotNil(t, exec.StageEvent.TriggerEvent)

		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, []protos.Execution_Result{protos.Execution_RESULT_FAILED}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
		assert.Equal(t, uint32(0), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.LastTimestamp)
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
		assert.Equal(t, uint32(5), res.TotalCount)
		assert.True(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
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
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		pastTime := execution.CreatedAt.Add(-1 * time.Hour)
		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 0, timestamppb.New(pastTime))
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Executions)
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.LastTimestamp)
	})

	t.Run("test pagination with hasNextPage", func(t *testing.T) {
		r := support.Setup(t)

		// Create 10 executions
		for i := 0; i < 10; i++ {
			event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "test.event", []byte(`{"key": "value"}`), []byte(`{}`))
			require.NoError(t, err)

			stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"input": "value"}, "test-stage-event")
			require.NoError(t, err)

			_, err = models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, stageEvent.ID)
			require.NoError(t, err)
		}

		// Request 5 items, should have next page
		res, err := ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 5, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 5)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.True(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Request 10 items, should NOT have next page
		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 10)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Request 15 items (more than exist), should NOT have next page
		res, err = ListStageExecutions(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), nil, nil, 15, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Executions, 10)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
	})
}
