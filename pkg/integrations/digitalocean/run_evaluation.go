package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const evalPollInterval = 30 * time.Second

const (
	evalChannelPassed = "passed"
	evalChannelFailed = "failed"
)

type RunEvaluation struct{}

type RunEvaluationSpec struct {
	TestCase string `json:"testCase" mapstructure:"testCase"`
	Agent    string `json:"agent" mapstructure:"agent"`
	RunName  string `json:"runName" mapstructure:"runName"`
}

func (r *RunEvaluation) Name() string {
	return "digitalocean.runEvaluation"
}

func (r *RunEvaluation) Label() string {
	return "Run Evaluation"
}

func (r *RunEvaluation) Description() string {
	return "Run a Gradient AI evaluation test case against an agent and report pass or fail"
}

func (r *RunEvaluation) Documentation() string {
	return `The Run Evaluation component triggers a Gradient AI evaluation test case against an agent, waits for it to complete, and reports whether the agent passed or failed.

## How it works

Runs a pre-configured evaluation test case against the selected agent. The test case already defines the prompts, metrics, and pass/fail thresholds. The component polls until the evaluation finishes, then fetches the results and routes to the appropriate output channel.

## Use Cases

- **Blue/green deployments**: Evaluate a staging agent before promoting it to production
- **Regression testing**: Automatically verify agent quality after knowledge base or configuration changes
- **Continuous validation**: Schedule periodic evaluations to detect quality drift

## Configuration

- **Test Case**: A pre-configured evaluation test case with prompts, metrics, and thresholds (required)
- **Agent**: The agent to evaluate (required)
- **Run Name**: A name for this evaluation run, visible in the DigitalOcean console (required, max 64 characters). Supports expressions for dynamic naming.

## Output Channels

- **Passed**: The evaluation completed and the agent met all pass criteria defined in the test case
- **Failed**: The evaluation completed but the agent did not meet the pass criteria, or the evaluation run itself errored

## Output

Returns the evaluation results including:
- **evaluationRunUUID**: UUID of the evaluation run
- **testCaseUUID / testCaseName**: The test case that was run
- **agentUUID / agentName**: The agent that was evaluated
- **passed**: Whether the agent passed the evaluation
- **status**: Final status of the evaluation run
- **starMetric**: The primary metric result (name, numberValue, stringValue)
- **runLevelMetrics**: All run-level metric results (name, numberValue, stringValue)
- **prompts**: Per-prompt results including input, output, ground truth, and per-prompt metric scores
- **startedAt / finishedAt**: Timing information
- **errorDescription**: Present on the Failed channel if the run itself errored

## Notes

- The evaluation typically takes 1–5 minutes depending on the number of prompts and complexity
- The component polls every 30 seconds until completion
- If the evaluation run itself fails (API error, timeout, etc.), the result is emitted to the Failed channel with the error description`
}

func (r *RunEvaluation) Icon() string {
	return "flask-conical"
}

func (r *RunEvaluation) Color() string {
	return "purple"
}

func (r *RunEvaluation) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: evalChannelPassed, Label: "Passed", Description: "The agent passed the evaluation"},
		{Name: evalChannelFailed, Label: "Failed", Description: "The agent failed the evaluation or the run errored"},
	}
}

func (r *RunEvaluation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "runName",
			Label:       "Run Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. post-kb-update-check",
			Description: "A name for this evaluation run, visible in the DigitalOcean console. Maximum 64 characters.",
		},
		{
			Name:        "testCase",
			Label:       "Evaluation Test Case",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a test case",
			Description: "The pre-configured evaluation test case to run, including prompts, metrics, and pass/fail thresholds. When using an expression, provide the test case UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "evaluation_test_case",
				},
			},
		},
		{
			Name:        "agent",
			Label:       "Agent",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an agent",
			Description: "The agent to evaluate. When using an expression, provide the agent UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent",
				},
			},
		},
	}
}

