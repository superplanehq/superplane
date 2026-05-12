package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
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

	zoneFragment, packID := splitCertificatePackReference(packValue)
	if strings.TrimSpace(packID) == "" {
		return errors.New("certificate pack id is required")
	}

	var zoneID string
	if strings.TrimSpace(zoneFragment) != "" {
		zoneID = resolveZoneID(strings.TrimSpace(zoneFragment), ctx.Integration)
	} else {
		zoneID, err = lookupZoneForCertificatePack(ctx.Logger, client, ctx.Integration, packID)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(zoneID) == "" {
		return fmt.Errorf("could not resolve Cloudflare zone for certificate pack %q", packValue)
	}

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

// splitCertificatePackReference splits a "{zone}/{pack_id}" resource id (from the picker) into parts.
// If there is no slash, the entire value is the pack ID and the zone must be discovered via the API.
func splitCertificatePackReference(value string) (zoneFragment, packID string) {
	if zonePart, packPart, ok := strings.Cut(value, "/"); ok && strings.TrimSpace(zonePart) != "" && strings.TrimSpace(packPart) != "" {
		return strings.TrimSpace(zonePart), strings.TrimSpace(packPart)
	}
	return "", strings.TrimSpace(value)
}

func lookupZoneForCertificatePack(logger *log.Entry, client *Client, integration core.IntegrationContext, packID string) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return "", fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if len(metadata.Zones) == 0 {
		return "", fmt.Errorf("cannot resolve zone for certificate pack %q: no zones in integration metadata", packID)
	}
	for _, zone := range metadata.Zones {
		packs, err := client.ListCertificatePacks(zone.ID)
		if err != nil {
			if logger != nil {
				logger.WithError(err).WithField("zone_id", zone.ID).WithField("zone_name", zone.Name).
					Warn("failed to list certificate packs for zone while resolving certificate pack, skipping zone")
			}
			continue
		}
		for _, pack := range packs {
			if pack.ID == packID {
				return zone.ID, nil
			}
		}
	}
	return "", fmt.Errorf("certificate pack %q not found in any configured zone", packID)
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
