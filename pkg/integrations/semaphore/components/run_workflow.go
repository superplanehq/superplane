package components

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
)

type RunWorkflow struct{}

type Spec struct {
	PipelineFile string      `json:"pipelineFile"`
	Ref          string      `json:"ref"`
	Parameters   []Parameter `json:"parameters"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (r *RunWorkflow) Name() string {
	return "run-workflow"
}

func (r *RunWorkflow) Label() string {
	return "Run Workflow"
}

func (r *RunWorkflow) Description() string {
	return "Run Semaphore Workflow and wait for it to finish"
}

func (r *RunWorkflow) Icon() string {
	return "workflow"
}

func (r *RunWorkflow) Color() string {
	return "blue"
}

func (r *RunWorkflow) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (r *RunWorkflow) Configuration() []components.ConfigurationField {
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
			Name:        "ref",
			Label:       "Ref",
			Type:        components.FieldTypeString,
			Description: "Branch or tag where the workflow is located",
			Required:    true,
		},
		{
			Name:        "pipelineFile",
			Label:       "Pipeline File",
			Type:        components.FieldTypeString,
			Description: "Path to the pipeline file",
			Required:    true,
		},
		{
			Name:        "parameters",
			Label:       "Parameters",
			Type:        components.FieldTypeList,
			Description: "Parameters to pass to the workflow",
			Required:    false,
			TypeOptions: &components.TypeOptions{
				List: &components.ListTypeOptions{
					ItemDefinition: &components.ListItemDefinition{
						Type: components.FieldTypeObject,
						Schema: []components.ConfigurationField{
							{
								Name:  "name",
								Label: "Parameter Name",
								Type:  components.FieldTypeString,
							},
							{
								Name:  "value",
								Label: "Parameter Value",
								Type:  components.FieldTypeString,
							},
						},
					},
				},
			},
		},
	}
}

func (r *RunWorkflow) Execute(ctx components.ExecutionContext) error {
	// TODO
	return nil
}

func (r *RunWorkflow) Actions() []components.Action {
	// TODO
	return []components.Action{}
}

func (r *RunWorkflow) HandleAction(ctx components.ActionContext) error {
	// TODO
	return fmt.Errorf("not supported yet")
}
