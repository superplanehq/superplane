package harness

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	RunPipelinePayloadType   = "harness.pipeline.finished"
	RunPipelinePollAction    = "poll"
	RunPipelineSuccess       = "success"
	RunPipelineFailed        = "failed"
	RunPipelinePollInterval  = 1 * time.Minute
	RunPipelineMaxPollErrors = 5
)

type RunPipeline struct{}

type RunPipelineSpec struct {
	OrgID              string   `json:"orgId" mapstructure:"orgId"`
	ProjectID          string   `json:"projectId" mapstructure:"projectId"`
	PipelineIdentifier string   `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Ref                string   `json:"ref" mapstructure:"ref"`
	InputSetReferences []string `json:"inputSetReferences" mapstructure:"inputSetReferences"`
	InputYAML          string   `json:"inputYAML" mapstructure:"inputYAML"`
}

type RunPipelineExecutionMetadata struct {
	OrgID              string `json:"orgId" mapstructure:"orgId"`
	ProjectID          string `json:"projectId" mapstructure:"projectId"`
	ExecutionID        string `json:"executionId" mapstructure:"executionId"`
	PipelineIdentifier string `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Status             string `json:"status" mapstructure:"status"`
	PlanExecutionURL   string `json:"planExecutionUrl,omitempty" mapstructure:"planExecutionUrl"`
	StartedAt          string `json:"startedAt,omitempty" mapstructure:"startedAt"`
	EndedAt            string `json:"endedAt,omitempty" mapstructure:"endedAt"`
	PollErrorCount     int    `json:"pollErrorCount,omitempty" mapstructure:"pollErrorCount"`
}

func (r *RunPipeline) Name() string {
	return "harness.runPipeline"
}

func (r *RunPipeline) Label() string {
	return "Run Pipeline"
}

func (r *RunPipeline) Description() string {
	return "Run a Harness pipeline and wait for completion"
}

func (r *RunPipeline) Documentation() string {
	return `The Run Pipeline component starts a Harness pipeline execution and waits for it to finish.

## Use Cases

- **CI/CD orchestration**: Trigger deploy pipelines from workflow events
- **Approval-based releases**: Run release pipelines after manual approvals
- **Scheduled automation**: Kick off recurring maintenance or validation pipelines

## How It Works

1. Starts a Harness pipeline execution
2. Stores the execution ID in node execution state
3. Watches execution completion via webhook (with polling fallback)
4. Routes output to:
   - **Success** when execution succeeds
   - **Failed** when execution fails, aborts, or expires

## Configuration

- **Org**: Harness organization identifier
- **Project**: Harness project identifier
- **Pipeline**: Harness pipeline identifier
- **Ref**: Optional git ref (` + "`refs/heads/main`" + ` or ` + "`refs/tags/v1.2.3`" + `)
- **Input Set References**: Optional input set identifiers
- **Runtime Input YAML**: Optional YAML override for runtime inputs`
}

func (r *RunPipeline) Icon() string {
	return "workflow"
}

func (r *RunPipeline) Color() string {
	return "gray"
}

func (r *RunPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: RunPipelineSuccess, Label: "Success"},
		{Name: RunPipelineFailed, Label: "Failed"},
	}
}

