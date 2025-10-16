package workflows

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, id string, pbWorkflow *pb.Workflow) (*pb.UpdateWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	existingWorkflow, err := models.FindWorkflow(uuid.MustParse(organizationID), workflowID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	nodes, edges, err := ParseWorkflow(registry, organizationID, pbWorkflow)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		//
		// Update the workflow record first
		//
		existingWorkflow.Name = pbWorkflow.Name
		existingWorkflow.Description = pbWorkflow.Description
		existingWorkflow.UpdatedAt = &now
		existingWorkflow.Edges = datatypes.NewJSONSlice(edges)
		err := tx.Save(&existingWorkflow).Error
		if err != nil {
			return err
		}

		//
		// Update the workflow node records
		//
		existingNodes, err := existingWorkflow.FindNodes()
		if err != nil {
			return err
		}

		//
		// Go through each node in the new workflow, creating / updating it,
		// and tracking which nodes we've seen, to delete nodes that are no longer in the workflow at the end.
		//
		for _, node := range nodes {
			err := upsertNode(tx, existingNodes, node, workflowID)
			if err != nil {
				return err
			}
		}

		return deleteNodes(tx, existingNodes, nodes, workflowID)
	})

	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkflowResponse{
		Workflow: SerializeWorkflow(existingWorkflow),
	}, nil
}

func findNode(nodes []models.WorkflowNode, nodeID string) *models.WorkflowNode {
	for _, node := range nodes {
		if node.NodeID == nodeID {
			return &node
		}
	}
	return nil
}

func upsertNode(tx *gorm.DB, existingNodes []models.WorkflowNode, node models.Node, workflowID uuid.UUID) error {
	now := time.Now()

	//
	// Node exists, just update it
	//
	existingNode := findNode(existingNodes, node.ID)
	if existingNode != nil {
		existingNode.Name = node.Name
		existingNode.Type = node.Type
		existingNode.Ref = datatypes.NewJSONType(node.Ref)
		existingNode.Configuration = datatypes.NewJSONType(node.Configuration)
		existingNode.UpdatedAt = &now
		return tx.Save(&existingNode).Error
	}

	//
	// Node doesn't exist, create it
	//
	workflowNode := models.WorkflowNode{
		WorkflowID:    workflowID,
		NodeID:        node.ID,
		Name:          node.Name,
		State:         models.WorkflowNodeStateReady,
		Type:          node.Type,
		Ref:           datatypes.NewJSONType(node.Ref),
		Configuration: datatypes.NewJSONType(node.Configuration),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	return tx.Create(&workflowNode).Error
}

func deleteNodes(tx *gorm.DB, existingNodes []models.WorkflowNode, newNodes []models.Node, workflowID uuid.UUID) error {
	for _, existingNode := range existingNodes {
		if !slices.ContainsFunc(newNodes, func(n models.Node) bool { return n.ID == existingNode.NodeID }) {
			err := models.DeleteWorkflowNode(tx, workflowID, existingNode.NodeID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
