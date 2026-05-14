package cloudflare

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("cloudflare", &Cloudflare{}, &CloudflareWebhookHandler{})
}

type Cloudflare struct{}

type Configuration struct {
	APIToken  string `json:"apiToken"`
	AccountID string `json:"accountId"`
}

type Metadata struct {
	Zones     []Zone `json:"zones"`
	AccountID string `json:"accountId"`
}

type KVNodeMetadata struct {
	NamespaceName string `json:"namespaceName"`
	KeyName       string `json:"keyName"`
}

func resolveKVNamespaceMetadata(ctx core.SetupContext, accountID, namespaceID string) (string, error) {
	if strings.Contains(namespaceID, "{{") || strings.Contains(accountID, "{{") {
		return namespaceID, nil
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	ns, err := client.GetKVNamespace(accountID, namespaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get KV namespace: %w", err)
	}
	return ns.Title, nil
}

func accountIDFromIntegration(ctx core.IntegrationContext) string {
	if ctx == nil {
		return ""
	}
	metadata := Metadata{}
	mapstructure.Decode(ctx.GetMetadata(), &metadata)
	if id := strings.TrimSpace(metadata.AccountID); id != "" {
		return id
	}
	cfg, err := ctx.GetConfig("accountId")
	if err != nil || len(cfg) == 0 {
		return ""
	}
	return strings.TrimSpace(string(cfg))
}

func resolveAccountID(specAccountID string, integration core.IntegrationContext) string {
	if specAccountID != "" {
		return specAccountID
	}
	return accountIDFromIntegration(integration)
}

type PoolNodeMetadata struct {
	PoolName string `json:"poolName"`
}

func resolvePoolMetadata(ctx core.SetupContext, accountID, poolID string) error {
	meta := PoolNodeMetadata{}
	if strings.Contains(poolID, "{{") || strings.Contains(accountID, "{{") {
		meta.PoolName = poolID
	} else {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		pool, err := client.GetPool(accountID, poolID)
		if err != nil {
			return fmt.Errorf("failed to get pool: %w", err)
		}
		meta.PoolName = pool.Name
	}
	return ctx.Metadata.Set(meta)
}

// splitLBID splits a composite load balancer ID of the form "zoneID/lbID"
// into its component parts.
func splitLBID(compositeID string) (zoneID, lbID string, err error) {
	parts := strings.SplitN(compositeID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid load balancer ID %q: expected format zoneId/lbId", compositeID)
	}
	return parts[0], parts[1], nil
}

func (c *Cloudflare) Name() string {
	return "cloudflare"
}

func (c *Cloudflare) Label() string {
	return "Cloudflare"
}

func (c *Cloudflare) Icon() string {
	return "cloud"
}

func (c *Cloudflare) Description() string {
	return "Manage Cloudflare zones, rules, and DNS"
}

func (c *Cloudflare) Instructions() string {
	return `
To connect Cloudflare to SuperPlane:

1. In the [Cloudflare dashboard](https://dash.cloudflare.com/), open the account you want to connect, then go to **Manage Account → Account API Tokens** and click **Create Token → Create Custom Token**. Creating an account-owned token requires **Super Administrator** access on the Cloudflare account.
2. Name the token and keep the first policy scoped to **Entire Account**. Select these permissions:
   - **Developer Platform** → **Workers KV Storage** → **Edit**
   - **Network Services** → **Account Load Balancers** → **Edit**
   - **Network Services** → **Load Balancing: Monitors and Pools** → **Edit**
   - **Account & Billing** → **Notifications** → **Edit**
3. Click **Add policy**. In the new policy, change the scope dropdown from **Entire Account** to **All Domains** or **Specified Domains** for only the domains SuperPlane should manage. The DNS and zone rows below are only available after switching this policy to a domain scope:
   - **DNS & Zones** → **Zone** → **Read**
   - **DNS & Zones** → **DNS** → **Edit**
   - **Rules & Configuration** → **Dynamic URL Redirects** → **Edit**
   - **Rules & Configuration** → **Origin** → **Edit**
   - **Network Services** → **Zone Load Balancers** → **Edit**
4. Optionally set an expiration date, review, create the token, and paste the generated token below. Cloudflare only shows the token once.
5. Copy the **Account ID** from the same account's home page right sidebar and paste it into **Account ID** below.
`
}

func (c *Cloudflare) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "Account-Owned API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Cloudflare account-owned API token with the permissions listed above",
		},
		{
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Cloudflare account ID. Required for KV storage, load balancing monitors/pools, and alerting webhooks.",
			Placeholder: "e.g. 01a7362d577a6c3019a474fd6f485823",
		},
	}
}

func (c *Cloudflare) Actions() []core.Action {
	return []core.Action{
		&CreateDNSRecord{},
		&CreateOriginRule{},
		&UpdateRedirectRule{},
		&UpdateOriginRule{},
		&UpdateDNSRecord{},
		&DeleteDNSRecord{},
		&CreateMonitor{},
		&DeleteMonitor{},
		&DeleteOriginRule{},
		&CreateKVNamespace{},
		&PutKVValue{},
		&GetKVValue{},
		&DeleteKVValue{},
		&DeleteKVNamespace{},
		&CreatePool{},
		&UpdatePool{},
		&GetPool{},
		&DeletePool{},
		&CreateLoadBalancer{},
		&GetLoadBalancer{},
		&UpdateLoadBalancer{},
		&DeleteLoadBalancer{},
	}
}

