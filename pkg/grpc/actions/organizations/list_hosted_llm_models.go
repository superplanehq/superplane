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
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	provider, err := models.NormalizeHostedLLMProvider(req.GetProvider())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
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

	modelsOut := make([]*pb.HostedLLMModel, 0, len(row.AllowedModels))
	for _, id := range row.AllowedModels {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		modelsOut = append(modelsOut, &pb.HostedLLMModel{Id: id, Name: id})
	}

	return &pb.ListHostedLLMModelsResponse{
		Enabled: true,
		Models:  modelsOut,
	}, nil
}
