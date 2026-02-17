package ecs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	stopTaskCheckRuleAvailabilityAction  = "checkRuleAvailability"
	stopTaskCheckRuleRetryInterval       = 10 * time.Second
	stopTaskInitialRuleAvailabilityCheck = 5 * time.Second
)

type StopTask struct{}

type StopTaskConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	Cluster string `json:"cluster" mapstructure:"cluster"`
	Task    string `json:"task" mapstructure:"task"`
	Reason  string `json:"reason" mapstructure:"reason"`
}

type StopTaskNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type StopTaskExecutionMetadata struct {
	Region  string `json:"region" mapstructure:"region"`
	Cluster string `json:"cluster" mapstructure:"cluster"`
	TaskARN string `json:"taskArn" mapstructure:"taskArn"`
}

type StopTaskStateChangeDetail struct {
	TaskARN    string `json:"taskArn" mapstructure:"taskArn"`
	LastStatus string `json:"lastStatus" mapstructure:"lastStatus"`
}

type stopTaskMessageData struct {
	TaskARN string
	Detail  map[string]any
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
	return `The Stop Task component requests ECS to stop a running task and waits for the task to reach STOPPED.

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
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := StopTaskNodeMetadata{}
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
		if err := ctx.Metadata.Set(StopTaskNodeMetadata{Region: config.Region}); err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return scheduleTaskStateChangeRuleProvision(
			ctx.Integration,
			ctx.Requests,
			stopTaskCheckRuleAvailabilityAction,
			stopTaskInitialRuleAvailabilityCheck,
			config.Region,
		)
	}

	subscriptionID, err := ctx.Integration.Subscribe(taskStateChangeSubscriptionPattern(config.Region, stopTaskEventDetailFilter))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(StopTaskNodeMetadata{
		Region:         config.Region,
		SubscriptionID: subscriptionID.String(),
	})
}

func (c *StopTask) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StopTask) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

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

	if strings.EqualFold(strings.TrimSpace(response.Task.LastStatus), "STOPPED") {
		return emitStopTaskOutput(ctx.ExecutionState, response.Task)
	}

	return persistStopTaskExecutionState(ctx, config, response.Task.TaskArn)
}

func persistStopTaskExecutionState(ctx core.ExecutionContext, config StopTaskConfiguration, taskARN string) error {
	executionMetadata := StopTaskExecutionMetadata{
		Region:  config.Region,
		Cluster: config.Cluster,
		TaskARN: taskARN,
	}
	if err := ctx.Metadata.Set(executionMetadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.ExecutionState.SetKV(ecsTaskExecutionKVTaskARN, taskARN); err != nil {
		return fmt.Errorf("failed to set execution kv: %w", err)
	}

	return nil
}

func (c *StopTask) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	messageData, err := decodeStopTaskMessage(ctx.Message)
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

	task, err := decodeStopTaskEventDetail(messageData.Detail)
	if err != nil {
		return fmt.Errorf("failed to decode task from event detail: %w", err)
	}
	if strings.TrimSpace(task.TaskArn) == "" {
		task.TaskArn = messageData.TaskARN
	}
	if strings.TrimSpace(task.LastStatus) == "" {
		task.LastStatus = "STOPPED"
	}

	return emitStopTaskOutput(executionCtx.ExecutionState, task)
}

func (c *StopTask) Actions() []core.Action {
	return []core.Action{
		{
			Name:           stopTaskCheckRuleAvailabilityAction,
			Description:    "Check if the EventBridge rule is available",
			UserAccessible: false,
		},
	}
}

func (c *StopTask) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case stopTaskCheckRuleAvailabilityAction:
		return c.checkRuleAvailability(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
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

func (c *StopTask) decodeAndValidateConfiguration(rawConfiguration any) (StopTaskConfiguration, error) {
	config := StopTaskConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return StopTaskConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if config.Region == "" {
		return StopTaskConfiguration{}, fmt.Errorf("region is required")
	}
	if config.Cluster == "" {
		return StopTaskConfiguration{}, fmt.Errorf("cluster is required")
	}
	if config.Task == "" {
		return StopTaskConfiguration{}, fmt.Errorf("task is required")
	}

	return config, nil
}

func (c *StopTask) normalizeConfig(config StopTaskConfiguration) StopTaskConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Cluster = strings.TrimSpace(config.Cluster)
	config.Task = strings.TrimSpace(config.Task)
	config.Reason = strings.TrimSpace(config.Reason)
	return config
}

var stopTaskEventDetailFilter = map[string]any{
	"lastStatus": "STOPPED",
}

func (c *StopTask) checkRuleAvailability(ctx core.ActionContext) error {
	metadata := StopTaskNodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	return subscribeWhenTaskStateChangeRuleAvailable(
		ctx,
		stopTaskCheckRuleAvailabilityAction,
		stopTaskCheckRuleRetryInterval,
		taskStateChangeSubscriptionPattern(metadata.Region, stopTaskEventDetailFilter),
		func(subscriptionID string) error {
			metadata.SubscriptionID = subscriptionID
			return ctx.Metadata.Set(metadata)
		},
	)
}

func decodeStopTaskMessage(message any) (*stopTaskMessageData, error) {
	event := common.EventBridgeEvent{}
	if err := mapstructure.Decode(message, &event); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	if event.Source != ecsEventBridgeSource || event.DetailType != ecsTaskStateChangeEventDetailType {
		return nil, nil
	}

	detail := StopTaskStateChangeDetail{}
	if err := mapstructure.Decode(event.Detail, &detail); err != nil {
		return nil, fmt.Errorf("failed to decode event detail: %w", err)
	}

	taskARN := strings.TrimSpace(detail.TaskARN)
	if taskARN == "" {
		return nil, fmt.Errorf("event detail is missing task ARN")
	}

	if !strings.EqualFold(strings.TrimSpace(detail.LastStatus), "STOPPED") {
		return nil, nil
	}

	return &stopTaskMessageData{
		TaskARN: taskARN,
		Detail:  event.Detail,
	}, nil
}

func emitStopTaskOutput(executionState core.ExecutionStateContext, task Task) error {
	return executionState.Emit(
		core.DefaultOutputChannel.Name,
		ecsTaskPayloadType,
		[]any{
			map[string]any{
				"task": task,
			},
		},
	)
}

func decodeStopTaskEventDetail(detail map[string]any) (Task, error) {
	encodedDetail, err := json.Marshal(detail)
	if err != nil {
		return Task{}, err
	}

	task := Task{}
	if err := json.Unmarshal(encodedDetail, &task); err != nil {
		return Task{}, err
	}

	return task, nil
}
