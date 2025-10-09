package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListEventExecutions(ctx context.Context, registry *registry.Registry, workflowID, eventID string) (*pb.ListEventExecutionsResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return nil, err
	}

	var executions []models.WorkflowNodeExecution
	query := database.Conn().
		Where("workflow_id = ?", workflowUUID).
		Where("root_event_id = ?", eventUUID).
		Order("created_at ASC")

	if err := query.Find(&executions).Error; err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeExecutions(executions)
	if err != nil {
		return nil, err
	}

	return &pb.ListEventExecutionsResponse{
		Executions: serialized,
	}, nil
}
