package elastic

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("elastic", &Elastic{}, &ElasticWebhookHandler{})
}

type Elastic struct{}

type Configuration struct {
	URL       string `json:"url"`
	KibanaURL string `json:"kibanaUrl"`
	APIKey    string `json:"apiKey"`
}

const installationInstructions = `
To connect Elastic to SuperPlane:

1. **Elasticsearch URL**:
   - Paste your Elasticsearch endpoint.
   - Example: ` + "`https://my-cluster.es.us-east-1.aws.found.io:9243`" + `.
2. **Kibana URL**:
   - Open Kibana in your browser and copy the URL from the address bar.
   - Keep only the base URL: protocol, host, and port.
   - Remove anything after that, such as ` + "`/app/...`" + `, anything starting with ` + "`?`" + `, or anything starting with ` + "`#`" + `.
   - Example browser URL: ` + "`https://my-cluster.kb.us-east-1.aws.found.io:9243/app/home`" + `.
   - Paste into SuperPlane: ` + "`https://my-cluster.kb.us-east-1.aws.found.io:9243`" + `.
3. **Where to find these URLs**:
   - **Elastic Cloud**: Go to **Elastic Cloud → Deployments**, open your deployment, and copy the **Elasticsearch endpoint** and **Kibana endpoint** from the Overview page.
   - **Self-managed Elastic**: Use the Elasticsearch server URL for **Elasticsearch URL**, and the normal browser URL you use to open Kibana for **Kibana URL**.
4. **API Key**:
   - In Kibana, go to **Stack Management → API Keys**.
   - Create an API key that can index documents in Elasticsearch and manage Kibana connectors.
   - Paste that API key into SuperPlane.
5. **Alert rules**:
   - SuperPlane automatically creates the Kibana webhook connector.
   - In Kibana, add that connector to the alert rules you want to send to SuperPlane.
`

func (e *Elastic) Name() string {
	return "elastic"
}

func (e *Elastic) Label() string {
	return "Elastic"
}

func (e *Elastic) Icon() string {
	return "elastic"
}

func (e *Elastic) Description() string {
	return "Index documents into Elasticsearch and receive Kibana alert webhooks"
}

func (e *Elastic) Instructions() string {
	return installationInstructions
}

func (e *Elastic) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "Elasticsearch URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Base URL used to send API requests to Elasticsearch, such as https://my-cluster.es.us-east-1.aws.found.io:9243.",
		},
		{
			Name:        "kibanaUrl",
			Label:       "Kibana URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Base URL of your Kibana instance. Copy the browser address up to the host and port only, without /app/... or query parameters.",
		},
		{
			Name:      "apiKey",
			Label:     "API Key",
			Type:      configuration.FieldTypeString,
			Required:  true,
			Sensitive: true,
		},
	}
}

func (e *Elastic) Components() []core.Component {
	return []core.Component{
		&IndexDocument{},
	}
}

func (e *Elastic) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertFires{},
	}
}

func (e *Elastic) Cleanup(_ core.IntegrationCleanupContext) error {
	return nil
}

func (e *Elastic) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.URL == "" {
		return fmt.Errorf("url is required")
	}

	if config.KibanaURL == "" {
		return fmt.Errorf("kibanaUrl is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("invalid Elasticsearch credentials: %v", err)
	}

	if err := client.ValidateKibana(); err != nil {
		return fmt.Errorf("invalid Kibana configuration: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (e *Elastic) HandleRequest(_ core.HTTPRequestContext) {}

const (
	ResourceTypeIndex               = "elastic.index"
	ResourceTypeKibanaRule          = "elastic.kibana.rule"
	ResourceTypeKibanaSpace         = "elastic.kibana.space"
	ResourceTypeKibanaAlertSeverity = "elastic.kibana.alert.severity"
	ResourceTypeKibanaAlertStatus   = "elastic.kibana.alert.status"
)

func (e *Elastic) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeIndex:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}
		indices, err := client.ListIndices()
		if err != nil {
			return nil, fmt.Errorf("error listing indices: %v", err)
		}
		resources := make([]core.IntegrationResource, 0, len(indices))
		for _, idx := range indices {
			resources = append(resources, core.IntegrationResource{ID: idx.Index, Name: idx.Index})
		}
		return resources, nil

	case ResourceTypeKibanaRule:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}
		rules, err := client.ListKibanaRules()
		if err != nil {
			return nil, fmt.Errorf("error listing Kibana rules: %v", err)
		}
		resources := make([]core.IntegrationResource, 0, len(rules))
		for _, r := range rules {
			resources = append(resources, core.IntegrationResource{ID: r.ID, Name: r.Name})
		}
		return resources, nil

	case ResourceTypeKibanaSpace:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}
		spaces, err := client.ListKibanaSpaces()
		if err != nil {
			return nil, fmt.Errorf("error listing Kibana spaces: %v", err)
		}
		resources := make([]core.IntegrationResource, 0, len(spaces))
		for _, s := range spaces {
			resources = append(resources, core.IntegrationResource{ID: s.ID, Name: s.Name})
		}
		return resources, nil

	case ResourceTypeKibanaAlertSeverity:
		return []core.IntegrationResource{
			{Type: ResourceTypeKibanaAlertSeverity, ID: "low", Name: "Low"},
			{Type: ResourceTypeKibanaAlertSeverity, ID: "medium", Name: "Medium"},
			{Type: ResourceTypeKibanaAlertSeverity, ID: "high", Name: "High"},
			{Type: ResourceTypeKibanaAlertSeverity, ID: "critical", Name: "Critical"},
		}, nil

	case ResourceTypeKibanaAlertStatus:
		return []core.IntegrationResource{
			{Type: ResourceTypeKibanaAlertStatus, ID: "active", Name: "Active"},
			{Type: ResourceTypeKibanaAlertStatus, ID: "flapping", Name: "Flapping"},
			{Type: ResourceTypeKibanaAlertStatus, ID: "recovered", Name: "Recovered"},
			{Type: ResourceTypeKibanaAlertStatus, ID: "untracked", Name: "Untracked"},
		}, nil
	}

	return nil, fmt.Errorf("unsupported resourceType %q", resourceType)
}

func (e *Elastic) Actions() []core.Action {
	return []core.Action{}
}

func (e *Elastic) HandleAction(_ core.IntegrationActionContext) error {
	return nil
}
