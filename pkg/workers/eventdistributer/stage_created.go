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

// HandleStageCreated processes a stage created message and forwards it to websocket clients
func HandleStageCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_added event")

	// Parse the protobuf message
	pbMsg := &pb.StageCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StageCreated message: %w", err)
	}

	// Fetch complete stage information using gRPC
	describeStageResp, err := actions.DescribeStage(context.Background(), &pb.DescribeStageRequest{
		CanvasIdOrName: pbMsg.CanvasId,
		Id:             pbMsg.StageId,
	})
	if err != nil {
		return fmt.Errorf("failed to describe stage: %w", err)
	}

	// Convert protobuf to a more websocket-friendly format with complete information
	wsEvent := map[string]interface{}{
		"event": "stage_added",
		"payload": map[string]interface{}{
			"id":            describeStageResp.Stage.Id,
			"canvas_id":     describeStageResp.Stage.CanvasId,
			"name":          describeStageResp.Stage.Name,
			"created_at":    describeStageResp.Stage.CreatedAt,
			"conditions":    describeStageResp.Stage.Conditions,
			"executor_spec": describeStageResp.Stage.Executor,
		},
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to all clients subscribed to this canvas
	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted stage_added event to canvas %s", pbMsg.CanvasId)

	return nil
}
