package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
)

type ListPipelines struct{}

type ListPipelinesSpec struct {
	Integration string `json:"integration"`
	Project     string `json:"project"`
}

func (l *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (l *ListPipelines) Label() string {
	return "List Pipelines"
}

func (l *ListPipelines) Description() string {
	return "List Semaphore pipelines"
}

func (l *ListPipelines) Icon() string {
	return "list"
}

func (l *ListPipelines) Color() string {
	return "gray"
}

func (l *ListPipelines) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (l *ListPipelines) Configuration() []configuration.Field {
	return []configuration.Field{
		// TODO: figure out how the component and its application are connected
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
	}
}

func (l *ListPipelines) ProcessQueueItem(ctx components.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (l *ListPipelines) Setup(ctx components.SetupContext) error {
	config := ListPipelinesSpec{}
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

	return nil
}

func (l *ListPipelines) Execute(ctx components.ExecutionContext) error {
	spec := ListPipelinesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	//
	// TODO: list pipelines and emit event
	//

	return nil
}

func (l *ListPipelines) Actions() []components.Action {
	return []components.Action{}
}

func (l *ListPipelines) HandleAction(ctx components.ActionContext) error {
	return nil
}
