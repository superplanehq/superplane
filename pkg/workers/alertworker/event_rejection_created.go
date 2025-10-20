package alertworker

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

	//
	// Only create alerts for errors in case of event rejections
	//
	if rejection.Reason != models.EventRejectionReasonError {
		return nil, fmt.Errorf("skipping alert creation for rejection reason: %s", rejection.Reason)
	}

	alert, err := models.NewAlert(
		rejection.Event.CanvasID,
		rejection.TargetID,
		rejection.TargetType,
		rejection.Message,
		models.AlertTypeError,
		models.AlertOriginTypeEventRejection,
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
