package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	// EnvSuperplaneMergeConfidenceOrderID is set on the merge confidence
	// agent node. The runner reads it, then removes it from the task
	// environment. The work order id stays in memory and is copied into
	// the runner token.
	EnvSuperplaneMergeConfidenceOrderID = "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID"

	// legacyMergeConfidenceToken is the old credential name. Strip it so a
	// canvas cannot pass that value into the agent.
	legacyMergeConfidenceToken = "SUPERPLANE_MERGE_CONFIDENCE_TOKEN"
)

// MergeConfidenceScope is the check list for one merge confidence agent run.
// EnabledChecks is the list from the agent prompt. The server reports a check
// only when its name is in this list.
type MergeConfidenceScope struct {
	OrganizationID  uuid.UUID
	FactoryID       uuid.UUID
	WorkOrderID     uuid.UUID
	CanvasRunID     uuid.UUID
	NodeExecutionID uuid.UUID
	EnabledChecks   []string
	// CheckLabels maps a check id to the step name. Built-in checks use their
	// own names when this map has no entry.
	CheckLabels map[string]string
}

var (
	mergeConfidenceChecksLine = regexp.MustCompile(`(?m)^Enabled checks: (none|(?:risk|performance|security|drift|reversibility)(?:, (?:risk|performance|security|drift|reversibility))*)\.$`)
	mergeCheckIDLine          = regexp.MustCompile(`(?m)^Merge check: ([a-z][a-z0-9-]{0,40})\.$`)
)

// MergeCheckID reads the check id from a step prompt. An empty string means the step is not a check.
func MergeCheckID(prompt string) string {
	match := mergeCheckIDLine.FindStringSubmatch(prompt)
	if match == nil {
		return ""
	}
	return match[1]
}

// ParseMergeConfidenceChecks reads the enabled-checks line from agent prompts.
// A missing line or "none" enables nothing.
func ParseMergeConfidenceChecks(prompts []string) []string {
	for _, prompt := range prompts {
		match := mergeConfidenceChecksLine.FindStringSubmatch(prompt)
		if match == nil {
			continue
		}
		if match[1] == "none" {
			return []string{}
		}
		return uniqueMergeConfidenceChecks(strings.Split(match[1], ", "))
	}
	return []string{}
}

// MergeConfidenceChecksFromSteps reads one check from each step that starts
// with a Merge check line. When no step has that line, it reads the older
// Enabled checks line and returns no labels.
func MergeConfidenceChecksFromSteps(steps []AgentStep) ([]string, map[string]string) {
	ids := make([]string, 0)
	labels := map[string]string{}
	for _, step := range steps {
		if step.Prompt == nil {
			continue
		}
		id := MergeCheckID(*step.Prompt)
		if id == "" {
			continue
		}
		if _, seen := labels[id]; seen {
			continue
		}
		ids = append(ids, id)
		label := strings.TrimSpace(step.Name)
		if label == "" {
			label = id
		}
		labels[id] = label
	}
	if len(ids) > 0 {
		return ids, labels
	}
	return ParseMergeConfidenceChecks(agentStepPrompts(steps)), nil
}

func uniqueMergeConfidenceChecks(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	enabled := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		enabled = append(enabled, name)
	}
	return enabled
}

func parseMergeConfidenceCheckLabels(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil
	}
	return labels
}

func parseMergeConfidenceChecksClaim(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	enabled := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		enabled = append(enabled, name)
	}
	return enabled
}

// AttachMergeConfidenceEnv reads the work order id and returns the check
// list in memory. Other agent runs are unchanged. The work order id is
// removed from the task environment.
func AttachMergeConfidenceEnv(ctx core.ExecutionContext, environment []TaskEnvironmentVariable, steps []AgentStep) ([]TaskEnvironmentVariable, *MergeConfidenceScope, error) {
	environment = withoutEnvironmentName(environment, legacyMergeConfidenceToken)
	orderID, ok := environmentValue(environment, EnvSuperplaneMergeConfidenceOrderID)
	if !ok {
		return environment, nil, nil
	}
	environment = withoutEnvironmentName(environment, EnvSuperplaneMergeConfidenceOrderID)

	workOrderID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence work order id is invalid")
	}
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence organization is invalid")
	}
	canvasID, err := uuid.Parse(strings.TrimSpace(ctx.WorkflowID))
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence canvas is invalid")
	}
	if ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return nil, nil, fmt.Errorf("merge confidence run is incomplete")
	}

	canvas, err := models.FindCanvasInTransaction(database.DB(context.Background()), orgID, canvasID)
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence canvas: %w", err)
	}
	if canvas.FactoryID == nil || *canvas.FactoryID == uuid.Nil {
		return nil, nil, fmt.Errorf("merge confidence requires a factory canvas")
	}

	factory, err := models.FindFactory(database.DB(context.Background()), orgID, *canvas.FactoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence workspace: %w", err)
	}
	if factory.OnboardingConfigValue().CodexSubscription {
		order, err := models.FindSubscriptionWorkOrderForRun(database.DB(context.Background()), orgID, factory.ID, ctx.RunID)
		if err != nil {
			return nil, nil, err
		}
		workOrderID = order.ID
	}

	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	if baseURL == "" {
		return nil, nil, fmt.Errorf("merge confidence requires a SuperPlane URL")
	}

	enabled, labels := MergeConfidenceChecksFromSteps(steps)
	if _, exists := environmentValue(environment, EnvSuperplaneBaseURL); !exists {
		environment = append(environment, TaskEnvironmentVariable{
			Name:  EnvSuperplaneBaseURL,
			Value: baseURL,
		})
	}
	return environment, &MergeConfidenceScope{
		OrganizationID:  orgID,
		FactoryID:       *canvas.FactoryID,
		WorkOrderID:     workOrderID,
		CanvasRunID:     ctx.RunID,
		NodeExecutionID: ctx.ID,
		EnabledChecks:   enabled,
		CheckLabels:     labels,
	}, nil
}

func agentStepPrompts(steps []AgentStep) []string {
	prompts := make([]string, 0, len(steps))
	for _, step := range steps {
		if step.Prompt == nil {
			continue
		}
		prompts = append(prompts, *step.Prompt)
	}
	return prompts
}

func environmentValue(environment []TaskEnvironmentVariable, name string) (string, bool) {
	for _, item := range environment {
		if item.Name != name {
			continue
		}
		return strings.TrimSpace(item.Value), true
	}
	return "", false
}

func withoutEnvironmentName(environment []TaskEnvironmentVariable, name string) []TaskEnvironmentVariable {
	filtered := make([]TaskEnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		if item.Name == name {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
