package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

type AlertAcknowledgedWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleAlertAcknowledged(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received alert_acknowledged event")

	pbMsg := &pb.AlertAcknowledged{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal AlertAcknowledged message: %w", err)
	}

	alertJSON, err := FindAndParseAlert(pbMsg.AlertId, pbMsg.CanvasId)
	if err != nil {
		return fmt.Errorf("failed to find and parse alert: %w", err)
	}

	event, err := json.Marshal(AlertAcknowledgedWebsocketEvent{
		Event:   "alert_acknowledged",
		Payload: json.RawMessage(alertJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, event)
	log.Debugf("Broadcasted alert_acknowledged event to canvas %s", pbMsg.CanvasId)

	return nil
}
