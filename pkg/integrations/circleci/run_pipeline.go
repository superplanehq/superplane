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
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	ProjectName string `json:"projectName" mapstructure:"projectName"`
}

type RunPipelineExecutionMetadata struct {
	Pipeline  *PipelineInfo  `json:"pipeline" mapstructure:"pipeline"`
	Workflows []WorkflowInfo `json:"workflows" mapstructure:"workflows"`
}

type PipelineInfo struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	CreatedAt string `json:"created_at"`
}

type WorkflowInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type RunPipelineSpec struct {
	ProjectSlug          string      `json:"projectSlug"`
	Location             string      `json:"location"`
	PipelineDefinitionID string      `json:"pipelineDefinitionId"` // Find in CircleCI: Project Settings → Project Setup
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
2. Waits for all workflows in the pipeline to complete (monitored via webhook and polling)
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
			Description: "CircleCI project slug (e.g. gh/org/repo). Find in CircleCI project settings.",
		},
		{
			Name:  "location",
			Label: "Location",
			Type:  configuration.FieldTypeGitRef,
		},
		{
			Name:        "pipelineDefinitionId",
			Label:       "Pipeline definition ID",
			Type:        configuration.FieldTypeString,
			Description: "Find in CircleCI: Project Settings → Project Setup.",
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

	if metadata.ProjectSlug == config.ProjectSlug {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	project, err := client.GetProject(config.ProjectSlug)
	if err != nil {
		return fmt.Errorf("project not found or inaccessible: %w", err)
	}

	err = ctx.Metadata.Set(RunPipelineNodeMetadata{
		ProjectSlug: config.ProjectSlug,
		ProjectName: project.Name,
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	// Request webhook for this project
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

	params := RunPipelineParams{
		Parameters: t.buildParameters(ctx, spec.Parameters),
	}
	var branch, tag string
	if spec.Location != "" {
		switch {
		case strings.HasPrefix(spec.Location, "refs/tags/"):
			tag = strings.TrimPrefix(spec.Location, "refs/tags/")
			params.Tag = tag
		case strings.HasPrefix(spec.Location, "ref/tags/"):
			tag = strings.TrimPrefix(spec.Location, "ref/tags/")
			params.Tag = tag
		case strings.HasPrefix(spec.Location, "refs/heads/"):
			branch = strings.TrimPrefix(spec.Location, "refs/heads/")
			params.Branch = branch
		case strings.HasPrefix(spec.Location, "ref/heads/"):
			branch = strings.TrimPrefix(spec.Location, "ref/heads/")
			params.Branch = branch
		default:
			branch = strings.TrimSpace(spec.Location)
			params.Branch = branch
		}
	}

	var response *RunPipelineResponse
	if spec.PipelineDefinitionID != "" {
		// Use pipeline/run API (required for GitHub App and Bitbucket Data Center)
		runParams := RunPipelineRunParams{
			DefinitionID: strings.TrimSpace(spec.PipelineDefinitionID),
			Parameters:   t.buildParameters(ctx, spec.Parameters),
		}
		if tag != "" {
			runParams.Config = map[string]string{"tag": tag}
			runParams.Checkout = map[string]string{"tag": tag}
		} else {
			if branch == "" {
				branch = "main"
			}
			runParams.Config = map[string]string{"branch": branch}
			runParams.Checkout = map[string]string{"branch": branch}
		}
		response, err = client.RunPipelineRun(spec.ProjectSlug, runParams)
	} else {
		response, err = client.RunPipeline(spec.ProjectSlug, params)
	}
	if err != nil {
		if strings.Contains(err.Error(), "400") &&
			(strings.Contains(err.Error(), "GitHub App") || strings.Contains(err.Error(), "not yet supported")) {
			return fmt.Errorf("GitHub App project: add Pipeline Definition ID (CircleCI Project Settings → Project Setup). See https://circleci.com/docs/triggers-overview/#run-a-pipeline-using-the-api")
		}
		return fmt.Errorf("error triggering pipeline: %v", err)
	}

	// Store pipeline info in metadata
	err = ctx.Metadata.Set(RunPipelineExecutionMetadata{
		Pipeline: &PipelineInfo{
			ID:        response.ID,
			Number:    response.Number,
			CreatedAt: response.CreatedAt,
		},
		Workflows: []WorkflowInfo{},
	})
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	// Associate pipeline ID with this execution for webhook handling
	err = ctx.ExecutionState.SetKV("pipeline", response.ID)
	if err != nil {
		return err
	}

	// Schedule polling as fallback
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (t *RunPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *RunPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Verify webhook signature first before any processing
	// CircleCI sends signature as "v1=<hex>" format
	signatureHeader := ctx.Headers.Get("circleci-signature")
	if signatureHeader == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	// Parse "v1=<hex>" format - extract just the hex part
	signature := signatureHeader
	if strings.HasPrefix(signatureHeader, "v1=") {
		signature = strings.TrimPrefix(signatureHeader, "v1=")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	// Parse webhook payload (same as OnPipelineCompleted)
	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}
	if eventType, _ := data["type"].(string); eventType != "workflow-completed" {
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

	workflowID, _ := workflowData["id"].(string)
	workflowName, _ := workflowData["name"].(string)
	workflowStatus, _ := workflowData["status"].(string)

	if workflowID == "" || workflowStatus == "" {
		return http.StatusBadRequest, fmt.Errorf("workflow data incomplete")
	}

	metadata := RunPipelineExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil
	}

	found := false
	for i, w := range metadata.Workflows {
		if w.ID == workflowID {
			metadata.Workflows[i].Status = workflowStatus
			found = true
			break
		}
	}

	if !found {
		metadata.Workflows = append(metadata.Workflows, WorkflowInfo{
			ID:     workflowID,
			Name:   workflowName,
			Status: workflowStatus,
		})
	}

	err = executionCtx.Metadata.Set(metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	var allDone, anyFailed bool
	if executionCtx.Integration != nil {
		client, err := NewClient(executionCtx.HTTP, executionCtx.Integration)
		if err == nil {
			allWorkflows, err := client.GetPipelineWorkflows(metadata.Pipeline.ID)
			if err == nil {
				updatedWorkflows := make([]WorkflowInfo, 0, len(allWorkflows))
				for _, w := range allWorkflows {
					updatedWorkflows = append(updatedWorkflows, WorkflowInfo{
						ID:     w.ID,
						Name:   w.Name,
						Status: w.Status,
					})
				}
				metadata.Workflows = updatedWorkflows
				_ = executionCtx.Metadata.Set(metadata)
				allDone, anyFailed = t.checkWorkflowsStatus(updatedWorkflows)
			}
		}
	}
	// When Integration is nil (e.g. webhook context from server doesn't set it), or API call failed,
	// use only the workflow data we have from webhook events (same as Semaphore Run Workflow).
	if !allDone {
		allDone, anyFailed = t.checkWorkflowsStatus(metadata.Workflows)
	}
	if !allDone {
		return http.StatusOK, nil
	}

	payload := map[string]any{
		"pipeline":  metadata.Pipeline,
		"workflows": metadata.Workflows,
	}

	if anyFailed {
		err = executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	} else {
		err = executionCtx.ExecutionState.Emit(SuccessOutputChannel, PayloadType, []any{payload})
	}

	if err != nil {
		return http.StatusInternalServerError, err
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	workflows, err := client.GetPipelineWorkflows(metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	updatedWorkflows := []WorkflowInfo{}
	for _, w := range workflows {
		updatedWorkflows = append(updatedWorkflows, WorkflowInfo{
			ID:     w.ID,
			Name:   w.Name,
			Status: w.Status,
		})
	}

	metadata.Workflows = updatedWorkflows
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	allDone, anyFailed := t.checkWorkflowsStatus(updatedWorkflows)

	if !allDone {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	payload := map[string]any{
		"pipeline":  metadata.Pipeline,
		"workflows": metadata.Workflows,
	}

	if anyFailed {
		return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(SuccessOutputChannel, PayloadType, []any{payload})
}

func (t *RunPipeline) checkWorkflowsStatus(workflows []WorkflowInfo) (allDone bool, anyFailed bool) {
	if len(workflows) == 0 {
		return false, false
	}

	allDone = true
	anyFailed = false

	for _, w := range workflows {
		if w.Status == "running" || w.Status == "on_hold" || w.Status == "not_run" || w.Status == "failing" {
			allDone = false
		}
		if w.Status == WorkflowStatusFailed || w.Status == WorkflowStatusCanceled || w.Status == "error" || w.Status == "failing" || w.Status == "unauthorized" {
			anyFailed = true
		}
	}

	return allDone, anyFailed
}

func (t *RunPipeline) buildParameters(ctx core.ExecutionContext, params []Parameter) map[string]string {
	parameters := make(map[string]string)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	parameters["SUPERPLANE_EXECUTION_ID"] = ctx.ID.String()
	parameters["SUPERPLANE_CANVAS_ID"] = ctx.WorkflowID

	return parameters
}

func (t *RunPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
