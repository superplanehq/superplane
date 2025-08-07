package eventdistributer

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	eventsources "github.com/superplanehq/superplane/pkg/grpc/actions/event_sources"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleEventSourceCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received event_source_added event")

	pbMsg := &pb.EventSourceCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal EventSourceCreated message: %w", err)
	}

	describeEventSourceResp, err := eventsources.DescribeEventSource(context.Background(), pbMsg.CanvasId, pbMsg.SourceId)
	if err != nil {
		return fmt.Errorf("failed to describe event source: %w", err)
	}

	wsEventJSON, err := json.Marshal(map[string]interface{}{
		"event":   "event_source_added",
		"payload": describeEventSourceResp.EventSource,
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)

	log.Debugf("Broadcasted event_source_added event to canvas %s", pbMsg.CanvasId)

	return nil
}
