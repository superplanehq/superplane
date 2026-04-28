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
	registry.RegisterIntegrationWithWebhookHandler("oci", &OCI{}, &WebhookHandler{})
}

type OCI struct{}

type Configuration struct {
	TenancyOCID string `json:"tenancyOcid" mapstructure:"tenancyOcid"`
	UserOCID    string `json:"userOcid" mapstructure:"userOcid"`
	Fingerprint string `json:"fingerprint" mapstructure:"fingerprint"`
	PrivateKey  string `json:"privateKey" mapstructure:"privateKey"`
	Region      string `json:"region" mapstructure:"region"`
}

// IntegrationMetadata holds resources created during integration setup.
type IntegrationMetadata struct {
	TopicID string `json:"topicId" mapstructure:"topicId"`
	// CompartmentRules maps compartment OCID → Events rule OCID.
	// One shared rule is created per compartment, reused across all triggers.
	CompartmentRules map[string]string `json:"compartmentRules" mapstructure:"compartmentRules"`
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

SuperPlane authenticates to OCI using API Key authentication tied to a dedicated service user with least-privilege permissions.

### Part 1 — Create a Dedicated Group and Service User

1. Open the [OCI Console](https://cloud.oracle.com/) and sign in.
2. Go to **Identity & Security → Domains → Default → Groups**.
3. Click **Create Group**.
4. Set the name to ` + "`SuperPlaneIntegration`" + ` and add a description, then click **Create**.
5. In the same Domain, go to **Users → Create User**.
6. Fill in the details:
   - **Username:** ` + "`superplane-integration`" + `
   - **Email:** use integrations@superplane.com or any valid email (not used for authentication)
   - **Description:** SuperPlane integration user
7. In the **Groups** section, assign them to the ` + "`SuperPlaneIntegration`" + ` group
8. Click **Create**.

### Part 2 — Create an IAM Policy

1. Go to **Identity & Security → Policies**.
2. Make sure you are in the **root compartment** (check the Compartment selector on the left).
3. Click **Create Policy**, name it ` + "`SuperPlanePolicies`" + `, and enable the **manual editor**.
4. Paste in the following statements, replacing ` + "`<your-compartment>`" + ` with your target compartment name and Click **Create**.:
` + "```" + `
Allow group SuperPlaneIntegration to manage instances
  in compartment <your-compartment>

Allow group SuperPlaneIntegration to manage compute-images
  in compartment <your-compartment>

Allow group SuperPlaneIntegration to use virtual-network-family
  in compartment <your-compartment>
` + "```" + `
 
### Part 3 — Generate API Keys for the Service User and Connect to Superplane

1. While still on the service user's page, go to **API keys → Add API key**.
2. Choose **Generate API Key Pair**, download the private key, and click **Add**.
3. Copy the **Configuration File Preview** values that appear to the UI:
    - **User OCID** (begins with ` + "`ocid1.user.`" + `)
    - **Fingerprint** (e.g. ` + "`12:34:56:…`" + `)
    - **Tenancy OCID** (begins with ` + "`ocid1.tenancy.`" + `)
4. Select the **Region** that matches your OCI tenancy's home region.
5. Open the downloaded private key file and paste its full contents into the **Private Key** field.
6. Click **Connect** to validate the credentials and save the integration.`
}

func (o *OCI) Configuration() []configuration.Field {
	return []configuration.Field{
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
			Label:       "Fingerprint",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "MD5 fingerprint of the uploaded API public key (e.g. 12:34:56:78:…)",
			Placeholder: "12:34:56:78:90:ab:cd:ef:12:34:56:78:90:ab:cd:ef",
		},
		{
			Name:        "tenancyOcid",
			Label:       "Tenancy OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your tenancy OCID (ocid1.tenancy.oc1..…)",
			Placeholder: "ocid1.tenancy.oc1..",
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
		{
			Name:        "privateKey",
			Label:       "Private Key (PEM)",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Sensitive:   true,
			Description: "PEM-encoded RSA private key corresponding to the uploaded public key",
			Placeholder: "-----BEGIN PRIVATE KEY-----\n…\n-----END PRIVATE KEY-----",
		},
	}
}

func (o *OCI) Actions() []core.Action {
	return []core.Action{
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("OCI credential validation failed: %w", err)
	}

	// Read existing metadata to check if the topic was already created.
	var metadata IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	// Create the shared ONS topic once; idempotent across re-syncs.
	// The per-trigger HTTPS subscription (webhook) is created in OnComputeInstanceCreated.Setup().
	if metadata.TopicID == "" {
		topicName := fmt.Sprintf("superplane-%s", ctx.Integration.ID())
		topic, err := client.CreateONSTopic(cfg.TenancyOCID, topicName)
		if err != nil {
			return fmt.Errorf("failed to create ONS topic: %w", err)
		}
		metadata.TopicID = topic.TopicID
		ctx.Integration.SetMetadata(metadata)
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	var metadata IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Warnf("failed to decode OCI integration metadata during cleanup: %v", err)
		return nil
	}

	if metadata.TopicID == "" && len(metadata.CompartmentRules) == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client during cleanup: %w", err)
	}

	for compartmentID, ruleID := range metadata.CompartmentRules {
		if err := client.DeleteEventsRule(ruleID); err != nil {
			ctx.Logger.Warnf("failed to delete Events rule %q (compartment %q) during cleanup: %v", ruleID, compartmentID, err)
		}
	}

	if err := client.DeleteONSTopic(metadata.TopicID); err != nil {
		ctx.Logger.Warnf("failed to delete ONS topic %q during cleanup: %v", metadata.TopicID, err)
	}

	return nil
}

func (o *OCI) HandleRequest(ctx core.HTTPRequestContext) {}

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
	{Label: "us-ashburn-1", Value: "us-ashburn-1"},
	{Label: "us-phoenix-1", Value: "us-phoenix-1"},
	{Label: "us-chicago-1", Value: "us-chicago-1"},
	{Label: "us-sanjose-1", Value: "us-sanjose-1"},
	{Label: "ca-montreal-1", Value: "ca-montreal-1"},
	{Label: "ca-toronto-1", Value: "ca-toronto-1"},
	{Label: "sa-saopaulo-1", Value: "sa-saopaulo-1"},
	{Label: "sa-vinhedo-1", Value: "sa-vinhedo-1"},
	{Label: "sa-santiago-1", Value: "sa-santiago-1"},
	{Label: "uk-london-1", Value: "uk-london-1"},
	{Label: "uk-cardiff-1", Value: "uk-cardiff-1"},
	{Label: "eu-frankfurt-1", Value: "eu-frankfurt-1"},
	{Label: "eu-amsterdam-1", Value: "eu-amsterdam-1"},
	{Label: "eu-madrid-1", Value: "eu-madrid-1"},
	{Label: "eu-paris-1", Value: "eu-paris-1"},
	{Label: "eu-stockholm-1", Value: "eu-stockholm-1"},
	{Label: "eu-milan-1", Value: "eu-milan-1"},
	{Label: "eu-zurich-1", Value: "eu-zurich-1"},
	{Label: "ap-tokyo-1", Value: "ap-tokyo-1"},
	{Label: "ap-osaka-1", Value: "ap-osaka-1"},
	{Label: "ap-seoul-1", Value: "ap-seoul-1"},
	{Label: "ap-chuncheon-1", Value: "ap-chuncheon-1"},
	{Label: "ap-sydney-1", Value: "ap-sydney-1"},
	{Label: "ap-melbourne-1", Value: "ap-melbourne-1"},
	{Label: "ap-mumbai-1", Value: "ap-mumbai-1"},
	{Label: "ap-hyderabad-1", Value: "ap-hyderabad-1"},
	{Label: "ap-singapore-1", Value: "ap-singapore-1"},
	{Label: "il-jerusalem-1", Value: "il-jerusalem-1"},
	{Label: "me-dubai-1", Value: "me-dubai-1"},
	{Label: "me-abudhabi-1", Value: "me-abudhabi-1"},
	{Label: "me-jeddah-1", Value: "me-jeddah-1"},
	{Label: "af-johannesburg-1", Value: "af-johannesburg-1"},
}

func (o *OCI) Hooks() []core.Hook {
	return []core.Hook{}
}

func (o *OCI) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
