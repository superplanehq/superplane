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

type UpsertRecord struct{}

type UpsertRecordConfiguration struct {
	HostedZoneID string   `json:"hostedZoneId" mapstructure:"hostedZoneId"`
	RecordName   string   `json:"recordName" mapstructure:"recordName"`
	RecordType   string   `json:"recordType" mapstructure:"recordType"`
	TTL          int      `json:"ttl" mapstructure:"ttl"`
	Values       []string `json:"values" mapstructure:"values"`
}

func (c *UpsertRecord) Name() string {
	return "aws.route53.upsertRecord"
}

func (c *UpsertRecord) Label() string {
	return "Route 53 • Upsert DNS Record"
}

func (c *UpsertRecord) Description() string {
	return "Create or update a DNS record in an AWS Route 53 hosted zone"
}

func (c *UpsertRecord) Documentation() string {
	return `The Upsert DNS Record component creates or updates a DNS record in an AWS Route 53 hosted zone.

## Use Cases

- **Idempotent updates**: Safely create or update DNS records without checking existence first
- **Rolling deployments**: Update DNS records to point to new infrastructure
- **Failover management**: Switch DNS records between primary and secondary endpoints

## How It Works

1. Connects to AWS Route 53 using the integration credentials
2. Creates the DNS record if it doesn't exist, or updates it if it does
3. Returns the change status and submission timestamp
`
}

func (c *UpsertRecord) Icon() string {
	return "aws"
}

func (c *UpsertRecord) Color() string {
	return "gray"
}

func (c *UpsertRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpsertRecord) Configuration() []configuration.Field {
	return recordConfigurationFields()
}

func (c *UpsertRecord) Setup(ctx core.SetupContext) error {
	var config UpsertRecordConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	return validateRecordConfiguration(config.HostedZoneID, config.RecordName, config.RecordType, config.Values)
}

func (c *UpsertRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpsertRecord) Execute(ctx core.ExecutionContext) error {
	var config UpsertRecordConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds)
	result, err := client.ChangeResourceRecordSets(config.HostedZoneID, "UPSERT", ResourceRecordSet{
		Name:   config.RecordName,
		Type:   config.RecordType,
		TTL:    config.TTL,
		Values: config.Values,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert DNS record: %w", err)
	}

	if result.Status == "INSYNC" {
		output := map[string]any{
			"change": map[string]any{
				"id":          result.ID,
				"status":      result.Status,
				"submittedAt": result.SubmittedAt,
			},
			"record": map[string]any{
				"name": config.RecordName,
				"type": config.RecordType,
			},
		}
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"aws.route53.change",
			[]any{output},
		)
	}

	if err := ctx.Metadata.Set(RecordChangePollMetadata{
		ChangeID:    result.ID,
		RecordName:  config.RecordName,
		RecordType:  config.RecordType,
		SubmittedAt: result.SubmittedAt,
	}); err != nil {
		return fmt.Errorf("failed to set poll metadata: %w", err)
	}
	return ctx.Requests.ScheduleActionCall(
		pollChangeActionName,
		map[string]any{},
		pollInterval,
	)
}

func (c *UpsertRecord) Actions() []core.Action {
	return []core.Action{
		{
			Name:        pollChangeActionName,
			Description: "Poll for change status",
		},
	}
}

func (c *UpsertRecord) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case pollChangeActionName:
		return pollChangeUntilSynced(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *UpsertRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpsertRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpsertRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpsertRecord) normalizeConfig(config UpsertRecordConfiguration) UpsertRecordConfiguration {
	config.HostedZoneID = strings.TrimSpace(config.HostedZoneID)
	config.RecordName = strings.TrimSpace(config.RecordName)
	config.RecordType = strings.TrimSpace(config.RecordType)
	config.Values = normalizeValues(config.Values)
	return config
}
