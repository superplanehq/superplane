package actions

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateStage(ctx context.Context, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	err := ValidateUUIDs(req.IdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = ValidateUUIDs(req.IdOrName)
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

	err = ValidateUUIDs(req.RequesterId)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "requester ID is invalid")
	}

	executor, err := validateExecutorSpec(ctx, req.Executor)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	inputValidator := inputs.NewValidator(
		inputs.WithInputs(req.Inputs),
		inputs.WithOutputs(req.Outputs),
		inputs.WithInputMappings(req.InputMappings),
		inputs.WithConnections(req.Connections),
	)

	err = inputValidator.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := validateConnections(canvas, req.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(req.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets, err := validateSecrets(req.Secrets)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = canvas.UpdateStage(
		stage.ID.String(),
		req.RequesterId,
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
		req.Connections,
		req.Inputs,
		req.Outputs,
		req.InputMappings,
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
