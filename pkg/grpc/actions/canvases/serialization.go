package canvases

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvas(canvas *models.Canvas, includeAdditionalStatusData bool) (*pb.Canvas, error) {
	nodeDefinitions, nodeStates, err := serializeCanvasNodes(canvas)
	if err != nil {
		return nil, err
	}

	var createdBy *pb.UserRef
	if canvas.CreatedBy != nil {
		idStr := canvas.CreatedBy.String()
		name := ""
		if user, err := models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), idStr); err == nil && user != nil {
			name = user.Name
		}
		createdBy = &pb.UserRef{Id: idStr, Name: name}
	}

	//
	// If we are not including information about events / executions / queue items, just return it.
	//
	if !includeAdditionalStatusData {
		return &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:             canvas.ID.String(),
				OrganizationId: canvas.OrganizationID.String(),
				Name:           canvas.Name,
				Description:    canvas.Description,
				CreatedAt:      timestamppb.New(*canvas.CreatedAt),
				UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
				CreatedBy:      createdBy,
				IsTemplate:     canvas.IsTemplate,
			},
			Spec: &pb.Canvas_Spec{
				Nodes: nodeDefinitions,
				Edges: actions.EdgesToProto(canvas.Edges),
			},
			Status: &pb.Canvas_Status{
				Nodes: nodeStates,
			},
		}, nil
	}

	//
	// Otherwise, fetch all the information needed before returning.
	//
	lastExecutions, err := models.FindLastExecutionPerNode(canvas.ID)
	if err != nil {
		return nil, err
	}

	executionIDs := make([]string, len(lastExecutions))
	for i, execution := range lastExecutions {
		executionIDs[i] = execution.ID.String()
	}

	childExecutions, err := models.FindChildExecutionsForMultiple(executionIDs)
	if err != nil {
		return nil, err
	}

	serializedExecutions, err := SerializeNodeExecutions(lastExecutions, childExecutions)
	if err != nil {
		return nil, err
	}

	// Fetch next queue items per node
	nextQueueItems, err := models.FindNextQueueItemPerNode(canvas.ID)
	if err != nil {
		return nil, err
	}

	serializedQueueItems, err := SerializeNodeQueueItems(nextQueueItems)
	if err != nil {
		return nil, err
	}

	// Fetch last events per node
	lastEvents, err := models.FindLastEventPerNode(canvas.ID)
	if err != nil {
		return nil, err
	}

	serializedEvents, err := SerializeCanvasEvents(lastEvents)
	if err != nil {
		return nil, err
	}

	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Id:             canvas.ID.String(),
			OrganizationId: canvas.OrganizationID.String(),
			Name:           canvas.Name,
			Description:    canvas.Description,
			CreatedAt:      timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
			CreatedBy:      createdBy,
			IsTemplate:     canvas.IsTemplate,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: nodeDefinitions,
			Edges: actions.EdgesToProto(canvas.Edges),
		},
		Status: &pb.Canvas_Status{
			Nodes:          nodeStates,
			LastExecutions: serializedExecutions,
			NextQueueItems: serializedQueueItems,
			LastEvents:     serializedEvents,
		},
	}, nil
}

func serializeCanvasNodes(canvas *models.Canvas) ([]*compb.NodeDefinition, []*pb.CanvasNodeState, error) {
	canvasNodes, err := models.FindCanvasNodes(canvas.ID)
	if err != nil {
		return nil, nil, err
	}

	if len(canvasNodes) == 0 {
		return []*compb.NodeDefinition{}, []*pb.CanvasNodeState{}, nil
	}

	nodeDefinitions := actions.NodeDefinitionsToProto(canvas.Nodes)
	nodeStates := serializeCanvasNodeStates(canvasNodes)
	return nodeDefinitions, nodeStates, nil
}

func serializeCanvasNodeStates(canvasNodes []models.CanvasNode) []*pb.CanvasNodeState {
	states := make([]*pb.CanvasNodeState, len(canvasNodes))
	for i, node := range canvasNodes {
		states[i] = serializeCanvasNodeState(&node)
	}

	return states
}

func serializeCanvasNodeState(node *models.CanvasNode) *pb.CanvasNodeState {
	state := &pb.CanvasNodeState{
		Id:    node.NodeID,
		State: node.State,
	}

	if node.StateReason != nil {
		state.StateReason = *node.StateReason
	}

	metadata, _ := structpb.NewStruct(node.Metadata.Data())
	state.Metadata = metadata
	return state
}

