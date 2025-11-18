package workflows

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeWorkflow(workflow *models.Workflow, includeStatus bool) *pb.Workflow {
	workflowNodes, err := models.FindWorkflowNodes(workflow.ID)
	if err != nil {
		return nil
	}

	// Only expose top-level nodes (no parents) to the UI
	nodes := make([]models.Node, 0, len(workflowNodes))
	for _, wn := range workflowNodes {
		if wn.ParentNodeID != nil {
			continue
		}
		nodes = append(nodes, models.Node{
			ID:            wn.NodeID,
			Name:          wn.Name,
			Type:          wn.Type,
			Ref:           wn.Ref.Data(),
			Configuration: wn.Configuration.Data(),
			Metadata:      wn.Metadata.Data(),
			Position:      wn.Position.Data(),
			IsCollapsed:   wn.IsCollapsed,
		})
	}

	var createdBy *pb.UserRef
	if workflow.CreatedBy != nil {
		idStr := workflow.CreatedBy.String()
		name := ""
		if user, err := models.FindMaybeDeletedUserByID(workflow.OrganizationID.String(), idStr); err == nil && user != nil {
			name = user.Name
		}
		createdBy = &pb.UserRef{Id: idStr, Name: name}
	}

	if !includeStatus {
		return &pb.Workflow{
			Metadata: &pb.Workflow_Metadata{
				Id:             workflow.ID.String(),
				OrganizationId: workflow.OrganizationID.String(),
				Name:           workflow.Name,
				Description:    workflow.Description,
				CreatedAt:      timestamppb.New(*workflow.CreatedAt),
				UpdatedAt:      timestamppb.New(*workflow.UpdatedAt),
				CreatedBy:      createdBy,
			},
			Spec: &pb.Workflow_Spec{
				Nodes: actions.NodesToProto(nodes),
				Edges: actions.EdgesToProto(workflow.Edges),
			},
			Status: nil,
		}
	}

	// Fetch last executions per node
	lastExecutions, err := models.FindLastExecutionPerNode(workflow.ID)
	if err != nil {
		return nil
	}

	executionIDs := make([]string, len(lastExecutions))
	for i, execution := range lastExecutions {
		executionIDs[i] = execution.ID.String()
	}

	childExecutions, err := models.FindChildExecutionsForMultiple(executionIDs)
	if err != nil {
		return nil
	}

	serializedExecutions, err := SerializeNodeExecutions(lastExecutions, childExecutions)
	if err != nil {
		return nil
	}

	// Fetch next queue items per node
	nextQueueItems, err := models.FindNextQueueItemPerNode(workflow.ID)
	if err != nil {
		return nil
	}

	serializedQueueItems, err := SerializeNodeQueueItems(nextQueueItems)
	if err != nil {
		return nil
	}

	// Fetch last events per node
	lastEvents, err := models.FindLastEventPerNode(workflow.ID)
	if err != nil {
		return nil
	}

	serializedEvents, err := SerializeWorkflowEvents(lastEvents)
	if err != nil {
		return nil
	}

	return &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Id:             workflow.ID.String(),
			OrganizationId: workflow.OrganizationID.String(),
			Name:           workflow.Name,
			Description:    workflow.Description,
			CreatedAt:      timestamppb.New(*workflow.CreatedAt),
			UpdatedAt:      timestamppb.New(*workflow.UpdatedAt),
			CreatedBy:      createdBy,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: actions.NodesToProto(nodes),
			Edges: actions.EdgesToProto(workflow.Edges),
		},
		Status: &pb.Workflow_Status{
			LastExecutions: serializedExecutions,
			NextQueueItems: serializedQueueItems,
			LastEvents:     serializedEvents,
		},
	}
}

func ParseWorkflow(registry *registry.Registry, orgID string, workflow *pb.Workflow) ([]models.Node, []models.Edge, error) {
	if workflow.Metadata == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "workflow metadata is required")
	}

	if workflow.Metadata.Name == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "workflow name is required")
	}

	if workflow.Spec == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "workflow spec is required")
	}

	// Allow empty workflows
	if len(workflow.Spec.Nodes) == 0 {
		return []models.Node{}, []models.Edge{}, nil
	}

	nodeIDs := make(map[string]bool)
	for i, node := range workflow.Spec.Nodes {
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

		if err := validateNodeRef(registry, orgID, node); err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node '%s' (%s): %v", node.Name, node.Id, err)
		}
	}

	for i, edge := range workflow.Spec.Edges {
		if edge.SourceId == "" || edge.TargetId == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source_id and target_id are required", i)
		}

		if !nodeIDs[edge.SourceId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if !nodeIDs[edge.TargetId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}
	}

	if err := actions.CheckForCycles(workflow.Spec.Nodes, workflow.Spec.Edges); err != nil {
		return nil, nil, err
	}

	return actions.ProtoToNodes(workflow.Spec.Nodes), actions.ProtoToEdges(workflow.Spec.Edges), nil
}

func validateNodeRef(registry *registry.Registry, organizationID string, node *compb.Node) error {
	switch node.Type {
	case compb.Node_TYPE_COMPONENT:
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

		return configuration.ValidateConfiguration(component.Configuration(), node.Configuration.AsMap())

	case compb.Node_TYPE_BLUEPRINT:
		if node.Blueprint == nil {
			return fmt.Errorf("blueprint reference is required for blueprint ref type")
		}

		if node.Blueprint.Id == "" {
			return fmt.Errorf("blueprint ID is required")
		}

		_, err := models.FindBlueprint(organizationID, node.Blueprint.Id)
		if err != nil {
			return fmt.Errorf("blueprint %s not found", node.Blueprint.Id)
		}

		return nil

	case compb.Node_TYPE_TRIGGER:
		if node.Trigger == nil {
			return fmt.Errorf("trigger reference is required for trigger ref type")
		}

		if node.Trigger.Name == "" {
			return fmt.Errorf("trigger name is required")
		}

		trigger, err := registry.GetTrigger(node.Trigger.Name)
		if err != nil {
			return fmt.Errorf("trigger %s not found", node.Trigger.Name)
		}

		return configuration.ValidateConfiguration(trigger.Configuration(), node.Configuration.AsMap())

	default:
		return fmt.Errorf("invalid node type: %s", node.Type)
	}
}
