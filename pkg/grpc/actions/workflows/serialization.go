package workflows

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeWorkflow(workflow *models.Workflow) *pb.Workflow {
	return &pb.Workflow{
		Id:             workflow.ID.String(),
		OrganizationId: workflow.OrganizationID.String(),
		Name:           workflow.Name,
		Description:    workflow.Description,
		CreatedAt:      timestamppb.New(*workflow.CreatedAt),
		UpdatedAt:      timestamppb.New(*workflow.UpdatedAt),
		Nodes:          actions.NodesToProto(workflow.Nodes),
		Edges:          actions.EdgesToProto(workflow.Edges),
	}
}

func ParseWorkflow(registry *registry.Registry, workflow *pb.Workflow) ([]models.Node, []models.Edge, error) {
	if workflow.Name == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "workflow name is required")
	}

	// Allow empty workflows
	if len(workflow.Nodes) == 0 {
		return []models.Node{}, []models.Edge{}, nil
	}

	nodeIDs := make(map[string]bool)
	for i, node := range workflow.Nodes {
		if node.Id == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %d: id is required", i)
		}

		if node.Name == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %s: name is required", node.Id)
		}

		if nodeIDs[node.Id] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %s: duplicate node id", node.Id)
		}

		nodeIDs[node.Id] = true

		if err := validateNodeRef(registry, node); err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %s: %v", node.Id, err)
		}
	}

	for i, edge := range workflow.Edges {
		if edge.SourceId == "" || edge.TargetId == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source_id and target_id are required", i)
		}

		if edge.TargetType != compb.Edge_REF_TYPE_NODE {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target_type must be set to NODE", i)
		}

		if !nodeIDs[edge.SourceId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if edge.TargetType == compb.Edge_REF_TYPE_NODE && !nodeIDs[edge.TargetId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}
	}

	if err := actions.CheckForCycles(workflow.Nodes, workflow.Edges); err != nil {
		return nil, nil, err
	}

	return actions.ProtoToNodes(workflow.Nodes), actions.ProtoToEdges(workflow.Edges), nil
}

func validateNodeRef(registry *registry.Registry, node *compb.Node) error {
	switch node.RefType {
	case compb.Node_REF_TYPE_COMPONENT:
		if node.Component == nil {
			return fmt.Errorf("component reference is required for component ref type")
		}

		if node.Component.Name == "" {
			return fmt.Errorf("component name is required")
		}

		component, err := registry.GetComponent(node.Component.Name)
		if err != nil {
			return fmt.Errorf("component %s not found", node.Component.Name)
		}

		// Validate component configuration
		configMap := node.Configuration.AsMap()
		configFields := component.Configuration()
		if err := components.ValidateConfiguration(configFields, configMap); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		return nil

	case compb.Node_REF_TYPE_BLUEPRINT:
		if node.Blueprint == nil {
			return fmt.Errorf("blueprint reference is required for blueprint ref type")
		}

		if node.Blueprint.Id == "" {
			return fmt.Errorf("blueprint ID is required")
		}

		_, err := models.FindBlueprintByID(node.Blueprint.Id)
		if err != nil {
			return fmt.Errorf("blueprint %s not found", node.Blueprint.Id)
		}

		return nil

	default:
		return fmt.Errorf("invalid ref type")
	}
}
