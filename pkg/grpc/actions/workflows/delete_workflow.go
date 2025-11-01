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
	"gorm.io/gorm"
)

func DeleteWorkflow(ctx context.Context, registry *registry.Registry, organizationID uuid.UUID, id string) (*pb.DeleteWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	workflow, err := models.FindWorkflow(organizationID, workflowID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		nodes, err := models.FindWorkflowNodes(workflow.ID)
		if err != nil {
			return err
		}

		for _, node := range nodes {
			err = models.DeleteWorkflowNode(database.Conn(), node)
			if err != nil {
				return err
			}
		}

		err = database.Conn().Delete(&workflow).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete workflow: %v", err)
	}

	return &pb.DeleteWorkflowResponse{}, nil
}
