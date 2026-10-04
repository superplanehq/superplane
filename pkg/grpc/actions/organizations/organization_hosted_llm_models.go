package organizations

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ListOrganizationHostedLLMModels(ctx context.Context, orgID string) (*pb.ListOrganizationHostedLLMModelsResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	tx := database.DB(ctx)
	candidates, err := installationHostedModelCandidates(tx)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list hosted models")
	}
	selected := make([]*pb.OrganizationHostedLLMModel, 0)
	for _, provider := range models.KnownHostedLLMProviders() {
		selection, err := models.FindOrganizationHostedModelAllowlist(tx, organizationID, provider)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to list hosted models")
		}
		for _, candidate := range candidates {
			if candidate.Provider != provider {
				continue
			}
			parsed, err := models.ParseSelectableLLMModelKey(candidate.Key)
			if err != nil {
				return nil, grpcerrors.Internal(err, "failed to list hosted models")
			}
			if selection == nil || slices.Contains(selection.AllowedModels, parsed.Model.ID) {
				selected = append(selected, candidate)
			}
		}
	}
	return &pb.ListOrganizationHostedLLMModelsResponse{Candidates: candidates, Selected: selected}, nil
}

func UpdateOrganizationHostedLLMModels(ctx context.Context, orgID string, req *pb.UpdateOrganizationHostedLLMModelsRequest) (*pb.UpdateOrganizationHostedLLMModelsResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	selected := make([]*pb.OrganizationHostedLLMModel, 0, len(req.GetAllowedModels()))
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		candidates, err := installationHostedModelCandidates(tx)
		if err != nil {
			return grpcerrors.Internal(err, "failed to update hosted models")
		}
		byKey := make(map[string]*pb.OrganizationHostedLLMModel, len(candidates))
		for _, candidate := range candidates {
			byKey[candidate.Key] = candidate
		}
		byProvider := make(map[string]datatypes.JSONSlice[string])
		seen := make(map[string]bool)
		for _, key := range req.GetAllowedModels() {
			candidate, ok := byKey[key]
			if !ok || seen[key] {
				err := fmt.Errorf("model is not an installation candidate or is duplicated: %s", key)
				return grpcerrors.InvalidArgument(err, "Select each model once from the installation model list.")
			}
			seen[key] = true
			parsed, err := models.ParseSelectableLLMModelKey(key)
			if err != nil {
				return grpcerrors.Internal(err, "failed to update hosted models")
			}
			byProvider[candidate.Provider] = append(byProvider[candidate.Provider], parsed.Model.ID)
			selected = append(selected, candidate)
		}
		for _, provider := range models.KnownHostedLLMProviders() {
			if _, err := models.UpsertOrganizationHostedModelAllowlist(tx, organizationID, provider, byProvider[provider]); err != nil {
				return grpcerrors.Internal(err, "failed to update hosted models")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateOrganizationHostedLLMModelsResponse{Selected: selected}, nil
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
