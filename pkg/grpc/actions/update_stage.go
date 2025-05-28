package actions

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateStage(ctx context.Context, encryptor encryptor.Encryptor, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	err := ValidateUUIDs(req.IdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
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
			return nil, status.Errorf(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	err = ValidateUUIDs(req.RequesterId)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "requester ID is invalid")
	}

	template, err := validateRunTemplate(ctx, encryptor, req.RunTemplate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	connections, err := validateConnections(canvas, req.Connections)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(req.Conditions)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = canvas.UpdateStage(stage.ID.String(), req.RequesterId, conditions, *template, connections)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	stage, err = canvas.FindStageByID(stage.ID.String())
	if err != nil {
		return nil, err
	}

	serialized, err := serializeStage(*stage, req.Connections)
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
