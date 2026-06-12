package ec2

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

const (
	allocateIPSourceAmazon        = "amazon"
	allocateIPSourceBYOIP         = "byoip"
	allocateIPSourceCustomerOwned = "customerOwned"
	allocateIPSourceIPAM          = "ipam"
)

var validAllocateIPSources = []string{
	allocateIPSourceAmazon,
	allocateIPSourceBYOIP,
	allocateIPSourceCustomerOwned,
	allocateIPSourceIPAM,
}

type AllocateElasticIP struct{}

type AllocateElasticIPConfiguration struct {
	Region                string       `json:"region" mapstructure:"region"`
	IPSource              string       `json:"ipSource" mapstructure:"ipSource"`
	PublicIPv4Pool        string       `json:"publicIpv4Pool" mapstructure:"publicIpv4Pool"`
	CustomerOwnedIPv4Pool string       `json:"customerOwnedIpv4Pool" mapstructure:"customerOwnedIpv4Pool"`
	IpamPoolID            string       `json:"ipamPoolId" mapstructure:"ipamPoolId"`
	Address               *string      `json:"address,omitempty" mapstructure:"address"`
	Tags                  []common.Tag `json:"tags" mapstructure:"tags"`
}

type AllocateElasticIPNodeMetadata struct {
	Region   string `json:"region" mapstructure:"region"`
	IPSource string `json:"ipSource" mapstructure:"ipSource"`
}

func (c *AllocateElasticIP) Name() string {
	return "aws.ec2.allocateElasticIP"
}

func (c *AllocateElasticIP) Label() string {
	return "EC2 • Allocate Elastic IP"
}

func (c *AllocateElasticIP) Description() string {
	return "Allocate a new Elastic IP address in a VPC"
}

func (c *AllocateElasticIP) Documentation() string {
	return `The Allocate Elastic IP component allocates a new Elastic IP address to your AWS account in the selected region.

## Use Cases

- **Static public IPs**: Reserve a public IPv4 address before launching or exposing a service
- **Failover workflows**: Allocate a replacement Elastic IP during disaster recovery
- **Pre-provisioning**: Reserve an address ahead of association with an instance or network interface
- **BYOIP and IPAM**: Allocate from your own address pools or VPC IPAM pools

## Configuration

- **Region**: AWS region where the Elastic IP will be allocated
- **IP source**: Where the address comes from:
  - **Amazon's pool**: Default public IPv4 address from AWS
  - **BYOIP pool**: Address from a public IPv4 pool you brought to your account
  - **Customer-owned pool**: Address from an on-premises pool for use with an Outpost
  - **IPAM pool**: Address from a VPC IPAM pool with a public IPv4 CIDR
- **Pool**: Required when using BYOIP, customer-owned, or IPAM sources (searchable pickers scoped to the selected region)
- **Address** (optional): Request a specific IPv4 address from the selected pool
- **Tags** (optional): Key/value tags applied to the Elastic IP at allocation time

## Output

Emits the allocated Elastic IP details on the default output channel:
- ` + "`allocationId`" + `, ` + "`publicIp`" + `, ` + "`domain`" + `, ` + "`region`" + `

## Important Notes

- Elastic IPs are allocated to your account and incur charges when not associated with a running instance
- The address is allocated for use in a VPC (` + "`domain: vpc`" + `)
- BYOIP, customer-owned, and IPAM pools must already exist in the target region
`
}

func (c *AllocateElasticIP) Icon() string {
	return "aws"
}

func (c *AllocateElasticIP) Color() string {
	return "gray"
}

