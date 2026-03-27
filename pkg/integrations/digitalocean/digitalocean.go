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
	return `## DigitalOcean Personal Access Token

Generate a [DigitalOcean Personal Access Token](https://cloud.digitalocean.com/account/api/tokens) and copy it.

- Token name: ` + "`SuperPlane Integration`" + `
- Expiration: **No expiry** (or choose an appropriate expiration)
- Scopes: **Full Access** (or customize as needed)

## Access Key (optional)

Only required for **Spaces Object Storage** components.

Create an [Access Key ID & Secret Access Key](https://cloud.digitalocean.com/spaces/access_keys) and copy the generated pair.

- Scope: **Full Access** (all buckets) or **Limited Access** (specific buckets)

> **Note:** The Personal Access Token and Secret Access Key are shown only once — store them somewhere safe before continuing.`
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
		&PutObject{},
		&CopyObject{},
		&DeleteObject{},
		&CreateApp{},
		&GetApp{},
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