func ValidateEdges(canvas *pb.Canvas) ([]models.Edge, error) {
	//
	// Build a map of node IDs and their types.
	//
	nodeIDs := make(map[string]bool)
	nodeTypeByID := make(map[string]compb.NodeDefinition_Type)
	for _, node := range canvas.Spec.Nodes {
		nodeIDs[node.Id] = true
		nodeTypeByID[node.Id] = node.Type
	}

	//
	// Validate each edge.
	//
	for i, edge := range canvas.Spec.Edges {
		if edge.SourceId == "" || edge.TargetId == "" {
			return nil, status.Errorf(codes.InvalidArgument, "edge %d: source_id and target_id are required", i)
		}

		if !nodeIDs[edge.SourceId] {
			return nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if !nodeIDs[edge.TargetId] {
			return nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}

		if nodeTypeByID[edge.SourceId] == compb.NodeDefinition_TYPE_WIDGET {
			return nil, status.Errorf(codes.InvalidArgument, "edge %d: widget nodes cannot be used as source nodes", i)
		}

		if nodeTypeByID[edge.TargetId] == compb.NodeDefinition_TYPE_WIDGET {
			return nil, status.Errorf(codes.InvalidArgument, "edge %d: widget nodes cannot be used as target nodes", i)
		}
	}

	return actions.ProtoToEdges(canvas.Spec.Edges), nil
}

func ValidateNodes(canvas *pb.Canvas) error {
	nodeIDs := make(map[string]bool)

	for i, node := range canvas.Spec.Nodes {
		if node.Id == "" {
			return status.Errorf(codes.InvalidArgument, "node %d: id is required", i)
		}

		if node.Name == "" {
			return status.Errorf(codes.InvalidArgument, "node %s: name is required", node.Id)
		}

		if nodeIDs[node.Id] {
			return status.Errorf(codes.InvalidArgument, "node %s: duplicate node id", node.Id)
		}

		nodeIDs[node.Id] = true
	}

	return nil
}

func ApplyNodeValidations(registry *registry.Registry, orgID string, canvas *pb.Canvas) map[string]string {
	nodeValidationErrors := make(map[string]string)
	for _, node := range canvas.Spec.Nodes {
		if err := validateNodeRef(registry, orgID, node); err != nil {
			nodeValidationErrors[node.Id] = err.Error()
		}
	}

	return nodeValidationErrors
}

func validateNodeRef(registry *registry.Registry, organizationID string, node *compb.NodeDefinition) error {
	switch node.Type {
	case compb.NodeDefinition_TYPE_COMPONENT:
		if node.Component == nil {
			return fmt.Errorf("component reference is required for component ref type")
		}

		if node.Component.Name == "" {
			return fmt.Errorf("component name is required")
		}

		component, err := findAndValidateComponent(registry, organizationID, node)
		if err != nil {
			return err
		}

		return configuration.ValidateConfiguration(component.Configuration(), node.Configuration.AsMap())

	case compb.NodeDefinition_TYPE_BLUEPRINT:
		if node.Blueprint == nil {
			return fmt.Errorf("blueprint reference is required for blueprint ref type")
		}

		if node.Blueprint.Id == "" {
			return fmt.Errorf("blueprint ID is required")
		}

		blueprint, err := models.FindBlueprint(organizationID, node.Blueprint.Id)
		if err != nil {
			return fmt.Errorf("blueprint %s not found", node.Blueprint.Id)
		}

		return configuration.ValidateConfiguration(blueprint.Configuration, node.Configuration.AsMap())

	case compb.NodeDefinition_TYPE_TRIGGER:
		if node.Trigger == nil {
			return fmt.Errorf("trigger reference is required for trigger ref type")
		}

		if node.Trigger.Name == "" {
			return fmt.Errorf("trigger name is required")
		}

		trigger, err := findAndValidateTrigger(registry, organizationID, node)
		if err != nil {
			return err
		}

		return configuration.ValidateConfiguration(trigger.Configuration(), node.Configuration.AsMap())

	case compb.NodeDefinition_TYPE_WIDGET:
		if node.Widget == nil {
			return fmt.Errorf("widget reference is required for widget ref type")
		}

		if node.Widget.Name == "" {
			return fmt.Errorf("widget name is required")
		}

		widget, err := findAndValidateWidget(registry, node)
		if err != nil {
			return err
		}

		return configuration.ValidateConfiguration(widget.Configuration(), node.Configuration.AsMap())

	default:
		return fmt.Errorf("invalid node type: %s", node.Type)
	}
}

func findAndValidateTrigger(registry *registry.Registry, organizationID string, node *compb.NodeDefinition) (core.Trigger, error) {
	parts := strings.SplitN(node.Trigger.Name, ".", 2)
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid trigger name: %s", node.Trigger.Name)
	}

	if len(parts) == 1 {
		return registry.GetTrigger(parts[0])
	}

	err := validateIntegration(organizationID, node.Integration)
	if err != nil {
		return nil, err
	}

	return registry.GetIntegrationTrigger(parts[0], node.Trigger.Name)
}

func findAndValidateWidget(registry *registry.Registry, node *compb.NodeDefinition) (core.Widget, error) {
	if node.Widget != nil && node.Widget.Name == "" {
		return nil, fmt.Errorf("widget name is required")
	}

	return registry.GetWidget(node.Widget.Name)
}

func findAndValidateComponent(registry *registry.Registry, organizationID string, node *compb.NodeDefinition) (core.Component, error) {
	parts := strings.SplitN(node.Component.Name, ".", 2)
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid component name: %s", node.Component.Name)
	}

	if len(parts) == 1 {
		return registry.GetComponent(parts[0])
	}

	err := validateIntegration(organizationID, node.Integration)
	if err != nil {
		return nil, err
	}

	return registry.GetIntegrationComponent(parts[0], node.Component.Name)
}

func validateIntegration(organizationID string, ref *compb.IntegrationRef) error {
	if ref == nil || ref.Id == "" {
		return fmt.Errorf("integration is required")
	}

	integrationID, err := uuid.Parse(ref.Id)
	if err != nil {
		return fmt.Errorf("invalid integration ID: %v", err)
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %v", err)
	}

	_, err = models.FindIntegration(orgID, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found or does not belong to this organization")
	}

	return nil
}
