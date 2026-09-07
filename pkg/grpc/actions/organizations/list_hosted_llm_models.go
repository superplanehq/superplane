package organizations

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

type llmModelListScope struct {
	OrganizationID uuid.UUID
	Provider       string
	FactoryID      *uuid.UUID
}

func parseLLMModelListScope(tx *gorm.DB, orgID, provider, factoryID, internalMessage string) (llmModelListScope, error) {
	organization, err := models.FindOrganizationByIDOrSlug(tx, orgID)
	if err != nil {
		return llmModelListScope{}, grpcerrors.InvalidArgument(err, "invalid organization id")
	}
	organizationID := organization.ID
	normalized, err := models.NormalizeHostedLLMProvider(provider)
	if err != nil {
		return llmModelListScope{}, grpcerrors.InvalidArgument(err, err.Error())
	}
	parsedFactoryID, err := parseOptionalFactoryID(factoryID)
	if err != nil {
		return llmModelListScope{}, grpcerrors.InvalidArgument(err, "invalid factory id")
	}
	if parsedFactoryID != nil {
		if _, err := models.FindFactory(tx, organizationID, *parsedFactoryID); err != nil {
			if errors.Is(err, models.ErrFactoryNotFound) {
				return llmModelListScope{}, grpcerrors.NotFound(err, "factory not found")
			}
			return llmModelListScope{}, grpcerrors.Internal(err, internalMessage)
		}
	}
	return llmModelListScope{
		OrganizationID: organizationID,
		Provider:       normalized,
		FactoryID:      parsedFactoryID,
	}, nil
}

func ListHostedLLMModels(
	ctx context.Context,
	orgID string,
	req *pb.ListHostedLLMModelsRequest,
) (*pb.ListHostedLLMModelsResponse, error) {
	tx := database.DB(ctx)
	scope, err := parseLLMModelListScope(tx, orgID, req.GetProvider(), req.GetFactoryId(), "failed to list hosted models")
	if err != nil {
		return nil, err
	}

	row, err := models.FindHostedLLMProvider(tx, scope.Provider)
	if err != nil {
		if errors.Is(err, models.ErrHostedLLMProviderNotFound) {
			return &pb.ListHostedLLMModelsResponse{}, nil
		}
		return nil, grpcerrors.Internal(err, "failed to list hosted models")
	}
	if !row.OffersHostedModels() {
		return &pb.ListHostedLLMModelsResponse{}, nil
	}

	allowed, err := models.ResolveSelectableLLMModels(
		tx,
		scope.OrganizationID,
		scope.FactoryID,
		scope.Provider,
		models.UsageFundingSourceHosted,
	)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list hosted models")
	}

	return &pb.ListHostedLLMModelsResponse{
		Enabled: true,
		Models:  serializeHostedLLMModels(allowed),
	}, nil
}

func serializeHostedLLMModels(ids []string) []*pb.HostedLLMModel {
	out := make([]*pb.HostedLLMModel, 0, len(ids))
	for _, id := range models.CompactModelIDs(ids) {
		out = append(out, &pb.HostedLLMModel{Id: id, Name: id})
	}
	return out
}

func parseOptionalFactoryID(value string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	id, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
