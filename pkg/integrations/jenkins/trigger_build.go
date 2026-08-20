package jenkins

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PayloadTypeTriggerBuild = "jenkins.build.triggered"

type TriggerBuild struct{}

type BuildParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TriggerBuildSpec struct {
	JobName    string           `json:"jobName" mapstructure:"jobName"`
	Parameters []BuildParameter `json:"parameters" mapstructure:"parameters"`
}

func (t *TriggerBuild) Name() string {
	return "jenkins.triggerBuild"
}

func (t *TriggerBuild) Label() string {
	return "Trigger Build"
}

func (t *TriggerBuild) Description() string {
	return "Trigger a Jenkins job build"
}

func (t *TriggerBuild) Documentation() string {
	return `The Trigger Build component starts a build for a Jenkins job.

## Use Cases

- **CI/CD orchestration**: Kick off Jenkins builds from SuperPlane workflows
- **Deployment automation**: Trigger deployment jobs as part of a larger pipeline
- **Parameterized builds**: Pass build parameters from upstream workflow data

## How It Works

1. Sends a build request to the configured Jenkins job (` + "`build`" + ` when no parameters
   are set, ` + "`buildWithParameters`" + ` otherwise)
2. Returns immediately with the queue URL for the requested build — it does not
   wait for the build to start or finish

## Configuration

- **Job Name**: The name of the Jenkins job to build (top-level jobs only in this version)
- **Parameters**: Optional build parameters as key-value pairs (supports expressions)

## Output

- **jobName**: The Jenkins job that was triggered
- **queueUrl**: The Jenkins build queue URL for the triggered build
`
}

func (t *TriggerBuild) Icon() string {
	return "jenkins"
}

func (t *TriggerBuild) Color() string {
	return "gray"
}

func (t *TriggerBuild) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (t *TriggerBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "jobName",
			Label:       "Job Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the Jenkins job to build",
		},
		{
			Name:        "parameters",
			Label:       "Parameters",
			Type:        configuration.FieldTypeList,
			Description: "Optional build parameters",
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

func (t *TriggerBuild) Setup(ctx core.SetupContext) error {
	spec := TriggerBuildSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.JobName == "" {
		return fmt.Errorf("jobName is required")
	}

	return nil
}

func (t *TriggerBuild) Execute(ctx core.ExecutionContext) error {
	spec := TriggerBuildSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	result, err := client.TriggerBuild(spec.JobName, buildParameterMap(spec.Parameters))
	if err != nil {
		return fmt.Errorf("failed to trigger build: %w", err)
	}

	payload := map[string]any{
		"jobName":  spec.JobName,
		"queueUrl": result.QueueURL,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeTriggerBuild, []any{payload})
}

func (t *TriggerBuild) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *TriggerBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *TriggerBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *TriggerBuild) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (t *TriggerBuild) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *TriggerBuild) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func buildParameterMap(params []BuildParameter) map[string]string {
	result := make(map[string]string, len(params))
	for _, param := range params {
		result[param.Name] = param.Value
	}

	return result
}