func (r *RunPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orgId",
			Label:       "Organization",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select Harness organization",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeOrg,
				},
			},
		},
		{
			Name:        "projectId",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select Harness project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
					Parameters: []configuration.ParameterRef{
						{
							Name: "orgId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "orgId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "pipelineIdentifier",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the pipeline to run",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePipeline,
					Parameters: []configuration.ParameterRef{
						{
							Name: "orgId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "orgId",
							},
						},
						{
							Name: "projectId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "projectId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "inputSetReferences",
			Label:       "Input Set References",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional Harness input sets to apply",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Input Set",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeGitRef,
			Required:    false,
			Togglable:   true,
			Default:     "refs/heads/main",
			Description: "Optional branch or tag to run against",
		},
		{
			Name:        "inputYAML",
			Label:       "Runtime Input YAML",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Optional runtime input YAML override",
		},
	}
}

func (r *RunPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunPipeline) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if ctx.Integration == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := validateHarnessScopeSelection(client, spec.OrgID, spec.ProjectID); err != nil {
		return err
	}

	if err := validateHarnessPipelineSelection(client, spec.OrgID, spec.ProjectID, spec.PipelineIdentifier); err != nil {
		return err
	}

	if err := ctx.Integration.RequestWebhook(WebhookConfiguration{
		PipelineIdentifier: spec.PipelineIdentifier,
		OrgID:              spec.OrgID,
		ProjectID:          spec.ProjectID,
		EventTypes:         defaultWebhookEventTypes,
	}); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to request Harness webhook provisioning, using polling fallback: %v", err)
		}
	}

	return nil
}

func (r *RunPipeline) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	client = client.withScope(spec.OrgID, spec.ProjectID)

	response, err := client.RunPipeline(RunPipelineRequest{
		PipelineIdentifier: spec.PipelineIdentifier,
		Ref:                spec.Ref,
		InputSetRefs:       spec.InputSetReferences,
		InputYAML:          spec.InputYAML,
	})
	if err != nil {
		return err
	}

	metadata := RunPipelineExecutionMetadata{
		OrgID:              spec.OrgID,
		ProjectID:          spec.ProjectID,
		ExecutionID:        response.ExecutionID,
		PipelineIdentifier: spec.PipelineIdentifier,
		Status:             "running",
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV("execution", response.ExecutionID); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
}

func (r *RunPipeline) Actions() []core.Action {
	return []core.Action{{Name: RunPipelinePollAction, UserAccessible: false}}
}

