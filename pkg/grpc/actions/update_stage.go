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
)

func UpdateStage(ctx context.Context, encryptor encryptor.Encryptor, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	err := ValidateUUIDs(req.Id, req.CanvasId, req.RequesterId)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvas(req.CanvasId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
	}

	_, err = models.FindStageByID(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "stage not found")
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

	err = canvas.UpdateStage(req.Id, req.RequesterId, conditions, *template, connections)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	stage, err := canvas.FindStageByID(req.Id)
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
