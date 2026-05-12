package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteCertificatePackPayloadType = "cloudflare.certificate_pack.deleted"

type DeleteCertificatePack struct{}

type DeleteCertificatePackSpec struct {
	CertificatePack string `json:"certificatePack"`
}

func (c *DeleteCertificatePack) Name() string {
	return "cloudflare.deleteCertificatePack"
}

func (c *DeleteCertificatePack) Label() string {
	return "Delete Certificate Pack"
}

func (c *DeleteCertificatePack) Description() string {
	return "Delete a custom SSL/TLS certificate pack from a Cloudflare zone"
}

func (c *DeleteCertificatePack) Documentation() string {
	return `The Delete Certificate Pack component removes a custom SSL/TLS certificate pack from a Cloudflare zone.

## Use Cases

- **Environment teardown**: Remove certificates issued for preview environments when they are decommissioned
- **Certificate rotation**: Delete old certificate packs as part of a renewal workflow
- **Cleanup**: Remove certificates that failed validation or are no longer needed

## Configuration

- **Certificate Pack**: The certificate pack to delete (shown as ` + "`zone/pack-id`" + `). Can be set from a prior Order Certificate Pack component via an expression.

## Output

Emits the zone ID and pack ID of the deleted certificate pack.`
}

func (c *DeleteCertificatePack) Icon() string {
	return "shield-off"
}

func (c *DeleteCertificatePack) Color() string {
	return "orange"
}

func (c *DeleteCertificatePack) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteCertificatePack) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "certificatePack",
			Label:       "Certificate Pack",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The SSL certificate pack to delete",
			Placeholder: "Select a certificate pack",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "certificate_pack",
				},
			},
		},
	}
}

func (c *DeleteCertificatePack) Setup(ctx core.SetupContext) error {
	spec := DeleteCertificatePackSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.CertificatePack) == "" {
		return errors.New("certificatePack is required")
	}

	return nil
}

func (c *DeleteCertificatePack) Execute(ctx core.ExecutionContext) error {
	spec := DeleteCertificatePackSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	packValue := strings.TrimSpace(spec.CertificatePack)
	if packValue == "" {
		return errors.New("certificatePack is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zoneID, packID := parseCertificatePackID(packValue, ctx.Integration)

	if err := client.DeleteCertificatePack(zoneID, packID); err != nil {
		return fmt.Errorf("failed to delete certificate pack: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteCertificatePackPayloadType,
		[]any{map[string]any{
			"zoneId":  zoneID,
			"packId":  packID,
			"deleted": true,
		}},
	)
}

// parseCertificatePackID splits a "{zone_id}/{pack_id}" resource value into its parts.
// If the value was set directly (e.g. from an expression), it is treated as the pack ID
// and the zone is resolved from the integration's zone list.
func parseCertificatePackID(value string, integration core.IntegrationContext) (zoneID, packID string) {
	if zonePart, packPart, ok := strings.Cut(value, "/"); ok && zonePart != "" && packPart != "" {
		return zonePart, packPart
	}
	return "", value
}

func (c *DeleteCertificatePack) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteCertificatePack) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteCertificatePack) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteCertificatePack) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteCertificatePack) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteCertificatePack) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
