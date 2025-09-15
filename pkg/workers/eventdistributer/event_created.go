package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleEventCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received event_created event")

	pbMsg := &pb.EventCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal EventCreated message: %w", err)
	}

	event, err := json.Marshal(map[string]any{
		"event": "event_created",
		"payload": map[string]any{
			"id":          pbMsg.EventId,
			"canvas_id":   pbMsg.CanvasId,
			"source_id":   pbMsg.SourceId,
			"source_type": actions.ProtoToEventSourceType(pbMsg.SourceType),
			"timestamp":   pbMsg.Timestamp.AsTime(),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, event)
	log.Debugf("Broadcasted new_stage_event event to canvas %s", pbMsg.CanvasId)

	return nil
}
