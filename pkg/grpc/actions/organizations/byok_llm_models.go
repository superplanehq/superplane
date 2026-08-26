package organizations

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ListBYOKLLMModels(
	ctx context.Context,
	reg *registry.Registry,
	orgID string,
	req *pb.ListBYOKLLMModelsRequest,
) (*pb.ListBYOKLLMModelsResponse, error) {
	tx := database.DB(ctx)
	scope, err := parseLLMModelListScope(tx, orgID, req.GetProvider(), req.GetFactoryId(), "failed to list byok models")
	if err != nil {
		return nil, err
	}

	selected, err := models.ResolveSelectableLLMModels(
		tx,
		scope.OrganizationID,
		scope.FactoryID,
		scope.Provider,
		models.UsageFundingSourceBYOK,
	)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list byok models")
	}

	integration, err := models.FindReadyBYOKIntegration(tx, scope.OrganizationID, scope.Provider)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list byok models")
	}

	resp := &pb.ListBYOKLLMModelsResponse{
		Selected: serializeHostedLLMModels(selected),
	}
	if integration == nil {
		return resp, nil
	}

	resp.Connected = true
	resp.IntegrationId = integration.ID.String()
	candidates, err := listBYOKCandidateModels(tx, reg, integration)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list byok models")
	}
	resp.Candidates = candidates
	resp.Selected = namedHostedLLMModels(selected, candidates)
	return resp, nil
}

func UpdateBYOKLLMModels(
	ctx context.Context,
	orgID string,
	req *pb.UpdateBYOKLLMModelsRequest,
) (*pb.UpdateBYOKLLMModelsResponse, error) {
	organizationID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	saved, err := models.UpsertOrganizationBYOKModelAllowlist(
		database.DB(ctx),
		organizationID,
		req.GetProvider(),
		datatypes.JSONSlice[string](req.GetAllowedModels()),
	)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "empty") {
			return nil, grpcerrors.InvalidArgument(err, err.Error())
		}
		return nil, grpcerrors.Internal(err, "failed to update byok models")
	}

	return &pb.UpdateBYOKLLMModelsResponse{
		Selected: serializeHostedLLMModels(saved.AllowedModels),
	}, nil
}

func listBYOKCandidateModels(tx *gorm.DB, reg *registry.Registry, instance *models.Integration) ([]*pb.HostedLLMModel, error) {
	if reg == nil {
		return nil, fmt.Errorf("integration registry is required")
	}
	integration, err := reg.GetIntegration(instance.AppName)
	if err != nil {
		return nil, err
	}

	integrationCtx := contexts.NewIntegrationContext(
		tx,
		nil,
		instance,
		reg.Encryptor,
		reg,
		nil,
	)
	resources, err := integration.ListResources("model", core.ListResourcesContext{
		Logger: log.WithFields(log.Fields{
			"integration_id":   instance.ID.String(),
			"integration_name": instance.AppName,
			"resource_type":    "model",
		}),
		HTTP:        reg.HTTPContext(),
		Integration: integrationCtx,
		Parameters:  map[string]string{"type": "model"},
	})
	if err != nil {
		return nil, err
	}

	out := make([]*pb.HostedLLMModel, 0, len(resources))
	seen := map[string]struct{}{}
	for _, resource := range resources {
		id := strings.TrimSpace(resource.ID)
		if id == "" {
			id = strings.TrimSpace(resource.Name)
		}
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(resource.Name)
		if name == "" {
			name = id
		}
		out = append(out, &pb.HostedLLMModel{Id: id, Name: name})
	}
	return out, nil
}

func namedHostedLLMModels(ids []string, candidates []*pb.HostedLLMModel) []*pb.HostedLLMModel {
	names := map[string]string{}
	for _, candidate := range candidates {
		names[candidate.GetId()] = candidate.GetName()
	}
	out := make([]*pb.HostedLLMModel, 0, len(ids))
	for _, id := range ids {
		name := names[id]
		if name == "" {
			name = id
		}
		out = append(out, &pb.HostedLLMModel{Id: id, Name: name})
	}
	return out
}
