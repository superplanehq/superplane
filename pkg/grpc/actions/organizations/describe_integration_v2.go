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

func DescribeIntegrationV2(ctx context.Context, registry *registry.Registry, orgID, id string) (*pb.DescribeIntegrationV2Response, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	integrationID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid integration ID")
	}

	integration, err := models.FindIntegrationV2(org, integrationID)
	if err != nil {
		return nil, err
	}

	proto, err := serializeIntegrationV2(integration)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeIntegrationV2Response{
		Integration: proto,
	}, nil
}

func serializeIntegrationV2(integration *models.IntegrationV2) (*pb.IntegrationV2, error) {
	proto := &pb.IntegrationV2{
		Metadata: &pb.IntegrationV2_Metadata{
			Id:              integration.ID.String(),
			Name:            integration.Name,
			IntegrationName: integration.IntegrationName,
		},
		Status: &pb.IntegrationV2_Status{
			Parameters: serializeParameters(integration.Parameters),
			// Capabilities: serializeCapabilities(integration.Capabilities),
			// NextStep:     serializeNextStep(integration.NextStep),
			// Secrets:      serializeSecrets(integration.Secrets),
		},
	}

	return proto, nil
}

func serializeParameters(parameters []models.IntegrationV2Parameter) []*pb.IntegrationV2_Parameter {
	proto := make([]*pb.IntegrationV2_Parameter, len(parameters))
	for i, parameter := range parameters {
		proto[i] = &pb.IntegrationV2_Parameter{
			Name:        parameter.Name,
			Label:       parameter.Label,
			Description: parameter.Description,
		}
	}

	return proto
}
