package ecs

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	runTaskMaxCount                     = 10
	runTaskCheckRuleAvailabilityAction  = "checkRuleAvailability"
	runTaskTimeoutAction                = "timeout"
	runTaskTimeoutCheckInterval         = 5 * time.Minute
	runTaskCheckRuleRetryInterval       = 10 * time.Second
	runTaskInitialRuleAvailabilityCheck = 5 * time.Second
)

var launchTypeOptions = []configuration.FieldOption{
	{Label: "Auto", Value: "AUTO"},
	{Label: "FARGATE", Value: "FARGATE"},
	{Label: "EC2", Value: "EC2"},
	{Label: "EXTERNAL", Value: "EXTERNAL"},
}

var allowedLaunchTypes = []string{"FARGATE", "EC2", "EXTERNAL"}
var startedTaskStatuses = []string{"RUNNING", "STOPPED"}

type RunTask struct{}

type RunTaskConfiguration struct {
	Region               string                                `json:"region" mapstructure:"region"`
	Cluster              string                                `json:"cluster" mapstructure:"cluster"`
	TaskDefinition       string                                `json:"taskDefinition" mapstructure:"taskDefinition"`
	Count                int                                   `json:"count" mapstructure:"count"`
	LaunchType           string                                `json:"launchType" mapstructure:"launchType"`
	CapacityProvider     []RunTaskCapacityProviderStrategyItem `json:"capacityProviderStrategy" mapstructure:"capacityProviderStrategy"`
	Group                string                                `json:"group" mapstructure:"group"`
	StartedBy            string                                `json:"startedBy" mapstructure:"startedBy"`
	PlatformVersion      string                                `json:"platformVersion" mapstructure:"platformVersion"`
	EnableExecuteCommand bool                                  `json:"enableExecuteCommand" mapstructure:"enableExecuteCommand"`
	NetworkConfiguration RunTaskNetworkConfiguration           `json:"networkConfiguration,omitempty" mapstructure:"networkConfiguration"`
	Overrides            RunTaskOverrides                      `json:"overrides,omitempty" mapstructure:"overrides"`
	TimeoutSeconds       int                                   `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
}

type RunTaskNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type RunTaskExecutionMetadata struct {
	Region         string   `json:"region" mapstructure:"region"`
	Cluster        string   `json:"cluster" mapstructure:"cluster"`
	TaskARNs       []string `json:"taskArns" mapstructure:"taskArns"`
	TimeoutSeconds int      `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
	StartedAt      string   `json:"startedAt" mapstructure:"startedAt"`
	DeadlineAt     string   `json:"deadlineAt" mapstructure:"deadlineAt"`
}

type RunTaskStateChangeDetail struct {
	TaskARN string `json:"taskArn" mapstructure:"taskArn"`
}

type RunTaskCapacityProviderStrategyItem struct {
	CapacityProvider string `json:"capacityProvider" mapstructure:"capacityProvider"`
	Weight           int    `json:"weight,omitempty" mapstructure:"weight"`
	Base             int    `json:"base,omitempty" mapstructure:"base"`
}

type RunTaskNetworkConfiguration struct {
	AwsvpcConfiguration *RunTaskAwsvpcConfiguration `json:"awsvpcConfiguration,omitempty" mapstructure:"awsvpcConfiguration"`
}

type RunTaskAwsvpcConfiguration struct {
	Subnets        []string `json:"subnets,omitempty" mapstructure:"subnets"`
	SecurityGroups []string `json:"securityGroups,omitempty" mapstructure:"securityGroups"`
	AssignPublicIP string   `json:"assignPublicIp,omitempty" mapstructure:"assignPublicIp"`
}

type RunTaskOverrides struct {
	ContainerOverrides []RunTaskContainerOverride `json:"containerOverrides,omitempty" mapstructure:"containerOverrides"`
}

