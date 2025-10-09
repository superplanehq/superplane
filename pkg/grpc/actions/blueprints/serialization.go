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
	return &pb.Blueprint{
		Id:             in.ID.String(),
		OrganizationId: in.OrganizationID.String(),
		Name:           in.Name,
		Description:    in.Description,
		CreatedAt:      timestamppb.New(*in.CreatedAt),
		UpdatedAt:      timestamppb.New(*in.UpdatedAt),
		Nodes:          actions.NodesToProto(in.Nodes),
		Edges:          actions.EdgesToProto(in.Edges),
		Configuration:  ConfigurationToProto(in.Configuration),
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

		if edge.TargetType != componentpb.Edge_REF_TYPE_NODE && edge.TargetType != componentpb.Edge_REF_TYPE_OUTPUT_BRANCH {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target_type must be set to either NODE or OUTPUT_BRANCH", i)
		}

		if !nodeIDs[edge.SourceId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if edge.TargetType == componentpb.Edge_REF_TYPE_NODE && !nodeIDs[edge.TargetId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}

		if edge.TargetType == componentpb.Edge_REF_TYPE_OUTPUT_BRANCH && !hasOutputBranch(blueprint.OutputBranches, edge.TargetId) {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target output branch %s not found", i, edge.TargetId)
		}
	}

	if err := validateAcyclic(blueprint.Nodes, blueprint.Edges); err != nil {
		return nil, nil, err
	}

	return actions.ProtoToNodes(blueprint.Nodes), actions.ProtoToEdges(blueprint.Edges), nil
}

func hasOutputBranch(outputBranches []*componentpb.OutputBranch, name string) bool {
	for _, outputBranch := range outputBranches {
		if outputBranch.Name == name {
			return true
		}
	}
	return false
}

func validateNodeRef(registry *registry.Registry, node *componentpb.Node) error {
	switch node.RefType {
	case componentpb.Node_REF_TYPE_COMPONENT:
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
		// Validate required fields
		if field.Name == "" {
			return nil, status.Errorf(codes.InvalidArgument, "configuration field %d: name is required", i)
		}
		if field.Label == "" {
			return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: label is required", field.Name)
		}

		// Type-specific validation
		switch field.Type {
		case components.FieldTypeNumber:
			if field.Min == nil {
				return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: min is required for number type", field.Name)
			}
			if field.Max == nil {
				return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: max is required for number type", field.Name)
			}
		case components.FieldTypeSelect, components.FieldTypeMultiSelect:
			if len(field.Options) == 0 {
				return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: options are required for %s type", field.Name, field.Type)
			}
		}

		// If field is not required, default value should be provided
		if !field.Required && field.DefaultValue == nil {
			return nil, status.Errorf(codes.InvalidArgument, "configuration field %s: default value is required when field is not required", field.Name)
		}

		result[i] = components.ConfigurationField{
			Name:        field.Name,
			Label:       field.Label,
			Type:        field.Type,
			Description: field.Description,
			Required:    field.Required,
		}

		// Handle default value
		if field.DefaultValue != nil {
			result[i].Default = *field.DefaultValue
		}

		// Handle options (for select/multi_select)
		if len(field.Options) > 0 {
			result[i].Options = make([]components.FieldOption, len(field.Options))
			for j, opt := range field.Options {
				result[i].Options[j] = components.FieldOption{
					Label: opt.Label,
					Value: opt.Value,
				}
			}
		}

		// Handle min/max (for number type)
		if field.Min != nil {
			min := int(*field.Min)
			result[i].Min = &min
		}
		if field.Max != nil {
			max := int(*field.Max)
			result[i].Max = &max
		}

		// Handle list item definition (for list type)
		if field.ListItem != nil {
			result[i].ListItem = &components.ListItemDefinition{
				Type: field.ListItem.Type,
			}
			if len(field.ListItem.Schema) > 0 {
				listItemSchema := make([]components.ConfigurationField, len(field.ListItem.Schema))
				for j, schemaField := range field.ListItem.Schema {
					listItemSchema[j] = components.ConfigurationField{
						Name:        schemaField.Name,
						Label:       schemaField.Label,
						Type:        schemaField.Type,
						Description: schemaField.Description,
						Required:    schemaField.Required,
					}
				}
				result[i].ListItem.Schema = listItemSchema
			}
		}

		// Handle object schema (for object type)
		if len(field.Schema) > 0 {
			objectSchema := make([]components.ConfigurationField, len(field.Schema))
			for j, schemaField := range field.Schema {
				objectSchema[j] = components.ConfigurationField{
					Name:        schemaField.Name,
					Label:       schemaField.Label,
					Type:        schemaField.Type,
					Description: schemaField.Description,
					Required:    schemaField.Required,
				}
			}
			result[i].Schema = objectSchema
		}
	}
	return result, nil
}
