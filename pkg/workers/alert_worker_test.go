package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/workers/alertworker"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test__AlertWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source: true,
	})
	defer r.Close()

	t.Run("handles event rejection created message", func(t *testing.T) {
		event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

		rejection, err := models.RejectEvent(
			event.ID,
			r.Source.ID,
			models.ConnectionTargetTypeStage,
			models.EventRejectionReasonError,
			"test error message",
		)
		require.NoError(t, err)

		pbMsg := &pb.EventRejectionCreated{
			RejectionId: rejection.ID.String(),
			Timestamp:   timestamppb.Now(),
		}

		messageBody, err := proto.Marshal(pbMsg)
		require.NoError(t, err)

		_, err = alertworker.HandleEventRejectionCreated(messageBody)
		require.NoError(t, err)

		alerts, err := models.ListAlerts(event.CanvasID, false, nil, nil)
		require.NoError(t, err)
		require.Len(t, alerts, 1)

		foundAlert := alerts[0]
		assert.Equal(t, event.CanvasID, foundAlert.CanvasID)
		assert.Equal(t, rejection.TargetID, foundAlert.SourceID)
		assert.Equal(t, rejection.TargetType, foundAlert.SourceType)
		assert.Equal(t, rejection.Message, foundAlert.Message)
		assert.Equal(t, models.AlertTypeError, foundAlert.Type)
	})
}
