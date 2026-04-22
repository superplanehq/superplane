package oci

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("oci", &OCI{})
}

type OCI struct{}

type Configuration struct {
	TenancyOCID string `json:"tenancyOcid" mapstructure:"tenancyOcid"`
	UserOCID    string `json:"userOcid" mapstructure:"userOcid"`
	Fingerprint string `json:"fingerprint" mapstructure:"fingerprint"`
	PrivateKey  string `json:"privateKey" mapstructure:"privateKey"`
	Region      string `json:"region" mapstructure:"region"`
}

func (o *OCI) Name() string {
	return "oci"
}

func (o *OCI) Label() string {
	return "Oracle Cloud Infrastructure"
}

func (o *OCI) Icon() string {
	return "oci"
}

func (o *OCI) Description() string {
	return "Manage Oracle Cloud Infrastructure resources in workflows"
}

func (o *OCI) Instructions() string {
	return `## Connect Oracle Cloud Infrastructure

SuperPlane authenticates to OCI using API Key authentication.

### Steps

1. Open the [OCI Console](https://cloud.oracle.com/) and sign in.
2. Go to **Profile → User settings → My profile → Tokens and keys → API keys → Add API key**.
3. Choose **Generate API Key Pair**, download the private key, and click **Add**.
4. After the key is added, copy the **Configuration File Preview** values:
   - **Tenancy OCID** (begins with ` + "`ocid1.tenancy.`" + `)
   - **User OCID** (begins with ` + "`ocid1.user.`" + `)
   - **Fingerprint** (e.g. ` + "`12:34:56:…`" + `)
5. Open the downloaded private key file and paste its full contents into the **Private Key** field below.
6. Select the **Region** that matches your OCI tenancy's home region.

> The API key's user must have the IAM policy ` + "`Allow group <group> to manage instances in compartment <compartment>`" + ` (or equivalent) to use the Compute components.`
}

func (o *OCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tenancyOcid",
			Label:       "Tenancy OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your tenancy OCID (ocid1.tenancy.oc1..…)",
			Placeholder: "ocid1.tenancy.oc1..",
		},
		{
			Name:        "userOcid",
			Label:       "User OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the IAM user whose API key is used",
			Placeholder: "ocid1.user.oc1..",
		},
		{
			Name:        "fingerprint",
			Label:       "Key Fingerprint",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "MD5 fingerprint of the uploaded API public key (e.g. 12:34:56:78:…)",
			Placeholder: "12:34:56:78:90:ab:cd:ef:12:34:56:78:90:ab:cd:ef",
		},
		{
			Name:        "privateKey",
			Label:       "Private Key (PEM)",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Sensitive:   true,
			Description: "PEM-encoded RSA private key corresponding to the uploaded public key",
			Placeholder: "-----BEGIN RSA PRIVATE KEY-----\n…\n-----END RSA PRIVATE KEY-----",
		},
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-ashburn-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: allRegions,
				},
			},
		},
	}
}

func (o *OCI) Components() []core.Component {
	return []core.Component{
		&CreateComputeInstance{},
	}
}

func (o *OCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnComputeInstanceCreated{},
	}
}

func (o *OCI) Sync(ctx core.SyncContext) error {
	cfg := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return err
	}

	// Validate credentials by calling the identity endpoint for the current user.
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("OCI credential validation failed: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OCI) HandleRequest(ctx core.HTTPRequestContext) {}

func (o *OCI) Actions() []core.Action {
	return []core.Action{}
}

func (o *OCI) HandleAction(ctx core.IntegrationActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func validateConfig(cfg Configuration) error {
	if strings.TrimSpace(cfg.TenancyOCID) == "" {
		return fmt.Errorf("tenancyOcid is required")
	}
	if strings.TrimSpace(cfg.UserOCID) == "" {
		return fmt.Errorf("userOcid is required")
	}
	if strings.TrimSpace(cfg.Fingerprint) == "" {
		return fmt.Errorf("fingerprint is required")
	}
	if strings.TrimSpace(cfg.PrivateKey) == "" {
		return fmt.Errorf("privateKey is required")
	}
	if strings.TrimSpace(cfg.Region) == "" {
		return fmt.Errorf("region is required")
	}
	return nil
}

var allRegions = []configuration.FieldOption{
	{Label: "US East (Ashburn)", Value: "us-ashburn-1"},
	{Label: "US West (Phoenix)", Value: "us-phoenix-1"},
	{Label: "US Midwest (Chicago)", Value: "us-chicago-1"},
	{Label: "US West (San Jose)", Value: "us-sanjose-1"},
	{Label: "Canada (Montreal)", Value: "ca-montreal-1"},
	{Label: "Canada (Toronto)", Value: "ca-toronto-1"},
	{Label: "Brazil (Sao Paulo)", Value: "sa-saopaulo-1"},
	{Label: "Brazil (Vinhedo)", Value: "sa-vinhedo-1"},
	{Label: "Chile (Santiago)", Value: "sa-santiago-1"},
	{Label: "UK South (London)", Value: "uk-london-1"},
	{Label: "UK West (Cardiff)", Value: "uk-cardiff-1"},
	{Label: "Germany (Frankfurt)", Value: "eu-frankfurt-1"},
	{Label: "Netherlands (Amsterdam)", Value: "eu-amsterdam-1"},
	{Label: "Spain (Madrid)", Value: "eu-madrid-1"},
	{Label: "France (Paris)", Value: "eu-paris-1"},
	{Label: "Sweden (Stockholm)", Value: "eu-stockholm-1"},
	{Label: "Italy (Milan)", Value: "eu-milan-1"},
	{Label: "Switzerland (Zurich)", Value: "eu-zurich-1"},
	{Label: "Japan (Tokyo)", Value: "ap-tokyo-1"},
	{Label: "Japan (Osaka)", Value: "ap-osaka-1"},
	{Label: "South Korea (Seoul)", Value: "ap-seoul-1"},
	{Label: "South Korea (Chuncheon)", Value: "ap-chuncheon-1"},
	{Label: "Australia (Sydney)", Value: "ap-sydney-1"},
	{Label: "Australia (Melbourne)", Value: "ap-melbourne-1"},
	{Label: "India (Mumbai)", Value: "ap-mumbai-1"},
	{Label: "India (Hyderabad)", Value: "ap-hyderabad-1"},
	{Label: "Singapore", Value: "ap-singapore-1"},
	{Label: "Israel (Jerusalem)", Value: "il-jerusalem-1"},
	{Label: "UAE (Dubai)", Value: "me-dubai-1"},
	{Label: "UAE (Abu Dhabi)", Value: "me-abudhabi-1"},
	{Label: "Saudi Arabia (Jeddah)", Value: "me-jeddah-1"},
	{Label: "South Africa (Johannesburg)", Value: "af-johannesburg-1"},
}
