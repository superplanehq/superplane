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
	TopicID      string `json:"topicId" mapstructure:"topicId"`
	EventsRuleID string `json:"eventsRuleId" mapstructure:"eventsRuleId"`
	// Deprecated: CompartmentRules was used in older versions to track per-compartment rules.
	// It is kept only for cleanup of legacy resources.
	CompartmentRules map[string]string `json:"compartmentRules,omitempty" mapstructure:"compartmentRules"`
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
2. Go to **Menu** → **Identity & Security → Domains → Default → User Management → Groups**.
3. Click **Create Group**.
4. Set the name to ` + "`SuperPlaneIntegration`" + ` and add a description, then click **Create**.
5. In the same **User Management** tab, go to **Users → Create User**.
6. Fill in the details:
   - **Lastname:** ` + "`superplane-integration`" + `
   - **Email:** use any valid email (not used for authentication)
7. In the **Groups** section, assign the user to the ` + "`SuperPlaneIntegration`" + ` group
8. Click **Create**.

### Part 2 — Create an IAM Policy

1. Go to **Identity & Security → Policies**.
2. Make sure you are in the **root compartment** (check the Compartment selector on the left).
3. Click **Create Policy**, name it ` + "`SuperPlanePolicies`" + `, add a description and enable the **manual editor**.
4. Paste in the following statements, replacing ` + "`<your-compartment>`" + ` with your target compartment name, and then Click **Create**.:
` + "```" + `
Allow group SuperPlaneIntegration to manage instances in tenancy
Allow group SuperPlaneIntegration to manage volumes in tenancy
Allow group SuperPlaneIntegration to manage volume-attachments in tenancy
Allow group SuperPlaneIntegration to manage virtual-network-family in tenancy
Allow group SuperPlaneIntegration to manage buckets in tenancy
Allow group SuperPlaneIntegration to manage objects in tenancy
Allow group SuperPlaneIntegration to manage objectstorage-namespaces in tenancy   
Allow group SuperPlaneIntegration to manage fn-app in tenancy
Allow group SuperPlaneIntegration to manage fn-function in tenancy
Allow group SuperPlaneIntegration to manage fn-invocation in tenancy
Allow group SuperPlaneIntegration to manage ons-topics in tenancy
Allow group SuperPlaneIntegration to manage ons-subscriptions in tenancy
Allow group SuperPlaneIntegration to inspect compartments in tenancy
Allow group SuperPlaneIntegration to inspect all-resources in tenancy
Allow group SuperPlaneIntegration to manage cloudevents-rules in tenancy
Allow group SuperPlaneIntegration to manage autonomous-database-family in tenancy
Allow service cloudEvents to use ons-topics in tenancy
` + "```" + `
 
### Part 3 — Generate API Keys for the Service User and Connect to SuperPlane

1. Go to **Menu** → **Identity & Security → Domains → Default → User Management → Users**.
2. Choose the service user you created, then go to **API Keys → Add API Key**.
3. Select **Generate API key pair**, download the private key file and then click **Add**.
4. Copy the **Configuration File Preview** values that appear to the UI:
    - **User OCID** (begins with ` + "`ocid1.user.`" + `)
    - **Fingerprint** (e.g. ` + "`12:34:56:…`" + `)
    - **Tenancy OCID** (begins with ` + "`ocid1.tenancy.`" + `)
5. Select the **Region** that matches your OCI tenancy's home region.
6. Open the downloaded private key file and paste its full contents into the **Private Key** field.
7. Click **Connect** to validate the credentials and save the integration.`
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

	// Create a single shared Events rule in the tenancy compartment, co-located with the topic.
	// The rule captures all compute launch events tenancy-wide; per-compartment filtering is
	// done server-side in the webhook handler. Creating the rule here (in the tenancy compartment)
	// avoids cross-compartment IAM issues that arise when the rule and topic are in different compartments.
	if metadata.EventsRuleID == "" {
		ruleName := fmt.Sprintf("superplane-%s", ctx.Integration.ID())
		condition := `{"eventType": ["com.oraclecloud.computeapi.launchinstance.end"]}`
		rule, err := client.CreateEventsRule(cfg.TenancyOCID, ruleName, condition, metadata.TopicID)
		if err != nil {
			return fmt.Errorf("failed to create Events rule: %w", err)
		}
		metadata.EventsRuleID = rule.ID
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

	if metadata.TopicID == "" && metadata.EventsRuleID == "" && len(metadata.CompartmentRules) == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client during cleanup: %w", err)
	}

	// Delete the single shared Events rule (current style).
	if metadata.EventsRuleID != "" {
		if err := client.DeleteEventsRule(metadata.EventsRuleID); err != nil {
			ctx.Logger.Warnf("failed to delete Events rule %q during cleanup: %v", metadata.EventsRuleID, err)
		}
	}

	// Delete any legacy per-compartment rules created by older versions.
	for compartmentID, ruleID := range metadata.CompartmentRules {
		if err := client.DeleteEventsRule(ruleID); err != nil {
			ctx.Logger.Warnf("failed to delete legacy Events rule %q (compartment %q) during cleanup: %v", ruleID, compartmentID, err)
		}
	}

	if metadata.TopicID != "" {
		if err := client.DeleteONSTopic(metadata.TopicID); err != nil {
			ctx.Logger.Warnf("failed to delete ONS topic %q during cleanup: %v", metadata.TopicID, err)
		}
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
