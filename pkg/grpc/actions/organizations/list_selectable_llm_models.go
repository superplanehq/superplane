package organizations

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListSelectableLLMModels(
	ctx context.Context,
	reg *registry.Registry,
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

	if factoryID != nil {
		f, err := models.FindFactory(tx, organization.ID, *factoryID)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to load workspace")
		}
		if f.OnboardingConfigValue().CodexSubscription {
			rawCaller, _ := authentication.GetUserIdFromMetadata(ctx)
			callerID, err := uuid.Parse(rawCaller)
			if err != nil {
				return nil, grpcerrors.Unauthenticated(err, "Sign in to view subscription models")
			}
			ownerID, err := f.ResolveSubscriptionCatalogOwner(tx, callerID, req.GetOrderId())
			if err != nil {
				return nil, grpcerrors.InvalidArgument(err, err.Error())
			}
			ids, err := models.FactoryCodexSubscriptionModels(tx, organization.ID, f.ID, ownerID)
			if err != nil {
				return nil, grpcerrors.Internal(err, "failed to load subscription models")
			}
			out := &pb.ListSelectableLLMModelsResponse{}
			for _, id := range ids {
				out.Models = append(out.Models, &pb.SelectableLLMModel{
					Source:   &pb.SelectableLLMNamedID{Id: "subscription", Name: "ChatGPT subscription"},
					Provider: &pb.SelectableLLMNamedID{Id: "openai", Name: "OpenAI"},
					Model:    &pb.SelectableLLMNamedID{Id: id, Name: id}, Key: id, Label: id,
				})
			}
			return out, nil
		}
	}

	byokAllowed, err := models.OrganizationBYOKAccess(tx, organization.ID, "")
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load model access")
	}
	if reg != nil && byokAllowed {
		enableAllConnectedBYOKModelsByDefault(tx, reg, organization.ID)
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
