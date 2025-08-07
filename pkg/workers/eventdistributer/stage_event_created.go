package eventdistributer

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/stages"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleStageEventCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received new_stage_event event")

	pbMsg := &pb.StageEventCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StageEventCreated message: %w", err)
	}

	describeStageResp, err := stages.DescribeStage(context.Background(), pbMsg.CanvasId, pbMsg.StageId)
	if err != nil {
		return err
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event": "new_stage_event",
		"payload": map[string]any{
			"id":        pbMsg.EventId,
			"stage_id":  pbMsg.StageId,
			"canvas_id": pbMsg.CanvasId,
			"source_id": pbMsg.SourceId,
			"timestamp": pbMsg.Timestamp.AsTime(),
			"stage":     describeStageResp.Stage,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted new_stage_event event to canvas %s", pbMsg.CanvasId)

	return nil
}
