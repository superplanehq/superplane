package alert_worker

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/proto"
)

func HandleEventRejectionCreated(messageBody []byte) (*models.Alert, error) {
	pbMsg := &pb.EventRejectionCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal EventRejectionCreated message: %w", err)
	}

	rejectionUUID, err := uuid.Parse(pbMsg.RejectionId)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rejection ID: %w", err)
	}

	rejection, err := models.FindEventRejectionByID(rejectionUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find rejection: %w", err)
	}

	alert, err := models.NewAlert(
		rejection.Event.CanvasID,
		rejection.Event.SourceID,
		rejection.Event.SourceType,
		rejection.Message,
		rejectionReasonToAlertType(rejection.Reason),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	err = alert.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	return alert, nil
}

func rejectionReasonToAlertType(reason string) string {
	switch reason {
	case models.EventRejectionReasonFiltered:
		return models.AlertTypeWarning
	case models.EventRejectionReasonError:
		return models.AlertTypeError
	default:
		return models.AlertTypeWarning
	}
}
