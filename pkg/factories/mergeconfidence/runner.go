package mergeconfidence

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const legacyMergeConfidenceToken = "SUPERPLANE_MERGE_CONFIDENCE_TOKEN"

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
func MergeConfidenceChecksFromSteps(steps []runner.AgentStep) ([]string, map[string]string) {
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

// AttachMergeConfidenceEnv reads the work order id and returns the check
// list in memory. Other agent runs are unchanged. The work order id is
// removed from the task environment.
func AttachMergeConfidenceEnv(tx *gorm.DB, ctx core.ExecutionContext, environment []runner.TaskEnvironmentVariable, steps []runner.AgentStep) ([]runner.TaskEnvironmentVariable, *runner.MergeConfidenceScope, error) {
	environment = withoutEnvironmentName(environment, legacyMergeConfidenceToken)
	orderID, ok := environmentValue(environment, runner.EnvSuperplaneMergeConfidenceOrderID)
	if !ok {
		return environment, nil, nil
	}
	environment = withoutEnvironmentName(environment, runner.EnvSuperplaneMergeConfidenceOrderID)

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

	canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence canvas: %w", err)
	}
	if canvas.FactoryID == nil || *canvas.FactoryID == uuid.Nil {
		return nil, nil, fmt.Errorf("merge confidence requires a factory canvas")
	}

	factory, err := models.FindFactory(tx, orgID, *canvas.FactoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence workspace: %w", err)
	}
	if factory.OnboardingConfigValue().CodexSubscription {
		order, err := models.FindSubscriptionWorkOrderForRun(tx, orgID, factory.ID, ctx.RunID)
		if err != nil {
			return nil, nil, err
		}
		workOrderID = order.ID
	}

	baseURL := runner.RunnerSuperplaneBaseURL(ctx.BaseURL)
	if baseURL == "" {
		return nil, nil, fmt.Errorf("merge confidence requires a SuperPlane URL")
	}

	repository, err := mergeConfidenceRepository(tx, ctx.RunID)
	if err != nil {
		return nil, nil, fmt.Errorf("merge confidence repository: %w", err)
	}

	enabled, labels := MergeConfidenceChecksFromSteps(steps)
	if _, exists := environmentValue(environment, runner.EnvSuperplaneBaseURL); !exists {
		environment = append(environment, runner.TaskEnvironmentVariable{
			Name:  runner.EnvSuperplaneBaseURL,
			Value: baseURL,
		})
	}
	return environment, &runner.MergeConfidenceScope{
		OrganizationID:  orgID,
		FactoryID:       *canvas.FactoryID,
		WorkOrderID:     workOrderID,
		CanvasRunID:     ctx.RunID,
		NodeExecutionID: ctx.ID,
		EnabledChecks:   enabled,
		CheckLabels:     labels,
		Repository:      repository,
	}, nil
}

// mergeConfidenceRepository reads the pull request repository from the run
// trigger, not from the task environment, so the agent cannot change it.
func mergeConfidenceRepository(db *gorm.DB, runID uuid.UUID) (string, error) {
	event, err := models.FindRootEventForRun(db, runID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	payload, ok := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return "", nil
	}
	repository, _ := payload["repository"].(map[string]any)
	name, _ := repository["full_name"].(string)
	for _, shape := range []struct{ field, number string }{{"pull_request", "number"}, {"pullrequest", "id"}} {
		pr, _ := payload[shape.field].(map[string]any)
		number, err := strconv.ParseInt(fmt.Sprint(pr[shape.number]), 10, 64)
		if err == nil && number > 0 {
			return strings.TrimSpace(name), nil
		}
	}
	return "", nil
}

func agentStepPrompts(steps []runner.AgentStep) []string {
	prompts := make([]string, 0, len(steps))
	for _, step := range steps {
		if step.Prompt == nil {
			continue
		}
		prompts = append(prompts, *step.Prompt)
	}
	return prompts
}

func environmentValue(environment []runner.TaskEnvironmentVariable, name string) (string, bool) {
	for _, item := range environment {
		if item.Name != name {
			continue
		}
		return strings.TrimSpace(item.Value), true
	}
	return "", false
}

func withoutEnvironmentName(environment []runner.TaskEnvironmentVariable, name string) []runner.TaskEnvironmentVariable {
	filtered := make([]runner.TaskEnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		if item.Name == name {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// PrepareRunner supplies trusted credentials as ordinary environment data.
func PrepareRunner(tx *gorm.DB, ctx *core.ExecutionContext) error {
	ctx.RunnerEnvironmentRemove = []string{legacyMergeConfidenceToken, runner.EnvSuperplaneMergeConfidenceOrderID}
	var configuration struct {
		EnvironmentFrom []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
		Environment     []runner.EnvironmentVariable  `mapstructure:"environment"`
		Steps           []runner.AgentStep            `mapstructure:"steps"`
		Prompt          string                        `mapstructure:"prompt"`
		Timeout         int                           `mapstructure:"executionTimeoutSeconds"`
	}
	decoder, err := runner.NewSpecDecoder(&configuration)
	if err != nil {
		return err
	}
	if err := decoder.Decode(ctx.Configuration); err != nil {
		return err
	}
	if err := runner.ValidateEnvironmentFrom(configuration.EnvironmentFrom); err != nil {
		return err
	}
	if err := runner.ValidateEnvironment(configuration.Environment); err != nil {
		return err
	}
	resolved, err := runner.ResolveEnvironment(ctx.Secrets, configuration.EnvironmentFrom, configuration.Environment)
	if err != nil {
		return err
	}
	if configuration.Timeout <= 0 {
		configuration.Timeout = runner.DefaultExecutionTimeoutSeconds
	}
	if len(configuration.Steps) == 0 && configuration.Prompt != "" {
		configuration.Steps = []runner.AgentStep{{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &configuration.Prompt}}
	}
	environment, merge, err := AttachMergeConfidenceEnv(tx, *ctx, resolved.Variables, configuration.Steps)
	if err != nil {
		return err
	}
	if merge == nil {
		return nil
	}
	scope := &runner.RunnerScope{
		OrganizationID: merge.OrganizationID, FactoryID: merge.FactoryID, WorkOrderID: merge.WorkOrderID,
		CanvasID: uuid.MustParse(ctx.WorkflowID), CanvasRunID: merge.CanvasRunID,
		NodeExecutionID: merge.NodeExecutionID, NodeID: ctx.NodeID,
		Flow: runner.RunnerFlowMergeConfidence, MergeChecks: merge.EnabledChecks,
		MergeCheckLabels: merge.CheckLabels, MergeRepository: merge.Repository,
	}
	prepared := runner.AttachScopedRunnerEnv(tx, *ctx, environment, configuration.Timeout, scope)
	ctx.RunnerEnvironment = map[string]string{}
	for _, item := range prepared {
		switch item.Name {
		case runner.EnvSuperplaneBaseURL, runner.EnvSuperplaneRunnerToken, runner.EnvSuperplaneTaskID:
			ctx.RunnerEnvironment[item.Name] = item.Value
		}
	}
	return nil
}
