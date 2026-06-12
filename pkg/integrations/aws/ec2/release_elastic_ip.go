package ec2

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type ReleaseElasticIP struct{}

type ReleaseElasticIPConfiguration struct {
	Region       string `json:"region" mapstructure:"region"`
	AllocationID string `json:"allocationId" mapstructure:"allocationId"`
}

type ReleaseElasticIPNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	AllocationID string `json:"allocationId" mapstructure:"allocationId"`
}

func (c *ReleaseElasticIP) Name() string {
	return "aws.ec2.releaseElasticIP"
}

func (c *ReleaseElasticIP) Label() string {
	return "EC2 • Release Elastic IP"
}

func (c *ReleaseElasticIP) Description() string {
	return "Release an Elastic IP address back to the AWS address pool"
}

func (c *ReleaseElasticIP) Documentation() string {
	return `The Release Elastic IP component releases an allocated Elastic IP address from your AWS account.

## Use Cases

- **Cost optimisation**: Release unused Elastic IPs to avoid idle charges
- **Cleanup workflows**: Remove temporary addresses after a workflow completes
- **Decommissioning**: Release addresses when tearing down infrastructure

## Configuration

- **Region**: AWS region where the Elastic IP was allocated
- **Elastic IP**: The allocated Elastic IP to release

## Output

Emits a release confirmation on the default output channel:
- ` + "`allocationId`" + `, ` + "`region`" + `

## Important Notes

- The Elastic IP must be disassociated before it can be released
- Released addresses may be allocated to another AWS account and cannot always be recovered
`
}

func (c *ReleaseElasticIP) Icon() string {
	return "aws"
}

func (c *ReleaseElasticIP) Color() string {
	return "gray"
}

func (c *ReleaseElasticIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ReleaseElasticIP) Configuration() []configuration.Field {
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
			Name:        "allocationId",
			Label:       "Elastic IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Elastic IP to release",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeElasticIPUnassociated,
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
	}
}

func (c *ReleaseElasticIP) Setup(ctx core.SetupContext) error {
	config := ReleaseElasticIPConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	allocationID, err := requireAllocationID(config.AllocationID)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(ReleaseElasticIPNodeMetadata{
		Region:       region,
		AllocationID: allocationID,
	})
}

func (c *ReleaseElasticIP) Execute(ctx core.ExecutionContext) error {
	config := ReleaseElasticIPConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	allocationID, err := requireAllocationID(config.AllocationID)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	if err := client.ReleaseAddress(allocationID); err != nil {
		return fmt.Errorf("failed to release Elastic IP: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ReleaseElasticIPPayloadType,
		[]any{
			map[string]any{
				"allocationId": allocationID,
				"region":       region,
			},
		},
	)
}

func (c *ReleaseElasticIP) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ReleaseElasticIP) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *ReleaseElasticIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ReleaseElasticIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ReleaseElasticIP) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ReleaseElasticIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
