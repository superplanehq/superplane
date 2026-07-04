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
	elasticIPOperationAssociate    = "associate"
	elasticIPOperationDisassociate = "disassociate"
)

var validElasticIPOperations = []string{
	elasticIPOperationAssociate,
	elasticIPOperationDisassociate,
}

type ManageElasticIP struct{}

type ManageElasticIPConfiguration struct {
	Region        string `json:"region" mapstructure:"region"`
	Operation     string `json:"operation" mapstructure:"operation"`
	AllocationID  string `json:"allocationId" mapstructure:"allocationId"`
	InstanceID    string `json:"instance" mapstructure:"instance"`
	AssociationID string `json:"associationId" mapstructure:"associationId"`
}

type ManageElasticIPNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	Operation    string `json:"operation" mapstructure:"operation"`
	InstanceName string `json:"instanceName,omitempty" mapstructure:"instanceName"`
}

func (c *ManageElasticIP) Name() string {
	return "aws.ec2.manageElasticIP"
}

func (c *ManageElasticIP) Label() string {
	return "EC2 • Manage Elastic IP"
}

func (c *ManageElasticIP) Description() string {
	return "Associate or disassociate an Elastic IP address with an EC2 instance"
}

func (c *ManageElasticIP) Documentation() string {
	return `The Manage Elastic IP component associates or disassociates an Elastic IP address with an EC2 instance.

## Use Cases

- **Static addressing**: Attach a reserved Elastic IP to an instance after launch
- **Failover**: Re-associate an Elastic IP to a replacement instance
- **Cleanup**: Disassociate an Elastic IP before releasing it

## Configuration

- **Region**: AWS region where the Elastic IP and instance reside
- **Operation**: Choose **Associate** or **Disassociate**
- **Elastic IP** (associate only): The allocated Elastic IP to attach
- **Instance** (associate only): EC2 instance to associate the Elastic IP with
- **Association** (disassociate only): The active Elastic IP association to remove

## Output

Emits operation-specific details on the default output channel:
- **Associate**: ` + "`associationId`" + `, ` + "`allocationId`" + `, ` + "`instanceId`" + `, ` + "`region`" + `
- **Disassociate**: ` + "`associationId`" + `, ` + "`region`" + `

## Important Notes

- The Elastic IP must be allocated before it can be associated
- Disassociating is required before releasing an Elastic IP
- Re-association to a different instance is allowed when the address is already associated elsewhere
`
}

func (c *ManageElasticIP) Icon() string {
	return "aws"
}

func (c *ManageElasticIP) Color() string {
	return "gray"
}

func (c *ManageElasticIP) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ManageElasticIP) Configuration() []configuration.Field {
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
			Name:        "operation",
			Label:       "Operation",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Whether to associate or disassociate the Elastic IP",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Associate", Value: elasticIPOperationAssociate},
						{Label: "Disassociate", Value: elasticIPOperationDisassociate},
					},
				},
			},
		},
		{
			Name:        "allocationId",
			Label:       "Elastic IP",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Elastic IP to associate",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "operation", Values: []string{elasticIPOperationAssociate}},
				{Field: "region", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "operation", Values: []string{elasticIPOperationAssociate}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeElasticIP,
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
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "EC2 instance to associate the Elastic IP with",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "operation", Values: []string{elasticIPOperationAssociate}},
				{Field: "region", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "operation", Values: []string{elasticIPOperationAssociate}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instance",
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
			Name:        "associationId",
			Label:       "Association",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Elastic IP association to remove",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "operation", Values: []string{elasticIPOperationDisassociate}},
				{Field: "region", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "operation", Values: []string{elasticIPOperationDisassociate}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeElasticIPAssociation,
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

func (c *ManageElasticIP) Setup(ctx core.SetupContext) error {
	config := ManageElasticIPConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	operation := strings.TrimSpace(config.Operation)
	if !slices.Contains(validElasticIPOperations, operation) {
		return fmt.Errorf("invalid operation %q: must be one of associate, disassociate", config.Operation)
	}

	if err := validateManageElasticIPConfiguration(config, operation); err != nil {
		return err
	}

	metadata := ManageElasticIPNodeMetadata{
		Region:    region,
		Operation: operation,
	}

	if operation == elasticIPOperationAssociate {
		instanceID := strings.TrimSpace(config.InstanceID)
		if instanceID != "" {
			metadata.InstanceName = resolveInstanceName(ctx, region, instanceID)
		}
	}

	return ctx.Metadata.Set(metadata)
}

func (c *ManageElasticIP) Execute(ctx core.ExecutionContext) error {
	config := ManageElasticIPConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	operation := strings.TrimSpace(config.Operation)
	if !slices.Contains(validElasticIPOperations, operation) {
		return fmt.Errorf("invalid operation %q: must be one of associate, disassociate", config.Operation)
	}

	if err := validateManageElasticIPConfiguration(config, operation); err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)

	switch operation {
	case elasticIPOperationAssociate:
		return c.executeAssociate(ctx, client, region, config)
	case elasticIPOperationDisassociate:
		return c.executeDisassociate(ctx, client, region, config)
	default:
		return fmt.Errorf("invalid operation %q", operation)
	}
}

func (c *ManageElasticIP) executeAssociate(ctx core.ExecutionContext, client *Client, region string, config ManageElasticIPConfiguration) error {
	allocationID, err := requireAllocationID(config.AllocationID)
	if err != nil {
		return err
	}

	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	output, err := client.AssociateAddress(AssociateAddressInput{
		AllocationID:       allocationID,
		InstanceID:         instanceID,
		AllowReassociation: true,
	})
	if err != nil {
		return fmt.Errorf("failed to associate Elastic IP: %w", err)
	}

	payload := map[string]any{
		"associationId": output.AssociationID,
		"allocationId":  allocationID,
		"instanceId":    instanceID,
		"region":        region,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ManageElasticIPAssociatePayloadType,
		[]any{payload},
	)
}

func (c *ManageElasticIP) executeDisassociate(ctx core.ExecutionContext, client *Client, region string, config ManageElasticIPConfiguration) error {
	associationID, err := requireAssociationID(config.AssociationID)
	if err != nil {
		return err
	}

	if err := client.DisassociateAddress(associationID); err != nil {
		return fmt.Errorf("failed to disassociate Elastic IP: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ManageElasticIPDisassociatePayloadType,
		[]any{
			map[string]any{
				"associationId": associationID,
				"region":        region,
			},
		},
	)
}

func validateManageElasticIPConfiguration(config ManageElasticIPConfiguration, operation string) error {
	switch operation {
	case elasticIPOperationAssociate:
		if _, err := requireAllocationID(config.AllocationID); err != nil {
			return err
		}
		if _, err := requireInstanceID(config.InstanceID); err != nil {
			return err
		}
	case elasticIPOperationDisassociate:
		if _, err := requireAssociationID(config.AssociationID); err != nil {
			return err
		}
	}

	return nil
}

func (c *ManageElasticIP) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ManageElasticIP) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *ManageElasticIP) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ManageElasticIP) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ManageElasticIP) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ManageElasticIP) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
