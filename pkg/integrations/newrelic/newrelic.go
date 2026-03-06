package newrelic

import (
	"context"
	"fmt"
	"regexp"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

var accountIDRegexp = regexp.MustCompile(`^\d+$`)

const (
	NewRelicIssuePayloadType = "newrelic.issue"
)

const installationInstructions = `### Getting your credentials

1. **Account ID**: Click your name in the bottom-left corner of New Relic. Your Account ID is displayed under the account name.

2. **User API Key**: Go to the **API Keys** page. Click **Create a key**. Select key type **User**. Give it a name (e.g. "SuperPlane") and click **Create a key**. This key is used for NerdGraph/NRQL queries — no additional permissions are needed.

3. **License Key**: On the same **API Keys** page, find the key with type **Ingest - License** and copy it. This key is used for sending metrics. If no license key exists, click **Create a key** and select **Ingest - License**.

4. **Region**: Choose **US** if your New Relic URL is ` + "`one.newrelic.com`" + `, or **EU** if it is ` + "`one.eu.newrelic.com`" + `.

### Webhook Setup

SuperPlane automatically creates a Webhook Notification Channel in your New Relic account when you add the **On Issue** trigger to your canvas. Just attach it to your alert workflow in New Relic to start receiving alerts.
`

func init() {
	registry.RegisterIntegrationWithWebhookHandler("newrelic", &NewRelic{}, &NewRelicWebhookHandler{})
}

type NewRelic struct{}

type Configuration struct {
	AccountID  string `json:"accountId" mapstructure:"accountId"`
	Region     string `json:"region" mapstructure:"region"`
	UserAPIKey string `json:"userApiKey" mapstructure:"userApiKey"`
	LicenseKey string `json:"licenseKey" mapstructure:"licenseKey"`
}

func (n *NewRelic) Name() string {
	return "newrelic"
}

func (n *NewRelic) Label() string {
	return "New Relic"
}

func (n *NewRelic) Icon() string {
	return "chart-bar"
}

func (n *NewRelic) Description() string {
	return "React to alerts and query telemetry data from New Relic"
}

func (n *NewRelic) Instructions() string {
	return installationInstructions
}

func (n *NewRelic) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your New Relic Account ID",
		},
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "US",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US", Value: "US"},
						{Label: "EU", Value: "EU"},
					},
				},
			},
			Description: "New Relic data center region",
		},
		{
			Name:        "userApiKey",
			Label:       "User API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "New Relic User API Key for NerdGraph and NRQL queries",
		},
		{
			Name:        "licenseKey",
			Label:       "License Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "New Relic License Key for metric ingestion",
		},
	}
}

func (n *NewRelic) Components() []core.Component {
	return []core.Component{
		&ReportMetric{},
		&RunNRQLQuery{},
	}
}

func (n *NewRelic) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (n *NewRelic) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (n *NewRelic) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.AccountID == "" {
		return fmt.Errorf("accountId is required")
	}

	if !accountIDRegexp.MatchString(config.AccountID) {
		return fmt.Errorf("accountId must be numeric")
	}

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Region != "US" && config.Region != "EU" {
		return fmt.Errorf("region must be US or EU, got %q", config.Region)
	}

	if config.UserAPIKey == "" {
		return fmt.Errorf("userApiKey is required")
	}

	if config.LicenseKey == "" {
		return fmt.Errorf("licenseKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateCredentials(context.Background())
	if err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (n *NewRelic) HandleRequest(ctx core.HTTPRequestContext) {
}

func (n *NewRelic) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (n *NewRelic) Actions() []core.Action {
	return []core.Action{}
}

func (n *NewRelic) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
