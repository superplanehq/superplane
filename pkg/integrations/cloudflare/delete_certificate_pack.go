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

Emits the deleted pack ID, resolved zone ID, zone name when known from integration metadata, certificate hostnames when returned by the Cloudflare API (before deletion), and ` + "`deleted: true`" + `.`
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
					Type:           "certificate_pack",
					UseNameAsValue: true,
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
	var packHosts []string
	if strings.TrimSpace(zoneFragment) != "" {
		zoneID = resolveZoneID(strings.TrimSpace(zoneFragment), ctx.Integration)
		packHosts = hostsForCertificatePack(ctx.Logger, client, zoneID, packID)
	} else {
		var lookupErr error
		zoneID, packID, packHosts, lookupErr = lookupCertificatePackReference(ctx.Logger, client, ctx.Integration, packID)
		if lookupErr != nil {
			return lookupErr
		}
	}
	if strings.TrimSpace(zoneID) == "" {
		return fmt.Errorf("could not resolve Cloudflare zone for certificate pack %q", packValue)
	}

	if err := client.DeleteCertificatePack(zoneID, packID); err != nil {
		return fmt.Errorf("failed to delete certificate pack: %w", err)
	}

	payload := map[string]any{
		"zoneId":  zoneID,
		"packId":  packID,
		"deleted": true,
	}
	if zn := resolveZoneName(zoneID, ctx.Integration); zn != "" {
		payload["zoneName"] = zn
	}
	if len(packHosts) > 0 {
		payload["hosts"] = packHosts
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteCertificatePackPayloadType,
		[]any{payload},
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

func lookupCertificatePackReference(logger *log.Entry, client *Client, integration core.IntegrationContext, reference string) (zoneID string, packID string, hosts []string, err error) {
	metadata := Metadata{}
	if decodeErr := mapstructure.Decode(integration.GetMetadata(), &metadata); decodeErr != nil {
		return "", "", nil, fmt.Errorf("failed to decode integration metadata: %w", decodeErr)
	}
	if len(metadata.Zones) == 0 {
		return "", "", nil, fmt.Errorf("cannot resolve zone for certificate pack %q: no zones in integration metadata", reference)
	}
	for _, zone := range metadata.Zones {
		packs, listErr := client.ListCertificatePacks(zone.ID)
		if listErr != nil {
			if logger != nil {
				logger.WithError(listErr).WithField("zone_id", zone.ID).WithField("zone_name", zone.Name).
					Warn("failed to list certificate packs for zone while resolving certificate pack, skipping zone")
			}
			continue
		}
		for _, pack := range packs {
			if certificatePackMatchesReference(zone.Name, pack, reference) {
				return zone.ID, pack.ID, pack.Hosts, nil
			}
		}
	}
	return "", "", nil, fmt.Errorf("certificate pack %q not found in any configured zone", reference)
}

func certificatePackMatchesReference(zoneName string, pack CertificatePack, reference string) bool {
	value := strings.TrimSpace(reference)
	return pack.ID == value || certificatePackResourceName(zoneName, pack) == value
}

func certificatePackResourceName(zoneName string, pack CertificatePack) string {
	label := pack.ID
	if len(pack.Hosts) > 0 {
		label = strings.Join(pack.Hosts, ", ")
	}
	if strings.TrimSpace(zoneName) == "" {
		return label
	}
	return fmt.Sprintf("%s - %s", zoneName, label)
}

func hostsForCertificatePack(logger *log.Entry, client *Client, zoneID, packID string) []string {
	packs, err := client.ListCertificatePacks(zoneID)
	if err != nil {
		if logger != nil {
			logger.WithError(err).WithField("zone_id", zoneID).
				Warn("failed to list certificate packs for zone while enriching delete payload")
		}
		return nil
	}
	for _, pack := range packs {
		if pack.ID == packID {
			return pack.Hosts
		}
	}
	return nil
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
