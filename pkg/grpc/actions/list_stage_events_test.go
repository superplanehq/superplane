package actions

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListStageEvents(t *testing.T) {
	r := support.Setup(t)

	t.Run("no org ID -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId:  uuid.New().String(),
			CanvasId: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId:        uuid.New().String(),
			OrganizationId: r.Org.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId:        uuid.New().String(),
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("stage with no stage events -> empty list", func(t *testing.T) {
		res, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			StageId:        r.Stage.ID.String(),
			CanvasId:       r.Canvas.ID.String(),
			OrganizationId: r.Org.String(),
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
		event := support.CreateStageEvent(t, r.Source, r.Stage)
		require.NoError(t, event.Approve(userID))

		res, err := ListStageEvents(context.Background(), &protos.ListStageEventsRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			StageId:        r.Stage.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Events, 2)

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

		// event with no approvals
		e = res.Events[1]
		assert.NotEmpty(t, e.Id)
		assert.NotEmpty(t, e.CreatedAt)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)
		assert.Equal(t, protos.StageEvent_STATE_REASON_UNKNOWN, e.StateReason)
		require.Len(t, e.Approvals, 0)
	})
}
