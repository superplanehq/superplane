package openrouter

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const fallbackModelsFileName = "openrouter_models.json"

// OrderedFallbackModels puts the selected model first, then other allowlisted
// ids in the given order. Duplicates and blank ids are dropped.
func OrderedFallbackModels(selected string, allowed []string) []string {
	selected = catalogModelID(selected)
	seen := make(map[string]struct{}, len(allowed)+1)
	out := make([]string, 0, len(allowed)+1)
	if selected != "" {
		out = append(out, selected)
		seen[selected] = struct{}{}
	}
	for _, id := range allowed {
		id = catalogModelID(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// FallbackModelsFile is the JSON list run.js reads when OpenRouter rate-limits.
func FallbackModelsFile(selected string, allowed []string) runner.BrokerTaskFile {
	models := OrderedFallbackModels(selected, allowed)
	raw, err := json.Marshal(models)
	if err != nil {
		raw = []byte("[]")
	}
	return runner.BrokerTaskFile{
		Path:    fallbackModelsFileName,
		Content: string(raw) + "\n",
		Mode:    "0644",
	}
}

func catalogModelID(model string) string {
	trimmed := strings.TrimSpace(model)
	return strings.TrimPrefix(trimmed, "openrouter/")
}

func byokOpenRouterFallbackModels(ctx core.ExecutionContext, selected string) []string {
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return OrderedFallbackModels(selected, nil)
	}
	ids, err := models.ResolveSelectableLLMModels(
		database.DB(context.Background()),
		orgID,
		nil,
		models.UsageProviderOpenRouter,
		models.UsageFundingSourceBYOK,
	)
	if err != nil {
		return OrderedFallbackModels(selected, nil)
	}
	return OrderedFallbackModels(selected, ids)
}
