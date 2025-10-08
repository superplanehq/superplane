package workflows

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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
		Nodes:          NodesToProto(workflow.Nodes),
		Edges:          EdgesToProto(workflow.Edges),
	}
}

func ProtoToNodes(nodes []*pb.WorkflowNode) []models.Node {
	result := make([]models.Node, len(nodes))
	for i, node := range nodes {
		result[i] = models.Node{
			ID:            node.Id,
			Name:          node.Name,
			RefType:       ProtoToRefType(node.RefType),
			Ref:           ProtoToNodeRef(node),
			Configuration: node.Configuration.AsMap(),
		}
	}
	return result
}

func NodesToProto(nodes []models.Node) []*pb.WorkflowNode {
	result := make([]*pb.WorkflowNode, len(nodes))
	for i, node := range nodes {
		configuration, err := structpb.NewStruct(node.Configuration)
		if err != nil {
			configuration = &structpb.Struct{}
		}

		result[i] = &pb.WorkflowNode{
			Id:            node.ID,
			Name:          node.Name,
			RefType:       RefTypeToProto(node.RefType),
			Configuration: configuration,
		}

		ref := node.Ref
		if ref.Component != nil {
			result[i].Component = &pb.WorkflowNode_ComponentRef{
				Name: ref.Component.Name,
			}
		}
		if ref.Blueprint != nil {
			result[i].Blueprint = &pb.WorkflowNode_BlueprintRef{
				Name: ref.Blueprint.Name,
			}
		}
	}
	return result
}

func ProtoToEdges(edges []*pb.WorkflowEdge) []models.Edge {
	result := make([]models.Edge, len(edges))
	for i, edge := range edges {
		result[i] = models.Edge{
			SourceID: edge.SourceId,
			TargetID: edge.TargetId,
			Branch:   edge.Branch,
		}
	}
	return result
}

func EdgesToProto(edges []models.Edge) []*pb.WorkflowEdge {
	result := make([]*pb.WorkflowEdge, len(edges))
	for i, edge := range edges {
		result[i] = &pb.WorkflowEdge{
			SourceId: edge.SourceID,
			TargetId: edge.TargetID,
			Branch:   edge.Branch,
		}
	}
	return result
}

func ProtoToRefType(refType pb.WorkflowNode_RefType) string {
	switch refType {
	case pb.WorkflowNode_REF_TYPE_COMPONENT:
		return models.NodeRefTypeComponent
	case pb.WorkflowNode_REF_TYPE_BLUEPRINT:
		return models.NodeRefTypeBlueprint
	default:
		return ""
	}
}

func RefTypeToProto(refType string) pb.WorkflowNode_RefType {
	switch refType {
	case models.NodeRefTypeComponent:
		return pb.WorkflowNode_REF_TYPE_COMPONENT
	case models.NodeRefTypeBlueprint:
		return pb.WorkflowNode_REF_TYPE_BLUEPRINT
	default:
		return pb.WorkflowNode_REF_TYPE_COMPONENT
	}
}

func ProtoToNodeRef(node *pb.WorkflowNode) models.NodeRef {
	ref := models.NodeRef{}

	switch node.RefType {
	case pb.WorkflowNode_REF_TYPE_COMPONENT:
		if node.Component != nil {
			ref.Component = &models.ComponentRef{
				Name: node.Component.Name,
			}
		}
	case pb.WorkflowNode_REF_TYPE_BLUEPRINT:
		if node.Blueprint != nil {
			ref.Blueprint = &models.BlueprintRef{
				Name: node.Blueprint.Name,
			}
		}
	}

	return ref
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

		if !nodeIDs[edge.SourceId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if !nodeIDs[edge.TargetId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}
	}

	if err := validateAcyclic(workflow.Nodes, workflow.Edges); err != nil {
		return nil, nil, err
	}

	return ProtoToNodes(workflow.Nodes), ProtoToEdges(workflow.Edges), nil
}

func validateNodeRef(registry *registry.Registry, node *pb.WorkflowNode) error {
	switch node.RefType {
	case pb.WorkflowNode_REF_TYPE_COMPONENT:
		if node.Component == nil {
			return fmt.Errorf("component reference is required for component ref type")
		}

		if node.Component.Name == "" {
			return fmt.Errorf("component name is required")
		}

		_, err := registry.GetComponent(node.Component.Name)
		if err != nil {
			return fmt.Errorf("component %s not found", node.Component.Name)
		}

		return nil

	case pb.WorkflowNode_REF_TYPE_BLUEPRINT:
		if node.Blueprint == nil {
			return fmt.Errorf("blueprint reference is required for blueprint ref type")
		}

		if node.Blueprint.Name == "" {
			return fmt.Errorf("blueprint name is required")
		}

		var blueprint models.Blueprint
		if err := database.Conn().Where("name = ?", node.Blueprint.Name).First(&blueprint).Error; err != nil {
			return fmt.Errorf("blueprint %s not found", node.Blueprint.Name)
		}

		return nil

	default:
		return fmt.Errorf("invalid ref type")
	}
}

// Verify if the workflow is acyclic using
// topological sort algorithm - kahn's - to detect cycles
func validateAcyclic(nodes []*pb.WorkflowNode, edges []*pb.WorkflowEdge) error {

	//
	// Build adjacency list
	//
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	//
	// Initialize all nodesm and build the graph
	//
	for _, node := range nodes {
		graph[node.Id] = []string{}
		inDegree[node.Id] = 0
	}

	for _, edge := range edges {
		graph[edge.SourceId] = append(graph[edge.SourceId], edge.TargetId)
		inDegree[edge.TargetId]++
	}

	// Kahn's algorithm for topological sort
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we visited all nodes, the graph is acyclic
	if visited != len(nodes) {
		return status.Error(codes.InvalidArgument, "workflow contains a cycle")
	}

	return nil
}

func SerializeWorkflowEvent(event *models.WorkflowEvent) *pb.WorkflowEvent {
	data, err := structpb.NewStruct(event.Data.Data())
	if err != nil {
		data = &structpb.Struct{}
	}

	result := &pb.WorkflowEvent{
		Id:         event.ID.String(),
		WorkflowId: event.WorkflowID.String(),
		Data:       data,
		State:      event.State,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
		UpdatedAt:  timestamppb.New(*event.UpdatedAt),
	}

	if event.ParentEventID != nil {
		result.ParentEventId = event.ParentEventID.String()
	}

	if event.BlueprintName != nil {
		result.BlueprintName = *event.BlueprintName
	}

	return result
}
