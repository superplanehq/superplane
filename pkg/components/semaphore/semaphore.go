package semaphore

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "semaphore"

func init() {
	registry.RegisterComponent(ComponentName, &Semaphore{})
}

type Semaphore struct{}

type NodeMetadata struct {
	Project *Project `json:"project"`
}

type ExecutionMetadata struct {
	Workflow *Workflow `json:"workflow"`
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

func (s *Semaphore) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (s *Semaphore) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "Semaphore integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "semaphore",
				},
			},
		},
		{
			Name:     "project",
			Label:    "Project",
			Type:     components.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []components.VisibilityCondition{
				{
					Field:  "integration",
					Values: []string{"*"},
				},
			},
			TypeOptions: &components.TypeOptions{
				Resource: &components.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:     "ref",
			Label:    "Workflow ref",
			Type:     components.FieldTypeString,
			Required: true,
		},
		{
			Name:     "pipelineFile",
			Label:    "Pipeline File",
			Type:     components.FieldTypeString,
			Required: true,
		},
		{
			Name:  "parameters",
			Label: "Parameters",
			Type:  components.FieldTypeList,
			TypeOptions: &components.TypeOptions{
				List: &components.ListTypeOptions{
					ItemDefinition: &components.ListItemDefinition{
						Type: components.FieldTypeObject,
						Schema: []components.ConfigurationField{
							{
								Name:     "name",
								Label:    "Name",
								Type:     components.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     components.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func (s *Semaphore) Setup(ctx components.SetupContext) error {
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

func (s *Semaphore) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	integration, err := ctx.IntegrationContext.GetIntegration(spec.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	semaphore, ok := integration.(*semaphore.SemaphoreResourceManager)
	if !ok {
		return fmt.Errorf("integration is not a semaphore integration")
	}

	params := map[string]any{
		"project_id":    spec.Project,
		"reference":     spec.Ref,
		"pipeline_file": spec.PipelineFile,
		"parameters":    s.buildParameters(spec.Parameters),
	}

	wf, err := semaphore.RunWorkflow(params)
	if err != nil {
		return ctx.ExecutionStateContext.Fail("failed to run workflow", err.Error())
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

func (s *Semaphore) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "cancel",
			Description:    "Cancel Semaphore workflow",
			UserAccessible: true,
			Parameters:     []components.ConfigurationField{},
		},
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (s *Semaphore) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return s.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (s *Semaphore) poll(ctx components.ActionContext) error {
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
	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {newMetadata},
	})
}

func (s *Semaphore) buildParameters(params []Parameter) map[string]any {
	result := make(map[string]any)
	for _, param := range params {
		result[param.Name] = param.Value
	}

	return result
}
