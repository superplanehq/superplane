package eventdistributer

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

// HandleStageUpdated processes a stage updated message and forwards it to websocket clients
func HandleStageUpdated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_updated event")

	// Parse the protobuf message
	pbMsg := &pb.StageUpdated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StageUpdated message: %w", err)
	}

	describeStageResp, err := stages.DescribeStage(context.Background(), &pb.DescribeStageRequest{
		CanvasIdOrName: pbMsg.CanvasId,
		Id:             pbMsg.StageId,
	})
	if err != nil {
		return fmt.Errorf("failed to describe stage: %w", err)
	}

	// Convert protobuf to a more websocket-friendly format
	wsEvent := map[string]interface{}{
		"event":   "stage_updated",
		"payload": describeStageResp.Stage,
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to all clients subscribed to this canvas
	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted stage_updated event to canvas %s", pbMsg.CanvasId)

	return nil
}
