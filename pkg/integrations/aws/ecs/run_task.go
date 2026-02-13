package ecs

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

var launchTypeOptions = []configuration.FieldOption{
	{Label: "Auto", Value: "AUTO"},
	{Label: "FARGATE", Value: "FARGATE"},
	{Label: "EC2", Value: "EC2"},
	{Label: "EXTERNAL", Value: "EXTERNAL"},
}

var allowedLaunchTypes = []string{"FARGATE", "EC2", "EXTERNAL"}

type RunTask struct{}

type RunTaskConfiguration struct {
	Region               string `json:"region" mapstructure:"region"`
	Cluster              string `json:"cluster" mapstructure:"cluster"`
	TaskDefinition       string `json:"taskDefinition" mapstructure:"taskDefinition"`
	Count                int    `json:"count" mapstructure:"count"`
	LaunchType           string `json:"launchType" mapstructure:"launchType"`
	Group                string `json:"group" mapstructure:"group"`
	StartedBy            string `json:"startedBy" mapstructure:"startedBy"`
	PlatformVersion      string `json:"platformVersion" mapstructure:"platformVersion"`
	EnableExecuteCommand bool   `json:"enableExecuteCommand" mapstructure:"enableExecuteCommand"`
	NetworkConfiguration any    `json:"networkConfiguration,omitempty" mapstructure:"networkConfiguration"`
	Overrides            any    `json:"overrides,omitempty" mapstructure:"overrides"`
}

func (c *RunTask) Name() string {
	return "aws.ecs.runTask"
}

func (c *RunTask) Label() string {
	return "ECS • Run Task"
}

func (c *RunTask) Description() string {
	return "Run a task in AWS ECS"
}

func (c *RunTask) Documentation() string {
	return `The Run Task component starts one or more ECS tasks.

## Use Cases

- **One-off workloads**: Execute ad-hoc jobs on ECS
- **Batch processing**: Trigger task runs from workflow events
- **Operational automation**: Run remediation or maintenance tasks

## Notes

- For Fargate tasks, set **Network Configuration** using the ECS awsvpcConfiguration format.
- Optional ECS API fields can be passed directly through **Overrides** and **Network Configuration**.
`
}

func (c *RunTask) Icon() string {
	return "aws"
}

func (c *RunTask) Color() string {
	return "gray"
}

func (c *RunTask) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunTask) Configuration() []configuration.Field {
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
			Name:     "taskDefinition",
			Label:    "Task Definition",
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
					Type:           "ecs.taskDefinition",
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
			Name:     "count",
			Label:    "Count",
			Type:     configuration.FieldTypeNumber,
			Required: true,
			Default:  "1",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
				},
			},
		},
		{
			Name:     "launchType",
			Label:    "Launch Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "AUTO",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: launchTypeOptions,
				},
			},
		},
		{
			Name:        "group",
			Label:       "Group",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "",
			Togglable:   true,
			Description: "Optional ECS task group",
		},
		{
			Name:        "startedBy",
			Label:       "Started By",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "",
			Togglable:   true,
			Description: "Optional identifier for who started the task",
		},
		{
			Name:        "platformVersion",
			Label:       "Platform Version",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "",
			Togglable:   true,
			Description: "Optional platform version (for example, for Fargate tasks)",
		},
		{
			Name:        "enableExecuteCommand",
			Label:       "Enable Execute Command",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Togglable:   true,
			Description: "Enable ECS Exec support for the task",
		},
		{
			Name:        "networkConfiguration",
			Label:       "Network Configuration",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{\"awsvpcConfiguration\":{\"subnets\":[],\"securityGroups\":[],\"assignPublicIp\":\"DISABLED\"}}",
			Togglable:   true,
			Description: "Optional ECS networkConfiguration object (for example, awsvpcConfiguration)",
		},
		{
			Name:        "overrides",
			Label:       "Overrides",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{\"containerOverrides\":[]}",
			Togglable:   true,
			Description: "Optional ECS task overrides object",
		},
	}
}

func (c *RunTask) Setup(ctx core.SetupContext) error {
	config := RunTaskConfiguration{}
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
	if config.TaskDefinition == "" {
		return fmt.Errorf("task definition is required")
	}
	if config.Count < 0 {
		return fmt.Errorf("count cannot be negative")
	}
	if config.LaunchType != "" && !slices.Contains(allowedLaunchTypes, config.LaunchType) {
		return fmt.Errorf("invalid launch type: %s", config.LaunchType)
	}

	return nil
}

func (c *RunTask) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunTask) Execute(ctx core.ExecutionContext) error {
	config := RunTaskConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.RunTask(RunTaskInput{
		Cluster:              config.Cluster,
		TaskDefinition:       config.TaskDefinition,
		Count:                config.Count,
		LaunchType:           config.LaunchType,
		Group:                config.Group,
		StartedBy:            config.StartedBy,
		PlatformVersion:      config.PlatformVersion,
		EnableExecuteCommand: config.EnableExecuteCommand,
		NetworkConfiguration: config.NetworkConfiguration,
		Overrides:            config.Overrides,
	})
	if err != nil {
		return fmt.Errorf("failed to run ECS task: %w", err)
	}

	if len(response.Tasks) == 0 && len(response.Failures) > 0 {
		failure := response.Failures[0]
		return fmt.Errorf("failed to run ECS task: %s (%s)", strings.TrimSpace(failure.Reason), strings.TrimSpace(failure.Detail))
	}

	output := map[string]any{
		"tasks":    response.Tasks,
		"failures": response.Failures,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecs.task",
		[]any{output},
	)
}

func (c *RunTask) Actions() []core.Action {
	return []core.Action{}
}

func (c *RunTask) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RunTask) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *RunTask) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunTask) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RunTask) normalizeConfig(config RunTaskConfiguration) RunTaskConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Cluster = strings.TrimSpace(config.Cluster)
	config.TaskDefinition = strings.TrimSpace(config.TaskDefinition)
	config.LaunchType = strings.ToUpper(strings.TrimSpace(config.LaunchType))
	if config.LaunchType == "AUTO" {
		config.LaunchType = ""
	}
	config.Group = strings.TrimSpace(config.Group)
	config.StartedBy = strings.TrimSpace(config.StartedBy)
	config.PlatformVersion = strings.TrimSpace(config.PlatformVersion)

	return config
}
