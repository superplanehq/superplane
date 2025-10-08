package blueprints

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeBlueprint(in *models.Blueprint) *pb.Blueprint {
	return &pb.Blueprint{
		Id:             in.ID.String(),
		OrganizationId: in.OrganizationID.String(),
		Name:           in.Name,
		Description:    in.Description,
		CreatedAt:      timestamppb.New(*in.CreatedAt),
		UpdatedAt:      timestamppb.New(*in.UpdatedAt),
		Nodes:          NodesToProto(in.Nodes),
		Edges:          EdgesToProto(in.Edges),
	}
}

func ProtoToNodes(nodes []*pb.BlueprintNode) []models.Node {
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

func NodesToProto(nodes []models.Node) []*pb.BlueprintNode {
	result := make([]*pb.BlueprintNode, len(nodes))
	for i, node := range nodes {
		result[i] = &pb.BlueprintNode{
			Id:      node.ID,
			Name:    node.Name,
			RefType: RefTypeToProto(node.RefType),
		}

		if node.Ref.Component != nil {
			result[i].Component = &pb.BlueprintNode_ComponentRef{
				Name: node.Ref.Component.Name,
			}
		}

		if node.Configuration != nil {
			result[i].Configuration, _ = structpb.NewStruct(node.Configuration)
		}
	}
	return result
}

func ProtoToEdges(edges []*pb.BlueprintEdge) []models.Edge {
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

func EdgesToProto(edges []models.Edge) []*pb.BlueprintEdge {
	result := make([]*pb.BlueprintEdge, len(edges))
	for i, edge := range edges {
		result[i] = &pb.BlueprintEdge{
			SourceId: edge.SourceID,
			TargetId: edge.TargetID,
			Branch:   edge.Branch,
		}
	}
	return result
}

func ProtoToRefType(refType pb.BlueprintNode_RefType) string {
	switch refType {
	case pb.BlueprintNode_REF_TYPE_COMPONENT:
		return "component"
	default:
		return ""
	}
}

func RefTypeToProto(refType string) pb.BlueprintNode_RefType {
	switch refType {
	case "component":
		return pb.BlueprintNode_REF_TYPE_COMPONENT
	default:
		return pb.BlueprintNode_REF_TYPE_COMPONENT
	}
}

func ProtoToNodeRef(node *pb.BlueprintNode) models.NodeRef {
	ref := models.NodeRef{}

	switch node.RefType {
	case pb.BlueprintNode_REF_TYPE_COMPONENT:
		if node.Component != nil {
			ref.Component = &models.ComponentRef{
				Name: node.Component.Name,
			}
		}
	}

	return ref
}

func ParseBlueprint(registry *registry.Registry, blueprint *pb.Blueprint) ([]models.Node, []models.Edge, error) {
	if blueprint.Name == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "blueprint name is required")
	}

	nodeIDs := make(map[string]bool)
	for i, node := range blueprint.Nodes {
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

	for i, edge := range blueprint.Edges {
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

	if err := validateAcyclic(blueprint.Nodes, blueprint.Edges); err != nil {
		return nil, nil, err
	}

	nodes := ProtoToNodes(blueprint.Nodes)
	edges := ProtoToEdges(blueprint.Edges)

	return nodes, edges, nil
}

func validateNodeRef(registry *registry.Registry, node *pb.BlueprintNode) error {
	switch node.RefType {
	case pb.BlueprintNode_REF_TYPE_COMPONENT:
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
	default:
		return fmt.Errorf("invalid ref type")
	}
}

func validateAcyclic(nodes []*pb.BlueprintNode, edges []*pb.BlueprintEdge) error {
	// Build adjacency list
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all nodes
	for _, node := range nodes {
		graph[node.Id] = []string{}
		inDegree[node.Id] = 0
	}

	// Build the graph
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
		return status.Error(codes.InvalidArgument, "blueprint contains a cycle")
	}

	return nil
}
