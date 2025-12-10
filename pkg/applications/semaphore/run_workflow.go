package semaphore

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
)

const PassedOutputChannel = "passed"
const FailedOutputChannel = "failed"

type RunWorkflow struct{}

type RunWorkflowNodeMetadata struct {
	Project *Project `json:"project"`
}

type RunWorkflowExecutionMetadata struct {
	Workflow *Workflow      `json:"workflow"`
	Data     map[string]any `json:"data,omitempty"`
}

type Workflow struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	State  string `json:"state"`
	Result string `json:"result"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RunWorkflowSpec struct {
	Integration  string      `json:"integration"`
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

func (r *RunWorkflow) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{
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

func (r *RunWorkflow) ProcessQueueItem(ctx components.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (r *RunWorkflow) Setup(ctx components.SetupContext) error {
	config := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	//
	// TODO: check if project exists
	// TODO: set up web hook to receive workflow updates
	//

	return nil
}

func (r *RunWorkflow) Execute(ctx components.ExecutionContext) error {
	spec := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	integration, err := ctx.IntegrationContext.GetIntegration(spec.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	project, err := integration.Get("project", spec.Project)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", spec.Project, err)
	}

	semaphore, ok := integration.(*semaphore.SemaphoreResourceManager)
	if !ok {
		return fmt.Errorf("integration is not a semaphore integration")
	}

	params := map[string]any{
		"project_id":    project.Id(),
		"reference":     spec.Ref,
		"pipeline_file": spec.PipelineFile,
		"parameters":    r.buildParameters(ctx, spec.Parameters),
	}

	if spec.CommitSha != "" {
		params["commit_sha"] = spec.CommitSha
	}

	wf, err := semaphore.RunWorkflow(params)
	if err != nil {
		return ctx.ExecutionStateContext.Fail("failed to run workflow", err.Error())
	}

	ctx.MetadataContext.Set(RunWorkflowExecutionMetadata{
		Workflow: &Workflow{
			ID:    wf.Id(),
			URL:   wf.URL(),
			State: "started",
		},
	})

	return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, 15*time.Second)
}

func (r *RunWorkflow) Actions() []components.Action {
	return []components.Action{
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

func (r *RunWorkflow) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return r.poll(ctx)
	case "finish":
		return r.finish(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (r *RunWorkflow) poll(ctx components.ActionContext) error {
	spec := RunWorkflowSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	metadata := RunWorkflowExecutionMetadata{}
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return err
	}

	//
	// If the execution already finished, we don't need to do anything.
	//
	if metadata.Workflow.State == "finished" {
		return nil
	}

	integration, err := ctx.IntegrationContext.GetIntegration(spec.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	resource, err := integration.Status("workflow", metadata.Workflow.ID, nil)
	if err != nil {
		return fmt.Errorf("error determing status for workflow %s: %v", resource.Id(), err)
	}

	//
	// If not finished, poll again in 1min.
	//
	if !resource.Finished() {
		return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, 15*time.Second)
	}

	result := "passed"
	if !resource.Successful() {
		result = "failed"
	}

	newMetadata := &RunWorkflowExecutionMetadata{
		Workflow: &Workflow{
			ID:     metadata.Workflow.ID,
			URL:    metadata.Workflow.URL,
			State:  "finished",
			Result: result,
		},
	}

	ctx.MetadataContext.Set(newMetadata)

	if result == "passed" {
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			PassedOutputChannel: {newMetadata},
		})
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		FailedOutputChannel: {newMetadata},
	})
}

func (r *RunWorkflow) finish(ctx components.ActionContext) error {
	metadata := RunWorkflowExecutionMetadata{}
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return err
	}

	if metadata.Workflow.State == "finished" {
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

	newMetadata := &RunWorkflowExecutionMetadata{
		Data: dataMap,
		Workflow: &Workflow{
			ID:     metadata.Workflow.ID,
			URL:    metadata.Workflow.URL,
			State:  "finished",
			Result: "passed",
		},
	}

	ctx.MetadataContext.Set(newMetadata)

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		PassedOutputChannel: {newMetadata},
	})
}

func (r *RunWorkflow) buildParameters(ctx components.ExecutionContext, params []Parameter) map[string]any {
	parameters := make(map[string]any)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	parameters["SUPERPLANE_EXECUTION_ID"] = ctx.ID
	parameters["SUPERPLANE_CANVAS_ID"] = ctx.WorkflowID

	return parameters
}
