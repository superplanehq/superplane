package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteDNSRecord struct{}

type DeleteDNSRecordSpec struct {
	Domain   string `json:"domain"`
	RecordID string `json:"recordId"`
}

func (d *DeleteDNSRecord) Name() string {
	return "digitalocean.deleteDNSRecord"
}

func (d *DeleteDNSRecord) Label() string {
	return "Delete DNS Record"
}

func (d *DeleteDNSRecord) Description() string {
	return "Delete a DNS record from a DigitalOcean domain"
}

func (d *DeleteDNSRecord) Documentation() string {
	return `The Delete DNS Record component permanently removes a DNS record from a DigitalOcean-managed domain.

## Use Cases

- **Cleanup**: Remove DNS records for decommissioned services
- **Rotation**: Delete old records as part of a DNS rotation workflow
- **Automated teardown**: Remove service discovery records when tearing down infrastructure

## Configuration

- **Domain**: The DigitalOcean-managed domain containing the record (required)
- **Record ID**: The ID of the DNS record to delete (required, supports expressions)

## Output

Returns information about the deleted record:
- **recordId**: The ID of the record that was deleted
- **domain**: The domain the record belonged to

## Important Notes

- This operation is **permanent** and cannot be undone
- Deleting a record that does not exist is treated as a success (idempotent)
- Record IDs can be obtained from the output of createDNSRecord or upsertDNSRecord`
}

func (d *DeleteDNSRecord) Icon() string {
	return "trash-2"
}

func (d *DeleteDNSRecord) Color() string {
	return "red"
}

func (d *DeleteDNSRecord) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteDNSRecord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "domain",
			Label:       "Domain",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The DigitalOcean-managed domain containing the record",
			Placeholder: "Select domain",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "domain",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "recordId",
			Label:       "Record ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The DNS record to delete",
			Placeholder: "Select a DNS record",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "dns_record",
					UseNameAsValue: false,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "domain",
							ValueFrom: &configuration.ParameterValueFrom{Field: "domain"},
						},
					},
				},
			},
		},
	}
}

func (d *DeleteDNSRecord) Setup(ctx core.SetupContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Domain == "" {
		return errors.New("domain is required")
	}

	if spec.RecordID == "" {
		return errors.New("recordId is required")
	}

	if err := resolveDNSRecordMetadata(ctx, spec.Domain, spec.RecordID); err != nil {
		return fmt.Errorf("error resolving record metadata: %v", err)
	}

	return nil
}

func (d *DeleteDNSRecord) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDNSRecordSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.Contains(spec.RecordID, "{{") {
		return fmt.Errorf("recordId expression was not resolved")
	}

	recordID, err := strconv.Atoi(spec.RecordID)
	if err != nil {
		return fmt.Errorf("invalid recordId %q: must be a number", spec.RecordID)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DeleteDNSRecord(spec.Domain, recordID)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			// Record already deleted, emit success (idempotent)
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.dns.record.deleted",
				[]any{map[string]any{"recordId": recordID, "domain": spec.Domain, "deleted": true}},
			)
		}
		return fmt.Errorf("failed to delete DNS record: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.dns.record.deleted",
		[]any{map[string]any{"recordId": recordID, "domain": spec.Domain, "deleted": true}},
	)
}

func (d *DeleteDNSRecord) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteDNSRecord) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteDNSRecord) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteDNSRecord) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeleteDNSRecord) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteDNSRecord) Cleanup(ctx core.SetupContext) error {
	return nil
}
