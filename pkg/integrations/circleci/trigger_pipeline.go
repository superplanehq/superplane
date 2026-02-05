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

const PayloadType = "circleci.workflow.finished"
const SuccessOutputChannel = "success"
const FailedOutputChannel = "failed"
const WorkflowStatusSuccess = "success"
const WorkflowStatusFailed = "failed"
const WorkflowStatusCanceled = "canceled"
const PollInterval = 5 * time.Minute

type TriggerPipeline struct{}

type TriggerPipelineNodeMetadata struct {
	Project *ProjectInfo `json:"project" mapstructure:"project"`
}

type TriggerPipelineExecutionMetadata struct {
	Pipeline *PipelineInfo  `json:"pipeline" mapstructure:"pipeline"`
	Workflow *WorkflowInfo  `json:"workflow,omitempty" mapstructure:"workflow,omitempty"`
	Extra    map[string]any `json:"extra,omitempty" mapstructure:"extra,omitempty"`
}

type PipelineInfo struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

type WorkflowInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url"`
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

func (c *TriggerPipeline) Name() string {
	return "circleci.triggerPipeline"
}

func (c *TriggerPipeline) Label() string {
	return "Trigger Pipeline"
}

func (c *TriggerPipeline) Description() string {
	return "Trigger a CircleCI pipeline and wait for completion"
}

func (c *TriggerPipeline) Documentation() string {
	return `The Trigger Pipeline component starts a CircleCI pipeline and waits for its workflow to complete.

## Use Cases

- **CI/CD orchestration**: Trigger builds and deployments from SuperPlane workflows
- **Pipeline automation**: Run CircleCI pipelines as part of workflow automation
- **Multi-stage deployments**: Coordinate complex deployment pipelines
- **Workflow chaining**: Chain multiple CircleCI pipelines together

## How It Works

1. Triggers a CircleCI pipeline with the specified branch/tag and parameters
2. Waits for the pipeline's workflows to complete (monitored via webhook and polling)
3. Routes execution based on workflow result:
   - **Success channel**: All workflows completed successfully
   - **Failed channel**: Any workflow failed or was cancelled

## Configuration

- **Project Slug**: The CircleCI project slug (e.g., gh/org/repo)
- **Branch**: Git branch to run the pipeline on (mutually exclusive with Tag)
- **Tag**: Git tag to run the pipeline on (mutually exclusive with Branch)
- **Parameters**: Optional pipeline parameters as key-value pairs (supports expressions)

## Output Channels

- **Success**: Emitted when all workflows complete successfully
- **Failed**: Emitted when any workflow fails or is cancelled

## Notes

- The component automatically sets up webhook monitoring for workflow completion
- Falls back to polling if webhook doesn't arrive
- Can be cancelled, which will stop monitoring (but won't stop the CircleCI pipeline)`
}

func (c *TriggerPipeline) Icon() string {
	return "circleci"
}

func (c *TriggerPipeline) Color() string {
	return "gray"
}

func (c *TriggerPipeline) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "workflow-completed",
		"workflow": map[string]any{
			"id":         "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
			"name":       "build-test-deploy",
			"status":     "success",
			"created_at": "2021-09-01T22:49:03.616Z",
			"stopped_at": "2021-09-01T22:49:34.170Z",
			"url":        "https://app.circleci.com/pipelines/github/circleci/webhook-service/130/workflows/fda08377-fe7e-46b1-8992-3a7aaecac9c3",
		},
		"pipeline": map[string]any{
			"id":         "1285fe1d-d3a6-44fc-8886-8979558254c4",
			"number":     130,
			"created_at": "2021-09-01T22:49:03.544Z",
		},
		"project": map[string]any{
			"id":   "84996744-a854-4f5e-aea3-04e2851dc1d2",
			"name": "webhook-service",
			"slug": "github/circleci/webhook-service",
		},
	}
}

