package organizations

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func ListSelectableLLMModels(
	ctx context.Context,
	orgID string,
	req *pb.ListSelectableLLMModelsRequest,
) (*pb.ListSelectableLLMModelsResponse, error) {
	tx := database.DB(ctx)
	organization, err := models.FindOrganizationByIDOrSlug(tx, orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}
	factoryID, err := parseOptionalFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid factory id")
	}
	if factoryID != nil {
		if _, err := models.FindFactory(tx, organization.ID, *factoryID); err != nil {
			if errors.Is(err, models.ErrFactoryNotFound) {
				return nil, grpcerrors.NotFound(err, "factory not found")
			}
			return nil, grpcerrors.Internal(err, "failed to list selectable models")
		}
	}

	listed, err := models.ListSelectableLLMModels(tx, organization.ID, factoryID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list selectable models")
	}
	return &pb.ListSelectableLLMModelsResponse{Models: serializeSelectableLLMModels(listed)}, nil
}

func serializeSelectableLLMModels(listed []models.SelectableLLMModel) []*pb.SelectableLLMModel {
	out := make([]*pb.SelectableLLMModel, 0, len(listed))
	for _, model := range listed {
		out = append(out, &pb.SelectableLLMModel{
			Source:   &pb.SelectableLLMNamedID{Id: model.Source.ID, Name: model.Source.Name},
			Provider: &pb.SelectableLLMNamedID{Id: model.Provider.ID, Name: model.Provider.Name},
			Model:    &pb.SelectableLLMNamedID{Id: model.Model.ID, Name: model.Model.Name},
			Key:      model.Key,
			Label:    model.Label,
		})
	}
	return out
}