func (c *AllocateElasticIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AllocateElasticIP) Configuration() []configuration.Field {
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
			Name:        "ipSource",
			Label:       "IP source",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     allocateIPSourceAmazon,
			Description: "The pool to allocate the Elastic IP address from",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Amazon's pool of IPv4 addresses", Value: allocateIPSourceAmazon},
						{Label: "Public IPv4 address pool (BYOIP)", Value: allocateIPSourceBYOIP},
						{Label: "Customer-owned pool (Outpost)", Value: allocateIPSourceCustomerOwned},
						{Label: "IPv4 IPAM pool", Value: allocateIPSourceIPAM},
					},
				},
			},
		},
		{
			Name:        "publicIpv4Pool",
			Label:       "Public IPv4 pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "BYOIP public IPv4 address pool to allocate from",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceBYOIP}},
				{Field: "region", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceBYOIP}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePublicIPv4Pool,
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
			Name:        "customerOwnedIpv4Pool",
			Label:       "Customer-owned pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Customer-owned address pool for Outposts",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceCustomerOwned}},
				{Field: "region", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceCustomerOwned}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCustomerOwnedIPv4Pool,
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
			Name:        "ipamPoolId",
			Label:       "IPAM pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "VPC IPAM pool with a public IPv4 CIDR provisioned for EC2",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceIPAM}},
				{Field: "region", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceIPAM}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIpamPool,
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
			Name:        "address",
			Label:       "Address",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional specific IPv4 address to allocate from the selected pool",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "ipSource", Values: []string{allocateIPSourceBYOIP, allocateIPSourceCustomerOwned, allocateIPSourceIPAM}},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Tags to apply to the allocated Elastic IP address",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "key",
								Label:    "Key",
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
	}
}

func (c *AllocateElasticIP) Setup(ctx core.SetupContext) error {
	config := AllocateElasticIPConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	ipSource, err := normalizeAllocateIPSource(config.IPSource)
	if err != nil {
		return err
	}

	if err := validateAllocateElasticIPConfiguration(config, ipSource); err != nil {
		return err
	}

	return ctx.Metadata.Set(AllocateElasticIPNodeMetadata{
		Region:   region,
		IPSource: ipSource,
	})
}

func (c *AllocateElasticIP) Execute(ctx core.ExecutionContext) error {
	config := AllocateElasticIPConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	ipSource, err := normalizeAllocateIPSource(config.IPSource)
	if err != nil {
		return err
	}

	if err := validateAllocateElasticIPConfiguration(config, ipSource); err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	output, err := client.AllocateAddress(buildAllocateAddressInput(config, ipSource))
	if err != nil {
		return fmt.Errorf("failed to allocate Elastic IP: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		AllocateElasticIPPayloadType,
		[]any{allocateAddressOutputToMap(output)},
	)
}

func normalizeAllocateIPSource(value string) (string, error) {
	ipSource := strings.TrimSpace(value)
	if ipSource == "" {
		return allocateIPSourceAmazon, nil
	}

	if !slices.Contains(validAllocateIPSources, ipSource) {
		return "", fmt.Errorf("invalid IP source %q: must be one of amazon, byoip, customerOwned, ipam", value)
	}

	return ipSource, nil
}

func validateAllocateElasticIPConfiguration(config AllocateElasticIPConfiguration, ipSource string) error {
	switch ipSource {
	case allocateIPSourceBYOIP:
		if _, err := requirePublicIPv4Pool(config.PublicIPv4Pool); err != nil {
			return err
		}
	case allocateIPSourceCustomerOwned:
		if _, err := requireCustomerOwnedIPv4Pool(config.CustomerOwnedIPv4Pool); err != nil {
			return err
		}
	case allocateIPSourceIPAM:
		if _, err := requireIpamPoolID(config.IpamPoolID); err != nil {
			return err
		}
	}

	return nil
}

func buildAllocateAddressInput(config AllocateElasticIPConfiguration, ipSource string) AllocateAddressInput {
	input := AllocateAddressInput{}

	switch ipSource {
	case allocateIPSourceBYOIP:
		input.PublicIPv4Pool = strings.TrimSpace(config.PublicIPv4Pool)
	case allocateIPSourceCustomerOwned:
		input.CustomerOwnedIPv4Pool = strings.TrimSpace(config.CustomerOwnedIPv4Pool)
	case allocateIPSourceIPAM:
		input.IpamPoolID = strings.TrimSpace(config.IpamPoolID)
	}

	if ipSource != allocateIPSourceAmazon && config.Address != nil {
		input.Address = strings.TrimSpace(*config.Address)
	}

	input.Tags = common.NormalizeTags(config.Tags)

	return input
}

func (c *AllocateElasticIP) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AllocateElasticIP) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *AllocateElasticIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AllocateElasticIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AllocateElasticIP) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AllocateElasticIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func allocateAddressOutputToMap(output *AllocateAddressOutput) map[string]any {
	return map[string]any{
		"allocationId": output.AllocationID,
		"publicIp":     output.PublicIP,
		"domain":       output.Domain,
		"region":       output.Region,
	}
}
