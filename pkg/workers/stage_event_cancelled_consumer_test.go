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
	"github.com/superplanehq/superplane/test/support"
)

func Test__StageEventCancelledConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
		Stage:       true,
	})

	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	w := NewStageEventCancelledConsumer(amqpURL)

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

	//
	// Cancel the stage event
	//
	ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
	_, err := stageevents.CancelStageEvent(ctx, r.Canvas.ID.String(), r.Stage.ID.String(), event.ID.String())
	require.NoError(t, err)

	//
	// Verify stage event is moved to processed state with cancelled reason
	// The consumer should receive the cancellation message and log it
	//
	require.Eventually(t, func() bool {
		event, _ := models.FindStageEventByID(event.ID.String(), event.StageID.String())
		return event.State == models.StageEventStateProcessed && event.StateReason == models.StageEventStateReasonCancelled
	}, time.Second, 200*time.Millisecond)
}