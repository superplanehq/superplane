package blueprints

import (
    "fmt"

    "github.com/superplanehq/superplane/pkg/components"
    "github.com/superplanehq/superplane/pkg/grpc/actions"
    "github.com/superplanehq/superplane/pkg/models"
    pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
    componentpb "github.com/superplanehq/superplane/pkg/protos/components"
    "github.com/superplanehq/superplane/pkg/registry"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeBlueprint(in *models.Blueprint) *pb.Blueprint {
    var createdBy *pb.UserRef
    if in.CreatedBy != nil {
        idStr := in.CreatedBy.String()
        name := ""
        if user, err := models.FindMaybeDeletedUserByID(in.OrganizationID.String(), idStr); err == nil && user != nil {
            name = user.Name
        }
        createdBy = &pb.UserRef{Id: idStr, Name: name}
    }

    return &pb.Blueprint{
        Id:             in.ID.String(),
        OrganizationId: in.OrganizationID.String(),
        Name:           in.Name,
        Description:    in.Description,
        CreatedAt:      timestamppb.New(*in.CreatedAt),
        UpdatedAt:      timestamppb.New(*in.UpdatedAt),
        Icon:           in.Icon,
        Color:          in.Color,
        Nodes:          actions.NodesToProto(in.Nodes),
        Edges:          actions.EdgesToProto(in.Edges),
        Configuration:  ConfigurationToProto(in.Configuration),
        OutputChannels: OutputChannelsToProto(in.OutputChannels),
        CreatedBy:      createdBy,
    }
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
	}

	if err := validateAcyclic(blueprint.Nodes, blueprint.Edges); err != nil {
		return nil, nil, err
	}

	return actions.ProtoToNodes(blueprint.Nodes), actions.ProtoToEdges(blueprint.Edges), nil
}

func validateNodeRef(registry *registry.Registry, node *componentpb.Node) error {
	switch node.Type {
	case componentpb.Node_TYPE_COMPONENT:
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
		return fmt.Errorf("invalid node type")
	}
}

func validateAcyclic(nodes []*componentpb.Node, edges []*componentpb.Edge) error {
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

func ConfigurationToProto(config []components.ConfigurationField) []*componentpb.ConfigurationField {
	if config == nil {
		return []*componentpb.ConfigurationField{}
	}

	result := make([]*componentpb.ConfigurationField, len(config))
	for i, field := range config {
		result[i] = actions.ConfigurationFieldToProto(field)
	}
	return result
}

func ProtoToConfiguration(config []*componentpb.ConfigurationField) ([]components.ConfigurationField, error) {
	if len(config) == 0 {
		return []components.ConfigurationField{}, nil
	}

	result := make([]components.ConfigurationField, len(config))
	for i, field := range config {
		if field.Name == "" {
			return nil, status.Errorf(codes.InvalidArgument, "configuration field %d: name is required", i)
		}
		if field.Label == "" {
			return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: label is required", field.Name)
		}

		if !field.Required && field.DefaultValue == nil {
			return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: default value is required when field is not required", field.Name)
		}

		// Type-specific validation
		if field.TypeOptions != nil {
			switch field.Type {
			case components.FieldTypeNumber:
				if field.TypeOptions.Number == nil || (field.TypeOptions.Number.Min == nil && field.TypeOptions.Number.Max == nil) {
					return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: number type options are required for number type", field.Name)
				}
			case components.FieldTypeSelect:
				if field.TypeOptions.Select == nil || len(field.TypeOptions.Select.Options) == 0 {
					return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: options are required for select type", field.Name)
				}
			case components.FieldTypeMultiSelect:
				if field.TypeOptions.MultiSelect == nil || len(field.TypeOptions.MultiSelect.Options) == 0 {
					return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: options are required for multi_select type", field.Name)
				}
			case components.FieldTypeIntegration:
				if field.TypeOptions.Integration == nil || field.TypeOptions.Integration.Type == "" {
					return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: integration type is required for integration type", field.Name)
				}
			}
		}

		result[i] = actions.ProtoToConfigurationField(field)
	}

	return result, nil
}

func OutputChannelsToProto(outputChannels []models.BlueprintOutputChannel) []*pb.OutputChannel {
	if outputChannels == nil {
		return []*pb.OutputChannel{}
	}

	result := make([]*pb.OutputChannel, len(outputChannels))
	for i, oc := range outputChannels {
		result[i] = &pb.OutputChannel{
			Name:              oc.Name,
			NodeId:            oc.NodeID,
			NodeOutputChannel: oc.NodeOutputChannel,
		}
	}
	return result
}

func ProtoToOutputChannels(outputChannels []*pb.OutputChannel) []models.BlueprintOutputChannel {
	if len(outputChannels) == 0 {
		return []models.BlueprintOutputChannel{}
	}

	result := make([]models.BlueprintOutputChannel, len(outputChannels))
	for i, oc := range outputChannels {
		result[i] = models.BlueprintOutputChannel{
			Name:              oc.Name,
			NodeID:            oc.NodeId,
			NodeOutputChannel: oc.NodeOutputChannel,
		}
	}
	return result
}
