package stageevents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ApproveStageEvent(ctx context.Context, req *pb.ApproveStageEventRequest) (*pb.ApproveStageEventResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "canvas not found")
		}

		return nil, err
	}

	err = actions.ValidateUUIDs(req.StageIdOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = canvas.FindStageByName(req.StageIdOrName)
	} else {
		stage, err = canvas.FindStageByID(req.StageIdOrName)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	err = actions.ValidateUUIDs(req.EventId, req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	logger := logging.ForStage(stage)
	event, err := models.FindStageEventByID(req.EventId, stage.ID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "event not found")
		}

		return nil, err
	}

	err = event.Approve(uuid.MustParse(req.RequesterId))
	if err != nil {
		if errors.Is(err, models.ErrEventAlreadyApprovedByRequester) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
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