func (c *TriggerPipeline) OutputChannels(configuration any) []core.OutputChannel {
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

func (c *TriggerPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g., gh/org/repo)",
			Placeholder: "e.g. gh/myorg/myrepo",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Git branch to run the pipeline on",
			Placeholder: "e.g. main",
		},
		{
			Name:        "tag",
			Label:       "Tag",
			Type:        configuration.FieldTypeString,
			Description: "Git tag to run the pipeline on (mutually exclusive with Branch)",
			Placeholder: "e.g. v1.0.0",
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

func (c *TriggerPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TriggerPipeline) Setup(ctx core.SetupContext) error {
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

	//
	// If this is the same project, nothing to do.
	//
	if metadata.Project != nil && config.ProjectSlug == metadata.Project.Slug {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.ProjectSlug)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.ProjectSlug, err)
	}

	err = ctx.Metadata.Set(TriggerPipelineNodeMetadata{
		Project: &ProjectInfo{
			ID:   project.ID,
			Name: project.Name,
			Slug: project.Slug,
			URL:  fmt.Sprintf("https://app.circleci.com/pipelines/%s", project.Slug),
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectSlug: project.Slug,
	})

	return nil
}

func (c *TriggerPipeline) Execute(ctx core.ExecutionContext) error {
	spec := TriggerPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	metadata := TriggerPipelineNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	request := &TriggerPipelineRequest{
		Branch:     spec.Branch,
		Tag:        spec.Tag,
		Parameters: c.buildParameters(ctx, spec.Parameters),
	}

	response, err := client.TriggerPipeline(spec.ProjectSlug, request)
	if err != nil {
		return fmt.Errorf("error triggering pipeline: %v", err)
	}

	ctx.Logger.Infof("New pipeline created - pipeline=%s, number=%d", response.ID, response.Number)

	ctx.Metadata.Set(TriggerPipelineExecutionMetadata{
		Pipeline: &PipelineInfo{
			ID:     response.ID,
			Number: response.Number,
			State:  response.State,
			URL:    fmt.Sprintf("https://app.circleci.com/pipelines/%s/%d", spec.ProjectSlug, response.Number),
		},
	})

	//
	// This is what allows the component to associate a CircleCI webhook
	// for a workflow finishing to a SuperPlane execution.
	//
	err = ctx.ExecutionState.SetKV("pipeline", response.ID)
	if err != nil {
		return err
	}

	//
	// We still set up the poller to check for workflow finishing,
	// just in case something wrong happens with the update through the webhook.
	//
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (c *TriggerPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TriggerPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// CircleCI uses circleci-signature header for webhook verification
	signature := ctx.Headers.Get("circleci-signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature header")
	}

	// Parse the signature - format is "v1=<hash>"
	signatureParts := strings.Split(signature, "=")
	if len(signatureParts) != 2 || signatureParts[0] != "v1" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signatureParts[1]); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	var payload map[string]any
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Check if this is a workflow-completed event
	eventType, ok := payload["type"].(string)
	if !ok || eventType != "workflow-completed" {
		// Silently ignore other event types
		return http.StatusOK, nil
	}

	pipelineData, ok := payload["pipeline"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("pipeline data missing from webhook payload")
	}

	pipelineID, _ := pipelineData["id"].(string)
	if pipelineID == "" {
		return http.StatusBadRequest, fmt.Errorf("pipeline id missing from webhook payload")
	}

	workflowData, ok := payload["workflow"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("workflow data missing from webhook payload")
	}

	workflowStatus, _ := workflowData["status"].(string)

	executionCtx, err := ctx.FindExecutionByKV("pipeline", pipelineID)

	//
	// We will receive hooks for pipelines that weren't started by SuperPlane,
	// so we just ignore them.
	//
	if err != nil {
		return http.StatusOK, nil
	}

	metadata := TriggerPipelineExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.Metadata.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	//
	// Already finished, do not do anything.
	//
	if metadata.Workflow != nil && (metadata.Workflow.Status == WorkflowStatusSuccess ||
		metadata.Workflow.Status == WorkflowStatusFailed ||
		metadata.Workflow.Status == WorkflowStatusCanceled) {
		return http.StatusOK, nil
	}

	workflowID, _ := workflowData["id"].(string)
	workflowName, _ := workflowData["name"].(string)
	workflowURL, _ := workflowData["url"].(string)

	metadata.Workflow = &WorkflowInfo{
		ID:     workflowID,
		Name:   workflowName,
		Status: workflowStatus,
		URL:    workflowURL,
	}
	err = executionCtx.Metadata.Set(metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error setting metadata: %v", err)
	}

	if workflowStatus == WorkflowStatusSuccess {
		err = executionCtx.ExecutionState.Emit(SuccessOutputChannel, PayloadType, []any{payload})
	} else {
		err = executionCtx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (c *TriggerPipeline) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
		{
			Name:           "finish",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:     "data",
					Type:     configuration.FieldTypeObject,
					Required: false,
					Default:  map[string]any{},
				},
			},
		},
	}
}

