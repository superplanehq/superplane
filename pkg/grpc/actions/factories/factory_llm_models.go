package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/datatypes"
)

func ListFactoryLLMModels(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryLLMModelsRequest,
) (*pb.ListFactoryLLMModelsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}
	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}
	if _, err := models.FindFactory(database.DB(ctx), orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}

	parent, err := models.ResolveSelectableLLMModels(database.DB(ctx), orgID, nil, req.GetProvider(), req.GetFundingSource())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}

	subset, err := models.FindFactoryLLMModelAllowlist(database.DB(ctx), factoryID, req.GetProvider(), req.GetFundingSource())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}

	inherit := subset == nil || len(compactFactoryModelIDs(subset.AllowedModels)) == 0
	selected := parent
	if !inherit {
		selected = compactFactoryModelIDs(subset.AllowedModels)
	}

	return &pb.ListFactoryLLMModelsResponse{
		Parent:        serializeFactoryLLMModels(parent),
		Selected:      serializeFactoryLLMModels(selected),
		InheritParent: inherit,
	}, nil
}

func UpdateFactoryLLMModels(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateFactoryLLMModelsRequest,
) (*pb.UpdateFactoryLLMModelsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory models")
	}
	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory models")
	}

	saved, err := models.UpsertFactoryLLMModelAllowlist(
		database.DB(ctx),
		orgID,
		factoryID,
		req.GetProvider(),
		req.GetFundingSource(),
		datatypes.JSONSlice[string](req.GetAllowedModels()),
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory models")
	}

	selected := compactFactoryModelIDs(saved.AllowedModels)
	return &pb.UpdateFactoryLLMModelsResponse{
		Selected:      serializeFactoryLLMModels(selected),
		InheritParent: len(selected) == 0,
	}, nil
}

func serializeFactoryLLMModels(ids []string) []*pb.FactoryLLMModel {
	out := make([]*pb.FactoryLLMModel, 0, len(ids))
	for _, id := range ids {
		out = append(out, &pb.FactoryLLMModel{Id: id, Name: id})
	}
	return out
}

func compactFactoryModelIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
