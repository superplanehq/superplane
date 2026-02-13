package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipeline []byte

const (
	GitLabPipelinePayloadType         = "gitlab.pipeline.finished"
	GitLabPipelinePassedOutputChannel = "passed"
	GitLabPipelineFailedOutputChannel = "failed"
	GitLabPipelineStatusSuccess       = "success"
	GitLabPipelineStatusFailed        = "failed"
	GitLabPipelineStatusCanceled      = "canceled"
	GitLabPipelineStatusCancelled     = "cancelled"
	GitLabPipelineStatusSkipped       = "skipped"
	GitLabPipelineStatusManual        = "manual"
	GitLabPipelineStatusBlocked       = "blocked"
	GitLabRunPipelinePollInterval     = 5 * time.Minute
	GitLabRunPipelinePollAction       = "poll"
	GitLabRunPipelineKVPipelineID     = "pipeline_id"
)

type RunPipeline struct{}

type RunPipelineSpec struct {
	Project   string                    `json:"project" mapstructure:"project"`
	Ref       string                    `json:"ref" mapstructure:"ref"`
	Inputs    []RunPipelineInputSpec    `json:"inputs" mapstructure:"inputs"`
	Variables []RunPipelineVariableSpec `json:"variables" mapstructure:"variables"`
}

type RunPipelineInputSpec struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type RunPipelineVariableSpec struct {
	Name         string `json:"name" mapstructure:"name"`
	Value        string `json:"value" mapstructure:"value"`
	VariableType string `json:"variableType" mapstructure:"variableType"`
}

type RunPipelineExecutionMetadata struct {
	Pipeline *Pipeline `json:"pipeline" mapstructure:"pipeline"`
}

func (r *RunPipeline) Name() string {
	return "gitlab.runPipeline"
}

func (r *RunPipeline) Label() string {
	return "Run Pipeline"
}

func (r *RunPipeline) Description() string {
	return "Run a GitLab pipeline and wait for completion"
}

func (r *RunPipeline) Documentation() string {
	return `The Run Pipeline component triggers a GitLab pipeline and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger GitLab pipelines from SuperPlane workflows
- **Deployment automation**: Run deployment pipelines with custom variables
- **Pipeline chaining**: Coordinate follow-up actions after pipeline completion

## How It Works

1. Creates a new pipeline using the selected project, ref, inputs, and variables
2. Waits for pipeline completion, primarily via webhook updates
3. Falls back to polling if webhook updates are delayed or unavailable
4. Routes execution to:
   - **Passed** channel when pipeline succeeds
   - **Failed** channel for failed, canceled, skipped, manual, or blocked outcomes`
}

func (r *RunPipeline) Icon() string {
	return "workflow"
}

func (r *RunPipeline) Color() string {
	return "orange"
}

func (r *RunPipeline) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputRunPipeline, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (r *RunPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  GitLabPipelinePassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  GitLabPipelineFailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (r *RunPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:     "ref",
			Label:    "Ref",
			Type:     configuration.FieldTypeGitRef,
			Required: true,
			Default:  "main",
		},
		{
			Name:  "inputs",
			Label: "Inputs",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Input",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "name",
								Label:              "Name",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		{
			Name:  "variables",
			Label: "Variables",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "name",
								Label:              "Name",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:    "variableType",
								Label:   "Variable Type",
								Type:    configuration.FieldTypeSelect,
								Default: "env_var",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Environment variable", Value: "env_var"},
											{Label: "File", Value: "file"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *RunPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunPipeline) Setup(ctx core.SetupContext) error {
	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Project) == "" {
		return fmt.Errorf("project is required")
	}

	if strings.TrimSpace(spec.Ref) == "" {
		return fmt.Errorf("ref is required")
	}

	if err := ensureProjectInMetadata(ctx.Metadata, ctx.Integration, spec.Project); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventType: "pipeline",
		ProjectID: spec.Project,
	})
}

