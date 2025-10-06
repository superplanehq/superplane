package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, id string) (*pb.DeleteWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	var workflow models.Workflow
	if err := database.Conn().Where("id = ? AND organization_id = ?", workflowID, organizationID).First(&workflow).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	if err := database.Conn().Delete(&workflow).Error; err != nil {
		return nil, err
	}

	return &pb.DeleteWorkflowResponse{}, nil
}
