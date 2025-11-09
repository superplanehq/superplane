package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, pbWorkflow *pb.Workflow) (*pb.CreateWorkflowResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

    nodes, edges, err := ParseWorkflow(registry, organizationID, pbWorkflow)
    if err != nil {
        return nil, err
    }

    // Expand blueprint nodes to include internal nodes (namespaced)
    expandedNodes, err := expandBlueprintNodes(organizationID, nodes)
    if err != nil {
        return nil, err
    }

	createdBy := uuid.MustParse(userID)

	now := time.Now()
	workflow := models.Workflow{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse(organizationID),
		Name:           pbWorkflow.Metadata.Name,
		Description:    pbWorkflow.Metadata.Description,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Edges:          datatypes.NewJSONSlice(edges),
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Create the workflow record
		//
		err := tx.Clauses(clause.Returning{}).Create(&workflow).Error
		if err != nil {
			return err
		}

		//
        // Create the workflow node records (including internal blueprint nodes)
        //
        for _, node := range expandedNodes {
            workflowNode := models.WorkflowNode{
                WorkflowID:    workflow.ID,
                NodeID:        node.ID,
                Name:          node.Name,
                State:         models.WorkflowNodeStateReady,
                Type:          node.Type,
                Ref:           datatypes.NewJSONType(node.Ref),
                Configuration: datatypes.NewJSONType(node.Configuration),
                Metadata:      datatypes.NewJSONType(node.Metadata),
                CreatedAt:     &now,
                UpdatedAt:     &now,
            }

			if err := tx.Create(&workflowNode).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkflowResponse{
		Workflow: SerializeWorkflow(&workflow, false),
	}, nil
}
