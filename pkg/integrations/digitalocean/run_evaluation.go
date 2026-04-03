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
	TestCaseID string `json:"testCaseId" mapstructure:"testCaseId"`
	AgentID    string `json:"agentId" mapstructure:"agentId"`
	RunName    string `json:"runName" mapstructure:"runName"`
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
- **evaluationRunId**: UUID of the evaluation run
- **testCaseId / testCaseName**: The test case that was run
- **agentId / agentName**: The agent that was evaluated
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
			Name:        "testCaseId",
			Label:       "Evaluation Test Case",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a test case",
			Description: "The pre-configured evaluation test case to run, including prompts, metrics, and pass/fail thresholds",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "evaluation_test_case",
				},
			},
		},
		{
			Name:        "agentId",
			Label:       "Agent",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an agent",
			Description: "The agent to evaluate",
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

	if spec.TestCaseID == "" {
		return errors.New("testCaseId is required")
	}

	if spec.AgentID == "" {
		return errors.New("agentId is required")
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
	EvalRunUUID  string `mapstructure:"evalRunUUID"`
	TestCaseID   string `mapstructure:"testCaseId"`
	TestCaseName string `mapstructure:"testCaseName"`
	AgentID      string `mapstructure:"agentId"`
	AgentName    string `mapstructure:"agentName"`
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
	runUUID, err := client.RunEvaluation(spec.TestCaseID, []string{spec.AgentID}, spec.RunName)
	if err != nil {
		return fmt.Errorf("failed to start evaluation run: %v", err)
	}

	// Get names from the node metadata set during Setup
	var nodeMeta EvalNodeMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &nodeMeta)

	meta := evalRunMetadata{
		EvalRunUUID:  runUUID,
		TestCaseID:   spec.TestCaseID,
		TestCaseName: nodeMeta.TestCaseName,
		AgentID:      spec.AgentID,
		AgentName:    nodeMeta.AgentName,
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
		return r.handleCompleted(ctx, client, meta, run)
	case "running", "queued", "in_progress", "pending":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, evalPollInterval)
	case "failed", "cancelled":
		return r.emitFailed(ctx, meta, run, run.ErrorDescription)
	default:
		// Unknown status — log it and fail so we can investigate
		return fmt.Errorf("evaluation run %s has unexpected status: %q (normalized: %q)", meta.EvalRunUUID, run.Status, state)
	}
}

func (r *RunEvaluation) handleCompleted(ctx core.ActionContext, client *Client, meta evalRunMetadata, run *EvaluationRun) error {
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
		"evaluationRunId": meta.EvalRunUUID,
		"testCaseId":      meta.TestCaseID,
		"testCaseName":    meta.TestCaseName,
		"agentId":         meta.AgentID,
		"agentName":       meta.AgentName,
		"passed":          run.PassStatus,
		"status":          evalRunState(run.Status),
		"startedAt":       run.StartedAt,
		"finishedAt":      run.FinishedAt,
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
	TestCaseID   string `json:"testCaseId" mapstructure:"testCaseId"`
	TestCaseName string `json:"testCaseName" mapstructure:"testCaseName"`
	AgentID      string `json:"agentId" mapstructure:"agentId"`
	AgentName    string `json:"agentName" mapstructure:"agentName"`
}

// resolveEvalMetadata fetches the test case and agent names from the API
func resolveEvalMetadata(ctx core.SetupContext, spec RunEvaluationSpec) error {
	meta := EvalNodeMetadata{
		TestCaseID: spec.TestCaseID,
		AgentID:    spec.AgentID,
	}

	isTestCaseExpr := strings.Contains(spec.TestCaseID, "{{")
	isAgentExpr := strings.Contains(spec.AgentID, "{{")

	if isTestCaseExpr {
		meta.TestCaseName = spec.TestCaseID
	}
	if isAgentExpr {
		meta.AgentName = spec.AgentID
	}

	// Check if already cached
	var existing EvalNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil &&
		existing.TestCaseID == spec.TestCaseID && existing.TestCaseName != "" &&
		existing.AgentID == spec.AgentID && existing.AgentName != "" {
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
			if tc.UUID == spec.TestCaseID {
				meta.TestCaseName = tc.Name
				break
			}
		}
		if meta.TestCaseName == "" {
			meta.TestCaseName = spec.TestCaseID
		}
	}

	if !isAgentExpr {
		agent, err := client.GetAgent(spec.AgentID)
		if err != nil {
			return fmt.Errorf("failed to fetch agent %q: %w", spec.AgentID, err)
		}
		meta.AgentName = agent.Name
	}

	return ctx.Metadata.Set(meta)
}
