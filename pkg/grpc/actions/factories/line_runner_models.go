package factories

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

const (
	runnerClaudeCode = "runnerClaudeCode"
	runnerCodex      = "runnerCodex"
	runnerOpenRouter = "runnerOpenRouter"
)

func ListFactoryLineRunnerModels(
	ctx context.Context,
	organizationID string,
	req *pb.ListFactoryLineRunnerModelsRequest,
) (*pb.ListFactoryLineRunnerModelsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line runner models")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line runner models")
	}

	ownerID, err := resolveModelCatalogOwner(ctx, db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve model owner")
	}
	ids, usesHostedFunding, err := listLineRunnerModels(db, orgID, factory.ID, req.GetLineName(), ownerID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line runner models")
	}

	return &pb.ListFactoryLineRunnerModelsResponse{
		Models:            serializeLineRunnerModels(ids),
		UsesHostedFunding: usesHostedFunding,
	}, nil
}

func listLineRunnerModels(tx *gorm.DB, orgID, factoryID uuid.UUID, lineName string, ownerIDs ...uuid.UUID) ([]string, bool, error) {
	lineName = strings.TrimSpace(lineName)
	if lineName == "" {
		return nil, false, invalidArgument("line_name is required")
	}

	factory, err := models.FindFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, false, err
	}
	line, err := factory.FindLineByName(tx, lineName)
	if err != nil {
		return nil, false, err
	}

	seen := map[string]struct{}{}
	usesHostedFunding := false
	var out []string
	for _, step := range line.Steps {
		if step.AppID == uuid.Nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionInTransaction(tx, step.AppID)
		if err != nil {
			return nil, false, err
		}
		for _, node := range []models.Node(version.Nodes) {
			if lineRunnerNodeUsesHostedFunding(node) {
				usesHostedFunding = true
			}
			ids, err := lineRunnerModelsForNode(tx, orgID, factoryID, node, ownerIDs...)
			if err != nil {
				return nil, false, err
			}
			for _, id := range ids {
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
				out = append(out, id)
			}
		}
	}
	sort.Strings(out)
	return out, usesHostedFunding, nil
}

// lineRunnerNodeUsesHostedFunding reports whether a runner node spends
// SuperPlane-hosted funds. The SuperPlane runner is always hosted; the other
// runners follow their credentials source.
func lineRunnerNodeUsesHostedFunding(node models.Node) bool {
	if node.ComponentName() == models.SuperPlaneRunnerComponent {
		return true
	}
	if _, ok := runnerNodeProvider(node); !ok {
		return false
	}
	return runnerFundingSource(node.Configuration) == models.UsageFundingSourceHosted
}

func lineRunnerModelsForNode(tx *gorm.DB, orgID, factoryID uuid.UUID, node models.Node, ownerIDs ...uuid.UUID) ([]string, error) {
	credentials, _ := node.Configuration["credentials"].(map[string]any)
	if node.ComponentName() == runnerCodex && credentials["source"] == "subscription" {
		if len(ownerIDs) != 1 {
			return nil, nil
		}
		return models.FactoryCodexSubscriptionModels(tx, orgID, factoryID, ownerIDs[0])
	}
	if node.ComponentName() == models.SuperPlaneRunnerComponent {
		return hostedSelectableLineRunnerModels(tx, orgID, factoryID, storedRunnerModel(node.Configuration))
	}

	provider, ok := runnerNodeProvider(node)
	if !ok {
		return nil, nil
	}
	ids, err := models.ResolveSelectableLLMModels(
		tx,
		orgID,
		&factoryID,
		provider,
		runnerFundingSource(node.Configuration),
	)
	if err != nil {
		return nil, err
	}
	stored := storedRunnerModel(node.Configuration)
	if stored != "" {
		include, err := storedRunnerModelIsAvailable(tx, orgID, factoryID, node)
		if err != nil {
			return nil, err
		}
		if include {
			ids = append(ids, stored)
		}
	}
	return concreteLineRunnerModels(ids), nil
}

// storedRunnerModelIsAvailable applies the execution gate to stored BYOK
// models. Hosted models stay available.
func storedRunnerModelIsAvailable(tx *gorm.DB, orgID, factoryID uuid.UUID, node models.Node) (bool, error) {
	if runnerFundingSource(node.Configuration) == models.UsageFundingSourceHosted {
		return true, nil
	}
	return models.RunnerAcceptsModel(tx, orgID, &factoryID, node.ComponentName(), node.Configuration, storedRunnerModel(node.Configuration))
}

func hostedSelectableLineRunnerModels(tx *gorm.DB, orgID, factoryID uuid.UUID, stored string) ([]string, error) {
	selectable, err := models.ListSelectableLLMModels(tx, orgID, &factoryID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(selectable)+1)
	for _, model := range selectable {
		if model.Source.ID != models.UsageFundingSourceHosted {
			continue
		}
		ids = append(ids, model.Key)
	}
	if stored != "" {
		ids = append(ids, stored)
	}
	return concreteLineRunnerModels(ids), nil
}

func concreteLineRunnerModels(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		resolved := models.ConcreteClaudeModelID(id, ids)
		if _, dup := seen[resolved]; dup {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	return out
}

func runnerNodeProvider(node models.Node) (string, bool) {
	if provider, _ := node.Configuration["llmProvider"].(string); strings.TrimSpace(provider) == models.UsageProviderCustom {
		return models.UsageProviderCustom, true
	}
	return runnerComponentProvider(node.ComponentName())
}

func runnerComponentProvider(component string) (string, bool) {
	switch component {
	case runnerClaudeCode:
		return models.UsageProviderAnthropic, true
	case runnerCodex:
		return models.UsageProviderOpenAI, true
	case runnerOpenRouter:
		return models.UsageProviderOpenRouter, true
	default:
		return "", false
	}
}

func storedRunnerModel(configuration map[string]any) string {
	model, _ := configuration["model"].(string)
	return strings.TrimSpace(model)
}

func runnerFundingSource(configuration map[string]any) string {
	credentials, _ := configuration["credentials"].(map[string]any)
	source, _ := credentials["source"].(string)
	switch strings.TrimSpace(source) {
	case "secret", "integration", "subscription":
		return models.UsageFundingSourceBYOK
	default:
		return models.UsageFundingSourceHosted
	}
}
