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
const PollInterval = 10 * time.Second

type TriggerPipeline struct{}

type TriggerPipelineNodeMetadata struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
}

type TriggerPipelineExecutionMetadata struct {
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

type TriggerPipelineSpec struct {
	ProjectSlug string      `json:"projectSlug"`
	Branch      string      `json:"branch"`
	Tag         string      `json:"tag"`
	Parameters  []Parameter `json:"parameters"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (t *TriggerPipeline) Name() string {
	return "circleci.triggerPipeline"
}

func (t *TriggerPipeline) Label() string {
	return "Trigger Pipeline"
}

func (t *TriggerPipeline) Description() string {
	return "Trigger a CircleCI pipeline and wait for completion"
}

func (t *TriggerPipeline) Documentation() string {
	return `The Trigger Pipeline component starts a CircleCI pipeline and waits for it to complete.

## Use Cases

- **CI/CD orchestration**: Trigger builds and deployments from SuperPlane workflows
- **Pipeline automation**: Run CircleCI pipelines as part of workflow automation
- **Multi-stage deployments**: Coordinate complex deployment pipelines
- **Workflow chaining**: Chain multiple CircleCI workflows together

## How It Works

1. Triggers a CircleCI pipeline with the specified branch or tag and parameters
2. Waits for all workflows in the pipeline to complete (monitored via webhook and polling)
3. Routes execution based on workflow results:
   - **Success channel**: All workflows completed successfully
   - **Failed channel**: Any workflow failed or was cancelled

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/username/repo)
- **Branch**: Git branch to run the pipeline on (optional)
- **Tag**: Git tag to run the pipeline on (optional)
- **Parameters**: Optional pipeline parameters as key-value pairs (supports expressions)

## Output Channels

- **Success**: Emitted when all workflows complete successfully
- **Failed**: Emitted when any workflow fails or is cancelled

## Notes

- The component automatically sets up webhook monitoring for workflow completion
- Falls back to polling if webhook doesn't arrive
- SuperPlane execution ID and canvas ID are automatically injected as pipeline parameters`
}

func (t *TriggerPipeline) Icon() string {
	return "workflow"
}

func (t *TriggerPipeline) Color() string {
	return "gray"
}

func (t *TriggerPipeline) OutputChannels(configuration any) []core.OutputChannel {
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

func (t *TriggerPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g., gh/username/repo)",
			Placeholder: "gh/username/repo",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Git branch to run pipeline on",
			Placeholder: "main",
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Description: "Git tag to run pipeline on",
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

func (t *TriggerPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *TriggerPipeline) Setup(ctx core.SetupContext) error {
	config := TriggerPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	metadata := TriggerPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If this is the same project, nothing to do
	if metadata.ProjectSlug == config.ProjectSlug {
		return nil
	}

	err = ctx.Metadata.Set(TriggerPipelineNodeMetadata{
		ProjectSlug: config.ProjectSlug,
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	// Request webhook for this project
	ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: config.ProjectSlug,
	})

	return nil
}

func (t *TriggerPipeline) Execute(ctx core.ExecutionContext) error {
	spec := TriggerPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Build pipeline parameters
	params := TriggerPipelineParams{
		Parameters: t.buildParameters(ctx, spec.Parameters),
	}

	if spec.Branch != "" {
		params.Branch = spec.Branch
	}

	if spec.Tag != "" {
		params.Tag = spec.Tag
	}

	// Trigger the pipeline
	response, err := client.TriggerPipeline(spec.ProjectSlug, params)
	if err != nil {
		return fmt.Errorf("error triggering pipeline: %v", err)
	}

	ctx.Logger.Infof("Pipeline triggered - id=%s, number=%d", response.ID, response.Number)

	// Store pipeline info in metadata
	ctx.Metadata.Set(TriggerPipelineExecutionMetadata{
		Pipeline: &PipelineInfo{
			ID:        response.ID,
			Number:    response.Number,
			CreatedAt: response.CreatedAt,
		},
		Workflows: []WorkflowInfo{},
	})

	// Associate pipeline ID with this execution for webhook handling
	err = ctx.ExecutionState.SetKV("pipeline", response.ID)
	if err != nil {
		return err
	}

	// Schedule polling as fallback
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (t *TriggerPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *TriggerPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Verify webhook signature
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

	var payload map[string]any
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Get pipeline data from payload
	pipelineData, ok := payload["pipeline"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("pipeline data missing from webhook payload")
	}

	pipelineID, _ := pipelineData["id"].(string)
	if pipelineID == "" {
		return http.StatusBadRequest, fmt.Errorf("pipeline id missing from webhook payload")
	}

	// Find the execution associated with this pipeline
	executionCtx, err := ctx.FindExecutionByKV("pipeline", pipelineID)
	if err != nil {
		// Not a pipeline we triggered, ignore
		return http.StatusOK, nil
	}

	// Get workflow data
	workflowData, ok := payload["workflow"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("workflow data missing from webhook payload")
	}

	workflowID, _ := workflowData["id"].(string)
	workflowName, _ := workflowData["name"].(string)
	workflowStatus, _ := workflowData["status"].(string)

	if workflowID == "" || workflowStatus == "" {
		return http.StatusBadRequest, fmt.Errorf("workflow data incomplete")
	}

	// Update execution metadata with workflow info
	metadata := TriggerPipelineExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	// Check if execution already finished
	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil
	}

	// Update or add workflow status
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

	// Check if all workflows are done
	allDone, anyFailed := t.checkWorkflowsStatus(metadata.Workflows)

	if !allDone {
		// Not all workflows finished yet
		return http.StatusOK, nil
	}

	// All workflows done, emit result
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

func (t *TriggerPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (t *TriggerPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return t.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *TriggerPipeline) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := TriggerPipelineExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Get workflows for the pipeline
	workflows, err := client.GetPipelineWorkflows(metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	// Update metadata with workflow statuses
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

	// Check if all workflows are done
	allDone, anyFailed := t.checkWorkflowsStatus(updatedWorkflows)

	if !allDone {
		// Schedule next poll
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	// All workflows done, emit result
	payload := map[string]any{
		"pipeline":  metadata.Pipeline,
		"workflows": metadata.Workflows,
	}

	if anyFailed {
		return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(SuccessOutputChannel, PayloadType, []any{payload})
}

func (t *TriggerPipeline) checkWorkflowsStatus(workflows []WorkflowInfo) (allDone bool, anyFailed bool) {
	if len(workflows) == 0 {
		return false, false
	}

	allDone = true
	anyFailed = false

	for _, w := range workflows {
		if w.Status == "running" || w.Status == "on_hold" || w.Status == "not_run" {
			allDone = false
		}
		if w.Status == WorkflowStatusFailed || w.Status == WorkflowStatusCanceled {
			anyFailed = true
		}
	}

	return allDone, anyFailed
}

func (t *TriggerPipeline) buildParameters(ctx core.ExecutionContext, params []Parameter) map[string]string {
	parameters := make(map[string]string)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	parameters["SUPERPLANE_EXECUTION_ID"] = ctx.ID.String()
	parameters["SUPERPLANE_CANVAS_ID"] = ctx.WorkflowID

	return parameters
}

func (t *TriggerPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
