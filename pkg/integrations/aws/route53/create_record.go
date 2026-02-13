package route53

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

type CreateRecord struct{}

type CreateRecordConfiguration struct {
	HostedZoneID string   `json:"hostedZoneId" mapstructure:"hostedZoneId"`
	RecordName   string   `json:"recordName" mapstructure:"recordName"`
	RecordType   string   `json:"recordType" mapstructure:"recordType"`
	TTL          int      `json:"ttl" mapstructure:"ttl"`
	Values       []string `json:"values" mapstructure:"values"`
}

type CreateRecordMetadata struct {
	HostedZoneID string `json:"hostedZoneId" mapstructure:"hostedZoneId"`
	RecordName   string `json:"recordName" mapstructure:"recordName"`
}

func (c *CreateRecord) Name() string {
	return "aws.route53.createRecord"
}

func (c *CreateRecord) Label() string {
	return "Route 53 • Create DNS Record"
}

func (c *CreateRecord) Description() string {
	return "Create a DNS record in an AWS Route 53 hosted zone"
}

func (c *CreateRecord) Documentation() string {
	return `The Create DNS Record component creates a new DNS record in an AWS Route 53 hosted zone.

## Use Cases

- **Domain management**: Create DNS records for new services or endpoints
- **Automated provisioning**: Set up DNS entries as part of infrastructure workflows
- **Multi-environment setup**: Create environment-specific DNS records automatically

## How It Works

1. Connects to AWS Route 53 using the integration credentials
2. Creates a new DNS record in the specified hosted zone
3. Returns the change status and submission timestamp
`
}

func (c *CreateRecord) Icon() string {
	return "aws"
}

func (c *CreateRecord) Color() string {
	return "gray"
}

func (c *CreateRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRecord) Configuration() []configuration.Field {
	return recordConfigurationFields()
}

func (c *CreateRecord) Setup(ctx core.SetupContext) error {
	var config CreateRecordConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if err := validateRecordConfiguration(config.HostedZoneID, config.RecordName, config.RecordType, config.Values); err != nil {
		return err
	}

	return ctx.Metadata.Set(CreateRecordMetadata{
		HostedZoneID: config.HostedZoneID,
		RecordName:   config.RecordName,
	})
}

func (c *CreateRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateRecord) Execute(ctx core.ExecutionContext) error {
	var config CreateRecordConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds)
	result, err := client.ChangeResourceRecordSets(config.HostedZoneID, "CREATE", ResourceRecordSet{
		Name:   config.RecordName,
		Type:   config.RecordType,
		TTL:    config.TTL,
		Values: config.Values,
	})
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	output := map[string]any{
		"changeId":    result.ID,
		"status":      result.Status,
		"submittedAt": result.SubmittedAt,
		"recordName":  config.RecordName,
		"recordType":  config.RecordType,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.route53.record",
		[]any{output},
	)
}

func (c *CreateRecord) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateRecord) normalizeConfig(config CreateRecordConfiguration) CreateRecordConfiguration {
	config.HostedZoneID = strings.TrimSpace(config.HostedZoneID)
	config.RecordName = strings.TrimSpace(config.RecordName)
	config.RecordType = strings.TrimSpace(config.RecordType)
	config.Values = normalizeValues(config.Values)
	return config
}
