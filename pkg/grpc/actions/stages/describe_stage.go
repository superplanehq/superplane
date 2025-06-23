package stages

import (
	"context"
	"errors"
	"fmt"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeStage(ctx context.Context, req *pb.DescribeStageRequest) (*pb.DescribeStageResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)
	stage, err := findStage(canvas, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "stage not found")
		}

		logger.Errorf("Error describing stage. Request: %v. Error: %v", req, err)
		return nil, err
	}

	//
	// TODO: we have to list all stages/sources because the API expects
	// the stage connection to use names, and the stage_connections table does not record that.
	//

	stages, err := canvas.ListStages()
	if err != nil {
		return nil, fmt.Errorf("failed to list stages for canvas: %w", err)
	}

	sources, err := canvas.ListEventSources()
	if err != nil {
		return nil, fmt.Errorf("failed to list event sources for canvas: %w", err)
	}

	connections, err := models.ListConnections(stage.ID, models.ConnectionTargetTypeStage)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections for stage: %w", err)
	}

	conn, err := actions.SerializeConnections(stages, sources, connections)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeStage(
		*stage,
		conn,
		serializeInputs(stage.Inputs),
		serializeOutputs(stage.Outputs),
		serializeInputMappings(stage.InputMappings),
	)

	if err != nil {
		return nil, err
	}

	response := &pb.DescribeStageResponse{
		Stage: serialized,
	}

	return response, nil
}

func findStage(canvas *models.Canvas, req *pb.DescribeStageRequest) (*models.Stage, error) {
	if req.Id == "" && req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must specify one of: id or name")
	}

	if req.Name != "" {
		return canvas.FindStageByName(req.Name)
	}

	ID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ID")
	}

	return canvas.FindStageByID(ID.String())
}