func (r *RunEvaluation) Setup(ctx core.SetupContext) error {
	spec := RunEvaluationSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.TestCase == "" {
		return errors.New("testCase is required")
	}

	if spec.Agent == "" {
		return errors.New("agent is required")
	}

	if spec.RunName == "" {
		return errors.New("runName is required")
	}

	if !strings.Contains(spec.RunName, "{{") && len(spec.RunName) > 64 {
		return errors.New("runName must be 64 characters or less")
	}

	if err := resolveEvalMetadata(ctx, spec); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

// evalRunMetadata is stored between poll ticks
type evalRunMetadata struct {
	EvalRunUUID   string `json:"evalRunUUID" mapstructure:"evalRunUUID"`
	TestCaseID    string `json:"testCaseId" mapstructure:"testCaseId"`
	TestCaseName  string `json:"testCaseName" mapstructure:"testCaseName"`
	WorkspaceUUID string `json:"workspaceUUID" mapstructure:"workspaceUUID"`
	AgentID       string `json:"agentId" mapstructure:"agentId"`
	AgentName     string `json:"agentName" mapstructure:"agentName"`
}

func (r *RunEvaluation) Execute(ctx core.ExecutionContext) error {
	spec := RunEvaluationSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// POST only returns the run UUID, not the full object
	runUUID, err := client.RunEvaluation(spec.TestCase, []string{spec.Agent}, spec.RunName)
	if err != nil {
		return fmt.Errorf("failed to start evaluation run: %v", err)
	}

	// Get names from the node metadata set during Setup
	var nodeMeta EvalNodeMetadata
	_ = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMeta)

	// If testCaseId or agentId was configured via an expression, the name stored
	// during Setup is the raw expression string. Fall back to the resolved ID instead.
	// Also resolve workspace UUID from the agent if not already set.
	testCaseName := nodeMeta.TestCaseName
	if strings.Contains(testCaseName, "{{") {
		testCaseName = spec.TestCase
	}

	agentName := nodeMeta.AgentName
	workspaceUUID := nodeMeta.WorkspaceUUID
	if strings.Contains(agentName, "{{") {
		agentName = spec.Agent
		if workspaceUUID == "" {
			if agent, agentErr := client.GetAgent(spec.Agent); agentErr == nil {
				if agent.Workspace != nil {
					workspaceUUID = agent.Workspace.UUID
				}
			}
		}
	}

	meta := evalRunMetadata{
		EvalRunUUID:   runUUID,
		TestCaseID:    spec.TestCase,
		TestCaseName:  testCaseName,
		WorkspaceUUID: workspaceUUID,
		AgentID:       spec.Agent,
		AgentName:     agentName,
	}

	if err := ctx.Metadata.Set(meta); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, evalPollInterval)
}

// evalRunState normalises a DO evaluation run status to a simple lowercase keyword.
// The DO API returns prefixed enum values like "EVALUATION_RUN_STATUS_COMPLETED",
// but may also return plain values. We match by suffix so both forms are handled.
func evalRunState(status string) string {
	lower := strings.ToLower(status)
	for _, state := range []string{"successful", "completed", "running", "queued", "failed", "cancelled", "in_progress", "pending"} {
		if strings.HasSuffix(lower, state) {
			return state
		}
	}
	return lower
}

func (r *RunEvaluation) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (r *RunEvaluation) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var meta evalRunMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	run, err := client.GetEvaluationRun(meta.EvalRunUUID)
	if err != nil {
		return fmt.Errorf("failed to get evaluation run: %v", err)
	}

	state := evalRunState(run.Status)

	switch state {
	case "successful", "completed":
		return r.handleCompleted(ctx, client, meta)
	case "running", "queued", "in_progress", "pending":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, evalPollInterval)
	case "failed", "cancelled":
		return r.emitFailed(ctx, meta, run, run.ErrorDescription)
	default:
		// Unknown status — log it and fail so we can investigate
		return fmt.Errorf("evaluation run %s has unexpected status: %q (normalized: %q)", meta.EvalRunUUID, run.Status, state)
	}
}

func (r *RunEvaluation) handleCompleted(ctx core.ActionContext, client *Client, meta evalRunMetadata) error {
	results, err := client.GetEvaluationRunResults(meta.EvalRunUUID)
	if err != nil {
		return fmt.Errorf("failed to get evaluation run results: %v", err)
	}

	output := buildEvalOutput(meta, &results.EvaluationRun, results.Prompts)

	if results.EvaluationRun.PassStatus {
		return ctx.ExecutionState.Emit(
			evalChannelPassed,
			"digitalocean.evaluation.passed",
			[]any{output},
		)
	}

	return ctx.ExecutionState.Emit(
		evalChannelFailed,
		"digitalocean.evaluation.failed",
		[]any{output},
	)
}

func (r *RunEvaluation) emitFailed(ctx core.ActionContext, meta evalRunMetadata, run *EvaluationRun, errorDesc string) error {
	output := buildEvalOutput(meta, run, nil)
	output["errorDescription"] = errorDesc

	return ctx.ExecutionState.Emit(
		evalChannelFailed,
		"digitalocean.evaluation.failed",
		[]any{output},
	)
}

