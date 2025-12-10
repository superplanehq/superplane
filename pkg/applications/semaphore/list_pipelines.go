package semaphore

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
)

type ListPipelines struct{}

type ListPipelinesSpec struct {
	Project string `json:"project"`
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
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeString,
			Required: true,
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

	//
	// TODO: check if project exists
	//

	return nil
}

func (l *ListPipelines) Execute(ctx components.ExecutionContext) error {
	spec := ListPipelinesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return err
	}

	response, err := client.ListPipelines(spec.Project)
	if err != nil {
		return err
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: response,
	})
}

func (l *ListPipelines) Actions() []components.Action {
	return []components.Action{}
}

func (l *ListPipelines) HandleAction(ctx components.ActionContext) error {
	return nil
}
