package circleci

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
	"github.com/superplanehq/superplane/pkg/crypto"
)

const PayloadType = "circleci.workflow.completed"
const SuccessOutputChannel = "success"
const FailedOutputChannel = "failed"
const WorkflowStatusSuccess = "success"
const WorkflowStatusFailed = "failed"
const WorkflowStatusCanceled = "canceled"
const PollInterval = 5 * time.Minute

type RunPipeline struct{}

type RunPipelineNodeMetadata struct {
	ProjectID              string `json:"projectId" mapstructure:"projectId"`
	ProjectSlug            string `json:"projectSlug" mapstructure:"projectSlug"`
	ProjectName            string `json:"projectName" mapstructure:"projectName"`
	PipelineDefinitionID   string `json:"pipelineDefinitionId" mapstructure:"pipelineDefinitionId"`
	PipelineDefinitionName string `json:"pipelineDefinitionName" mapstructure:"pipelineDefinitionName"`
}

type RunPipelineExecutionMetadata struct {
	Pipeline PipelineInfo `json:"pipeline" mapstructure:"pipeline"`
}

type PipelineInfo struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	CreatedAt   string `json:"created_at"`
	PipelineURL string `json:"pipeline_url"`
}

type WorkflowInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type RunPipelineSpec struct {
	ProjectSlug          string      `json:"projectSlug"`
	Location             string      `json:"location"`
	PipelineDefinitionID string      `json:"pipelineDefinitionId"`
	Parameters           []Parameter `json:"parameters"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (t *RunPipeline) Name() string {
	return "circleci.runPipeline"
}

func (t *RunPipeline) Label() string {
	return "Run Pipeline"
}

func (t *RunPipeline) Description() string {
	return "Run a CircleCI pipeline and wait for completion"
}

func (t *RunPipeline) Documentation() string {
	return `The Run Pipeline component starts a CircleCI pipeline and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger builds and deployments from SuperPlane workflows
- **Pipeline automation**: Run CircleCI pipelines as part of workflow automation
- **Multi-stage deployments**: Coordinate complex deployment pipelines
- **Workflow chaining**: Chain multiple CircleCI workflows together

## How It Works

1. Triggers a CircleCI pipeline with the specified location (branch or tag) and parameters
2. Waits for all workflows in the pipeline to complete (monitored via webhook)
3. Routes execution based on workflow results:
   - **Success channel**: All workflows completed successfully
   - **Failed channel**: Any workflow failed or was cancelled

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/username/repo)
- **Location**: Branch or tag to run the pipeline
- **Pipeline definition ID**: Find in CircleCI: Project Settings → Project Setup.
- **Parameters**: Optional pipeline parameters as key-value pairs (supports expressions)

## Output Channels

