package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SubmitIntegrationSetupStep(ctx context.Context, registry *registry.Registry, orgID, id, stepName string, inputs any) (*pb.SubmitIntegrationSetupStepResponse, error) {
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

	// TODO: call OnSetupStepSubmit()

	proto, err := serializeIntegrationV2(integration)
	if err != nil {
		return nil, err
	}

	return &pb.SubmitIntegrationSetupStepResponse{
		Integration: proto,
	}, nil
}
