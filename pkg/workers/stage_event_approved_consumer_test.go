package workers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	stageevents "github.com/superplanehq/superplane/pkg/grpc/actions/stage_events"
	"github.com/superplanehq/superplane/pkg/models"
	superplane "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__StageEventApprovedConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
		Stage:       true,
		Approvals:   2,
	})

	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	w := NewStageEventApprovedConsumer(amqpURL)

	go w.Start()
	defer w.Stop()

	//
	// give the worker a few milliseconds to start before we start running the tests
	//
	time.Sleep(100 * time.Millisecond)

	//
	// Create stage event
	//
	event := support.CreateStageEvent(t, r.Source, r.Stage)
	require.NoError(t, event.UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonApproval))

	//
	// Approve event once
	//
	ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
	_, err := stageevents.ApproveStageEvent(ctx, &superplane.ApproveStageEventRequest{
		CanvasIdOrName: r.Canvas.ID.String(),
		StageIdOrName:  r.Stage.ID.String(),
		EventId:        event.ID.String(),
	})

	require.NoError(t, err)

	//
	// Verify stage event is not moved to pending yet,
	// because the stage requires 2 approvals.
	//
	require.Never(t, func() bool {
		event, _ := models.FindStageEventByID(event.ID.String(), event.StageID.String())
		return event.State == models.StageEventStatePending
	}, time.Second, 200*time.Millisecond)

	//
	// Approve event again
	//
	ctx = authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
	_, err = stageevents.ApproveStageEvent(ctx, &superplane.ApproveStageEventRequest{
		CanvasIdOrName: r.Canvas.ID.String(),
		StageIdOrName:  r.Stage.ID.String(),
		EventId:        event.ID.String(),
	})

	require.NoError(t, err)

	//
	// Verify stage event is moved to pending state after the 2nd approval.
	//
	require.Eventually(t, func() bool {
		event, _ := models.FindStageEventByID(event.ID.String(), event.StageID.String())
		return event.State == models.StageEventStatePending
	}, time.Second, 200*time.Millisecond)
}