func (c *TriggerPipeline) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	case "finish":
		return c.finish(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *TriggerPipeline) poll(ctx core.ActionContext) error {
	spec := TriggerPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := TriggerPipelineExecutionMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	//
	// If the workflow already finished, we don't need to do anything.
	//
	if metadata.Workflow != nil && (metadata.Workflow.Status == WorkflowStatusSuccess ||
		metadata.Workflow.Status == WorkflowStatusFailed ||
		metadata.Workflow.Status == WorkflowStatusCanceled) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Get pipeline workflows
	workflows, err := client.ListPipelineWorkflows(metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	// Check if all workflows are done
	allDone := true
	anyFailed := false
	var lastWorkflow *Workflow

	for _, wf := range workflows {
		lastWorkflow = &wf
		switch wf.Status {
		case WorkflowStatusSuccess:
			// Continue checking
		case WorkflowStatusFailed, WorkflowStatusCanceled:
			anyFailed = true
		default:
			allDone = false
		}
	}

	//
	// If not finished, poll again.
	//
	if !allDone {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	if lastWorkflow != nil {
		metadata.Workflow = &WorkflowInfo{
			ID:     lastWorkflow.ID,
			Name:   lastWorkflow.Name,
			Status: lastWorkflow.Status,
			URL:    fmt.Sprintf("https://app.circleci.com/pipelines/%s/%d/workflows/%s", spec.ProjectSlug, metadata.Pipeline.Number, lastWorkflow.ID),
		}
		err = ctx.Metadata.Set(metadata)
		if err != nil {
			return err
		}
	}

	payload := map[string]any{
		"pipeline": map[string]any{
			"id":     metadata.Pipeline.ID,
			"number": metadata.Pipeline.Number,
			"url":    metadata.Pipeline.URL,
		},
	}

	if metadata.Workflow != nil {
		payload["workflow"] = map[string]any{
			"id":     metadata.Workflow.ID,
			"name":   metadata.Workflow.Name,
			"status": metadata.Workflow.Status,
			"url":    metadata.Workflow.URL,
		}
	}

	if anyFailed {
		return ctx.ExecutionState.Emit(FailedOutputChannel, PayloadType, []any{payload})
	}

	return ctx.ExecutionState.Emit(SuccessOutputChannel, PayloadType, []any{payload})
}

func (c *TriggerPipeline) finish(ctx core.ActionContext) error {
	metadata := TriggerPipelineExecutionMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Workflow != nil && (metadata.Workflow.Status == WorkflowStatusSuccess ||
		metadata.Workflow.Status == WorkflowStatusFailed ||
		metadata.Workflow.Status == WorkflowStatusCanceled) {
		return fmt.Errorf("workflow already finished")
	}

	data, ok := ctx.Parameters["data"]
	if !ok {
		data = map[string]any{}
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("data is invalid")
	}

	metadata.Extra = dataMap
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	return nil
}

func (c *TriggerPipeline) buildParameters(ctx core.ExecutionContext, params []Parameter) map[string]any {
	parameters := make(map[string]any)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	parameters["SUPERPLANE_EXECUTION_ID"] = ctx.ID.String()
	parameters["SUPERPLANE_CANVAS_ID"] = ctx.WorkflowID

	return parameters
}

func (c *TriggerPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
