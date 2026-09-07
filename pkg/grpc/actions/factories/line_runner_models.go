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
	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line runner models")
	}

	ids, err := listLineRunnerModels(database.DB(ctx), orgID, factoryID, req.GetLineName())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list line runner models")
	}

	return &pb.ListFactoryLineRunnerModelsResponse{
		Models: serializeFactoryLLMModels(ids),
	}, nil
}

func listLineRunnerModels(tx *gorm.DB, orgID, factoryID uuid.UUID, lineName string) ([]string, error) {
	lineName = strings.TrimSpace(lineName)
	if lineName == "" {
		return nil, invalidArgument("line_name is required")
	}

	factory, err := models.FindFactory(tx, orgID, factoryID)
	if err != nil {
		return nil, err
	}
	line, err := factory.FindLineByName(tx, lineName)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var out []string
	for _, step := range line.Steps {
		if step.AppID == uuid.Nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionInTransaction(tx, step.AppID)
		if err != nil {
			return nil, err
		}
		for _, node := range []models.Node(version.Nodes) {
			provider, ok := runnerComponentProvider(node.ComponentName())
			if !ok {
				continue
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
			if stored := storedRunnerModel(node.Configuration); stored != "" {
				ids = append(ids, stored)
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
	return out, nil
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
	case "secret", "integration":
		return models.UsageFundingSourceBYOK
	default:
		return models.UsageFundingSourceHosted
	}
}
