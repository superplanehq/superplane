package openrouter

import (
	"context"
	"encoding/json"
	"hash/fnv"
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

// RotateFallbackModels moves the start of the list by a hash of seed so
// concurrent runs do not all call the same model first. An empty seed keeps
// the original order.
func RotateFallbackModels(models []string, seed string) []string {
	n := len(models)
	if n <= 1 || strings.TrimSpace(seed) == "" {
		return models
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	offset := int(h.Sum32() % uint32(n))
	if offset == 0 {
		return models
	}
	out := make([]string, n)
	copy(out, models[offset:])
	copy(out[n-offset:], models[:offset])
	return out
}

// FallbackModelList is the ordered OpenRouter model chain shipped to the runner.
func FallbackModelList(selected string, allowed []string, seed string) []string {
	return RotateFallbackModels(OrderedFallbackModels(selected, allowed), seed)
}

// FallbackRotateSeed spreads concurrent executions. Tests that omit IDs keep
// a stable selected-first order.
func FallbackRotateSeed(ctx core.ExecutionContext) string {
	if ctx.ID == uuid.Nil && ctx.RunID == uuid.Nil {
		return ""
	}
	return ctx.ID.String() + ":" + ctx.RunID.String()
}

// FallbackModelsFile is the JSON list run.js reads for OpenRouter model rotation.
func FallbackModelsFile(models []string) runner.BrokerTaskFile {
	if models == nil {
		models = []string{}
	}
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

func byokOpenRouterAllowedModels(ctx core.ExecutionContext) []string {
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return nil
	}
	ids, err := models.ResolveSelectableLLMModels(
		database.DB(context.Background()),
		orgID,
		nil,
		models.UsageProviderOpenRouter,
		models.UsageFundingSourceBYOK,
	)
	if err != nil {
		return nil
	}
	return ids
}
