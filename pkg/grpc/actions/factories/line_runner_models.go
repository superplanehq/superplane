package factories

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
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

	options, usesHostedFunding, err := listLineRunnerModelOptions(db, orgID, factory.ID, req.GetLineName())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line runner models")
	}

	return &pb.ListFactoryLineRunnerModelsResponse{
		Models:            serializeLineRunnerModels(options),
		UsesHostedFunding: usesHostedFunding,
	}, nil
}

// lineRunnerModel is one model that a line runner can use, with the thinking
// levels that the runner can send to it.
type lineRunnerModel struct {
	ID             string
	ThinkingLevels []string
}

func listLineRunnerModels(tx *gorm.DB, orgID, factoryID uuid.UUID, lineName string) ([]string, bool, error) {
	options, usesHostedFunding, err := listLineRunnerModelOptions(tx, orgID, factoryID, lineName)
	if err != nil {
		return nil, false, err
	}
	var ids []string
	for _, option := range options {
		ids = append(ids, option.ID)
	}
	return ids, usesHostedFunding, nil
}

func listLineRunnerModelOptions(tx *gorm.DB, orgID, factoryID uuid.UUID, lineName string) ([]lineRunnerModel, bool, error) {
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

	usesHostedFunding := false
	var out []lineRunnerModel
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
			ids, err := lineRunnerModelsForNode(tx, orgID, factoryID, node)
			if err != nil {
				return nil, false, err
			}
			thinkingLevels, err := lineRunnerThinkingLevels(tx, orgID, factoryID, node)
			if err != nil {
				return nil, false, err
			}
			for _, id := range ids {
				out = append(out, lineRunnerModel{ID: id, ThinkingLevels: thinkingLevels(id)})
			}
		}
	}
	merged := mergeLineRunnerModels(out)
	sort.Slice(merged, func(i, j int) bool { return merged[i].ID < merged[j].ID })
	return merged, usesHostedFunding, nil
}

// mergeLineRunnerModels keeps one entry per model id in first-seen order.
// Two runners can offer the same model; the entry keeps the levels of both.
func mergeLineRunnerModels(options []lineRunnerModel) []lineRunnerModel {
	merged := map[string]int{}
	var out []lineRunnerModel
	for _, option := range options {
		index, seen := merged[option.ID]
		if !seen {
			merged[option.ID] = len(out)
			out = append(out, lineRunnerModel{ID: option.ID, ThinkingLevels: runner.SupportedThinkingLevels(option.ThinkingLevels)})
			continue
		}
		out[index].ThinkingLevels = runner.SupportedThinkingLevels(append(out[index].ThinkingLevels, option.ThinkingLevels...))
	}
	return out
}

// lineRunnerThinkingLevels returns a lookup of the thinking levels that a
// runner node can send to each of its models. A ChatGPT subscription reports
// its levels in the Codex model catalog. The other runners use the model
// catalog for the provider of their CLI.
func lineRunnerThinkingLevels(tx *gorm.DB, orgID, factoryID uuid.UUID, node models.Node) (func(string) []string, error) {
	credentials, _ := node.Configuration["credentials"].(map[string]any)
	if node.ComponentName() == runnerCodex && credentials["source"] == "subscription" {
		catalog, err := models.FactoryCodexSubscriptionCatalog(tx, orgID, factoryID)
		if err != nil {
			return nil, err
		}
		return codexSubscriptionThinkingLevels(catalog), nil
	}
	if node.ComponentName() == models.SuperPlaneRunnerComponent {
		return hostedModelThinkingLevels, nil
	}
	provider, ok := runnerNodeProvider(node)
	if !ok {
		return func(string) []string { return nil }, nil
	}
	return func(id string) []string { return runner.ModelThinkingLevels(provider, id) }, nil
}

func hostedModelThinkingLevels(key string) []string {
	parsed, err := models.ParseSelectableLLMModelKey(key)
	if err != nil {
		return nil
	}
	return runner.ModelThinkingLevels(parsed.Provider.ID, parsed.Model.ID)
}

func codexSubscriptionThinkingLevels(catalog []models.CodexModel) func(string) []string {
	levels := make(map[string][]string, len(catalog))
	for _, model := range catalog {
		id := strings.TrimSpace(model.Model)
		levels[id] = runner.SupportedThinkingLevels(append(levels[id], model.ReasoningEfforts()...))
	}
	return func(id string) []string { return levels[strings.TrimSpace(id)] }
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

func lineRunnerModelsForNode(tx *gorm.DB, orgID, factoryID uuid.UUID, node models.Node) ([]string, error) {
	credentials, _ := node.Configuration["credentials"].(map[string]any)
	if node.ComponentName() == runnerCodex && credentials["source"] == "subscription" {
		return models.FactoryCodexSubscriptionModels(tx, orgID, factoryID)
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
