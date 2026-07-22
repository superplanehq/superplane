package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func ListIntegrationTools(ctx context.Context, reg *registry.Registry, orgID string, integrationID string) (*pb.ListIntegrationToolsResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid installation ID")
	}

	instance, err := models.FindIntegration(org, ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "integration not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load integration")
	}

	if instance.State != models.IntegrationStateReady {
		return &pb.ListIntegrationToolsResponse{
			Tools: []*pb.IntegrationTool{},
		}, nil
	}

	integration, err := reg.GetIntegration(instance.AppName)
	if err != nil {
		return nil, grpcerrors.FailedPrecondition(nil, fmt.Sprintf("integration %s is unavailable", instance.AppName))
	}

	tools := []*pb.IntegrationTool{}
	for _, action := range integration.Actions() {
		if _, ok := registry.AsIntegrationTool(action); !ok {
			continue
		}

		t := &pb.IntegrationTool{
			Name:        action.Name(),
			Label:       action.Label(),
			Description: action.Description(),
		}

		for _, configField := range action.Configuration() {
			t.Parameters = append(t.Parameters, actions.ConfigurationFieldToProto(configField))
		}

		tools = append(tools, t)
	}

	return &pb.ListIntegrationToolsResponse{
		Tools: tools,
	}, nil
}
