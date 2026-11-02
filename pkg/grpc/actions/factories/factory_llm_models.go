package factories

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ListFactoryLLMModels(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryLLMModelsRequest,
) (*pb.ListFactoryLLMModelsResponse, error) {
	provider, source, err := factoryLLMPolicyScope(req.GetProvider(), req.GetFundingSource())
	if err != nil {
		return nil, err
	}
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}
	factory, err := findFactory(database.DB(ctx), orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}
	parent, names, err := factoryLLMPolicyParent(ctx, database.DB(ctx), factory, provider, source)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}

	subset, err := models.FindFactoryLLMModelAllowlist(database.DB(ctx), factory.ID, provider, source)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory models")
	}

	inherit := subset.InheritsParent()
	selected := parent
	var configured []string
	if subset != nil {
		configured = models.CompactModelIDs(subset.AllowedModels)
	}
	if !inherit {
		selected = models.IntersectModelIDs(parent, subset.AllowedModels)
	}

	return &pb.ListFactoryLLMModelsResponse{
		Parent:           serializeFactoryLLMModels(parent, names),
		Selected:         serializeFactoryLLMModels(selected, names),
		InheritParent:    inherit,
		ConfiguredModels: serializeFactoryLLMModels(configured, names),
	}, nil
}

func UpdateFactoryLLMModels(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateFactoryLLMModelsRequest,
) (*pb.UpdateFactoryLLMModelsResponse, error) {
	if req == nil {
		return nil, grpcerrors.InvalidArgument(nil, "request is required")
	}
	provider, source, err := factoryLLMPolicyScope(req.GetProvider(), req.GetFundingSource())
	if err != nil {
		return nil, err
	}
	if req.GetInheritParent() && len(req.GetAllowedModels()) > 0 {
		return nil, grpcerrors.InvalidArgument(models.ErrInheritedModelSelection, "Use all available models cannot include selected model IDs.")
	}
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory models")
	}
	var factoryID uuid.UUID
	var saved *models.FactoryLLMModelAllowlist
	var parent []string
	var names map[string]string
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := findFactory(tx, orgID, req.GetFactoryId())
		if err != nil {
			return err
		}
		factoryID = factory.ID
		parent, names, err = factoryLLMPolicyParent(ctx, tx, factory, provider, source)
		if err != nil {
			return err
		}
		if source == models.UsageFundingSourceSubscription {
			previous, err := models.FindFactoryLLMModelAllowlist(tx, factory.ID, provider, source)
			if err != nil {
				return err
			}
			if err := validateSubscriptionPolicyAdditions(req.GetAllowedModels(), parent, previous); err != nil {
				return err
			}
		}
		saved, err = models.UpsertFactoryLLMModelAllowlist(tx, orgID, factory.ID, provider, source, datatypes.JSONSlice[string](req.GetAllowedModels()), req.InheritParent)
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory models")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(factoryID.String(), "", "model_policy"); err != nil {
		log.WithError(err).Warn("Failed to publish workspace model policy change")
	}
	selected := parent
	if !saved.InheritsParent() {
		selected = models.IntersectModelIDs(parent, saved.AllowedModels)
	}
	return &pb.UpdateFactoryLLMModelsResponse{
		Selected:      serializeFactoryLLMModels(selected, names),
		InheritParent: saved.InheritsParent(),
	}, nil
}

func factoryLLMPolicyScope(provider, source string) (string, string, error) {
	normalized, err := models.NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return "", "", grpcerrors.InvalidArgument(err, "Select a supported model provider.")
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if !slices.Contains([]string{models.UsageFundingSourceHosted, models.UsageFundingSourceBYOK, models.UsageFundingSourceSubscription}, source) {
		return "", "", grpcerrors.InvalidArgument(nil, "Select a supported model source.")
	}
	if source == models.UsageFundingSourceSubscription && normalized != models.UsageProviderOpenAI {
		return "", "", grpcerrors.InvalidArgument(nil, "ChatGPT subscription model policies require provider openai.")
	}
	return normalized, source, nil
}

func factoryLLMPolicyParent(ctx context.Context, tx *gorm.DB, factory *models.Factory, provider, source string) ([]string, map[string]string, error) {
	if source == models.UsageFundingSourceSubscription {
		callerID, err := codexCallerID(ctx)
		if err != nil {
			return nil, nil, err
		}
		if _, err := models.ActiveCodexMember(tx, factory.OrganizationID, callerID); err != nil {
			return nil, nil, err
		}
		connections, err := models.ListEligibleCodexConnections(tx, factory.OrganizationID)
		if err != nil {
			return nil, nil, err
		}
		var available []string
		names := map[string]string{}
		for _, connection := range connections {
			for _, model := range connection.AvailableModels {
				available = append(available, model.Model)
				names[model.Model] = strings.TrimSpace(model.DisplayName)
			}
		}
		available = models.CompactModelIDs(available)
		return available, names, nil
	}
	if source == models.UsageFundingSourceBYOK {
		if err := models.RequireOrganizationBYOKAccess(tx, factory.OrganizationID, provider); err != nil {
			return nil, nil, err
		}
	}
	parent, err := models.ResolveSelectableLLMModels(tx, factory.OrganizationID, nil, provider, source)
	return parent, nil, err
}

func validateSubscriptionPolicyAdditions(requested, available []string, previous *models.FactoryLLMModelAllowlist) error {
	var configured []string
	if previous != nil {
		configured = models.CompactModelIDs(previous.AllowedModels)
	}
	for _, id := range models.CompactModelIDs(requested) {
		if slices.Contains(available, id) || slices.Contains(configured, id) {
			continue
		}
		return grpcerrors.InvalidArgument(nil, "The selected model is not available in your ChatGPT subscription: "+id)
	}
	return nil
}

func serializeFactoryLLMModels(ids []string, names map[string]string) []*pb.FactoryLLMModel {
	out := make([]*pb.FactoryLLMModel, 0, len(ids))
	for _, id := range models.CompactModelIDs(ids) {
		name := names[id]
		if name == "" {
			name = id
		}
		out = append(out, &pb.FactoryLLMModel{Id: id, Name: name})
	}
	return out
}

func serializeLineRunnerModels(ids []string) []*pb.FactoryLLMModel {
	out := make([]*pb.FactoryLLMModel, 0, len(ids))
	for _, id := range models.CompactModelIDs(ids) {
		name := id
		if parsed, err := models.ParseSelectableLLMModelKey(id); err == nil {
			name = parsed.Label
		}
		out = append(out, &pb.FactoryLLMModel{Id: id, Name: name})
	}
	return out
}
