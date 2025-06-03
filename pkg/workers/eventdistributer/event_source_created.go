package eventdistributer

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

// HandleEventSourceCreated processes an event source created message and forwards it to websocket clients
func HandleEventSourceCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received event_source_added event")

	// Parse the protobuf message
	pbMsg := &pb.EventSourceCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal EventSourceCreated message: %w", err)
	}

	// Fetch complete event source information using gRPC
	describeEventSourceResp, err := actions.DescribeEventSource(context.Background(), &pb.DescribeEventSourceRequest{
		CanvasIdOrName: pbMsg.CanvasId,
		Id:             pbMsg.SourceId,
	})
	if err != nil {
		return fmt.Errorf("failed to describe event source: %w", err)
	}

	// Convert protobuf to a more websocket-friendly format with complete information
	// Use only the fields that exist in the EventSource structure
	wsEvent := map[string]interface{}{
		"event":   "event_source_added",
		"payload": describeEventSourceResp.EventSource,
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to all clients subscribed to this canvas
	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted event_source_added event to canvas %s", pbMsg.CanvasId)

	return nil
}