func (r *RunPipeline) Execute(ctx core.ExecutionContext) error {
	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.CreatePipeline(context.Background(), spec.Project, &CreatePipelineRequest{
		Ref:       normalizePipelineRef(spec.Ref),
		Inputs:    r.buildInputs(spec.Inputs),
		Variables: r.buildVariables(spec.Variables),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	var rawNodeMetadata any
	if ctx.NodeMetadata != nil {
		rawNodeMetadata = ctx.NodeMetadata.Get()
	}

	if pipeline.WebURL == "" {
		pipeline.WebURL = r.resolvePipelineURL(pipeline, rawNodeMetadata)
	}
	if pipeline.URL == "" {
		pipeline.URL = pipeline.WebURL
	}

	metadata := RunPipelineExecutionMetadata{Pipeline: pipeline}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(GitLabRunPipelineKVPipelineID, strconv.Itoa(pipeline.ID)); err != nil {
		return err
	}

	ctx.Logger.Infof("Started GitLab pipeline %d on project %s (ref=%s)", pipeline.ID, spec.Project, spec.Ref)
	return ctx.Requests.ScheduleActionCall(GitLabRunPipelinePollAction, map[string]any{}, GitLabRunPipelinePollInterval)
}

func (r *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	eventType := ctx.Headers.Get("X-Gitlab-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-Gitlab-Event header")
	}

	if eventType != "Pipeline Hook" {
		return http.StatusOK, nil
	}

	code, err := verifyWebhookToken(ctx)
	if err != nil {
		return code, err
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	pipeline, err := metadataFromPipelineWebhookPayload(payload)
	if err != nil {
		return http.StatusBadRequest, err
	}

	executionCtx, err := ctx.FindExecutionByKV(GitLabRunPipelineKVPipelineID, strconv.Itoa(pipeline.ID))
	if err != nil {
		// Ignore hooks for pipelines not started by SuperPlane
		return http.StatusOK, nil
	}

	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Pipeline != nil && isGitLabPipelineDone(metadata.Pipeline.Status) {
		return http.StatusOK, nil
	}

	if metadata.Pipeline == nil {
		metadata.Pipeline = &Pipeline{}
	}
	mergePipelineMetadata(metadata.Pipeline, pipeline)

	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to set metadata: %w", err)
	}

	if !isGitLabPipelineDone(pipeline.Status) {
		return http.StatusOK, nil
	}

	if err := r.emitPipelineResult(executionCtx, metadata); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (r *RunPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           GitLabRunPipelinePollAction,
			UserAccessible: false,
		},
	}
}

