package components

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
)

type RunWorkflow struct{}

type Spec struct {
	Workflow string  `json:"workflow"`
	Ref      string  `json:"ref"`
	Inputs   []Input `json:"inputs"`
}

type Input struct {
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
	return "Run GitHub Workflow and wait for it to finish"
}

func (r *RunWorkflow) Icon() string {
	return "github"
}

func (r *RunWorkflow) Color() string {
	return "gray"
}

func (r *RunWorkflow) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (r *RunWorkflow) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "GitHub integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "github",
				},
			},
		},
		{
			Name:     "repository",
			Label:    "Repository",
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
					Type: "repository",
				},
			},
		},
		{
			Name:        "workflow",
			Label:       "Workflow",
			Type:        components.FieldTypeString,
			Description: "Name or path to the workflow to run",
			Required:    true,
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        components.FieldTypeString,
			Description: "Ref of the workflow to run",
			Required:    true,
		},
		{
			Name:        "inputs",
			Type:        components.FieldTypeList,
			Description: "Inputs to pass to the workflow",
			Required:    false,
			TypeOptions: &components.TypeOptions{
				List: &components.ListTypeOptions{
					ItemDefinition: &components.ListItemDefinition{
						Type: components.FieldTypeObject,
						Schema: []components.ConfigurationField{
							{
								Name:  "name",
								Label: "Input Name",
								Type:  components.FieldTypeString,
							},
							{
								Name:  "value",
								Label: "Input Value",
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
