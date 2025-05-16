package actions

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ApproveStageEvent(ctx context.Context, req *pb.ApproveStageEventRequest) (*pb.ApproveStageEventResponse, error) {
	err := ValidateUUIDs(req.OrganizationId, req.CanvasId, req.StageId, req.EventId, req.RequesterId)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvasByID(req.CanvasId, req.OrganizationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
		}

		return nil, err
	}

	stage, err := canvas.FindStageByID(req.StageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	logger := logging.ForStage(stage)
	event, err := models.FindStageEventByID(req.EventId, req.StageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "event not found")
		}

		return nil, err
	}

	err = event.Approve(uuid.MustParse(req.RequesterId))
	if err != nil {
		if errors.Is(err, models.ErrEventAlreadyApprovedByRequester) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		logger.Errorf("failed to approve event: %v", err)
		return nil, err
	}

	logger.Infof("event %s approved", event.ID)

	err = messages.NewStageEventApprovedMessage(canvas.ID.String(), event).Publish()
	if err != nil {
		logger.Errorf("failed to publish event approved message: %v", err)
	}

	serialized, err := serializeStageEvent(*event)
	if err != nil {
		logger.Errorf("failed to serialize stage event: %v", err)
		return nil, err
	}

	return &pb.ApproveStageEventResponse{
		Event: serialized,
	}, nil
}