func (r *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case GitLabRunPipelinePollAction:
		return r.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (r *RunPipeline) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Pipeline == nil || metadata.Pipeline.ID == 0 {
		return fmt.Errorf("pipeline metadata is missing")
	}

	if isGitLabPipelineDone(metadata.Pipeline.Status) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(spec.Project, metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	mergePipelineMetadata(metadata.Pipeline, pipeline)
	if metadata.Pipeline.WebURL == "" && metadata.Pipeline.URL != "" {
		metadata.Pipeline.WebURL = metadata.Pipeline.URL
	}
	if metadata.Pipeline.URL == "" && metadata.Pipeline.WebURL != "" {
		metadata.Pipeline.URL = metadata.Pipeline.WebURL
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if !isGitLabPipelineDone(pipeline.Status) {
		return ctx.Requests.ScheduleActionCall(GitLabRunPipelinePollAction, map[string]any{}, GitLabRunPipelinePollInterval)
	}

	return r.emitPipelineResultInActionContext(ctx, metadata)
}

func (r *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (r *RunPipeline) buildInputs(inputs []RunPipelineInputSpec) []PipelineInput {
	result := make([]PipelineInput, 0, len(inputs))
	for _, input := range inputs {
		if strings.TrimSpace(input.Name) == "" {
			continue
		}

		result = append(result, PipelineInput{
			Name:  input.Name,
			Value: input.Value,
		})
	}

	return result
}

func (r *RunPipeline) buildVariables(variables []RunPipelineVariableSpec) []PipelineVariable {
	result := make([]PipelineVariable, 0, len(variables))
	for _, variable := range variables {
		if strings.TrimSpace(variable.Name) == "" {
			continue
		}
		result = append(result, PipelineVariable{
			Key:          variable.Name,
			Value:        variable.Value,
			VariableType: defaultPipelineVariableType(variable.VariableType),
		})
	}

	return result
}

func (r *RunPipeline) resolvePipelineURL(pipeline *Pipeline, rawNodeMetadata any) string {
	if pipeline.WebURL != "" {
		return pipeline.WebURL
	}

	if pipeline.URL != "" {
		return pipeline.URL
	}

	nodeMetadata := NodeMetadata{}
	if err := mapstructure.Decode(rawNodeMetadata, &nodeMetadata); err != nil {
		return ""
	}

	if nodeMetadata.Project == nil || nodeMetadata.Project.URL == "" {
		return ""
	}

	return fmt.Sprintf("%s/-/pipelines/%d", strings.TrimSuffix(nodeMetadata.Project.URL, "/"), pipeline.ID)
}

func (r *RunPipeline) emitPipelineResult(ctx *core.ExecutionContext, metadata RunPipelineExecutionMetadata) error {
	channel := GitLabPipelineFailedOutputChannel
	if metadata.Pipeline != nil && metadata.Pipeline.Status == GitLabPipelineStatusSuccess {
		channel = GitLabPipelinePassedOutputChannel
	}

	return ctx.ExecutionState.Emit(channel, GitLabPipelinePayloadType, []any{
		map[string]any{
			"pipeline": metadata.Pipeline,
		},
	})
}

func (r *RunPipeline) emitPipelineResultInActionContext(ctx core.ActionContext, metadata RunPipelineExecutionMetadata) error {
	channel := GitLabPipelineFailedOutputChannel
	if metadata.Pipeline != nil && metadata.Pipeline.Status == GitLabPipelineStatusSuccess {
		channel = GitLabPipelinePassedOutputChannel
	}

	return ctx.ExecutionState.Emit(channel, GitLabPipelinePayloadType, []any{
		map[string]any{
			"pipeline": metadata.Pipeline,
		},
	})
}

func metadataFromPipelineWebhookPayload(payload map[string]any) (*Pipeline, error) {
	attrs, ok := payload["object_attributes"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("pipeline attributes missing from webhook payload")
	}

	pipelineID, ok := intFromAny(attrs["id"])
	if !ok {
		return nil, fmt.Errorf("pipeline id missing from webhook payload")
	}

	status, ok := attrs["status"].(string)
	if !ok || status == "" {
		return nil, fmt.Errorf("pipeline status missing from webhook payload")
	}

	pipelineIID, _ := intFromAny(attrs["iid"])
	ref, _ := attrs["ref"].(string)
	url, _ := attrs["url"].(string)
	sha, _ := attrs["sha"].(string)
	source, _ := attrs["source"].(string)
	createdAt, _ := attrs["created_at"].(string)
	updatedAt, _ := attrs["updated_at"].(string)
	finishedAt, _ := attrs["finished_at"].(string)
	duration, _ := floatFromAny(attrs["duration"])
	projectID := projectIDFromPayload(payload)

	return &Pipeline{
		ID:         pipelineID,
		IID:        pipelineIID,
		ProjectID:  projectID,
		Status:     status,
		Source:     source,
		Ref:        ref,
		SHA:        sha,
		WebURL:     url,
		URL:        url,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		FinishedAt: finishedAt,
		Duration:   duration,
	}, nil
}

func projectIDFromPayload(payload map[string]any) int {
	project, ok := payload["project"].(map[string]any)
	if !ok {
		return 0
	}

	projectID, _ := intFromAny(project["id"])
	return projectID
}

func isGitLabPipelineDone(status string) bool {
	switch status {
	case GitLabPipelineStatusSuccess,
		GitLabPipelineStatusFailed,
		GitLabPipelineStatusCanceled,
		GitLabPipelineStatusCancelled,
		GitLabPipelineStatusSkipped,
		GitLabPipelineStatusManual,
		GitLabPipelineStatusBlocked:
		return true
	default:
		return false
	}
}

func floatFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func mergePipelineMetadata(target, incoming *Pipeline) {
	if target == nil || incoming == nil {
		return
	}

	if incoming.ID != 0 {
		target.ID = incoming.ID
	}
	if incoming.IID != 0 {
		target.IID = incoming.IID
	}
	if incoming.ProjectID != 0 {
		target.ProjectID = incoming.ProjectID
	}
	if incoming.Status != "" {
		target.Status = incoming.Status
	}
	if incoming.Source != "" {
		target.Source = incoming.Source
	}
	if incoming.Ref != "" {
		target.Ref = incoming.Ref
	}
	if incoming.SHA != "" {
		target.SHA = incoming.SHA
	}
	if incoming.BeforeSHA != "" {
		target.BeforeSHA = incoming.BeforeSHA
	}
	if incoming.WebURL != "" {
		target.WebURL = incoming.WebURL
	}
	if incoming.URL != "" {
		target.URL = incoming.URL
	}
	if incoming.CreatedAt != "" {
		target.CreatedAt = incoming.CreatedAt
	}
	if incoming.UpdatedAt != "" {
		target.UpdatedAt = incoming.UpdatedAt
	}
	if incoming.StartedAt != "" {
		target.StartedAt = incoming.StartedAt
	}
	if incoming.FinishedAt != "" {
		target.FinishedAt = incoming.FinishedAt
	}
	if incoming.CommittedAt != "" {
		target.CommittedAt = incoming.CommittedAt
	}
	if incoming.Duration != 0 {
		target.Duration = incoming.Duration
	}
	if incoming.QueuedDuration != 0 {
		target.QueuedDuration = incoming.QueuedDuration
	}
	if incoming.Coverage != "" {
		target.Coverage = incoming.Coverage
	}
	if incoming.User != nil {
		target.User = incoming.User
	}
	if incoming.DetailedStatus != nil {
		target.DetailedStatus = incoming.DetailedStatus
	}
	if incoming.YamlErrors != nil {
		target.YamlErrors = incoming.YamlErrors
	}
	if incoming.Tag {
		target.Tag = true
	}
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func defaultPipelineVariableType(variableType string) string {
	if variableType == "file" {
		return "file"
	}
	return "env_var"
}
