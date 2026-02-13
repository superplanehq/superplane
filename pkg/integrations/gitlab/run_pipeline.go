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
	PipelinePayloadType         = "gitlab.pipeline.finished"
	PipelinePassedOutputChannel = "passed"
	PipelineFailedOutputChannel = "failed"

	RunPipelinePollInterval = 5 * time.Minute
	RunPipelinePollAction   = "poll"
	RunPipelineKVPipelineID = "pipeline_id"
)

type RunPipeline struct{}

type RunPipelineSpec struct {
	Project string                 `json:"project" mapstructure:"project"`
	Ref     string                 `json:"ref" mapstructure:"ref"`
	Inputs  []RunPipelineInputSpec `json:"inputs" mapstructure:"inputs"`
}

type RunPipelineInputSpec struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type RunPipelineExecutionMetadata struct {
	Pipeline *PipelineMetadata `json:"pipeline" mapstructure:"pipeline"`
}

type PipelineMetadata struct {
	ID     int    `json:"id"`
	IID    int    `json:"iid"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
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
- **Deployment automation**: Run deployment pipelines with inputs
- **Pipeline chaining**: Coordinate follow-up actions after pipeline completion`
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
			Name:  PipelinePassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  PipelineFailedOutputChannel,
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

	nodeMetadata := NodeMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.CreatePipeline(context.Background(), spec.Project, &CreatePipelineRequest{
		Ref:    normalizePipelineRef(spec.Ref),
		Inputs: r.buildInputs(spec.Inputs),
	})

	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	metadata := RunPipelineExecutionMetadata{Pipeline: &PipelineMetadata{
		ID:     pipeline.ID,
		IID:    pipeline.IID,
		Status: pipeline.Status,
		URL:    pipeline.WebURL,
	}}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(RunPipelineKVPipelineID, strconv.Itoa(pipeline.ID)); err != nil {
		return err
	}

	ctx.Logger.Infof("Started GitLab pipeline %d on project %s (ref=%s)", pipeline.ID, spec.Project, spec.Ref)
	return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
}

func (r *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Pipeline == nil || metadata.Pipeline.ID == 0 {
		ctx.Logger.Info("No pipeline to cancel")
		return nil
	}

	if isPipelineDone(metadata.Pipeline.Status) {
		ctx.Logger.Infof("Pipeline %d already done - %s", metadata.Pipeline.ID, metadata.Pipeline.Status)
		return nil
	}

	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.CancelPipeline(context.Background(), spec.Project, metadata.Pipeline.ID); err != nil {
		return fmt.Errorf("failed to cancel pipeline: %w", err)
	}

	err = ctx.Metadata.Set(RunPipelineExecutionMetadata{Pipeline: &PipelineMetadata{
		ID:     metadata.Pipeline.ID,
		IID:    metadata.Pipeline.IID,
		URL:    metadata.Pipeline.URL,
		Status: PipelineStatusCanceled,
	}})

	if err != nil {
		return err
	}

	ctx.Logger.Infof("Cancel request sent for pipeline %d", metadata.Pipeline.ID)
	return nil
}

func (r *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	spec := RunPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

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

	newMetadata, err := metadataFromWebhook(payload)
	if err != nil {
		return http.StatusBadRequest, err
	}

	executionCtx, err := ctx.FindExecutionByKV(RunPipelineKVPipelineID, strconv.Itoa(newMetadata.Pipeline.ID))

	//
	// Ignore hooks for pipelines not started by SuperPlane
	//
	if err != nil {
		return http.StatusOK, nil
	}

	metadata := RunPipelineExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	//
	// If pipeline is already done, do nothing.
	//
	if metadata.Pipeline != nil && isPipelineDone(metadata.Pipeline.Status) {
		ctx.Logger.Infof("Pipeline %d is already done - %s", newMetadata.Pipeline.ID, metadata.Pipeline.Status)
		return http.StatusOK, nil
	}

	//
	// Set new metadata
	//
	if err := executionCtx.Metadata.Set(newMetadata); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to set metadata: %w", err)
	}

	//
	// If pipeline is not done, do not complete execution and emit yet.
	//
	if !isPipelineDone(newMetadata.Pipeline.Status) {
		ctx.Logger.Infof("Pipeline %d is not done - %s", newMetadata.Pipeline.ID, newMetadata.Pipeline.Status)
		return http.StatusOK, nil
	}

	//
	// Fetch pipeline from API so we have the latest status,
	// and so the data emitted by webhook update and by polling is the same.
	//
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create client: %w", err)
	}

	pipeline, err := client.GetPipeline(spec.Project, newMetadata.Pipeline.ID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get pipeline: %w", err)
	}

	//
	// Emit on proper channel
	//
	channel := PipelineFailedOutputChannel
	if pipeline.Status == PipelineStatusSuccess {
		channel = PipelinePassedOutputChannel
	}

	err = executionCtx.ExecutionState.Emit(channel, PipelinePayloadType, []any{
		map[string]any{
			"pipeline": pipeline,
		},
	})

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit pipeline result: %w", err)
	}

	ctx.Logger.Infof("Pipeline %d completed - %s", pipeline.ID, pipeline.Status)
	return http.StatusOK, nil
}

func (r *RunPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           RunPipelinePollAction,
			UserAccessible: false,
		},
	}
}

func (r *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case RunPipelinePollAction:
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

	//
	// If pipeline is already done, do nothing.
	//
	if isPipelineDone(metadata.Pipeline.Status) {
		return nil
	}

	//
	// Otherwise, poll, update metadata and emit result if pipeline is done.
	//
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(spec.Project, metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	newMetadata := RunPipelineExecutionMetadata{Pipeline: &PipelineMetadata{
		ID:     pipeline.ID,
		IID:    pipeline.IID,
		Status: pipeline.Status,
		URL:    pipeline.URL,
	}}

	if err := ctx.Metadata.Set(newMetadata); err != nil {
		return err
	}

	if !isPipelineDone(pipeline.Status) {
		return ctx.Requests.ScheduleActionCall(RunPipelinePollAction, map[string]any{}, RunPipelinePollInterval)
	}

	channel := PipelineFailedOutputChannel
	if metadata.Pipeline != nil && metadata.Pipeline.Status == PipelineStatusSuccess {
		channel = PipelinePassedOutputChannel
	}

	return ctx.ExecutionState.Emit(channel, PipelinePayloadType, []any{
		map[string]any{
			"pipeline": pipeline,
		},
	})
}

func (r *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (r *RunPipeline) buildInputs(inputs []RunPipelineInputSpec) map[string]string {
	result := make(map[string]string, len(inputs))
	for _, input := range inputs {
		if strings.TrimSpace(input.Name) == "" {
			continue
		}

		result[input.Name] = input.Value
	}

	return result
}

func metadataFromWebhook(payload map[string]any) (*RunPipelineExecutionMetadata, error) {
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
	url, _ := attrs["url"].(string)

	return &RunPipelineExecutionMetadata{
		Pipeline: &PipelineMetadata{
			ID:     pipelineID,
			IID:    pipelineIID,
			Status: status,
			URL:    url,
		},
	}, nil
}

func isPipelineDone(status string) bool {
	switch status {
	case PipelineStatusSuccess,
		PipelineStatusFailed,
		PipelineStatusCanceled,
		PipelineStatusCancelled,
		PipelineStatusSkipped,
		PipelineStatusManual,
		PipelineStatusBlocked:
		return true
	default:
		return false
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
