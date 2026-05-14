package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
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
	ValidityDays         string   `json:"validityDays,omitempty"`
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
- **Certificate Authority**: The CA to use — Let's Encrypt (free, automated), Google, or SSL.com.
- **Validation Method**: How domain ownership is verified — TXT record, HTTP file, or email.
- **Certificate Validity Period**: How long the certificate should be valid. Available for Google and SSL.com certificates.
- **Cloudflare Branding**: Whether to include Cloudflare branding on the certificate (optional).

## Output

Emits the resolved zone ID, zone name when known from integration metadata, pack ID, and the ordered certificate pack object (including covered hostnames). Status will typically be ` + "`initializing`" + ` or ` + "`pending_validation`" + ` immediately after ordering.`
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
	validitySupportedCA := []configuration.VisibilityCondition{
		{Field: "certificateAuthority", Values: []string{"google", "ssl_com"}},
	}

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
						{Label: "Email", Value: "email"},
					},
				},
			},
		},
		{
			Name:                 "validityDays",
			Label:                "Certificate Validity Period",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              "90",
			Description:          "How long the certificate should be valid",
			VisibilityConditions: validitySupportedCA,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "certificateAuthority", Values: []string{"google", "ssl_com"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "2 weeks", Value: "14"},
						{Label: "1 month", Value: "30"},
						{Label: "3 months", Value: "90"},
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

	certificateAuthority := strings.TrimSpace(spec.CertificateAuthority)
	if !slices.Contains([]string{"lets_encrypt", "google", "ssl_com"}, certificateAuthority) {
		return fmt.Errorf("unsupported certificateAuthority: %s", certificateAuthority)
	}

	if strings.TrimSpace(spec.ValidationMethod) == "" {
		return errors.New("validationMethod is required")
	}

	validationMethod := strings.TrimSpace(spec.ValidationMethod)
	if !slices.Contains([]string{"txt", "http", "email"}, validationMethod) {
		return fmt.Errorf("unsupported validationMethod: %s", validationMethod)
	}

	if certificateAuthoritySupportsValidityDays(certificateAuthority) {
		if strings.TrimSpace(spec.ValidityDays) == "" {
			return errors.New("validityDays is required for the selected certificateAuthority")
		}

		if _, err := parseCertificateValidityDays(spec.ValidityDays); err != nil {
			return err
		}
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

	if certificateAuthoritySupportsValidityDays(req.CertificateAuthority) {
		validityDays, err := parseCertificateValidityDays(spec.ValidityDays)
		if err != nil {
			return err
		}
		req.ValidityDays = &validityDays
	}

	pack, err := client.OrderCertificatePack(zoneID, req)
	if err != nil {
		return fmt.Errorf("failed to order certificate pack: %w", err)
	}

	payload := map[string]any{
		"zoneId": zoneID,
		"packId": pack.ID,
		"pack":   pack,
	}
	if zn := resolveZoneName(zoneID, ctx.Integration); zn != "" {
		payload["zoneName"] = zn
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		OrderCertificatePackPayloadType,
		[]any{payload},
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

func certificateAuthoritySupportsValidityDays(certificateAuthority string) bool {
	return slices.Contains([]string{"google", "ssl_com"}, certificateAuthority)
}

func parseCertificateValidityDays(value string) (int, error) {
	validityDays, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("validityDays must be one of: 14, 30, 90")
	}

	if !slices.Contains([]int{14, 30, 90}, validityDays) {
		return 0, fmt.Errorf("validityDays must be one of: 14, 30, 90")
	}

	return validityDays, nil
}