func buildEvalOutput(meta evalRunMetadata, run *EvaluationRun, prompts []EvaluationPrompt) map[string]any {
	output := map[string]any{
		"evaluationRunUUID": meta.EvalRunUUID,
		"testCaseUUID":      meta.TestCaseID,
		"testCaseName":      meta.TestCaseName,
		"workspaceUUID":     meta.WorkspaceUUID,
		"agentUUID":         meta.AgentID,
		"agentName":         meta.AgentName,
		"passed":            run.PassStatus,
		"status":            evalRunState(run.Status),
		"startedAt":         run.StartedAt,
		"finishedAt":        run.FinishedAt,
	}

	if run.StarMetricResult != nil {
		output["starMetric"] = map[string]any{
			"metricName":  run.StarMetricResult.MetricName,
			"numberValue": run.StarMetricResult.NumberValue,
			"stringValue": run.StarMetricResult.StringValue,
		}
	}

	if len(run.RunLevelMetrics) > 0 {
		metrics := make([]map[string]any, 0, len(run.RunLevelMetrics))
		for _, m := range run.RunLevelMetrics {
			metrics = append(metrics, map[string]any{
				"metricName":  m.MetricName,
				"numberValue": m.NumberValue,
				"stringValue": m.StringValue,
			})
		}
		output["runLevelMetrics"] = metrics
	}

	if len(prompts) > 0 {
		promptResults := make([]map[string]any, 0, len(prompts))
		for _, p := range prompts {
			pr := map[string]any{
				"input":       p.Input,
				"output":      p.Output,
				"groundTruth": p.GroundTruth,
			}

			if len(p.Metrics) > 0 {
				pMetrics := make([]map[string]any, 0, len(p.Metrics))
				for _, m := range p.Metrics {
					pMetrics = append(pMetrics, map[string]any{
						"metricName":  m.MetricName,
						"numberValue": m.NumberValue,
						"stringValue": m.StringValue,
					})
				}
				pr["metrics"] = pMetrics
			}

			promptResults = append(promptResults, pr)
		}
		output["prompts"] = promptResults
	}

	return output
}

func (r *RunEvaluation) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunEvaluation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunEvaluation) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (r *RunEvaluation) Cleanup(ctx core.SetupContext) error {
	return nil
}

// EvalNodeMetadata stores metadata about an evaluation for display in the UI
type EvalNodeMetadata struct {
	TestCaseID    string `json:"testCaseId" mapstructure:"testCaseId"`
	TestCaseName  string `json:"testCaseName" mapstructure:"testCaseName"`
	WorkspaceUUID string `json:"workspaceUUID" mapstructure:"workspaceUUID"`
	AgentID       string `json:"agentId" mapstructure:"agentId"`
	AgentName     string `json:"agentName" mapstructure:"agentName"`
}

// resolveEvalMetadata fetches the test case and agent names from the API
func resolveEvalMetadata(ctx core.SetupContext, spec RunEvaluationSpec) error {
	meta := EvalNodeMetadata{
		TestCaseID: spec.TestCase,
		AgentID:    spec.Agent,
	}

	isTestCaseExpr := strings.Contains(spec.TestCase, "{{")
	isAgentExpr := strings.Contains(spec.Agent, "{{")

	if isTestCaseExpr {
		meta.TestCaseName = spec.TestCase
	}
	if isAgentExpr {
		meta.AgentName = spec.Agent
	}

	// Check if already cached
	var existing EvalNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil &&
		existing.TestCaseID == spec.TestCase && existing.TestCaseName != "" &&
		existing.AgentID == spec.Agent && existing.AgentName != "" {
		return nil
	}

	if isTestCaseExpr && isAgentExpr {
		return ctx.Metadata.Set(meta)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if !isTestCaseExpr {
		testCases, err := client.ListEvaluationTestCases()
		if err != nil {
			return fmt.Errorf("failed to list evaluation test cases: %w", err)
		}
		for _, tc := range testCases {
			if tc.UUID == spec.TestCase {
				meta.TestCaseName = tc.Name
				meta.WorkspaceUUID = tc.WorkspaceUUID
				break
			}
		}
		if meta.TestCaseName == "" {
			meta.TestCaseName = spec.TestCase
		}
	}

	if !isAgentExpr {
		agent, err := client.GetAgent(spec.Agent)
		if err != nil {
			return fmt.Errorf("failed to fetch agent %q: %w", spec.Agent, err)
		}
		meta.AgentName = agent.Name
		if agent.Workspace != nil && meta.WorkspaceUUID == "" {
			meta.WorkspaceUUID = agent.Workspace.UUID
		}
	}

	return ctx.Metadata.Set(meta)
}
