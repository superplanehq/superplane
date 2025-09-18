package stages

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

func CancelStageExecution(ctx context.Context, canvasID string, stageIdOrName string, executionID string) (*pb.CancelStageExecutionResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	id, err := uuid.Parse(executionID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid execution ID")
	}

	err = actions.ValidateUUIDs(stageIdOrName)
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
	execution, err := models.FindExecutionByID(id, stage.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "event not found")
		}

		return nil, err
	}

	err = execution.Cancel(uuid.MustParse(userID))
	if err != nil {
		if errors.Is(err, models.ErrExecutionAlreadyCancelled) || errors.Is(err, models.ErrExecutionCannotBeCancelled) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		logger.Errorf("failed to cancel execution: %v", err)
		return nil, err
	}

	logger.Infof("execution %s marked for cancellation", execution.ID)

	err = messages.NewExecutionCancelledMessage(canvasID, execution).Publish()
	if err != nil {
		logger.Errorf("failed to publish execution cancellation message: %v", err)
	}

	serialized, err := serializeExecution(*execution)
	if err != nil {
		logger.Errorf("failed to serialize execution: %v", err)
		return nil, err
	}

	return &pb.CancelStageExecutionResponse{
		Execution: serialized,
	}, nil
}
