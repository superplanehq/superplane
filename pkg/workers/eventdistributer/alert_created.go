package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/alerts"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type AlertCreatedWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleAlertCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received alert_created event")

	pbMsg := &pb.AlertCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal AlertCreated message: %w", err)
	}

	alertJSON, err := FindAndParseAlert(pbMsg.AlertId, pbMsg.CanvasId)
	if err != nil {
		return fmt.Errorf("failed to find and parse alert: %w", err)
	}

	event, err := json.Marshal(AlertCreatedWebsocketEvent{
		Event:   "alert_created",
		Payload: json.RawMessage(alertJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, event)
	log.Debugf("Broadcasted alert_created event to canvas %s", pbMsg.CanvasId)

	return nil
}

func FindAndParseAlert(alertID string, canvasID string) ([]byte, error) {
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse alert ID: %w", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse canvas ID: %w", err)
	}

	alert, err := models.FindAlertByID(alertUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find alert: %w", err)
	}

	serializedAlert := alerts.SerializeAlert(alert)
	alertJSON, err := protojson.Marshal(serializedAlert)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize alert: %w", err)
	}

	return alertJSON, nil
}
