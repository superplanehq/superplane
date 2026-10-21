package factories

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/yaml"
)

// This is the rules line written by the risk-category settings form.
var mergeRiskRulesLine = regexp.MustCompile(`(?m)^(?:[^=\n]+? = [1-5] \((?:very_low|low|medium|high|critical)\)\.\s*)+$`)

func configureMergeConfidenceChecks(node *yaml.Node, raw any) error {
	if raw == nil {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("read saved merge confidence checks: %w", err)
	}
	var savedSteps []runner.AgentStep
	if err := json.Unmarshal(encoded, &savedSteps); err != nil {
		return fmt.Errorf("read saved merge confidence checks: %w", err)
	}
	ids, labels := runner.MergeConfidenceChecksFromSteps(savedSteps)
	generated, _ := node.Configuration["steps"].([]any)
	defaults := mergeCheckStepsByID(generated)
	custom := savedMergeCheckSteps(raw)
	selected := make([]any, 0, len(ids))
	for _, id := range ids {
		step := defaults[id]
		if step == nil {
			if slices.Contains(strings.Split(defaultMergeConfidenceChecks, ", "), id) {
				return fmt.Errorf("merge confidence check %q is no longer available; review the selected checks before resetting", id)
			}
			step = custom[id]
		}
		if step == nil {
			return fmt.Errorf("could not preserve merge confidence check %q", id)
		}
		if label, present := labels[id]; present {
			step["name"] = label
		}
		if id == "risk" {
			applySavedRiskRules(step, savedSteps)
		}
		selected = append(selected, step)
	}
	node.Configuration["steps"] = replaceMergeCheckSteps(generated, selected)
	return nil
}

func mergeCheckStepsByID(steps []any) map[string]map[string]any {
	checks := map[string]map[string]any{}
	for _, item := range steps {
		step := cloneMergeCheckStep(item)
		prompt, _ := step["prompt"].(string)
		if id := runner.MergeCheckID(prompt); id != "" {
			checks[id] = step
		}
	}
	return checks
}

func savedMergeCheckSteps(value any) map[string]map[string]any {
	steps, _ := value.([]any)
	return mergeCheckStepsByID(steps)
}

func replaceMergeCheckSteps(generated, selected []any) []any {
	result := make([]any, 0, len(generated)+len(selected))
	inserted := false
	for _, item := range generated {
		step, _ := item.(map[string]any)
		prompt, _ := step["prompt"].(string)
		if runner.MergeCheckID(prompt) != "" {
			if !inserted {
				result = append(result, selected...)
				inserted = true
			}
			continue
		}
		result = append(result, item)
	}
	if !inserted {
		result = append(result, selected...)
	}
	return result
}

func applySavedRiskRules(step map[string]any, saved []runner.AgentStep) {
	for _, previous := range saved {
		if previous.Prompt == nil {
			continue
		}
		prompt := *previous.Prompt
		rules := strings.TrimSpace(mergeRiskRulesLine.FindString(prompt))
		if rules == "" && !strings.Contains(prompt, "Apply these category rules.") {
			continue
		}
		current, _ := step["prompt"].(string)
		step["prompt"] = mergeRiskRulesLine.ReplaceAllStringFunc(current, func(string) string { return rules + "\n\n" })
		return
	}
}

func cloneMergeCheckStep(item any) map[string]any {
	step, _ := item.(map[string]any)
	return maps.Clone(step)
}
