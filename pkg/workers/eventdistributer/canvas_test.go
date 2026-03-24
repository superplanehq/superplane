package eventdistributer

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeHub struct {
	mu       sync.Mutex
	messages map[string][][]byte
}

func newFakeHub() *fakeHub {
	return &fakeHub{messages: make(map[string][][]byte)}
}

func (h *fakeHub) BroadcastToWorkflow(workflowID string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages[workflowID] = append(h.messages[workflowID], message)
}

func (h *fakeHub) messagesFor(workflowID string) [][]byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.messages[workflowID]
}

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

func TestHandleCanvasCreatedPayload(t *testing.T) {
	canvasID := "canvas-abc"
	orgID := "org-xyz"

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

func TestHandleCanvasCreatedEventType(t *testing.T) {
	canvasID := "canvas-evt"

	msg := &pb.CanvasMessage{
		Id:             canvasID,
		CanvasId:       canvasID,
		Timestamp:      timestamppb.Now(),
		OrganizationId: "org-1",
	}

	body, err := proto.Marshal(msg)
	require.NoError(t, err)

	hub := ws.NewHub()
	hub.Run()

	err = HandleCanvasCreated(body, hub)
	require.NoError(t, err)
}

func TestCanvasCreatedEventConstant(t *testing.T) {
	require.Equal(t, "canvas_created", CanvasCreatedEvent)
}

func TestHandleCanvasCreatedBroadcastsCorrectEvent(t *testing.T) {
	canvasID := "canvas-broadcast"

	msg := &pb.CanvasMessage{
		Id:             canvasID,
		CanvasId:       canvasID,
		Timestamp:      timestamppb.Now(),
		OrganizationId: "org-broadcast",
	}

	body, err := proto.Marshal(msg)
	require.NoError(t, err)

	var marshaledEvent []byte
	originalFn := handleCanvasState

	err = originalFn(body, ws.NewHub(), CanvasCreatedEvent)
	require.NoError(t, err)

	expected := CanvasStateWebsocketEvent{
		Event: CanvasCreatedEvent,
		Payload: CanvasStatePayload{
			ID:       canvasID,
			CanvasID: canvasID,
		},
	}
	marshaledEvent, err = json.Marshal(expected)
	require.NoError(t, err)
	require.NotEmpty(t, marshaledEvent)

	var unmarshaled CanvasStateWebsocketEvent
	err = json.Unmarshal(marshaledEvent, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, CanvasCreatedEvent, unmarshaled.Event)
	require.Equal(t, canvasID, unmarshaled.Payload.CanvasID)
}
