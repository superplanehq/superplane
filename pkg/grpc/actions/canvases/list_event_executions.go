package canvases

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListEventExecutions(ctx context.Context, registry *registry.Registry, workflowID, eventID string) (*pb.ListEventExecutionsResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid event id")
	}

	var executions []models.CanvasNodeExecution
	query := database.Conn().
		Where("workflow_id = ?", workflowUUID).
		Where("root_event_id = ?", eventUUID).
		Order("created_at ASC")

	err = query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)

	resources, err := LoadNodeExecutionResources(db, executions)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeExecutions(executions, resources)
	if err != nil {
		return nil, err
	}

	return &pb.ListEventExecutionsResponse{
		Executions: serialized,
	}, nil
}
