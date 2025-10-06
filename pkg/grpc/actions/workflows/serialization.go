package workflows

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
		Nodes:          NodesToProto(workflow.Nodes),
		Edges:          EdgesToProto(workflow.Edges),
	}
}

func ProtoToNodes(nodes []*pb.WorkflowNode) []models.Node {
	result := make([]models.Node, len(nodes))
	for i, node := range nodes {
		result[i] = models.Node{
			ID:      node.Id,
			Name:    node.Name,
			RefType: ProtoToRefType(node.RefType),
			Ref:     ProtoToNodeRef(node),
		}
	}
	return result
}

func NodesToProto(nodes []models.Node) []*pb.WorkflowNode {
	result := make([]*pb.WorkflowNode, len(nodes))
	for i, node := range nodes {
		result[i] = &pb.WorkflowNode{
			Id:      node.ID,
			Name:    node.Name,
			RefType: RefTypeToProto(node.RefType),
		}

		ref := node.Ref
		if ref.Primitive != nil {
			result[i].Primitive = &pb.WorkflowNode_PrimitiveRef{
				Name: ref.Primitive.Name,
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
	case pb.WorkflowNode_REF_TYPE_PRIMITIVE:
		return "primitive"
	case pb.WorkflowNode_REF_TYPE_BLUEPRINT:
		return "blueprint"
	default:
		return ""
	}
}

func RefTypeToProto(refType string) pb.WorkflowNode_RefType {
	switch refType {
	case "primitive":
		return pb.WorkflowNode_REF_TYPE_PRIMITIVE
	case "blueprint":
		return pb.WorkflowNode_REF_TYPE_BLUEPRINT
	default:
		return pb.WorkflowNode_REF_TYPE_PRIMITIVE
	}
}

func ProtoToNodeRef(node *pb.WorkflowNode) models.NodeRef {
	ref := models.NodeRef{}

	switch node.RefType {
	case pb.WorkflowNode_REF_TYPE_PRIMITIVE:
		if node.Primitive != nil {
			ref.Primitive = &models.PrimitiveRef{
				Name: node.Primitive.Name,
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
	case pb.WorkflowNode_REF_TYPE_PRIMITIVE:
		if node.Primitive == nil {
			return fmt.Errorf("primitive reference is required for primitive ref type")
		}

		if node.Primitive.Name == "" {
			return fmt.Errorf("primitive name is required")
		}

		_, err := registry.GetPrimitive(node.Primitive.Name)
		if err != nil {
			return fmt.Errorf("primitive %s not found", node.Primitive.Name)
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
