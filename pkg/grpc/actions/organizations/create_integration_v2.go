package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateIntegrationV2(
	ctx context.Context,
	registry *registry.Registry,
	orgID string,
	integrationName, name string,
) (*pb.CreateIntegrationV2Response, error) {
	_, err := registry.GetIntegration(integrationName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "integration %s not found", integrationName)
	}

	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization")
	}

	//
	// Check if an integration with this name already exists in the organization
	//
	_, err = models.FindIntegrationV2ByName(org, name)
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "an integration with the name %s already exists in this organization", name)
	}

	newIntegration, err := models.CreateIntegrationV2(org, integrationName, name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create integration")
	}

	proto, err := serializeIntegrationV2(newIntegration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize integration: %v", err)
	}

	return &pb.CreateIntegrationV2Response{
		Integration: proto,
	}, nil
}
