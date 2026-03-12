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

type UpdateService struct{}

type UpdateServiceConfiguration struct {
	ServiceMutationConfiguration `mapstructure:",squash"`
	Service                      string `json:"service" mapstructure:"service"`
}

type UpdateServiceNodeMetadata struct {
	Region  string `json:"region" mapstructure:"region"`
	Cluster string `json:"cluster" mapstructure:"cluster"`
	Service string `json:"service" mapstructure:"service"`
}

func (c *UpdateService) Name() string {
	return "aws.ecs.updateService"
}

func (c *UpdateService) Label() string {
	return "ECS • Update Service"
}

func (c *UpdateService) Description() string {
	return "Update an AWS ECS service configuration"
}

func (c *UpdateService) Documentation() string {
	return `The Update Service component updates configuration for an existing ECS service.

## Use Cases

- **Deployments**: Roll out a new task definition
- **Scaling workflows**: Change desired count dynamically
- **Operational tuning**: Update deployment, network, or tag behavior

## Notes

- You can pass advanced ECS UpdateService fields through **Additional ECS API Arguments**.
- Do not combine **Launch Type** with **Capacity Provider Strategy**.
`
}

func (c *UpdateService) Icon() string {
	return "aws"
}

func (c *UpdateService) Color() string {
	return "gray"
}

func (c *UpdateService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateService) Configuration() []configuration.Field {
	fields := []configuration.Field{
		ecsRegionField(),
		ecsClusterField(),
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "ECS service to update (name or ARN)",
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
					Type:           "ecs.service",
					UseNameAsValue: true,
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
		ecsTaskDefinitionField(false),
	}

	return append(fields, ecsServiceMutationFields(nil, true, true)...)
}

func (c *UpdateService) Setup(ctx core.SetupContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(UpdateServiceNodeMetadata{
		Region:  config.Region,
		Cluster: config.Cluster,
		Service: config.Service,
	})
}

func (c *UpdateService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateService) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeAndValidateConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.UpdateService(UpdateServiceInput{
		Service:         config.Service,
		ServiceMutation: config.ServiceMutationConfiguration.toInput(),
	})
	if err != nil {
		return fmt.Errorf("failed to update ECS service: %w", err)
	}

	if strings.TrimSpace(response.Service.ServiceArn) == "" {
		return fmt.Errorf("failed to update ECS service: response did not include a service")
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

func (c *UpdateService) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateService) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateService) decodeAndValidateConfiguration(rawConfiguration any) (UpdateServiceConfiguration, error) {
	config := UpdateServiceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return UpdateServiceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if err := config.ServiceMutationConfiguration.validateBase(); err != nil {
		return UpdateServiceConfiguration{}, err
	}
	if config.Service == "" {
		return UpdateServiceConfiguration{}, fmt.Errorf("service is required")
	}

	return config, nil
}

func (c *UpdateService) normalizeConfig(config UpdateServiceConfiguration) UpdateServiceConfiguration {
	config.ServiceMutationConfiguration = config.ServiceMutationConfiguration.normalize()
	config.Service = strings.TrimSpace(config.Service)
	return config
}
