package factories

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/yaml"
)

// This is the rules line written by the risk-category settings form.
var mergeRiskRulesLine = regexp.MustCompile(`(?m)^(?:[^=\n]+? = [1-5] \((?:very_low|low|medium|high|critical)\)\.\s*)+$`)

// Legacy installations keep their body switch until the agent owns these settings.
func configureMergeConfidencePublication(canvas *yaml.Canvas) error {
	var agent, body map[string]any
	for _, node := range canvas.Spec.Nodes {
		switch node.ID {
		case "assess-risk":
			agent = node.Configuration
		case "publish-merge-confidence":
			body = node.Configuration
		}
	}
	publish := body["enabled"] != false
	if saved, present := agent["publishResults"]; present {
		publish = saved != false
	}
	bodyEnabled, commentEnabled := agent["resultsBody"] != false, agent["resultsComment"] == true
	if publish && !bodyEnabled && !commentEnabled {
		return fmt.Errorf("select at least one destination for merge confidence results")
	}
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		switch node.ID {
		case "start-assessment-check":
			node.Configuration["enabled"] = agent["publishPRCheck"] == true
			node.Configuration["activationGeneration"] = savedActivationGeneration(agent["prCheckActivationGeneration"])
		case "publish-merge-confidence":
			node.Configuration["enabled"] = publish && bodyEnabled
			node.Configuration["activationGeneration"] = savedActivationGeneration(agent["resultsBodyActivationGeneration"])
		case "comment-merge-confidence":
			node.Configuration["enabled"] = publish && commentEnabled
			node.Configuration["activationGeneration"] = savedActivationGeneration(agent["resultsCommentActivationGeneration"])
		}
	}
	return nil
}

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
	ids, labels := mergeconfidence.MergeConfidenceChecksFromSteps(savedSteps)
	generated, _ := node.Configuration["steps"].([]any)
	defaults := mergeCheckStepsByID(generated)
	custom := savedMergeCheckSteps(raw)
	selected := make([]any, 0, len(ids))
	for _, id := range ids {
		defaultID := id
		if id == "drift" {
			if slices.Contains(ids, "correctness") {
				continue
			}
			defaultID = "correctness"
		}
		step := defaults[defaultID]
		if step == nil {
			if slices.Contains(strings.Split(defaultMergeConfidenceChecks, ", "), defaultID) {
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
	if len(ids) == 0 {
		// Runners require a prompt even when the saved selection enables no checks.
		selected = []any{map[string]any{
			"name":   "No checks enabled",
			"type":   "prompt",
			"prompt": "Enabled checks: none.\nNo merge confidence checks are enabled. Do not inspect the repository or call the report_merge_check tool.",
		}}
	}
	node.Configuration["steps"] = replaceMergeCheckSteps(generated, selected)
	return nil
}

func mergeCheckStepsByID(steps []any) map[string]map[string]any {
	checks := map[string]map[string]any{}
	for _, item := range steps {
		step := cloneMergeCheckStep(item)
		prompt, _ := step["prompt"].(string)
		if id := mergeconfidence.MergeCheckID(prompt); id != "" {
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
		if mergeconfidence.MergeCheckID(prompt) != "" {
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
