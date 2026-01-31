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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvas(canvas *models.Canvas, includeStatus bool) (*pb.Canvas, error) {
	serializedNodes, err := serializeCanvasNodes(canvas)
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

	if !includeStatus {
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
				Nodes: serializedNodes,
				Edges: actions.EdgesToProto(canvas.Edges),
			},
			Status: nil,
		}, nil
	}

	// Fetch last executions per node
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
			Nodes: serializedNodes,
			Edges: actions.EdgesToProto(canvas.Edges),
		},
		Status: &pb.Canvas_Status{
			LastExecutions: serializedExecutions,
			NextQueueItems: serializedQueueItems,
			LastEvents:     serializedEvents,
		},
	}, nil
}

func serializeCanvasNodes(canvas *models.Canvas) ([]*compb.Node, error) {
	serialized := actions.NodesToProto(canvas.Nodes)
	if len(serialized) == 0 {
		return serialized, nil
	}

	canvasNodes, err := models.FindCanvasNodes(canvas.ID)
	if err != nil {
		return nil, err
	}

	pausedByID := make(map[string]bool, len(canvasNodes))
	for _, node := range canvasNodes {
		pausedByID[node.NodeID] = node.State == models.CanvasNodeStatePaused
	}

	for _, node := range serialized {
		if paused, ok := pausedByID[node.Id]; ok {
			node.Paused = paused
		}
	}

	return serialized, nil
}

func ParseCanvas(registry *registry.Registry, orgID string, canvas *pb.Canvas) ([]models.Node, []models.Edge, error) {
	if canvas.Metadata == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "canvas metadata is required")
	}

	if canvas.Metadata.Name == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	if canvas.Spec == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "canvas spec is required")
	}

	// Allow empty canvases
	if len(canvas.Spec.Nodes) == 0 {
		return []models.Node{}, []models.Edge{}, nil
	}

	nodeIDs := make(map[string]bool)
	nodeTypeByID := make(map[string]compb.Node_Type)
	nodeValidationErrors := make(map[string]string)

	for i, node := range canvas.Spec.Nodes {
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
		nodeTypeByID[node.Id] = node.Type

		if err := validateNodeRef(registry, orgID, node); err != nil {
			nodeValidationErrors[node.Id] = err.Error()
		}
	}

	// Find shadowed names within connected components
	nodeWarnings := actions.FindShadowedNameWarnings(canvas.Spec.Nodes, canvas.Spec.Edges)

	for i, edge := range canvas.Spec.Edges {
		if edge.SourceId == "" || edge.TargetId == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source_id and target_id are required", i)
		}

		if !nodeIDs[edge.SourceId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if !nodeIDs[edge.TargetId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}

		if nodeTypeByID[edge.SourceId] == compb.Node_TYPE_WIDGET {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: widget nodes cannot be used as source nodes", i)
		}

		if nodeTypeByID[edge.TargetId] == compb.Node_TYPE_WIDGET {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: widget nodes cannot be used as target nodes", i)
		}
	}

	if err := actions.CheckForCycles(canvas.Spec.Nodes, canvas.Spec.Edges); err != nil {
		return nil, nil, err
	}

	// Convert proto nodes to models, adding validation errors and warnings where applicable
	nodes := actions.ProtoToNodes(canvas.Spec.Nodes)
	for i := range nodes {
		if errorMsg, hasError := nodeValidationErrors[nodes[i].ID]; hasError {
			nodes[i].ErrorMessage = &errorMsg
		} else {
			nodes[i].ErrorMessage = nil
		}

		if warningMsg, hasWarning := nodeWarnings[nodes[i].ID]; hasWarning {
			nodes[i].WarningMessage = &warningMsg
		} else {
			nodes[i].WarningMessage = nil
		}
	}

	return nodes, actions.ProtoToEdges(canvas.Spec.Edges), nil
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

		component, err := findAndValidateComponent(registry, organizationID, node)
		if err != nil {
			return err
		}

		return configuration.ValidateConfiguration(component.Configuration(), node.Configuration.AsMap())

	case compb.Node_TYPE_BLUEPRINT:
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

	case compb.Node_TYPE_TRIGGER:
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

	case compb.Node_TYPE_WIDGET:
		if node.Widget == nil {
			return fmt.Errorf("widget reference is required for widget ref type")
		}

		if node.Widget.Name == "" {
			return fmt.Errorf("widget name is required")
		}

		widget, err := findAndValidateWidget(registry, organizationID, node)
		if err != nil {
			return err
		}

		return configuration.ValidateConfiguration(widget.Configuration(), node.Configuration.AsMap())

	default:
		return fmt.Errorf("invalid node type: %s", node.Type)
	}
}

func findAndValidateTrigger(registry *registry.Registry, organizationID string, node *compb.Node) (core.Trigger, error) {
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

func findAndValidateWidget(registry *registry.Registry, organizationID string, node *compb.Node) (core.Widget, error) {
	if node.Widget != nil && node.Widget.Name == "" {
		return nil, fmt.Errorf("widget name is required")
	}

	return registry.GetWidget(node.Widget.Name)
}

func findAndValidateComponent(registry *registry.Registry, organizationID string, node *compb.Node) (core.Component, error) {
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
