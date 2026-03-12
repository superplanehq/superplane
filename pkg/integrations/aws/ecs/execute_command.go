package ecs

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type ExecuteCommand struct{}

type ExecuteCommandConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Cluster     string `json:"cluster" mapstructure:"cluster"`
	Task        string `json:"task" mapstructure:"task"`
	Container   string `json:"container" mapstructure:"container"`
	Command     string `json:"command" mapstructure:"command"`
	Interactive bool   `json:"interactive" mapstructure:"interactive"`
}

type ExecuteCommandNodeMetadata struct {
	Region  string `json:"region" mapstructure:"region"`
	Cluster string `json:"cluster" mapstructure:"cluster"`
	Task    string `json:"task" mapstructure:"task"`
}

func (c *ExecuteCommand) Name() string {
	return "aws.ecs.executeCommand"
}

func (c *ExecuteCommand) Label() string {
	return "ECS • Execute Command"
}

func (c *ExecuteCommand) Description() string {
	return "Execute a command in a running AWS ECS task container"
}

func (c *ExecuteCommand) Documentation() string {
	return `The Execute Command component runs ECS Exec against a running task container.

## Use Cases

- **Operational debugging**: Run diagnostics inside a live task
- **Runtime inspection**: Check process state or config from workflows
- **Automated remediation**: Trigger one-off commands in containerized services

## Notes

- ECS Exec must be enabled and properly configured for the task/service.
- Interactive mode opens an ECS session and returns session connection details.
`
}

func (c *ExecuteCommand) Icon() string {
	return "aws"
}

func (c *ExecuteCommand) Color() string {
	return "gray"
}

func (c *ExecuteCommand) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ExecuteCommand) Configuration() []configuration.Field {
	return []configuration.Field{
		ecsRegionField(),
		ecsClusterField(),
		{
			Name:        "task",
			Label:       "Task",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Running ECS task (task ARN or ID) to run the command in",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
				{
					Field:  "cluster",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ecs.task",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "cluster",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "cluster",
							},
						},
					},
				},
			},
		},
		{
			Name:        "interactive",
			Label:       "Interactive",
			Type:        configuration.FieldTypeBool,
			Required:    true,
			Default:     false,
			Description: "Run in interactive mode (returns ECS session details)",
		},
		{
			Name:        "command",
			Label:       "Command",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Command to execute inside the container",
		},
		{
			Name:        "container",
			Label:       "Container",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "",
			Togglable:   true,
			Description: "Container name to execute the command on (only required for tasks with multiple containers)",
		},
	}
}

func (c *ExecuteCommand) Setup(ctx core.SetupContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(ExecuteCommandNodeMetadata{
		Region:  config.Region,
		Cluster: config.Cluster,
		Task:    config.Task,
	})
}

func (c *ExecuteCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ExecuteCommand) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.ExecuteCommand(ExecuteCommandInput{
		Cluster:     config.Cluster,
		Task:        config.Task,
		Container:   config.Container,
		Command:     config.Command,
		Interactive: config.Interactive,
	})
	if err != nil {
		return fmt.Errorf("failed to execute ECS command: %w", err)
	}
	if strings.TrimSpace(response.TaskArn) == "" {
		return fmt.Errorf("failed to execute ECS command: response did not include a task")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecs.executeCommand",
		[]any{
			map[string]any{
				"command": *response,
			},
		},
	)
}

func (c *ExecuteCommand) Actions() []core.Action {
	return []core.Action{}
}

func (c *ExecuteCommand) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ExecuteCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ExecuteCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ExecuteCommand) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ExecuteCommand) decodeAndValidateConfiguration(rawConfiguration any) (ExecuteCommandConfiguration, error) {
	config := ExecuteCommandConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return ExecuteCommandConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if config.Region == "" {
		return ExecuteCommandConfiguration{}, fmt.Errorf("region is required")
	}
	if config.Cluster == "" {
		return ExecuteCommandConfiguration{}, fmt.Errorf("cluster is required")
	}
	if config.Task == "" {
		return ExecuteCommandConfiguration{}, fmt.Errorf("task is required")
	}
	if config.Command == "" {
		return ExecuteCommandConfiguration{}, fmt.Errorf("command is required")
	}

	return config, nil
}

func (c *ExecuteCommand) normalizeConfig(config ExecuteCommandConfiguration) ExecuteCommandConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Cluster = strings.TrimSpace(config.Cluster)
	config.Task = strings.TrimSpace(config.Task)
	config.Container = strings.TrimSpace(config.Container)
	config.Command = strings.TrimSpace(config.Command)
	return config
}
