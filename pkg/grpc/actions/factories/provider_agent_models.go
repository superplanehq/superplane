package factories

import (
	"slices"
	"strings"
)

// providerAgentModelSpec matches the onboarding agent plan. The run model
// prefers a known id from the key's model list. The planning model prefers
// an id that contains the hint. An empty list keeps the alias the agent CLI
// resolves itself.
type providerAgentModelSpec struct {
	defaultModel         string
	defaultPlanningModel string
	planningHint         string
	preferred            []string
}

var providerAgentModelSpecs = map[string]providerAgentModelSpec{
	modelSourceAnthropic: {
		defaultModel:         "sonnet",
		defaultPlanningModel: "opus",
		planningHint:         "opus",
		preferred:            []string{"opus-5-5", "opus-5.5", "sonnet"},
	},
	modelSourceOpenAI: {
		defaultModel:         "gpt-5",
		defaultPlanningModel: "gpt-5",
		planningHint:         "gpt-5",
		preferred:            []string{"gpt-5"},
	},
	modelSourceOpenRouter: {
		defaultModel:         "anthropic/claude-sonnet-4-6",
		defaultPlanningModel: "anthropic/claude-opus-4-6",
		planningHint:         "opus",
		preferred:            []string{"grok-4.7", "sonnet"},
	},
}

func agentModelsForSource(source string, modelIDs []string) (string, string) {
	spec, ok := providerAgentModelSpecs[source]
	if !ok {
		return "", ""
	}

	ids := uniqueSortedModelIDs(modelIDs)
	model := pickPreferredModel(spec.preferred, ids)
	if model == "" {
		model = spec.defaultModel
	}
	return model, planningModelFor(spec, ids, model)
}

func planningModelFor(spec providerAgentModelSpec, modelIDs []string, model string) string {
	if len(modelIDs) == 0 {
		return spec.defaultPlanningModel
	}
	if match := pickModelContaining(modelIDs, spec.planningHint); match != "" {
		return match
	}
	return model
}

func pickPreferredModel(preferred []string, modelIDs []string) string {
	for _, hint := range preferred {
		if match := pickModelContaining(modelIDs, hint); match != "" {
			return match
		}
	}
	if len(modelIDs) == 0 {
		return ""
	}
	return modelIDs[0]
}

func pickModelContaining(modelIDs []string, hint string) string {
	needle := strings.ToLower(strings.TrimSpace(hint))
	if needle == "" {
		return ""
	}
	for _, id := range modelIDs {
		if strings.Contains(strings.ToLower(id), needle) {
			return id
		}
	}
	return ""
}

func uniqueSortedModelIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	// Numeric runs keep "opus-4" before "opus-10", matching onboarding order.
	slices.SortFunc(out, compareModelIDs)
	return out
}

func compareModelIDs(left, right string) int {
	a := strings.ToLower(left)
	b := strings.ToLower(right)
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		aDigit := isDigit(a[i])
		bDigit := isDigit(b[j])
		if aDigit && bDigit {
			iStart, jStart := i, j
			for i < len(a) && isDigit(a[i]) {
				i++
			}
			for j < len(b) && isDigit(b[j]) {
				j++
			}
			if cmp := compareNumericRuns(a[iStart:i], b[jStart:j]); cmp != 0 {
				return cmp
			}
			continue
		}
		if a[i] != b[j] {
			if a[i] < b[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case len(a)-i == len(b)-j:
		return 0
	case i == len(a):
		return -1
	default:
		return 1
	}
}

func compareNumericRuns(left, right string) int {
	left = strings.TrimLeft(left, "0")
	right = strings.TrimLeft(right, "0")
	if left == "" {
		left = "0"
	}
	if right == "" {
		right = "0"
	}
	if len(left) != len(right) {
		if len(left) < len(right) {
			return -1
		}
		return 1
	}
	return strings.Compare(left, right)
}

func isDigit(value byte) bool {
	return value >= '0' && value <= '9'
}
