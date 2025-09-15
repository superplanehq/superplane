package canvases

import (
	"context"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test__ListEventRejections(t *testing.T) {
	r := support.Setup(t)

	// Create some events and rejections for testing
	for i := 0; i < 3; i++ {
		event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "data1"}`), []byte(`{"x-header": "value1"}`))
		require.NoError(t, err)

		// Reject the event targeting our stage
		_, err = models.RejectEvent(event.ID, r.Stage.ID, models.SourceTypeStage, models.EventRejectionReasonFiltered, "Test rejection")
		require.NoError(t, err)
	}

	t.Run("invalid target ID -> error", func(t *testing.T) {
		_, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, "invalid-uuid", 0, nil)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid target id", s.Message())
	})

	t.Run("invalid target type -> error", func(t *testing.T) {
		_, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_UNKNOWN, r.Stage.ID.String(), 0, nil)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid target type", s.Message())
	})

	t.Run("list event rejections for stage", func(t *testing.T) {
		res, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Rejections, 3)
		assert.Equal(t, uint32(3), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Verify rejection data
		rejection := res.Rejections[0]
		assert.NotEmpty(t, rejection.Id)
		assert.Equal(t, pb.Connection_TYPE_STAGE, rejection.TargetType)
		assert.Equal(t, r.Stage.ID.String(), rejection.TargetId)
		assert.Equal(t, pb.EventRejection_REJECTION_REASON_FILTERED, rejection.Reason)
		assert.Equal(t, "Test rejection", rejection.Message)
		assert.NotNil(t, rejection.RejectedAt)
		assert.NotNil(t, rejection.Event)

		// Test with non-existent target ID
		res, err = ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, uuid.NewString(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Rejections)
		assert.Equal(t, uint32(0), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.Nil(t, res.LastTimestamp)
	})

	t.Run("limit parameter", func(t *testing.T) {
		// Test with limit of 2
		res, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 2, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Rejections, 2)
		assert.Equal(t, uint32(3), res.TotalCount)
		assert.True(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Test with default limit (0)
		res, err = ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Rejections, 3)
		assert.Equal(t, uint32(3), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Test with high limit
		res, err = ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 100, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Rejections, 3)
		assert.Equal(t, uint32(3), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
	})

	t.Run("before parameter", func(t *testing.T) {
		// Create additional rejection with specific timing
		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "before1"}`), []byte(`{}`))
		require.NoError(t, err)

		rejection1, err := models.RejectEvent(event1.ID, r.Stage.ID, models.SourceTypeStage, models.EventRejectionReasonError, "Before test 1")
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "before2"}`), []byte(`{}`))
		require.NoError(t, err)

		_, err = models.RejectEvent(event2.ID, r.Stage.ID, models.SourceTypeStage, models.EventRejectionReasonError, "Before test 2")
		require.NoError(t, err)

		// Query with before timestamp
		beforeTime := timestamppb.New(*rejection1.RejectedAt)
		res, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 0, beforeTime)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, uint32(5), res.TotalCount)
		assert.False(t, res.HasNextPage)

		if len(res.Rejections) > 0 {
			assert.NotNil(t, res.LastTimestamp)
			// Verify that all returned rejections are before the specified time
			for _, rejection := range res.Rejections {
				rejectionTime := rejection.RejectedAt.AsTime()
				assert.True(t, rejectionTime.Before(*rejection1.RejectedAt))
			}
		}

		// Verify the newest rejection (after beforeTime) is not included
		rejectionIDs := getRejectionEventIDs(res.Rejections)
		assert.NotContains(t, rejectionIDs, event2.ID.String())
	})

	t.Run("test pagination with hasNextPage", func(t *testing.T) {
		r := support.Setup(t)

		// Create 10 rejections for pagination testing
		for i := 0; i < 10; i++ {
			event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "pagination"}`), []byte(`{}`))
			require.NoError(t, err)

			_, err = models.RejectEvent(event.ID, r.Stage.ID, models.SourceTypeStage, models.EventRejectionReasonFiltered, "Pagination test")
			require.NoError(t, err)
		}

		// Test with limit that triggers hasNextPage = true
		res, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 5, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Rejections, 5)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.True(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Test with limit equal to total count
		res, err = ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 10, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Rejections, 10)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)

		// Test with limit greater than total count
		res, err = ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 15, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Rejections, 10)
		assert.Equal(t, uint32(10), res.TotalCount)
		assert.False(t, res.HasNextPage)
		assert.NotNil(t, res.LastTimestamp)
	})

	t.Run("test different target types", func(t *testing.T) {
		r := support.Setup(t)

		// Create connection group for testing
		connectionGroup := support.CreateConnectionGroup(t, "test-group", r.Canvas, r.Source, 30, models.ConnectionGroupTimeoutBehaviorDrop)

		// Create rejections for different target types
		event1, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "stage"}`), []byte(`{}`))
		require.NoError(t, err)

		event2, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "webhook", []byte(`{"test": "connection_group"}`), []byte(`{}`))
		require.NoError(t, err)

		// Reject event1 by stage
		_, err = models.RejectEvent(event1.ID, r.Stage.ID, models.SourceTypeStage, models.EventRejectionReasonFiltered, "Stage rejection")
		require.NoError(t, err)

		// Reject event2 by connection group
		_, err = models.RejectEvent(event2.ID, connectionGroup.ID, models.SourceTypeConnectionGroup, models.EventRejectionReasonError, "Connection group rejection")
		require.NoError(t, err)

		// Query stage rejections
		res, err := ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_STAGE, r.Stage.ID.String(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Rejections, 1)
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.Equal(t, pb.Connection_TYPE_STAGE, res.Rejections[0].TargetType)
		assert.Equal(t, "Stage rejection", res.Rejections[0].Message)

		// Query connection group rejections
		res, err = ListEventRejections(context.Background(), r.Canvas.ID.String(), pb.Connection_TYPE_CONNECTION_GROUP, connectionGroup.ID.String(), 0, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.Rejections, 1)
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.Equal(t, pb.Connection_TYPE_CONNECTION_GROUP, res.Rejections[0].TargetType)
		assert.Equal(t, "Connection group rejection", res.Rejections[0].Message)
	})
}

func getRejectionEventIDs(rejections []*pb.EventRejection) []string {
	ids := make([]string, len(rejections))
	for i, rejection := range rejections {
		if rejection.Event != nil {
			ids[i] = rejection.Event.Id
		}
	}
	return ids
}