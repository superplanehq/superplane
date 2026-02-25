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

type CreateService struct{}

type CreateServiceConfiguration struct {
	ServiceMutationConfiguration `mapstructure:",squash"`
	ServiceName                  string `json:"serviceName" mapstructure:"serviceName"`
	SchedulingStrategy           string `json:"schedulingStrategy" mapstructure:"schedulingStrategy"`
	ClientToken                  string `json:"clientToken" mapstructure:"clientToken"`
}

type CreateServiceNodeMetadata struct {
	Region         string `json:"region" mapstructure:"region"`
	Cluster        string `json:"cluster" mapstructure:"cluster"`
	ServiceName    string `json:"serviceName" mapstructure:"serviceName"`
	TaskDefinition string `json:"taskDefinition" mapstructure:"taskDefinition"`
}

func (c *CreateService) Name() string {
	return "aws.ecs.createService"
}

func (c *CreateService) Label() string {
	return "ECS • Create Service"
}

func (c *CreateService) Description() string {
	return "Create an AWS ECS service"
}

func (c *CreateService) Documentation() string {
	return `The Create Service component creates a new ECS service in a cluster.

## Use Cases

- **Provisioning workflows**: Create a service during environment setup
- **Deployment automation**: Roll out new workloads from workflows
- **Infrastructure orchestration**: Configure ECS service settings as part of release pipelines

## Notes

- You can pass advanced ECS CreateService fields through **Additional ECS API Arguments**.
- Do not combine **Launch Type** with **Capacity Provider Strategy**.
`
}

func (c *CreateService) Icon() string {
	return "aws"
}

func (c *CreateService) Color() string {
	return "gray"
}

func (c *CreateService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateService) Configuration() []configuration.Field {
	fields := []configuration.Field{
		ecsRegionField(),
		ecsClusterField(),
		{
			Name:        "serviceName",
			Label:       "Service Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the ECS service to create",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "cluster",
					Values: []string{"*"},
				},
			},
		},
		ecsTaskDefinitionField(true),
		{
			Name:        "schedulingStrategy",
			Label:       "Scheduling Strategy",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     serviceSchedulingStrategyReplica,
			Description: "REPLICA for a fixed number of tasks, or DAEMON for one task per instance",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: serviceSchedulingStrategyOptions,
				},
			},
		},
		{
			Name:        "clientToken",
			Label:       "Client Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional idempotency token",
		},
	}

	return append(fields, ecsServiceMutationFields(1, false, false)...)
}

func (c *CreateService) Setup(ctx core.SetupContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(CreateServiceNodeMetadata{
		Region:         config.Region,
		Cluster:        config.Cluster,
		ServiceName:    config.ServiceName,
		TaskDefinition: config.TaskDefinition,
	})
}

func (c *CreateService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateService) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.CreateService(CreateServiceInput{
		ServiceName:        config.ServiceName,
		SchedulingStrategy: config.SchedulingStrategy,
		ClientToken:        config.ClientToken,
		ServiceMutation:    config.ServiceMutationConfiguration.toInput(),
	})
	if err != nil {
		return fmt.Errorf("failed to create ECS service: %w", err)
	}

	if strings.TrimSpace(response.Service.ServiceArn) == "" {
		return fmt.Errorf("failed to create ECS service: response did not include a service")
	}

	output := map[string]any{
		"service": response.Service,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecs.service",
		[]any{output},
	)
}

func (c *CreateService) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateService) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateService) decodeAndValidateConfiguration(rawConfiguration any) (CreateServiceConfiguration, error) {
	config := CreateServiceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return CreateServiceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if err := config.ServiceMutationConfiguration.validateBase(); err != nil {
		return CreateServiceConfiguration{}, err
	}
	if config.ServiceName == "" {
		return CreateServiceConfiguration{}, fmt.Errorf("service name is required")
	}
	if config.TaskDefinition == "" {
		return CreateServiceConfiguration{}, fmt.Errorf("task definition is required")
	}
	if !slices.Contains([]string{serviceSchedulingStrategyReplica, serviceSchedulingStrategyDaemon}, config.SchedulingStrategy) {
		return CreateServiceConfiguration{}, fmt.Errorf("invalid scheduling strategy: %s", config.SchedulingStrategy)
	}
	if config.SchedulingStrategy == serviceSchedulingStrategyDaemon && config.DesiredCount != nil {
		return CreateServiceConfiguration{}, fmt.Errorf("desired count cannot be set when scheduling strategy is DAEMON")
	}

	return config, nil
}

func (c *CreateService) normalizeConfig(config CreateServiceConfiguration) CreateServiceConfiguration {
	config.ServiceMutationConfiguration = config.ServiceMutationConfiguration.normalize()
	config.ServiceName = strings.TrimSpace(config.ServiceName)
	config.SchedulingStrategy = strings.ToUpper(strings.TrimSpace(config.SchedulingStrategy))
	if config.SchedulingStrategy == "" {
		config.SchedulingStrategy = serviceSchedulingStrategyReplica
	}
	config.ClientToken = strings.TrimSpace(config.ClientToken)

	if config.SchedulingStrategy != serviceSchedulingStrategyDaemon && config.DesiredCount == nil {
		desiredCount := 1
		config.DesiredCount = &desiredCount
	}

	return config
}
