package digitalocean

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("digitalocean", &DigitalOcean{})
}

type DigitalOcean struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	AccountEmail string `json:"accountEmail"`
	AccountUUID  string `json:"accountUUID"`
}

func (d *DigitalOcean) Name() string {
	return "digitalocean"
}

func (d *DigitalOcean) Label() string {
	return "DigitalOcean"
}

func (d *DigitalOcean) Icon() string {
	return "digitalocean"
}

func (d *DigitalOcean) Description() string {
	return "Manage and monitor your DigitalOcean infrastructure"
}

func (d *DigitalOcean) Instructions() string {
	return `## Create a DigitalOcean Personal Access Token

1. Open the [DigitalOcean API Tokens page](https://cloud.digitalocean.com/account/api/tokens)
2. Click **Generate New Token**
3. Configure the token:
   - **Token name**: SuperPlane Integration
   - **Expiration**: No expiry (or choose an appropriate expiration)
   - **Scopes**: Select **Full Access** (or customize as needed)
4. Click **Generate Token**
5. Copy the token and paste it below

> **Note**: The token is only shown once. Store it securely if needed elsewhere.

## Spaces Access Key ID and Secret Access Key (optional)

Spaces Access Key and Secret Key are only required if you plan to use **Spaces Object Storage** components (e.g. Get Object). Other components such as Droplets, DNS, Load Balancers, and Snapshots work with the API Token alone.

To generate Spaces access keys:

1. Open the [Spaces Access Keys page](https://cloud.digitalocean.com/spaces/access_keys)
2. Click **Create Access Key**
3. Select the access scope:
   - **Full Access** — works across all buckets
   - **Limited Access** — scoped to specific buckets with Read or Read/Write/Delete permissions
4. Copy both the **Access Key ID** and the **Secret Access Key** immediately — the secret is only shown once`
}

func (d *DigitalOcean) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "DigitalOcean Personal Access Token",
		},
		{
			Name:        "spacesAccessKey",
			Label:       "Spaces Access Key ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Required for Spaces Object Storage components",
		},
		{
			Name:        "spacesSecretKey",
			Label:       "Spaces Secret Access Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Required for Spaces Object Storage components",
		},
	}
}

func (d *DigitalOcean) Components() []core.Component {
	return []core.Component{
		&CreateDroplet{},
		&GetDroplet{},
		&DeleteDroplet{},
		&ManageDropletPower{},
		&CreateSnapshot{},
		&DeleteSnapshot{},
		&CreateDNSRecord{},
		&DeleteDNSRecord{},
		&UpsertDNSRecord{},
		&CreateLoadBalancer{},
		&DeleteLoadBalancer{},
		&AssignReservedIP{},
		&CreateAlertPolicy{},
		&GetAlertPolicy{},
		&UpdateAlertPolicy{},
		&DeleteAlertPolicy{},
		&GetDropletMetrics{},
		&GetObject{},
		&CreateApp{},
		&DeleteApp{},
		&UpdateApp{},
	}
}

func (d *DigitalOcean) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (d *DigitalOcean) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	account, err := client.GetAccount()
	if err != nil {
		return fmt.Errorf("error fetching account: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		AccountEmail: account.Email,
		AccountUUID:  account.UUID,
	})
	ctx.Integration.Ready()
	return nil
}

func (d *DigitalOcean) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *DigitalOcean) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (d *DigitalOcean) Actions() []core.Action {
	return []core.Action{}
}

func (d *DigitalOcean) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
