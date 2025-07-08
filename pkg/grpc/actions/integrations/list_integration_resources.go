package integrations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListIntegrationResources(ctx context.Context, encryptor crypto.Encryptor, req *pb.ListIntegrationResourcesRequest) (*pb.ListIntegrationResourcesResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.IdOrName)
	var integration *models.Integration
	if err != nil {
		integration, err = models.FindIntegrationByName(authorization.DomainCanvas, canvas.ID, req.IdOrName)
	} else {
		integration, err = models.FindDomainIntegrationByID(authorization.DomainCanvas, canvas.ID, uuid.MustParse(req.IdOrName))
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "integration not found")
	}

	i, err := integrations.NewIntegration(ctx, integration, encryptor)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "integration not found")
	}

	resourceType, err := protoToResourceType(req.ResourceType)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid resource type")
	}

	resources, err := i.ListResources(resourceType)
	if err != nil {
		log.Errorf("error listing integration resources: %v", err)
		return nil, status.Error(codes.Internal, "error listing integration resources")
	}

	return &pb.ListIntegrationResourcesResponse{
		Resources: serializeIntegrationResources(resources),
	}, nil
}

func serializeIntegrationResources(resources []integrations.IntegrationResource) []*pb.IntegrationResource {
	out := []*pb.IntegrationResource{}
	for _, resource := range resources {
		out = append(out, &pb.IntegrationResource{
			Id:   resource.ID(),
			Name: resource.Name(),
			Type: resourceTypeToProto(resource.Type()),
		})
	}
	return out
}

func protoToResourceType(resourceType pb.IntegrationResource_Type) (string, error) {
	switch resourceType {
	case pb.IntegrationResource_TYPE_TASK:
		return integrations.ResourceTypeTask, nil
	case pb.IntegrationResource_TYPE_PROJECT:
		return integrations.ResourceTypeProject, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid resource type")
	}
}

func resourceTypeToProto(resourceType string) pb.IntegrationResource_Type {
	switch resourceType {
	case integrations.ResourceTypeTask:
		return pb.IntegrationResource_TYPE_TASK
	case integrations.ResourceTypeProject:
		return pb.IntegrationResource_TYPE_PROJECT
	default:
		return pb.IntegrationResource_TYPE_NONE
	}
}
