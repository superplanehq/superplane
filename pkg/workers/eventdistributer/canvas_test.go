package eventdistributer

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHandleCanvasCreated(t *testing.T) {
	canvasID := "canvas-123"
	orgID := "org-456"

	msg := &pb.CanvasMessage{
		Id:             canvasID,
		CanvasId:       canvasID,
		Timestamp:      timestamppb.Now(),
		OrganizationId: orgID,
	}

	body, err := proto.Marshal(msg)
	require.NoError(t, err)

	hub := ws.NewHub()
	hub.Run()

	err = HandleCanvasCreated(body, hub)
	require.NoError(t, err)
}

func TestHandleCanvasCreatedMissingCanvasID(t *testing.T) {
	msg := &pb.CanvasMessage{
		Id:        "",
		CanvasId:  "",
		Timestamp: timestamppb.Now(),
	}

	body, err := proto.Marshal(msg)
	require.NoError(t, err)

	hub := ws.NewHub()
	hub.Run()

	err = HandleCanvasCreated(body, hub)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing canvas id")
}

func TestHandleCanvasCreatedInvalidProtobuf(t *testing.T) {
	hub := ws.NewHub()
	hub.Run()

	err := HandleCanvasCreated([]byte("not-a-protobuf"), hub)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal")
}

func TestCanvasCreatedEventConstant(t *testing.T) {
	require.Equal(t, "canvas_created", CanvasCreatedEvent)
}
