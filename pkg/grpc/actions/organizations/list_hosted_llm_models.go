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
)

func ListHostedLLMModels(
	ctx context.Context,
	orgID string,
	req *pb.ListHostedLLMModelsRequest,
) (*pb.ListHostedLLMModelsResponse, error) {
	organizationID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	provider, err := models.NormalizeHostedLLMProvider(req.GetProvider())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}

	factoryID, err := parseOptionalFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid factory id")
	}
	if factoryID != nil {
		if _, err := models.FindFactory(database.DB(ctx), organizationID, *factoryID); err != nil {
			if errors.Is(err, models.ErrFactoryNotFound) {
				return nil, grpcerrors.NotFound(err, "factory not found")
			}
			return nil, grpcerrors.Internal(err, "failed to list hosted models")
		}
	}

	row, err := models.FindHostedLLMProvider(database.DB(ctx), provider)
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
		database.DB(ctx),
		organizationID,
		factoryID,
		provider,
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
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
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