- **Success**: Emitted when all workflows complete successfully
- **Failed**: Emitted when any workflow fails or is cancelled
`
}

func (t *RunPipeline) Icon() string {
	return "workflow"
}

func (t *RunPipeline) Color() string {
	return "gray"
}

func (t *RunPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  SuccessOutputChannel,
			Label: "Success",
		},
		{
			Name:  FailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (t *RunPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug. Find in CircleCI project settings.",
		},
		{
			Name:    "location",
			Label:   "Location",
			Type:    configuration.FieldTypeGitRef,
			Default: "refs/heads/main",
		},
		{
			Name:        "pipelineDefinitionId",
			Label:       "Pipeline definition",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select a pipeline definition from the project.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePipelineDefinition,
					Parameters: []configuration.ParameterRef{
						{
							Name: "projectSlug",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "projectSlug",
							},
						},
					},
				},
			},
		},
		{
			Name:  "parameters",
			Label: "Parameters",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Parameter",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Name",
								Type:     configuration.FieldTypeString,
								Required: true,
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

func (t *RunPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *RunPipeline) Setup(ctx core.SetupContext) error {
	config := RunPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	metadata := RunPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if strings.TrimSpace(config.PipelineDefinitionID) == "" {
		return fmt.Errorf("pipeline definition ID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Check if project changed
	projectChanged := metadata.ProjectSlug != config.ProjectSlug
	var project *ProjectResponse
	if projectChanged {
		project, err = client.GetProject(config.ProjectSlug)
		if err != nil {
			return fmt.Errorf("project not found or inaccessible: %w", err)
		}
	} else {
		// Use existing project ID from metadata
		project = &ProjectResponse{
			ID:   metadata.ProjectID,
			Slug: metadata.ProjectSlug,
			Name: metadata.ProjectName,
		}
	}

	// Check if pipeline definition changed
	pipelineDefinitionChanged := metadata.PipelineDefinitionID != config.PipelineDefinitionID
	var pipelineDefinitionName string
	if pipelineDefinitionChanged || projectChanged {
		// Fetch pipeline definitions to get the name
		definitions, err := client.GetPipelineDefinitions(project.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch pipeline definitions: %w", err)
		}

		// Find the matching pipeline definition
		found := false
		for _, def := range definitions {
			if def.ID == config.PipelineDefinitionID {
				pipelineDefinitionName = def.Name
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("pipeline definition with ID %s not found", config.PipelineDefinitionID)
		}
	} else {
		// Use existing pipeline definition name from metadata
		pipelineDefinitionName = metadata.PipelineDefinitionName
	}

	// Always update metadata to ensure it's current (in case it was reset or cleared)
	err = ctx.Metadata.Set(RunPipelineNodeMetadata{
		ProjectID:              project.ID,
		ProjectSlug:            project.Slug,
		ProjectName:            project.Name,
		PipelineDefinitionID:   config.PipelineDefinitionID,
		PipelineDefinitionName: pipelineDefinitionName,
	})
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	if !IsValidLocation(config.Location) {
		return fmt.Errorf("branch or tag is required, got: %s", config.Location)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: config.ProjectSlug,
		Events:      []string{"workflow-completed"},
	})
}

func (t *RunPipeline) Execute(ctx core.ExecutionContext) error {
	spec := RunPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	runPipelineConfig := map[string]string{}
	if t.getBranch(spec.Location) != "" {
		runPipelineConfig["branch"] = t.getBranch(spec.Location)
	}
	if t.getTag(spec.Location) != "" {
		runPipelineConfig["tag"] = t.getTag(spec.Location)
	}

	runParams := RunPipelineParams{
		DefinitionID: strings.TrimSpace(spec.PipelineDefinitionID),
		Parameters:   t.buildParameters(spec.Parameters),
		Config:       runPipelineConfig,
		Checkout:     runPipelineConfig,
	}

	var response *RunPipelineResponse
	response, err = client.RunPipeline(spec.ProjectSlug, runParams)
	if err != nil {
		return fmt.Errorf("failed to run pipeline: %w", err)
	}

	metadata := RunPipelineExecutionMetadata{
		Pipeline: PipelineInfo{
			ID:          response.ID,
			Number:      response.Number,
			CreatedAt:   response.CreatedAt,
			PipelineURL: fmt.Sprintf("https://app.circleci.com/pipelines/%s/%d", spec.ProjectSlug, response.Number),
		},
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	err = ctx.ExecutionState.SetKV("pipeline", response.ID)
	if err != nil {
		return err
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (t *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signatureHeader := ctx.Headers.Get("circleci-signature")
	if signatureHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	signature, _ := strings.CutPrefix(signatureHeader, "v1=")

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	eventType, _ := data["type"].(string)
	if eventType != "workflow-completed" {
		return http.StatusOK, nil
	}

	pipelineData, ok := data["pipeline"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("pipeline data missing from webhook payload")
	}

	pipelineID, _ := pipelineData["id"].(string)
	if pipelineID == "" {
		return http.StatusBadRequest, fmt.Errorf("pipeline id missing from webhook payload")
	}

	executionCtx, err := ctx.FindExecutionByKV("pipeline", pipelineID)
	if err != nil {
		return http.StatusOK, nil
	}

	workflowData, ok := data["workflow"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("workflow data missing from webhook payload")
	}

	workflowStatus, _ := workflowData["status"].(string)

	var metadata RunPipelineExecutionMetadata
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode metadata: %w", err)
	}

	if t.isFailed(workflowStatus) {
		payload := map[string]any{"pipeline": metadata.Pipeline}
		err = executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to emit output: %w", err)
		}

		return http.StatusOK, nil
	}

	err = executionCtx.Requests.ScheduleActionCall("poll", map[string]any{}, 3*time.Second)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to schedule poll action: %w", err)
	}

	return http.StatusOK, nil
}

func (t *RunPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (t *RunPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return t.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *RunPipeline) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunPipelineExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Pipeline.ID == "" {
		return fmt.Errorf("pipeline ID is missing from execution metadata")
	}

	// Always fetch workflows fresh
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	if pipeline.State == "errored" {
		payload := map[string]any{"pipeline": pipeline}
		err = ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
		if err != nil {
			return fmt.Errorf("failed to emit output: %w", err)
		}

		return nil
	}

	workflows, err := client.GetPipelineWorkflows(metadata.Pipeline.ID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	if len(workflows) == 0 {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	firstWorkflow := workflows[0]

	if t.isRunning(firstWorkflow.Status) {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	payload := map[string]any{"pipeline": metadata.Pipeline}
	channel := SuccessOutputChannel
	if t.isFailed(firstWorkflow.Status) {
		channel = FailedOutputChannel
	}

	return ctx.ExecutionState.Emit(channel, PayloadType, []any{payload})
}

func (t *RunPipeline) buildParameters(params []Parameter) map[string]string {
	parameters := make(map[string]string)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	return parameters
}

func (t *RunPipeline) getBranch(location string) string {
	if strings.HasPrefix(location, "refs/heads/") {
		return strings.TrimPrefix(location, "refs/heads/")
	}

	return ""
}

func (t *RunPipeline) getTag(location string) string {
	if strings.HasPrefix(location, "refs/tags/") {
		return strings.TrimPrefix(location, "refs/tags/")
	}

	return ""
}

func IsValidLocation(location string) bool {
	return strings.HasPrefix(location, "refs/heads/") || strings.HasPrefix(location, "refs/tags/")
}

func (t *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (t *RunPipeline) isFailed(workflowStatus string) bool {
	return workflowStatus == WorkflowStatusFailed || workflowStatus == WorkflowStatusCanceled || workflowStatus == "error" || workflowStatus == "failing" || workflowStatus == "unauthorized"
}

func (t *RunPipeline) isRunning(workflowStatus string) bool {
	return workflowStatus == "running" || workflowStatus == "on_hold" || workflowStatus == "not_run"
}
