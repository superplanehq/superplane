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

type DeleteRecord struct{}

type DeleteRecordConfiguration struct {
	HostedZoneID string   `json:"hostedZoneId" mapstructure:"hostedZoneId"`
	RecordName   string   `json:"recordName" mapstructure:"recordName"`
	RecordType   string   `json:"recordType" mapstructure:"recordType"`
	TTL          int      `json:"ttl" mapstructure:"ttl"`
	Values       []string `json:"values" mapstructure:"values"`
}

func (c *DeleteRecord) Name() string {
	return "aws.route53.deleteRecord"
}

func (c *DeleteRecord) Label() string {
	return "Route 53 • Delete DNS Record"
}

func (c *DeleteRecord) Description() string {
	return "Delete a DNS record from an AWS Route 53 hosted zone"
}

func (c *DeleteRecord) Documentation() string {
	return `The Delete DNS Record component deletes a DNS record from an AWS Route 53 hosted zone.

## Use Cases

- **Cleanup**: Remove DNS records when decommissioning services
- **Environment teardown**: Delete DNS entries for temporary environments
- **Migration**: Remove old DNS records after migrating to new endpoints

## How It Works

1. Connects to AWS Route 53 using the integration credentials
2. Deletes the specified DNS record from the hosted zone
3. The record name, type, TTL, and values must match the existing record exactly
4. Returns the change status and submission timestamp
`
}

func (c *DeleteRecord) Icon() string {
	return "aws"
}

func (c *DeleteRecord) Color() string {
	return "gray"
}

func (c *DeleteRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteRecord) Configuration() []configuration.Field {
	return recordConfigurationFields()
}

func (c *DeleteRecord) Setup(ctx core.SetupContext) error {
	var config DeleteRecordConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	return validateRecordConfiguration(config.HostedZoneID, config.RecordName, config.RecordType, config.Values)
}

func (c *DeleteRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteRecord) Execute(ctx core.ExecutionContext) error {
	var config DeleteRecordConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds)
	result, err := client.ChangeResourceRecordSets(config.HostedZoneID, "DELETE", ResourceRecordSet{
		Name:   config.RecordName,
		Type:   config.RecordType,
		TTL:    config.TTL,
		Values: config.Values,
	})
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
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

func (c *DeleteRecord) Actions() []core.Action {
	return []core.Action{
		{
			Name:        pollChangeActionName,
			Description: "Poll for change status",
		},
	}
}

func (c *DeleteRecord) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case pollChangeActionName:
		return pollChangeUntilSynced(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *DeleteRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteRecord) normalizeConfig(config DeleteRecordConfiguration) DeleteRecordConfiguration {
	config.HostedZoneID = strings.TrimSpace(config.HostedZoneID)
	config.RecordName = strings.TrimSpace(config.RecordName)
	config.RecordType = strings.TrimSpace(config.RecordType)
	config.Values = normalizeValues(config.Values)
	return config
}
