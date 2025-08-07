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

func HandleStageUpdated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_updated event")

	pbMsg := &pb.StageUpdated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StageUpdated message: %w", err)
	}

	describeStageResp, err := stages.DescribeStage(context.Background(), pbMsg.CanvasId, pbMsg.StageId)
	if err != nil {
		return fmt.Errorf("failed to describe stage: %w", err)
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event":   "stage_updated",
		"payload": describeStageResp.Stage,
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted stage_updated event to canvas %s", pbMsg.CanvasId)

	return nil
}
