package stages

import (
	"context"
	"errors"
	"fmt"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeStage(ctx context.Context, canvasID string, idOrName string) (*pb.DescribeStageResponse, error) {
	stage, err := findStage(canvasID, idOrName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "stage not found")
		}

		log.Errorf("Error describing stage %s in canvas %s: %v", canvasID, idOrName, err)
		return nil, err
	}

	connections, err := models.ListConnections(stage.ID, models.ConnectionTargetTypeStage)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections for stage: %w", err)
	}

	conn, err := actions.SerializeConnections(connections)
	if err != nil {
		return nil, err
	}

	statusInfo, err := models.GetStagesStatusInfo([]models.Stage{*stage})
	if err != nil {
		return nil, fmt.Errorf("failed to get stage status info: %w", err)
	}

	var stageStatus *models.StageStatusInfo
	if info, exists := statusInfo[stage.ID]; exists {
		stageStatus = info
	}

	serialized, err := serializeStage(
		*stage,
		conn,
		serializeInputs(stage.Inputs),
		serializeOutputs(stage.Outputs),
		serializeInputMappings(stage.InputMappings),
		stageStatus,
	)

	if err != nil {
		return nil, err
	}

	response := &pb.DescribeStageResponse{
		Stage: serialized,
	}

	return response, nil
}

func findStage(canvasID string, idOrName string) (*models.Stage, error) {
	if idOrName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must specify either the ID or name of the stage")
	}

	_, err := uuid.Parse(idOrName)
	if err == nil {
		return models.FindStageByID(canvasID, idOrName)
	}

	return models.FindStageByName(canvasID, idOrName)
}
