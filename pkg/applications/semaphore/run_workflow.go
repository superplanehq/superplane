package semaphore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

const PassedOutputChannel = "passed"
const FailedOutputChannel = "failed"
const PipelineStateDone = "done"
const PipelineResultPassed = "passed"
const PollInterval = 5 * time.Minute

type RunWorkflow struct{}

type RunWorkflowNodeMetadata struct {
	Project *Project `json:"project" mapstructure:"project"`
}

type RunWorkflowExecutionMetadata struct {
	Workflow *WorkflowMetadata `json:"workflow" mapstructure:"workflow"`
	Pipeline *PipelineMetadata `json:"pipeline" mapstructure:"pipeline"`
	Data     map[string]any    `json:"data,omitempty" mapstructure:"data,omitempty"`
}

type WorkflowMetadata struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type PipelineMetadata struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Result string `json:"result"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RunWorkflowSpec struct {
	Project      string      `json:"project"`
	Ref          string      `json:"ref"`
	PipelineFile string      `json:"pipelineFile"`
	CommitSha    string      `json:"commitSha"`
	Parameters   []Parameter `json:"parameters"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (r *RunWorkflow) Name() string {
	return "semaphore.runWorkflow"
}

func (r *RunWorkflow) Label() string {
	return "Run Workflow"
}

func (r *RunWorkflow) Description() string {
	return "Run Semaphore workflow"
}

func (r *RunWorkflow) Icon() string {
	return "workflow"
}

func (r *RunWorkflow) Color() string {
	return "gray"
}

func (r *RunWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  PassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  FailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (r *RunWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:        "pipelineFile",
			Label:       "Pipeline file",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. .semaphore/semaphore.yml",
		},
		{
			Name:     "ref",
			Label:    "Pipeline file location",
			Type:     configuration.FieldTypeGitRef,
			Required: true,
		},
		{
			Name:  "commitSha",
			Label: "Commit SHA",
			Type:  configuration.FieldTypeString,
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

func (r *RunWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (r *RunWorkflow) Setup(ctx core.SetupContext) error {
	config := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	metadata := RunWorkflowNodeMetadata{}
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// If this is the same project, nothing to do.
	//
	if metadata.Project != nil && (config.Project == metadata.Project.ID || config.Project == metadata.Project.Name) {
		return nil
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.Project, err)
	}

	ctx.MetadataContext.Set(RunWorkflowNodeMetadata{
		Project: &Project{
			ID:   project.Metadata.ProjectID,
			Name: project.Metadata.ProjectName,
			URL:  fmt.Sprintf("%s/projects/%s", string(client.OrgURL), project.Metadata.ProjectID),
		},
	})

	ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Project: project.Metadata.ProjectName,
	})

	return nil
}

func (r *RunWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	metadata := RunWorkflowNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return err
	}

	params := map[string]any{
		"project_id":    metadata.Project.ID,
		"reference":     spec.Ref,
		"pipeline_file": spec.PipelineFile,
		"parameters":    r.buildParameters(ctx, spec.Parameters),
	}

	if spec.CommitSha != "" {
		params["commit_sha"] = spec.CommitSha
	}

	response, err := client.RunWorkflow(params)
	if err != nil {
		return ctx.ExecutionStateContext.Fail("failed to run workflow", err.Error())
	}

	ctx.MetadataContext.Set(RunWorkflowExecutionMetadata{
		Workflow: &WorkflowMetadata{
			ID:  response.WorkflowID,
			URL: fmt.Sprintf("%s/workflows/%s", string(client.OrgURL), response.WorkflowID),
		},
		Pipeline: &PipelineMetadata{
			ID: response.PipelineID,
		},
	})

	//
	// This is what allows the component to associate a semaphore webhook
	// for a pipeline finishing to a SuperPlane execution.
	//
	err = ctx.ExecutionStateContext.SetKV("pipeline", response.PipelineID)
	if err != nil {
		return err
	}

	//
	// We still set up the poller to check for pipeline finishing,
	// just in case something wrong happens with the update through the webhook.
	//
	return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (r *RunWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *RunWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Semaphore-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	type Hook struct {
		Pipeline struct {
			ID     string `json:"id"`
			State  string `json:"state"`
			Result string `json:"result"`
		} `json:"pipeline"`
	}

	hook := Hook{}
	err = json.Unmarshal(ctx.Body, &hook)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	executionCtx, err := ctx.FindExecutionByKV("pipeline", hook.Pipeline.ID)

	//
	// We will receive hooks for pipelines that weren't started by SuperPlane,
	// so we just ignore them.
	//
	if err != nil {
		return http.StatusOK, nil
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(executionCtx.MetadataContext.Get(), &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding metadata: %v", err)
	}

	//
	// Already finished, do not do anything.
	//
	if metadata.Pipeline.State == PipelineStateDone {
		return http.StatusOK, nil
	}

	metadata.Pipeline.State = hook.Pipeline.State
	metadata.Pipeline.Result = hook.Pipeline.Result
	executionCtx.MetadataContext.Set(metadata)

	if metadata.Pipeline.Result == PipelineResultPassed {
		err = executionCtx.ExecutionStateContext.Pass(map[string][]any{
			PassedOutputChannel: {metadata},
		})
	} else {
		err = executionCtx.ExecutionStateContext.Pass(map[string][]any{
			FailedOutputChannel: {metadata},
		})
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (r *RunWorkflow) Actions() []core.Action {
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

func (r *RunWorkflow) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return r.poll(ctx)
	case "finish":
		return r.finish(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (r *RunWorkflow) poll(ctx core.ActionContext) error {
	spec := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return err
	}

	//
	// If the pipeline already finished, we don't need to do anything.
	//
	if metadata.Pipeline.State == PipelineStateDone {
		return nil
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(metadata.Pipeline.ID)
	if err != nil {
		return err
	}

	//
	// If not finished, poll again in 1min.
	//
	if pipeline.State != PipelineStateDone {
		return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, PollInterval)
	}

	metadata.Pipeline.State = pipeline.State
	metadata.Pipeline.Result = pipeline.Result
	ctx.MetadataContext.Set(metadata)

	if pipeline.Result == PipelineResultPassed {
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			PassedOutputChannel: {metadata},
		})
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		FailedOutputChannel: {metadata},
	})
}

func (r *RunWorkflow) finish(ctx core.ActionContext) error {
	metadata := RunWorkflowExecutionMetadata{}
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Pipeline.State == PipelineStateDone {
		return fmt.Errorf("pipeline already finished")
	}

	data, ok := ctx.Parameters["data"]
	if !ok {
		data = map[string]any{}
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("data is invalid")
	}

	metadata.Data = dataMap
	ctx.MetadataContext.Set(metadata)
	return nil
}

func (r *RunWorkflow) buildParameters(ctx core.ExecutionContext, params []Parameter) map[string]any {
	parameters := make(map[string]any)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	parameters["SUPERPLANE_EXECUTION_ID"] = ctx.ID
	parameters["SUPERPLANE_CANVAS_ID"] = ctx.WorkflowID

	return parameters
}