func (c *Cloudflare) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnLoadBalancingHealthAlert{},
	}
}

func (c *Cloudflare) Sync(ctx core.SyncContext) error {
	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if configuration.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zones, err := client.ListZones()
	if err != nil {
		return fmt.Errorf("error listing zones: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Zones: zones, AccountID: configuration.AccountID})
	ctx.Integration.Ready()
	return nil
}

func (c *Cloudflare) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *Cloudflare) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "zone":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		zones, err := client.ListZones()
		if err != nil {
			return nil, fmt.Errorf("error listing zones: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(zones))
		for _, zone := range zones {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: zone.Name,
				ID:   zone.ID,
			})
		}
		return resources, nil

	case "redirect_rule":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		var resources []core.IntegrationResource
		for _, zone := range metadata.Zones {
			rules, err := client.ListRedirectRules(zone.ID)
			if err != nil {
				continue
			}

			for _, rule := range rules {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s - %s", zone.Name, rule.Description),
					ID:   fmt.Sprintf("%s/%s", zone.ID, rule.ID),
				})
			}
		}
		return resources, nil

	case "dns_record":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		var resources []core.IntegrationResource
		for _, zone := range metadata.Zones {
			records, err := client.ListDNSRecords(zone.ID)
			if err != nil {
				continue
			}

			for _, record := range records {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s (%s)", record.Name, record.Type),
					ID:   fmt.Sprintf("%s/%s", zone.ID, record.ID),
				})
			}
		}
		return resources, nil

	case "origin_rule":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		metadata := Metadata{}
		if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
			return nil, fmt.Errorf("failed to decode application metadata: %w", err)
		}

		var resources []core.IntegrationResource
		for _, zone := range metadata.Zones {
			rules, err := client.ListOriginRules(zone.ID)
			if err != nil {
				continue
			}

			for _, rule := range rules {
				name := rule.Description
				if name == "" {
					name = rule.Expression
				}

				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s - %s", zone.Name, name),
					ID:   fmt.Sprintf("%s/%s", zone.ID, rule.ID),
				})
			}
		}
		return resources, nil

	case "namespace":
		accountID := ctx.Parameters["accountId"]
		if accountID == "" {
			accountID = accountIDFromIntegration(ctx.Integration)
		}
		if accountID == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		namespaces, err := client.ListKVNamespaces(accountID)
		if err != nil {
			return nil, fmt.Errorf("error listing KV namespaces: %w", err)
		}

		var nsResources []core.IntegrationResource
		for _, ns := range namespaces {
			nsResources = append(nsResources, core.IntegrationResource{
				Type: resourceType,
				Name: ns.Title,
				ID:   ns.ID,
			})
		}
		return nsResources, nil

	case "kv_key":
		accountID := ctx.Parameters["accountId"]
		if accountID == "" {
			accountID = accountIDFromIntegration(ctx.Integration)
		}
		namespaceID := ctx.Parameters["namespace"]
		if accountID == "" || namespaceID == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		keys, err := client.ListKVKeys(accountID, namespaceID)
		if err != nil {
			return nil, fmt.Errorf("error listing KV keys: %w", err)
		}

		var keyResources []core.IntegrationResource
		for _, key := range keys {
			keyResources = append(keyResources, core.IntegrationResource{
				Type: resourceType,
				Name: key.Name,
				ID:   key.Name,
			})
		}
		return keyResources, nil

	case "monitor":
		accountID := ctx.Parameters["accountId"]
		if accountID == "" {
			accountID = accountIDFromIntegration(ctx.Integration)
		}
		if accountID == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		monitors, err := client.ListMonitors(accountID)
		if err != nil {
			return nil, fmt.Errorf("error listing monitors: %w", err)
		}

		var resources []core.IntegrationResource
		for _, m := range monitors {
			name := m.Description
			if name == "" {
				name = m.ID
			}
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: name,
				ID:   m.ID,
			})
		}
		return resources, nil

	case "pool":
		accountID := ctx.Parameters["accountId"]
		if accountID == "" {
			accountID = accountIDFromIntegration(ctx.Integration)
		}
		if accountID == "" {
			return []core.IntegrationResource{}, nil
		}

		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		pools, err := client.ListPools(accountID)
		if err != nil {
			return nil, fmt.Errorf("error listing pools: %w", err)
		}

		var resources []core.IntegrationResource
		for _, p := range pools {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: p.Name,
				ID:   p.ID,
			})
		}
		return resources, nil

	case "load_balancer":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		zones, err := client.ListZones()
		if err != nil {
			return nil, fmt.Errorf("error listing zones: %w", err)
		}

		var resources []core.IntegrationResource
		for _, zone := range zones {
			lbs, err := client.ListLoadBalancers(zone.ID)
			if err != nil {
				ctx.Logger.WithError(err).WithField("zone_id", zone.ID).WithField("zone_name", zone.Name).Warn("failed to list load balancers for zone, skipping")
				continue
			}
			for _, lb := range lbs {
				resources = append(resources, core.IntegrationResource{
					Type: resourceType,
					Name: fmt.Sprintf("%s (%s)", lb.Name, zone.Name),
					ID:   fmt.Sprintf("%s/%s", zone.ID, lb.ID),
				})
			}
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (c *Cloudflare) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (c *Cloudflare) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Cloudflare) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
