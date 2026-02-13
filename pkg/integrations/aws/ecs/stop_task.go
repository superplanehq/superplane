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

type StopTask struct{}

type StopTaskConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	Cluster string `json:"cluster" mapstructure:"cluster"`
	Task    string `json:"task" mapstructure:"task"`
	Reason  string `json:"reason" mapstructure:"reason"`
}

func (c *StopTask) Name() string {
	return "aws.ecs.stopTask"
}

func (c *StopTask) Label() string {
	return "ECS • Stop Task"
}

func (c *StopTask) Description() string {
	return "Stop a running AWS ECS task"
}

func (c *StopTask) Documentation() string {
	return `The Stop Task component requests ECS to stop a running task.

## Use Cases

- **Operational control**: Stop ad-hoc or long-running tasks from workflows
- **Remediation**: Terminate unhealthy tasks during automated incident response
- **Cost control**: Stop no-longer-needed background workloads

## Notes

- ECS sends a SIGTERM signal and then force-stops the task if it does not exit gracefully.
- **Reason** is optional and appears in ECS task stop metadata when provided.
`
}

func (c *StopTask) Icon() string {
	return "aws"
}

func (c *StopTask) Color() string {
	return "gray"
}

func (c *StopTask) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *StopTask) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:     "cluster",
			Label:    "Cluster",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "ecs.cluster",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:     "task",
			Label:    "Task",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
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
			Name:        "reason",
			Label:       "Reason",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "",
			Togglable:   true,
			Description: "Optional reason stored with the task stop request",
		},
	}
}

func (c *StopTask) Setup(ctx core.SetupContext) error {
	config := StopTaskConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if config.Region == "" {
		return fmt.Errorf("region is required")
	}
	if config.Cluster == "" {
		return fmt.Errorf("cluster is required")
	}
	if config.Task == "" {
		return fmt.Errorf("task is required")
	}

	return nil
}

func (c *StopTask) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StopTask) Execute(ctx core.ExecutionContext) error {
	config := StopTaskConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.StopTask(config.Cluster, config.Task, config.Reason)
	if err != nil {
		return fmt.Errorf("failed to stop ECS task: %w", err)
	}

	if strings.TrimSpace(response.Task.TaskArn) == "" {
		return fmt.Errorf("failed to stop ECS task: response did not include a task")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecs.task",
		[]any{
			map[string]any{
				"task": response.Task,
			},
		},
	)
}

func (c *StopTask) Actions() []core.Action {
	return []core.Action{}
}

func (c *StopTask) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *StopTask) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *StopTask) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *StopTask) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *StopTask) normalizeConfig(config StopTaskConfiguration) StopTaskConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Cluster = strings.TrimSpace(config.Cluster)
	config.Task = strings.TrimSpace(config.Task)
	config.Reason = strings.TrimSpace(config.Reason)
	return config
}
