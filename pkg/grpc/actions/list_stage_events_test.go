package actions

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListStageEvents(t *testing.T) {
	r := support.Setup(t)

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId:  uuid.New().String(),
			CanvasId: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage with no stage events -> empty list", func(t *testing.T) {
		res, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId:  r.Stage.ID.String(),
			CanvasId: r.Canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Events)
	})

	t.Run("stage with stage events - list", func(t *testing.T) {
		// event without approval
		support.CreateStageEvent(t, r.Source, r.Stage)

		// event with approval
		userID := uuid.New()
		approvedEvent := support.CreateStageEvent(t, r.Source, r.Stage)
		require.NoError(t, approvedEvent.Approve(userID))

		// event with execution
		eventWithExecution := support.CreateStageEvent(t, r.Source, r.Stage)
		execution, err := models.CreateStageExecution(r.Stage.ID, eventWithExecution.ID)
		require.NoError(t, err)
		require.NoError(t, eventWithExecution.UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonExecution))

		res, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			CanvasId: r.Canvas.ID.String(),
			StageId:  r.Stage.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 3)

		// event with execution
		e := res.Events[0]
		assert.NotEmpty(t, e.Id)
		assert.NotEmpty(t, e.CreatedAt)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_WAITING, e.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_EXECUTION, e.StateReason)
		require.NotNil(t, e.Execution)
		assert.Equal(t, execution.ID.String(), e.Execution.Id)
		assert.Empty(t, e.Execution.ReferenceId)
		assert.Equal(t, protos.Execution_STATE_PENDING, e.Execution.State)
		assert.Equal(t, protos.Execution_RESULT_UNKNOWN, e.Execution.Result)
		assert.NotNil(t, e.Execution.CreatedAt)
		assert.Nil(t, e.Execution.StartedAt)
		assert.Nil(t, e.Execution.FinishedAt)
		require.Len(t, e.Approvals, 0)

		// event with approvals
		e = res.Events[1]
		assert.NotEmpty(t, e.Id)
		assert.NotEmpty(t, e.CreatedAt)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_UNKNOWN, e.StateReason)
		require.Len(t, e.Approvals, 1)
		assert.Equal(t, userID.String(), e.Approvals[0].ApprovedBy)
		assert.NotEmpty(t, userID, e.Approvals[0].ApprovedAt)
		require.Nil(t, e.Execution)

		// event with no approvals
		e = res.Events[2]
		assert.NotEmpty(t, e.Id)
		assert.NotEmpty(t, e.CreatedAt)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_UNKNOWN, e.StateReason)
		require.Len(t, e.Approvals, 0)
		require.Nil(t, e.Execution)
	})
}