type RunTaskContainerOverride struct {
	Name              string                        `json:"name,omitempty" mapstructure:"name"`
	Command           []string                      `json:"command,omitempty" mapstructure:"command"`
	Environment       []RunTaskContainerEnvironment `json:"environment,omitempty" mapstructure:"environment"`
	CPU               int                           `json:"cpu,omitempty" mapstructure:"cpu"`
	Memory            int                           `json:"memory,omitempty" mapstructure:"memory"`
	MemoryReservation int                           `json:"memoryReservation,omitempty" mapstructure:"memoryReservation"`
}

type RunTaskContainerEnvironment struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type runTaskMessageData struct {
	TaskARN string
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
	return `The Run Task component starts one or more ECS tasks and completes based on task lifecycle events.

## Use Cases

- **One-off workloads**: Execute ad-hoc jobs on ECS
- **Batch processing**: Trigger task runs from workflow events
- **Operational automation**: Run remediation or maintenance tasks

## Completion behavior

- Always waits for tasks to leave startup states (for example, PENDING) before completing.
- If **Timeout (seconds)** is set, waits for all tracked tasks to reach STOPPED, or completes with timeout when that deadline is reached.

## Notes

- For Fargate tasks, set **Network Configuration** using the ECS awsvpcConfiguration format.
- Use **Capacity Provider Strategy** when you want ECS to choose capacity providers; it cannot be combined with **Launch Type**.
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
					Max: func() *int { max := runTaskMaxCount; return &max }(),
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
			Name:        "capacityProviderStrategy",
			Label:       "Capacity Provider Strategy",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []any{},
			Togglable:   true,
			Description: "Optional capacity provider strategy (cannot be used with Launch Type)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Strategy",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "capacityProvider",
								Label:    "Capacity Provider",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:      "weight",
								Label:     "Weight",
								Type:      configuration.FieldTypeNumber,
								Required:  false,
								Default:   "0",
								Togglable: true,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 0; return &min }(),
									},
								},
							},
							{
								Name:      "base",
								Label:     "Base",
								Type:      configuration.FieldTypeNumber,
								Required:  false,
								Default:   "0",
								Togglable: true,
								TypeOptions: &configuration.TypeOptions{
									Number: &configuration.NumberTypeOptions{
										Min: func() *int { min := 0; return &min }(),
									},
								},
							},
						},
					},
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
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "awsvpcConfiguration",
							Label:    "AWS VPC Configuration",
							Type:     configuration.FieldTypeObject,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								Object: &configuration.ObjectTypeOptions{
									Schema: []configuration.Field{
										{
											Name:     "subnets",
											Label:    "Subnets",
											Type:     configuration.FieldTypeList,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												List: &configuration.ListTypeOptions{
													ItemLabel: "Subnet",
													ItemDefinition: &configuration.ListItemDefinition{
														Type: configuration.FieldTypeString,
													},
												},
											},
										},
										{
											Name:     "securityGroups",
											Label:    "Security Groups",
											Type:     configuration.FieldTypeList,
											Required: false,
											TypeOptions: &configuration.TypeOptions{
												List: &configuration.ListTypeOptions{
													ItemLabel: "Security Group",
													ItemDefinition: &configuration.ListItemDefinition{
														Type: configuration.FieldTypeString,
													},
												},
											},
										},
										{
											Name:     "assignPublicIp",
											Label:    "Assign Public IP",
											Type:     configuration.FieldTypeSelect,
											Required: false,
											Default:  "DISABLED",
											TypeOptions: &configuration.TypeOptions{
												Select: &configuration.SelectTypeOptions{
													Options: []configuration.FieldOption{
														{Label: "Disabled", Value: "DISABLED"},
														{Label: "Enabled", Value: "ENABLED"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "overrides",
			Label:       "Overrides",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Default:     "{\"containerOverrides\":[]}",
			Togglable:   true,
			Description: "Optional ECS task overrides object",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "containerOverrides",
							Label:    "Container Overrides",
							Type:     configuration.FieldTypeList,
							Required: false,
							TypeOptions: &configuration.TypeOptions{
								List: &configuration.ListTypeOptions{
									ItemLabel: "Container Override",
									ItemDefinition: &configuration.ListItemDefinition{
										Type: configuration.FieldTypeObject,
										Schema: []configuration.Field{
											{
												Name:     "name",
												Label:    "Name",
												Type:     configuration.FieldTypeString,
												Required: false,
											},
											{
												Name:     "command",
												Label:    "Command",
												Type:     configuration.FieldTypeList,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													List: &configuration.ListTypeOptions{
														ItemLabel: "Argument",
														ItemDefinition: &configuration.ListItemDefinition{
															Type: configuration.FieldTypeString,
														},
													},
												},
											},
											{
												Name:     "environment",
												Label:    "Environment",
												Type:     configuration.FieldTypeList,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													List: &configuration.ListTypeOptions{
														ItemLabel: "Environment Variable",
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
																	Required: false,
																},
															},
														},
													},
												},
											},
											{
												Name:     "cpu",
												Label:    "CPU",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 0; return &min }(),
													},
												},
											},
											{
												Name:     "memory",
												Label:    "Memory",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 0; return &min }(),
													},
												},
											},
											{
												Name:     "memoryReservation",
												Label:    "Memory Reservation",
												Type:     configuration.FieldTypeNumber,
												Required: false,
												TypeOptions: &configuration.TypeOptions{
													Number: &configuration.NumberTypeOptions{
														Min: func() *int { min := 0; return &min }(),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "0",
			Togglable:   true,
			Description: "When set, wait for all tracked tasks to stop before this timeout is reached",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
				},
			},
		},
	}
}

func (c *RunTask) Setup(ctx core.SetupContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := RunTaskNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.SubscriptionID != "" && metadata.Region == config.Region {
		return nil
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if integrationMetadata.EventBridge == nil {
		return fmt.Errorf("event bridge metadata is not configured")
	}

	if !hasTaskStateChangeRule(integrationMetadata) {
		if err := ctx.Metadata.Set(RunTaskNodeMetadata{Region: config.Region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return scheduleTaskStateChangeRuleProvision(
			ctx.Integration,
			ctx.Requests,
			runTaskCheckRuleAvailabilityAction,
			runTaskInitialRuleAvailabilityCheck,
			config.Region,
		)
	}

	subscriptionID, err := ctx.Integration.Subscribe(taskStateChangeSubscriptionPattern(config.Region, nil))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(RunTaskNodeMetadata{
		Region:         config.Region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (c *RunTask) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunTask) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	response, err := c.runTask(ctx, config)
	if err != nil {
		return err
	}

	if err := validateRunTaskResponse(response); err != nil {
		return err
	}

	taskARNs := taskARNs(response.Tasks)
	waitForStarted, waitForStopped := shouldWaitForTaskLifecycle(response.Tasks, config.TimeoutSeconds)
	if !(waitForStarted || waitForStopped) {
		return emitRunTaskOutput(ctx.ExecutionState, response.Tasks, response.Failures, false)
	}

	if len(taskARNs) == 0 {
		return fmt.Errorf("run task response did not include task ARNs")
	}

	executionMetadata := buildRunTaskExecutionMetadata(config, taskARNs, time.Now().UTC())
	if err := persistRunTaskExecutionState(ctx, executionMetadata); err != nil {
		return err
	}

	return scheduleRunTaskTimeoutIfNeeded(ctx, config.TimeoutSeconds)
}

func (c *RunTask) runTask(ctx core.ExecutionContext, config RunTaskConfiguration) (*RunTaskResponse, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.RunTask(RunTaskInput{
		Cluster:              config.Cluster,
		TaskDefinition:       config.TaskDefinition,
		Count:                config.Count,
		LaunchType:           config.LaunchType,
		CapacityProvider:     config.CapacityProvider,
		Group:                config.Group,
		StartedBy:            config.StartedBy,
		PlatformVersion:      config.PlatformVersion,
		EnableExecuteCommand: config.EnableExecuteCommand,
		NetworkConfiguration: config.NetworkConfiguration,
		Overrides:            config.Overrides,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run ECS task: %w", err)
	}

	return response, nil
}

func validateRunTaskResponse(response *RunTaskResponse) error {
	if len(response.Tasks) == 0 && len(response.Failures) > 0 {
		failure := response.Failures[0]
		return fmt.Errorf("failed to run ECS task: %s (%s)", failure.Reason, failure.Detail)
	}

	return nil
}

func shouldWaitForTaskLifecycle(tasks []Task, timeoutSeconds int) (bool, bool) {
	waitForStarted := !allTasksStarted(tasks)
	waitForStopped := timeoutSeconds > 0 && !allTasksStopped(tasks)
	return waitForStarted, waitForStopped
}

func buildRunTaskExecutionMetadata(config RunTaskConfiguration, taskARNs []string, startedAt time.Time) RunTaskExecutionMetadata {
	executionMetadata := RunTaskExecutionMetadata{
		Region:         config.Region,
		Cluster:        config.Cluster,
		TaskARNs:       taskARNs,
		TimeoutSeconds: config.TimeoutSeconds,
		StartedAt:      startedAt.Format(time.RFC3339Nano),
	}
	if config.TimeoutSeconds > 0 {
		executionMetadata.DeadlineAt = startedAt.Add(time.Duration(config.TimeoutSeconds) * time.Second).Format(time.RFC3339Nano)
	}

	return executionMetadata
}

func persistRunTaskExecutionState(ctx core.ExecutionContext, executionMetadata RunTaskExecutionMetadata) error {
	if err := ctx.Metadata.Set(executionMetadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	for _, taskARN := range executionMetadata.TaskARNs {
		if err := ctx.ExecutionState.SetKV(ecsTaskExecutionKVTaskARN, taskARN); err != nil {
			return fmt.Errorf("failed to set execution kv: %w", err)
		}
	}

	return nil
}

func scheduleRunTaskTimeoutIfNeeded(ctx core.ExecutionContext, timeoutSeconds int) error {
	if timeoutSeconds <= 0 {
		return nil
	}

	timeoutCheckInterval := nextTimeoutCheckInterval(time.Duration(timeoutSeconds) * time.Second)
	if err := ctx.Requests.ScheduleActionCall(
		runTaskTimeoutAction,
		map[string]any{},
		timeoutCheckInterval,
	); err != nil {
		return fmt.Errorf("failed to schedule timeout action: %w", err)
	}

	return nil
}

func (c *RunTask) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	messageData, err := decodeRunTaskMessage(ctx.Message)
	if err != nil {
		return err
	}
	if messageData == nil {
		return nil
	}

	executionCtx, err := ctx.FindExecutionByKV(ecsTaskExecutionKVTaskARN, messageData.TaskARN)
	if err != nil {
		return err
	}
	if executionCtx == nil {
		return nil
	}

	executionMetadata, err := decodeRunTaskExecutionMetadata(executionCtx.Metadata.Get())
	if err != nil {
		return err
	}

	tasks, failures, err := describeTasks(executionCtx.HTTP, executionCtx.Integration, executionMetadata)
	if err != nil {
		return err
	}

	shouldEmit, timedOut := shouldEmitRunTaskCompletion(executionMetadata, tasks, time.Now().UTC())
	if !shouldEmit {
		return nil
	}

	return emitRunTaskOutput(executionCtx.ExecutionState, tasks, failures, timedOut)
}

func (c *RunTask) Actions() []core.Action {
	return []core.Action{
		{
			Name:           runTaskCheckRuleAvailabilityAction,
			Description:    "Check if the EventBridge rule is available",
			UserAccessible: false,
		},
		{
			Name:           runTaskTimeoutAction,
			Description:    "Complete waiting execution after timeout",
			UserAccessible: false,
		},
	}
}

func (c *RunTask) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case runTaskCheckRuleAvailabilityAction:
		return c.checkRuleAvailability(ctx)

	case runTaskTimeoutAction:
		return c.handleTimeout(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
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

func (c *RunTask) decodeAndValidateConfiguration(rawConfiguration any) (RunTaskConfiguration, error) {
	config := RunTaskConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return RunTaskConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if !hasConfigKey(rawConfiguration, "count") {
		config.Count = 1
	}

	config = c.normalizeConfig(config)
	if config.Region == "" {
		return RunTaskConfiguration{}, fmt.Errorf("region is required")
	}
	if config.Cluster == "" {
		return RunTaskConfiguration{}, fmt.Errorf("cluster is required")
	}
	if config.TaskDefinition == "" {
		return RunTaskConfiguration{}, fmt.Errorf("task definition is required")
	}
	if config.Count < 1 {
		return RunTaskConfiguration{}, fmt.Errorf("count must be at least 1")
	}
	if config.Count > runTaskMaxCount {
		return RunTaskConfiguration{}, fmt.Errorf("count cannot exceed %d", runTaskMaxCount)
	}
	if config.LaunchType != "" && len(config.CapacityProvider) > 0 {
		return RunTaskConfiguration{}, fmt.Errorf("launch type cannot be combined with capacity provider strategy")
	}
	for _, strategy := range config.CapacityProvider {
		if strategy.CapacityProvider == "" {
			return RunTaskConfiguration{}, fmt.Errorf("capacity provider is required for each strategy item")
		}
		if strategy.Weight < 0 {
			return RunTaskConfiguration{}, fmt.Errorf("capacity provider weight cannot be negative")
		}
		if strategy.Base < 0 {
			return RunTaskConfiguration{}, fmt.Errorf("capacity provider base cannot be negative")
		}
	}
	if config.LaunchType != "" && !slices.Contains(allowedLaunchTypes, config.LaunchType) {
		return RunTaskConfiguration{}, fmt.Errorf("invalid launch type: %s", config.LaunchType)
	}
	if config.TimeoutSeconds < 0 {
		return RunTaskConfiguration{}, fmt.Errorf("timeout seconds cannot be negative")
	}

	return config, nil
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
	for i := range config.CapacityProvider {
		config.CapacityProvider[i].CapacityProvider = strings.TrimSpace(config.CapacityProvider[i].CapacityProvider)
	}
	if config.NetworkConfiguration.AwsvpcConfiguration != nil {
		config.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIP = strings.TrimSpace(config.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIP)
	}
	for i := range config.Overrides.ContainerOverrides {
		config.Overrides.ContainerOverrides[i].Name = strings.TrimSpace(config.Overrides.ContainerOverrides[i].Name)
		for j := range config.Overrides.ContainerOverrides[i].Environment {
			config.Overrides.ContainerOverrides[i].Environment[j].Name = strings.TrimSpace(config.Overrides.ContainerOverrides[i].Environment[j].Name)
		}
	}
	return config
}

func (c RunTaskNetworkConfiguration) ToMap() map[string]any {
	configuration := map[string]any{}
	if c.AwsvpcConfiguration == nil {
		return configuration
	}

	awsvpcConfiguration := map[string]any{}
	if len(c.AwsvpcConfiguration.Subnets) > 0 {
		awsvpcConfiguration["subnets"] = c.AwsvpcConfiguration.Subnets
	}
	if len(c.AwsvpcConfiguration.SecurityGroups) > 0 {
		awsvpcConfiguration["securityGroups"] = c.AwsvpcConfiguration.SecurityGroups
	}
	if c.AwsvpcConfiguration.AssignPublicIP != "" {
		awsvpcConfiguration["assignPublicIp"] = c.AwsvpcConfiguration.AssignPublicIP
	}

	configuration["awsvpcConfiguration"] = awsvpcConfiguration
	return configuration
}

func (o RunTaskOverrides) ToMap() map[string]any {
	overrides := map[string]any{}
	if len(o.ContainerOverrides) == 0 {
		return overrides
	}

	containerOverrides := make([]map[string]any, 0, len(o.ContainerOverrides))
	for _, containerOverride := range o.ContainerOverrides {
		containerOverrides = append(containerOverrides, containerOverride.ToMap())
	}
	overrides["containerOverrides"] = containerOverrides
	return overrides
}

func (o RunTaskContainerOverride) ToMap() map[string]any {
	containerOverride := map[string]any{}
	if o.Name != "" {
		containerOverride["name"] = o.Name
	}
	if len(o.Command) > 0 {
		containerOverride["command"] = o.Command
	}
	if len(o.Environment) > 0 {
		containerOverride["environment"] = o.Environment
	}
	if o.CPU > 0 {
		containerOverride["cpu"] = o.CPU
	}
	if o.Memory > 0 {
		containerOverride["memory"] = o.Memory
	}
	if o.MemoryReservation > 0 {
		containerOverride["memoryReservation"] = o.MemoryReservation
	}

	return containerOverride
}

func hasConfigKey(configuration any, key string) bool {
	configurationMap, ok := configuration.(map[string]any)
	if !ok {
		return false
	}

	_, exists := configurationMap[key]
	return exists
}

func (c *RunTask) checkRuleAvailability(ctx core.ActionContext) error {
	metadata := RunTaskNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	return subscribeWhenTaskStateChangeRuleAvailable(
		ctx,
		runTaskCheckRuleAvailabilityAction,
		runTaskCheckRuleRetryInterval,
		taskStateChangeSubscriptionPattern(metadata.Region, nil),
		func(subscriptionID string) error {
			metadata.SubscriptionID = subscriptionID
			return ctx.Metadata.Set(metadata)
		},
	)
}

func (c *RunTask) handleTimeout(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	executionMetadata := RunTaskExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &executionMetadata); err != nil {
		return fmt.Errorf("failed to decode execution metadata: %w", err)
	}

	if executionMetadata.TimeoutSeconds <= 0 {
		return nil
	}

	now := time.Now().UTC()
	if !executionTimedOut(executionMetadata, now) {
		deadline, err := parseExecutionDeadline(executionMetadata)
		if err != nil {
			return fmt.Errorf("failed to parse execution deadline: %w", err)
		}

		remaining := deadline.Sub(now)
		if remaining <= 0 {
			remaining = time.Second
		}

		return ctx.Requests.ScheduleActionCall(
			runTaskTimeoutAction,
			map[string]any{},
			nextTimeoutCheckInterval(remaining),
		)
	}

	tasks, failures, err := describeTasks(ctx.HTTP, ctx.Integration, executionMetadata)
	if err != nil {
		return err
	}

	return emitRunTaskOutput(ctx.ExecutionState, tasks, failures, !allTasksStopped(tasks))
}

func decodeRunTaskMessage(message any) (*runTaskMessageData, error) {
	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(message, &event); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	if event.Source != ecsEventBridgeSource || event.DetailType != ecsTaskStateChangeEventDetailType {
		return nil, nil
	}

	detail := RunTaskStateChangeDetail{}
	if err := mapstructure.Decode(event.Detail, &detail); err != nil {
		return nil, fmt.Errorf("failed to decode event detail: %w", err)
	}

	taskARN := strings.TrimSpace(detail.TaskARN)
	if taskARN == "" {
		return nil, fmt.Errorf("event detail is missing task ARN")
	}

	return &runTaskMessageData{
		TaskARN: taskARN,
	}, nil
}

func decodeRunTaskExecutionMetadata(metadata any) (RunTaskExecutionMetadata, error) {
	executionMetadata := RunTaskExecutionMetadata{}
	if err := mapstructure.Decode(metadata, &executionMetadata); err != nil {
		return RunTaskExecutionMetadata{}, fmt.Errorf("failed to decode execution metadata: %w", err)
	}
	if len(executionMetadata.TaskARNs) == 0 {
		return RunTaskExecutionMetadata{}, fmt.Errorf("execution metadata missing task ARNs")
	}

	return executionMetadata, nil
}

func shouldEmitRunTaskCompletion(executionMetadata RunTaskExecutionMetadata, tasks []Task, now time.Time) (bool, bool) {
	if executionTimedOut(executionMetadata, now) && !allTasksStopped(tasks) {
		return true, true
	}

	if !allTasksStarted(tasks) {
		return false, false
	}

	if executionMetadata.TimeoutSeconds > 0 && !allTasksStopped(tasks) {
		return false, false
	}

	return true, false
}

func nextTimeoutCheckInterval(remaining time.Duration) time.Duration {
	if remaining <= runTaskTimeoutCheckInterval {
		return remaining
	}

	return runTaskTimeoutCheckInterval
}

func taskARNs(tasks []Task) []string {
	arns := make([]string, 0, len(tasks))
	seen := map[string]struct{}{}
	for _, task := range tasks {
		taskARN := strings.TrimSpace(task.TaskArn)
		if taskARN == "" {
			continue
		}
		if _, ok := seen[taskARN]; ok {
			continue
		}
		seen[taskARN] = struct{}{}
		arns = append(arns, taskARN)
	}
	return arns
}

func allTasksStarted(tasks []Task) bool {
	if len(tasks) == 0 {
		return false
	}

	for _, task := range tasks {
		if !slices.Contains(startedTaskStatuses, strings.ToUpper(strings.TrimSpace(task.LastStatus))) {
			return false
		}
	}
	return true
}

func allTasksStopped(tasks []Task) bool {
	if len(tasks) == 0 {
		return false
	}

	for _, task := range tasks {
		if strings.ToUpper(strings.TrimSpace(task.LastStatus)) != "STOPPED" {
			return false
		}
	}
	return true
}

func executionTimedOut(metadata RunTaskExecutionMetadata, now time.Time) bool {
	if metadata.TimeoutSeconds <= 0 {
		return false
	}

	deadline, err := parseExecutionDeadline(metadata)
	if err != nil {
		return false
	}

	return !now.Before(deadline)
}

func parseExecutionDeadline(metadata RunTaskExecutionMetadata) (time.Time, error) {
	deadline := strings.TrimSpace(metadata.DeadlineAt)
	if deadline != "" {
		return time.Parse(time.RFC3339Nano, deadline)
	}

	startedAt := strings.TrimSpace(metadata.StartedAt)
	if startedAt == "" {
		return time.Time{}, fmt.Errorf("missing startedAt")
	}

	startedAtTime, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid startedAt: %w", err)
	}

	return startedAtTime.Add(time.Duration(metadata.TimeoutSeconds) * time.Second), nil
}

func describeTasks(
	httpCtx core.HTTPContext,
	integrationCtx core.IntegrationContext,
	metadata RunTaskExecutionMetadata,
) ([]Task, []Failure, error) {
	region := strings.TrimSpace(metadata.Region)
	if region == "" {
		return nil, nil, fmt.Errorf("execution metadata missing region")
	}

	cluster := strings.TrimSpace(metadata.Cluster)
	if cluster == "" {
		return nil, nil, fmt.Errorf("execution metadata missing cluster")
	}

	if len(metadata.TaskARNs) == 0 {
		return nil, nil, fmt.Errorf("execution metadata missing task ARNs")
	}

	creds, err := common.CredentialsFromInstallation(integrationCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(httpCtx, creds, region)
	response, err := client.DescribeTasks(cluster, metadata.TaskARNs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe ECS tasks: %w", err)
	}

	return response.Tasks, response.Failures, nil
}

func emitRunTaskOutput(
	executionState core.ExecutionStateContext,
	tasks []Task,
	failures []Failure,
	timedOut bool,
) error {
	return executionState.Emit(
		core.DefaultOutputChannel.Name,
		ecsTaskPayloadType,
		[]any{
			map[string]any{
				"tasks":    tasks,
				"failures": failures,
				"timedOut": timedOut,
			},
		},
	)
}
