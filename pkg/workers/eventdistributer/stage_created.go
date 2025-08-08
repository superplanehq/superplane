package eventdistributer

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type StageCreatedWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleStageCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_added event")

	pbMsg := &pb.StageCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StageCreated message: %w", err)
	}

	describeStageResp, err := stages.DescribeStage(context.Background(), pbMsg.CanvasId, pbMsg.StageId)
	if err != nil {
		return fmt.Errorf("failed to describe stage: %w", err)
	}

	stageJSON, err := protojson.Marshal(describeStageResp.Stage)
	if err != nil {
		return fmt.Errorf("failed to marshal stage to JSON: %w", err)
	}

	// Convert to JSON for websocket transmission
	event, err := json.Marshal(StageCreatedWebsocketEvent{
		Event:   "stage_added",
		Payload: json.RawMessage(stageJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, event)
	log.Debugf("Broadcasted stage_added event to canvas %s", pbMsg.CanvasId)

	return nil
}
