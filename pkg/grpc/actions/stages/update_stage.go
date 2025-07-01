package stages

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateStage(ctx context.Context, specValidator executors.SpecValidator, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(req.IdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.IdOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = canvas.FindStageByName(req.IdOrName)
	} else {
		stage, err = canvas.FindStageByID(req.IdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	if req.Stage == nil || req.Stage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage spec is required")
	}

	executor, err := specValidator.Validate(req.Stage.Spec.Executor)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	inputValidator := inputs.NewValidator(
		inputs.WithInputs(req.Stage.Spec.Inputs),
		inputs.WithOutputs(req.Stage.Spec.Outputs),
		inputs.WithInputMappings(req.Stage.Spec.InputMappings),
		inputs.WithConnections(req.Stage.Spec.Connections),
	)

	err = inputValidator.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := actions.ValidateConnections(canvas, req.Stage.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(req.Stage.Spec.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets, err := validateSecrets(req.Stage.Spec.Secrets)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = canvas.UpdateStage(
		stage.ID.String(),
		userID,
		conditions,
		*executor,
		connections,
		inputValidator.SerializeInputs(),
		inputValidator.SerializeInputMappings(),
		inputValidator.SerializeOutputs(),
		secrets,
	)

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	stage, err = canvas.FindStageByID(stage.ID.String())
	if err != nil {
		return nil, err
	}

	serialized, err := serializeStage(
		*stage,
		req.Stage.Spec.Connections,
		req.Stage.Spec.Inputs,
		req.Stage.Spec.Outputs,
		req.Stage.Spec.InputMappings,
	)

	if err != nil {
		return nil, err
	}

	response := &pb.UpdateStageResponse{
		Stage: serialized,
	}

	err = messages.NewStageCreatedMessage(stage).Publish()

	if err != nil {
		logging.ForStage(stage).Errorf("failed to publish stage created message: %v", err)
	}

	return response, nil
}
