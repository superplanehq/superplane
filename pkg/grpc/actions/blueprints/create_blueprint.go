package blueprints

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func CreateBlueprint(ctx context.Context, registry *registry.Registry, organizationID string, blueprint *pb.Blueprint) (*pb.CreateBlueprintResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	nodes, edges, err := ParseBlueprint(registry, blueprint)
	if err != nil {
		return nil, err
	}

	outputChannels, err := ParseOutputChannels(registry, blueprint.Nodes, blueprint.OutputChannels)
	if err != nil {
		return nil, err
	}

	err = ValidateNodeConfigurations(nodes, registry)
	if err != nil {
		return nil, err
	}

	configuration, err := ProtoToConfiguration(blueprint.Configuration)
	if err != nil {
		return nil, err
	}

	createdBy := uuid.MustParse(userID)

	orgID, _ := uuid.Parse(organizationID)
	now := time.Now()
	model := &models.Blueprint{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           blueprint.Name,
		Description:    blueprint.Description,
		Icon:           blueprint.Icon,
		Color:          blueprint.Color,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Nodes:          nodes,
		Edges:          edges,
		Configuration:  datatypes.NewJSONSlice(configuration),
		OutputChannels: datatypes.NewJSONSlice(outputChannels),
	}

	if err := database.Conn().Create(model).Error; err != nil {
		return nil, err
	}

	return &pb.CreateBlueprintResponse{
		Blueprint: SerializeBlueprint(model),
	}, nil
}

func ParseOutputChannels(registry *registry.Registry, nodes []*componentpb.Node, outputChannels []*pb.OutputChannel) ([]models.BlueprintOutputChannel, error) {
	channels := []models.BlueprintOutputChannel{}
	for _, outputChannel := range outputChannels {
		if outputChannel.Name == "" {
			return nil, fmt.Errorf("output channel name is required")
		}

		if outputChannel.NodeId == "" {
			return nil, fmt.Errorf("output channel node id is required")
		}

		if outputChannel.NodeOutputChannel == "" {
			return nil, fmt.Errorf("output channel node output channel is required")
		}

		err := validateOutputChannelReference(registry, nodes, outputChannel)
		if err != nil {
			return nil, err
		}

		channels = append(channels, models.BlueprintOutputChannel{
			Name:              outputChannel.Name,
			NodeID:            outputChannel.NodeId,
			NodeOutputChannel: outputChannel.NodeOutputChannel,
		})
	}

	return channels, nil
}

func validateOutputChannelReference(registry *registry.Registry, nodes []*componentpb.Node, outputChannel *pb.OutputChannel) error {
	//
	// Check if the node referenced exists
	//
	node := findNode(nodes, outputChannel.NodeId)
	if node == nil {
		return fmt.Errorf("output channel %s references a node that does not exist: %s", outputChannel.Name, outputChannel.NodeId)
	}

	//
	// Check if the node has the output channel referenced
	//
	component, err := registry.GetComponent(node.Component.Name)
	if err != nil {
		return err
	}

	for _, c := range component.OutputChannels(nil) {
		if c.Name == outputChannel.NodeOutputChannel {
			return nil
		}
	}

	return fmt.Errorf("output channel %s references an output channel that does not exist on node %s", outputChannel.NodeOutputChannel, outputChannel.NodeId)
}

func findNode(nodes []*componentpb.Node, id string) *componentpb.Node {
	for _, node := range nodes {
		if node.Id == id {
			return node
		}
	}

	return nil
}
