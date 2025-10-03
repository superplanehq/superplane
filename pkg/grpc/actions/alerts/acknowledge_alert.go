package alerts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func AcknowledgeAlert(ctx context.Context, canvasID string, alertID string) (*pb.AcknowledgeAlertResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas ID: %w", err)
	}

	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		return nil, fmt.Errorf("invalid alert ID: %w", err)
	}

	alert, err := models.FindAlertByID(alertUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find alert: %w", err)
	}

	alert.Acknowledge()

	if err := alert.Update(); err != nil {
		return nil, fmt.Errorf("failed to update alert: %w", err)
	}

	serialized := SerializeAlert(alert)

	err = messages.NewAlertAcknowledgedMessage(alert).Publish()
	if err != nil {
		log.Errorf("failed to publish alert acknowledged message: %v", err)
	}

	response := &pb.AcknowledgeAlertResponse{
		Alert: serialized,
	}

	return response, nil
}
