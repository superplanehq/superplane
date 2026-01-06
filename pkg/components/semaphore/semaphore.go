package semaphore

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "semaphore"
const PassedOutputChannel = "passed"
const FailedOutputChannel = "failed"

func init() {
	registry.RegisterComponent(ComponentName, &Semaphore{})
}

type Semaphore struct{}

type NodeMetadata struct {
	Project *Project `json:"project"`
}

type ExecutionMetadata struct {
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

type Spec struct {
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

func (s *Semaphore) Name() string {
	return ComponentName
}

func (s *Semaphore) Label() string {
	return "Semaphore"
}

func (s *Semaphore) Description() string {
	return "Run Semaphore workflow"
}

func (s *Semaphore) Icon() string {
	return "workflow"
}

func (s *Semaphore) Color() string {
	return "gray"
}

func (s *Semaphore) OutputChannels(configuration any) []core.OutputChannel {
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

func (s *Semaphore) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "integration",
			Label:    "Semaphore integration",
			Type:     configuration.FieldTypeIntegration,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Integration: &configuration.IntegrationTypeOptions{
					Type: "semaphore",
				},
			},
		},
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeString,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "integration",
					Values: []string{"*"},
				},
			},
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

func (s *Semaphore) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (s *Semaphore) Setup(ctx core.SetupContext) error {
	config := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Integration == "" {
		return fmt.Errorf("integration is required")
	}

	_, err = uuid.Parse(config.Integration)
	if err != nil {
		return fmt.Errorf("integration ID is invalid: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	integration, err := ctx.IntegrationContext.GetIntegration(config.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	resource, err := integration.Get("project", config.Project)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", config.Project, err)
	}

	ctx.MetadataContext.Set(NodeMetadata{
		Project: &Project{
			ID:   resource.Id(),
			Name: resource.Name(),
			URL:  resource.URL(),
		},
	})

	return nil
}

func (s *Semaphore) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
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
		"parameters":    s.buildParameters(ctx, spec.Parameters),
	}

	if spec.CommitSha != "" {
		params["commit_sha"] = spec.CommitSha
	}

	wf, err := semaphore.RunWorkflow(params)
	if err != nil {
		return ctx.ExecutionStateContext.Fail(models.WorkflowNodeExecutionResultReasonError, err.Error())
	}

	ctx.MetadataContext.Set(ExecutionMetadata{
		Workflow: &Workflow{
			ID:    wf.Id(),
			URL:   wf.URL(),
			State: "started",
		},
	})

	return ctx.RequestContext.ScheduleActionCall("poll", map[string]any{}, 15*time.Second)
}

func (s *Semaphore) Actions() []core.Action {
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

func (s *Semaphore) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (s *Semaphore) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return s.poll(ctx)
	case "finish":
		return s.finish(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (s *Semaphore) poll(ctx core.ActionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	metadata := ExecutionMetadata{}
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

	newMetadata := &ExecutionMetadata{
		Workflow: &Workflow{
			ID:     metadata.Workflow.ID,
			URL:    metadata.Workflow.URL,
			State:  "finished",
			Result: result,
		},
	}

	ctx.MetadataContext.Set(newMetadata)

	if result == "passed" {
		return ctx.ExecutionStateContext.Emit(
			PassedOutputChannel,
			"semaphore.workflow.finished",
			[]any{metadata},
		)
	}

	return ctx.ExecutionStateContext.Emit(
		FailedOutputChannel,
		"semaphore.workflow.finished",
		[]any{metadata},
	)
}

func (s *Semaphore) finish(ctx core.ActionContext) error {
	metadata := ExecutionMetadata{}
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

	newMetadata := &ExecutionMetadata{
		Data: dataMap,
		Workflow: &Workflow{
			ID:     metadata.Workflow.ID,
			URL:    metadata.Workflow.URL,
			State:  "finished",
			Result: "passed",
		},
	}

	ctx.MetadataContext.Set(newMetadata)

	return ctx.ExecutionStateContext.Emit(
		PassedOutputChannel,
		"semaphore.workflow.finished",
		[]any{metadata},
	)
}

func (s *Semaphore) Cancel(ctx core.ExecutionContext) error {
	metadata := ExecutionMetadata{}
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Workflow == nil {
		return fmt.Errorf("no workflow found in metadata")
	}

	if ctx.ExecutionStateContext.IsFinished() {
		return nil
	}

	spec := Spec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	integration, err := ctx.IntegrationContext.GetIntegration(spec.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	semaphore, ok := integration.(*semaphore.SemaphoreResourceManager)
	if !ok {
		return fmt.Errorf("integration is not a semaphore integration")
	}

	err = semaphore.Cancel("workflow", metadata.Workflow.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel semaphore workflow: %w", err)
	}

	metadata.Workflow.State = "finished"
	metadata.Workflow.Result = "cancelled"
	ctx.MetadataContext.Set(metadata)

	return nil
}

func (s *Semaphore) buildParameters(ctx core.ExecutionContext, params []Parameter) map[string]any {
	parameters := make(map[string]any)
	for _, param := range params {
		parameters[param.Name] = param.Value
	}

	parameters["SUPERPLANE_EXECUTION_ID"] = ctx.ID
	parameters["SUPERPLANE_CANVAS_ID"] = ctx.WorkflowID

	return parameters
}
