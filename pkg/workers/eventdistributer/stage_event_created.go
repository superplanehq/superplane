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

// HandleStageEventCreated processes a stage event created message and forwards it to websocket clients
func HandleStageEventCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received new_stage_event event")

	// Parse the protobuf message
	pbMsg := &pb.StageEventCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StageEventCreated message: %w", err)
	}

	// First get the stage information
	describeStageResp, err := actions.DescribeStage(context.Background(), &pb.DescribeStageRequest{
		CanvasIdOrName: pbMsg.CanvasId,
		Id:             pbMsg.StageId,
	})
	if err != nil {
		log.Warnf("Failed to describe stage: %v, continuing with basic info", err)
	}

	// Prepare event payload
	payload := map[string]interface{}{
		"id":        pbMsg.EventId,
		"stage_id":  pbMsg.StageId,
		"canvas_id": pbMsg.CanvasId,
		"source_id": pbMsg.SourceId,
		"timestamp": pbMsg.Timestamp.AsTime(),
	}

	// Add stage information if available
	if describeStageResp != nil && describeStageResp.Stage != nil {
		payload["stage"] = describeStageResp.Stage
	}

	// Convert to a websocket-friendly format
	wsEvent := map[string]interface{}{
		"event":   "new_stage_event",
		"payload": payload,
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to all clients subscribed to this canvas
	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted new_stage_event event to canvas %s", pbMsg.CanvasId)

	return nil
}
