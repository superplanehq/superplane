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

type DescribeService struct{}

type DescribeServiceConfiguration struct {
	Region  string `json:"region" mapstructure:"region"`
	Cluster string `json:"cluster" mapstructure:"cluster"`
	Service string `json:"service" mapstructure:"service"`
}

func (c *DescribeService) Name() string {
	return "aws.ecs.describeService"
}

func (c *DescribeService) Label() string {
	return "ECS • Describe Service"
}

func (c *DescribeService) Description() string {
	return "Describe an AWS ECS service"
}

func (c *DescribeService) Documentation() string {
	return `The Describe Service component fetches details about a single ECS service.

## Use Cases

- **Deployment checks**: Inspect running/desired task counts before or after deployment
- **Operational visibility**: Fetch service status and task definition details in workflows
- **Automation branching**: Route workflow execution based on ECS service state
`
}

func (c *DescribeService) Icon() string {
	return "aws"
}

func (c *DescribeService) Color() string {
	return "gray"
}

func (c *DescribeService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DescribeService) Configuration() []configuration.Field {
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
			Name:     "service",
			Label:    "Service",
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
	}
}

func (c *DescribeService) Setup(ctx core.SetupContext) error {
	config := DescribeServiceConfiguration{}
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
	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	return nil
}

func (c *DescribeService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DescribeService) Execute(ctx core.ExecutionContext) error {
	config := DescribeServiceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.DescribeServices(config.Cluster, []string{config.Service})
	if err != nil {
		return fmt.Errorf("failed to describe ECS service: %w", err)
	}

	if len(response.Services) == 0 {
		if len(response.Failures) > 0 {
			failure := response.Failures[0]
			return fmt.Errorf("failed to describe ECS service %s: %s (%s)", config.Service, failure.Reason, failure.Detail)
		}
		return fmt.Errorf("service not found: %s", config.Service)
	}

	output := map[string]any{
		"service": response.Services[0],
	}
	if len(response.Failures) > 0 {
		output["failures"] = response.Failures
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecs.service",
		[]any{output},
	)
}

func (c *DescribeService) Actions() []core.Action {
	return []core.Action{}
}

func (c *DescribeService) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DescribeService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DescribeService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DescribeService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DescribeService) normalizeConfig(config DescribeServiceConfiguration) DescribeServiceConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Cluster = strings.TrimSpace(config.Cluster)
	config.Service = strings.TrimSpace(config.Service)
	return config
}
