package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleStageEventApproved(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_event_approved event")

	var msg pb.StageEventApproved
	if err := proto.Unmarshal(messageBody, &msg); err != nil {
		log.Errorf("Failed to unmarshal StageEventApproved message: %v", err)
		return err
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event": "stage_event_approved",
		"payload": map[string]any{
			"id":        msg.EventId,
			"stage_id":  msg.StageId,
			"canvas_id": msg.CanvasId,
			"source_id": msg.SourceId,
			"approved":  true,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(msg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted stage_event_approved event to canvas %s", msg.CanvasId)

	return nil
}
