package organizations

import (
	"context"
	"sort"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func ListOrganizationHostedLLMModels(ctx context.Context, orgID string) (*pb.ListOrganizationHostedLLMModelsResponse, error) {
	if _, err := resolveOrganizationID(ctx, orgID); err != nil {
		return nil, err
	}
	tx := database.DB(ctx)
	candidates, err := installationHostedModelCandidates(tx)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list hosted models")
	}
	return &pb.ListOrganizationHostedLLMModelsResponse{Candidates: candidates, Selected: candidates}, nil
}

func installationHostedModelCandidates(tx *gorm.DB) ([]*pb.OrganizationHostedLLMModel, error) {
	providers, err := models.ListHostedLLMProviders(tx)
	if err != nil {
		return nil, err
	}
	candidates := make([]*pb.OrganizationHostedLLMModel, 0)
	for _, provider := range providers {
		if !provider.OffersHostedModels() {
			continue
		}
		for _, model := range models.CompactModelIDs(provider.AllowedModels) {
			candidates = append(candidates, &pb.OrganizationHostedLLMModel{
				Key:      models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, provider.Provider, model),
				Label:    models.HostedLLMTechnicalName(provider.Provider, model),
				Provider: provider.Provider,
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Label != candidates[j].Label {
			return candidates[i].Label < candidates[j].Label
		}
		return candidates[i].Key < candidates[j].Key
	})
	return candidates, nil
}
