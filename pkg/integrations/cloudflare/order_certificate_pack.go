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

const OrderCertificatePackPayloadType = "cloudflare.certificate_pack.ordered"

type OrderCertificatePack struct{}

type OrderCertificatePackSpec struct {
	Zone                 string   `json:"zone"`
	Hosts                []string `json:"hosts"`
	CertificateAuthority string   `json:"certificateAuthority"`
	ValidationMethod     string   `json:"validationMethod"`
	CloudflareBranding   *bool    `json:"cloudflareBranding,omitempty"`
}

func (c *OrderCertificatePack) Name() string {
	return "cloudflare.orderCertificatePack"
}

func (c *OrderCertificatePack) Label() string {
	return "Order Certificate Pack"
}

func (c *OrderCertificatePack) Description() string {
	return "Order a custom SSL/TLS certificate pack for a Cloudflare zone"
}

func (c *OrderCertificatePack) Documentation() string {
	return `The Order Certificate Pack component orders an advanced SSL/TLS certificate pack for custom hostnames in a Cloudflare zone.

## Use Cases

- **Preview environments**: Automatically provision SSL certificates for dynamically created preview subdomains
- **Custom domains**: Issue certificates for customer-owned domains behind Cloudflare
- **Wildcard coverage**: Order a certificate that covers both the apex and wildcard subdomains

## Configuration

- **Zone**: The Cloudflare zone to issue the certificate in
- **Hosts**: One or more hostnames the certificate should cover. Include both apex and wildcard variants (e.g., ` + "`example.com`" + ` and ` + "`*.example.com`" + `) if needed.
- **Certificate Authority**: The CA to use — Let's Encrypt (free, automated), Google, DigiCert, or SSL.com.
- **Validation Method**: How domain ownership is verified — TXT record, HTTP file, CNAME, or email.
- **Cloudflare Branding**: Whether to include Cloudflare branding on the certificate (optional).

## Output

Emits the ordered certificate pack including its ID, status, and covered hostnames. Status will typically be ` + "`initializing`" + ` or ` + "`pending_validation`" + ` immediately after ordering.`
}

func (c *OrderCertificatePack) Icon() string {
	return "shield-check"
}

func (c *OrderCertificatePack) Color() string {
	return "orange"
}

func (c *OrderCertificatePack) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *OrderCertificatePack) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare zone to order the certificate for",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		},
		{
			Name:        "hosts",
			Label:       "Hosts",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Hostnames the certificate should cover",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Hostname",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:     "certificateAuthority",
			Label:    "Certificate Authority",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "lets_encrypt",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Let's Encrypt", Value: "lets_encrypt"},
						{Label: "Google", Value: "google"},
						{Label: "DigiCert", Value: "digicert"},
						{Label: "SSL.com", Value: "ssl_com"},
					},
				},
			},
		},
		{
			Name:     "validationMethod",
			Label:    "Validation Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "txt",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "TXT Record", Value: "txt"},
						{Label: "HTTP File", Value: "http"},
						{Label: "CNAME", Value: "cname"},
						{Label: "Email", Value: "email"},
					},
				},
			},
		},
		{
			Name:        "cloudflareBranding",
			Label:       "Cloudflare Branding",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     false,
			Description: "Whether to include Cloudflare branding on the certificate",
		},
	}
}

func (c *OrderCertificatePack) Setup(ctx core.SetupContext) error {
	spec := OrderCertificatePackSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	return validateOrderCertificatePackSpec(spec)
}

func validateOrderCertificatePackSpec(spec OrderCertificatePackSpec) error {
	if strings.TrimSpace(spec.Zone) == "" {
		return errors.New("zone is required")
	}

	if len(spec.Hosts) == 0 {
		return errors.New("at least one host is required")
	}

	for i, h := range spec.Hosts {
		if strings.TrimSpace(h) == "" {
			return fmt.Errorf("hosts[%d] must not be blank", i)
		}
	}

	if strings.TrimSpace(spec.CertificateAuthority) == "" {
		return errors.New("certificateAuthority is required")
	}

	if strings.TrimSpace(spec.ValidationMethod) == "" {
		return errors.New("validationMethod is required")
	}

	return nil
}

func (c *OrderCertificatePack) Execute(ctx core.ExecutionContext) error {
	spec := OrderCertificatePackSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := validateOrderCertificatePackSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zoneID := resolveZoneID(spec.Zone, ctx.Integration)

	req := OrderCertificatePackRequest{
		CertificateAuthority: strings.TrimSpace(spec.CertificateAuthority),
		Hosts:                spec.Hosts,
		Type:                 "advanced",
		ValidationMethod:     strings.TrimSpace(spec.ValidationMethod),
		CloudflareBranding:   spec.CloudflareBranding,
	}

	pack, err := client.OrderCertificatePack(zoneID, req)
	if err != nil {
		return fmt.Errorf("failed to order certificate pack: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		OrderCertificatePackPayloadType,
		[]any{map[string]any{
			"zoneId": zoneID,
			"packId": pack.ID,
			"pack":   pack,
		}},
	)
}

func (c *OrderCertificatePack) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *OrderCertificatePack) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *OrderCertificatePack) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *OrderCertificatePack) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *OrderCertificatePack) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *OrderCertificatePack) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
