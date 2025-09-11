package stageevents

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListStageEvents(t *testing.T) {
	r := support.Setup(t)

	states := []protos.StageEvent_State{}

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), uuid.NewString(), r.Stage.ID.String(), states, []protos.StageEvent_StateReason{}, 0, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), r.Canvas.ID.String(), uuid.NewString(), states, []protos.StageEvent_StateReason{}, 0, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage with no stage events -> empty list", func(t *testing.T) {
		res, err := ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), states, []protos.StageEvent_StateReason{}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
		assert.Equal(t, uint32(0), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.LastTimestamp)
	})

	t.Run("stage with stage events - list", func(t *testing.T) {
		// event without approval, no inputs
		support.CreateStageEvent(t, r.Source, r.Stage)

		// event with approval, with inputs
		userID := uuid.New()
		approvedEvent := support.CreateStageEventWithData(t, r.Source, r.Stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`), map[string]any{
			"VERSION": "v1",
		})

		require.NoError(t, approvedEvent.Approve(userID))

		// event with execution, inputs and outputs
		eventWithExecution := support.CreateStageEventWithData(t, r.Source, r.Stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`), map[string]any{
			"VERSION": "v1",
		})

		// execution should not be showed in list_stage_events
		execution, err := models.CreateStageExecution(r.Canvas.ID, r.Stage.ID, eventWithExecution.ID)
		require.NoError(t, err)
		require.NoError(t, eventWithExecution.UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonExecution))
		require.NoError(t, execution.UpdateOutputs(map[string]any{
			"VERSION": "v1",
			"VALUE_1": "value1",
		}))

		execution, err = models.FindExecutionByID(execution.ID)
		require.NoError(t, err)

		res, err := ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), states, []protos.StageEvent_StateReason{}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 2)
		assert.Equal(t, uint32(2), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// event with approvals
		e := res.Events[0]
		assert.NotEmpty(t, e.Id)
		assert.NotEmpty(t, e.CreatedAt)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_UNKNOWN, e.StateReason)
		require.Len(t, e.Approvals, 1)
		assert.Equal(t, userID.String(), e.Approvals[0].ApprovedBy)
		assert.NotEmpty(t, userID, e.Approvals[0].ApprovedAt)
		require.Len(t, e.Inputs, 1)
		assert.Equal(t, "VERSION", e.Inputs[0].Name)
		assert.Equal(t, "v1", e.Inputs[0].Value)
		assert.Equal(t, "", e.Name)

		require.NotNil(t, e.TriggerEvent)
		assert.Equal(t, "push", e.TriggerEvent.Type)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, e.TriggerEvent.SourceType)
		assert.Equal(t, r.Source.ID.String(), e.TriggerEvent.SourceId)

		// event with no approvals
		e = res.Events[1]
		assert.NotEmpty(t, e.Id)
		assert.NotEmpty(t, e.CreatedAt)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_UNKNOWN, e.StateReason)
		require.Len(t, e.Approvals, 0)
		require.Len(t, e.Inputs, 0)
		assert.Equal(t, "", e.Name)

		require.NotNil(t, e.TriggerEvent)
		assert.Equal(t, "push", e.TriggerEvent.Type)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, e.TriggerEvent.SourceType)
		assert.Equal(t, r.Source.ID.String(), e.TriggerEvent.SourceId)
	})

	t.Run("stage events include trigger event data", func(t *testing.T) {
		r := support.Setup(t)

		event, err := models.CreateEvent(r.Source.ID, r.Canvas.ID, r.Source.Name, models.SourceTypeEventSource, "webhook.received", []byte(`{"branch": "main", "commit": "abc123"}`), []byte(`{"x-github-event": "push"}`))
		require.NoError(t, err)

		stageEvent, err := models.CreateStageEvent(r.Stage.ID, event, models.StageEventStatePending, "", map[string]any{"branch": "main", "commit": "abc123"}, "deploy-main")
		require.NoError(t, err)

		res, err := ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.StageEvent_State{}, []protos.StageEvent_StateReason{}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 1)
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		e := res.Events[0]

		assert.Equal(t, stageEvent.ID.String(), e.Id)
		assert.Equal(t, "deploy-main", e.Name)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)

		require.Len(t, e.Inputs, 2)
		inputMap := make(map[string]string)
		for _, input := range e.Inputs {
			inputMap[input.Name] = input.Value
		}
		assert.Equal(t, "main", inputMap["branch"])
		assert.Equal(t, "abc123", inputMap["commit"])

		require.NotNil(t, e.TriggerEvent)
		assert.Equal(t, event.ID.String(), e.TriggerEvent.Id)
		assert.Equal(t, "webhook.received", e.TriggerEvent.Type)
		assert.Equal(t, protos.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE, e.TriggerEvent.SourceType)
		assert.Equal(t, r.Source.ID.String(), e.TriggerEvent.SourceId)
		assert.Equal(t, r.Source.Name, e.TriggerEvent.SourceName)
		assert.Equal(t, protos.Event_STATE_PENDING, e.TriggerEvent.State)
		assert.NotNil(t, e.TriggerEvent.ReceivedAt)

		require.NotNil(t, e.TriggerEvent.Raw)
		assert.Equal(t, "main", e.TriggerEvent.Raw.Fields["branch"].GetStringValue())
		assert.Equal(t, "abc123", e.TriggerEvent.Raw.Fields["commit"].GetStringValue())

		require.NotNil(t, e.TriggerEvent.Headers)
		assert.Equal(t, "push", e.TriggerEvent.Headers.Fields["x-github-event"].GetStringValue())
	})

	t.Run("test pagination with hasNextPage", func(t *testing.T) {
		r := support.Setup(t)

		for i := 0; i < 10; i++ {
			support.CreateStageEvent(t, r.Source, r.Stage)
		}

		res, err := ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.StageEvent_State{}, []protos.StageEvent_StateReason{}, 5, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 5)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.True(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		res, err = ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.StageEvent_State{}, []protos.StageEvent_StateReason{}, 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 10)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		res, err = ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.StageEvent_State{}, []protos.StageEvent_StateReason{}, 15, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 10)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
	})

	t.Run("test filtering with new response fields", func(t *testing.T) {
		r := support.Setup(t)

		event1 := support.CreateStageEvent(t, r.Source, r.Stage)
		event2 := support.CreateStageEvent(t, r.Source, r.Stage)

		require.NoError(t, event1.UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonExecution))

		res, err := ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.StageEvent_State{protos.StageEvent_STATE_PENDING}, []protos.StageEvent_StateReason{}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 1)
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
		assert.Equal(t, event2.ID.String(), res.Events[0].Id)

		res, err = ListStageEvents(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String(), []protos.StageEvent_State{protos.StageEvent_STATE_WAITING}, []protos.StageEvent_StateReason{}, 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 1)
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
		assert.Equal(t, event1.ID.String(), res.Events[0].Id)
	})
}
