package stageevents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DiscardStageEvent(ctx context.Context, canvasID string, stageIdOrName string, eventID string) (*pb.DiscardStageEventResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(stageIdOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = models.FindStageByName(canvasID, stageIdOrName)
	} else {
		stage, err = models.FindStageByID(canvasID, stageIdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	logger := logging.ForStage(stage)
	event, err := models.FindStageEventByID(eventID, stage.ID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "event not found")
		}

		return nil, err
	}

	err = event.Discard(uuid.MustParse(userID))
	if err != nil {
		if errors.Is(err, models.ErrEventAlreadyDiscarded) || errors.Is(err, models.ErrEventCannotBeDiscarded) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		logger.Errorf("failed to cancel event: %v", err)
		return nil, err
	}

	logger.Infof("event %s discarded", event.ID)

	err = messages.NewStageEventDiscardedMessage(canvasID, event).Publish()
	if err != nil {
		logger.Errorf("failed to publish event discarded message: %v", err)
	}

	serialized, err := actions.SerializeStageEvent(*event)
	if err != nil {
		logger.Errorf("failed to serialize stage event: %v", err)
		return nil, err
	}

	return &pb.DiscardStageEventResponse{
		Event: serialized,
	}, nil
}