func (r *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case RunPipelinePollAction:
		return r.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (r *RunPipeline) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if strings.TrimSpace(metadata.ExecutionID) == "" {
		return fmt.Errorf("execution ID is missing from metadata")
	}
	spec, err := decodeRunPipelineSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	metadata.OrgID = firstNonEmpty(metadata.OrgID, spec.OrgID)
	metadata.ProjectID = firstNonEmpty(metadata.ProjectID, spec.ProjectID)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	client = client.withScope(metadata.OrgID, metadata.ProjectID)

	summary, err := client.GetExecutionSummary(metadata.ExecutionID)
	if err != nil {
		metadata.PollErrorCount++
		metadata.Status = "error"

		if setErr := ctx.Metadata.Set(metadata); setErr != nil {
			return setErr
		}

		if metadata.PollErrorCount >= RunPipelineMaxPollErrors {
			payload := map[string]any{
				"executionId":        metadata.ExecutionID,
				"pipelineIdentifier": metadata.PipelineIdentifier,
				"status":             canonicalStatus(metadata.Status),
				"planExecutionUrl":   metadata.PlanExecutionURL,
				"startedAt":          metadata.StartedAt,
				"endedAt":            metadata.EndedAt,
				"error":              err.Error(),
			}
			return ctx.ExecutionState.Emit(RunPipelineFailed, RunPipelinePayloadType, []any{payload})
		}

		return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
	}

	metadata.Status = summary.Status
	metadata.PlanExecutionURL = summary.PlanExecutionURL
	metadata.StartedAt = summary.StartedAt
	metadata.EndedAt = summary.EndedAt
	metadata.PollErrorCount = 0
	if metadata.PipelineIdentifier == "" {
		metadata.PipelineIdentifier = summary.PipelineIdentifier
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if !isTerminalStatus(summary.Status) {
		return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
	}

	return r.emitResult(ctx.ExecutionState, metadata, summary.Status, nil)
}

func (r *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := authorizeWebhook(ctx); err != nil {
		return http.StatusForbidden, err
	}

	if ctx.FindExecutionByKV == nil {
		return http.StatusOK, nil
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	event := extractPipelineWebhookEvent(payload)
	if !isPipelineCompletedEventType(event.EventType) {
		return http.StatusOK, nil
	}

	if !isTerminalStatus(event.Status) {
		return http.StatusOK, nil
	}

	executionID := strings.TrimSpace(event.ExecutionID)
	if executionID == "" {
		return http.StatusOK, nil
	}

	executionCtx, err := ctx.FindExecutionByKV("execution", executionID)
	if err != nil {
		// Ignore unrelated webhook events for executions not created by SuperPlane.
		return http.StatusOK, nil
	}

	if executionCtx.ExecutionState != nil && executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil
	}
	if executionCtx.Metadata == nil {
		return http.StatusInternalServerError, fmt.Errorf("missing metadata context")
	}

	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	webhookSummary := executionSummaryFromWebhookPayload(event, payload)
	metadata.ExecutionID = firstNonEmpty(strings.TrimSpace(metadata.ExecutionID), webhookSummary.ExecutionID)
	metadata.PipelineIdentifier = firstNonEmpty(strings.TrimSpace(metadata.PipelineIdentifier), webhookSummary.PipelineIdentifier)
	metadata.Status = firstNonEmpty(strings.TrimSpace(webhookSummary.Status), metadata.Status)
	metadata.PlanExecutionURL = firstNonEmpty(strings.TrimSpace(webhookSummary.PlanExecutionURL), metadata.PlanExecutionURL)
	metadata.StartedAt = firstNonEmpty(strings.TrimSpace(webhookSummary.StartedAt), metadata.StartedAt)
	metadata.EndedAt = firstNonEmpty(strings.TrimSpace(webhookSummary.EndedAt), metadata.EndedAt)
	metadata.PollErrorCount = 0

	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update metadata: %w", err)
	}

	if err := r.emitResult(executionCtx.ExecutionState, metadata, metadata.Status, payload); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (r *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeRunPipelineSpec(value any) (RunPipelineSpec, error) {
	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(value, &spec); err != nil {
		return RunPipelineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.OrgID = strings.TrimSpace(spec.OrgID)
	if spec.OrgID == "" {
		return RunPipelineSpec{}, fmt.Errorf("orgId is required")
	}

	spec.ProjectID = strings.TrimSpace(spec.ProjectID)
	if spec.ProjectID == "" {
		return RunPipelineSpec{}, fmt.Errorf("projectId is required")
	}

	spec.PipelineIdentifier = strings.TrimSpace(spec.PipelineIdentifier)
	if spec.PipelineIdentifier == "" {
		return RunPipelineSpec{}, fmt.Errorf("pipelineIdentifier is required")
	}

	spec.Ref = strings.TrimSpace(spec.Ref)
	spec.InputYAML = strings.TrimSpace(spec.InputYAML)

	filtered := make([]string, 0, len(spec.InputSetReferences))
	for _, ref := range spec.InputSetReferences {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	spec.InputSetReferences = filtered

	return spec, nil
}

func (r *RunPipeline) emitResult(
	executionState core.ExecutionStateContext,
	metadata RunPipelineExecutionMetadata,
	status string,
	raw map[string]any,
) error {
	if executionState == nil {
		return fmt.Errorf("missing execution state context")
	}

	payload := map[string]any{
		"executionId":        metadata.ExecutionID,
		"pipelineIdentifier": metadata.PipelineIdentifier,
		"status":             canonicalStatus(status),
		"planExecutionUrl":   metadata.PlanExecutionURL,
		"startedAt":          metadata.StartedAt,
		"endedAt":            metadata.EndedAt,
	}
	if raw != nil {
		payload["raw"] = raw
	}

	channel := RunPipelineFailed
	if isSuccessStatus(status) {
		channel = RunPipelineSuccess
	}

	return executionState.Emit(channel, RunPipelinePayloadType, []any{payload})
}
