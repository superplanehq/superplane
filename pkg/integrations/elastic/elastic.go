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
   - **Elastic Cloud**: Go to **Elastic Cloud → Deployments**, open your deployment, go to **Manage**, and copy the **Elasticsearch endpoint** from the **Application endpoints, cluster and component IDs** section.
   - **Self-managed Elastic**: Use your Elasticsearch server URL.
   - Example: ` + "`https://my-cluster.es.us-east-1.aws.found.io:9243`" + `.
2. **Kibana URL**:
   - **Elastic Cloud**: From the same **Manage** section for that deployment, copy the **Kibana endpoint**.
   - **Self-managed Elastic**: Use the base URL of your Kibana instance.
   - Keep only the base URL: protocol, host, and port.
   - Example: ` + "`https://my-cluster.kb.us-east-1.aws.found.io:9243`" + `.
3. **API Key**:
   - In Kibana, go to **Stack Management → API Keys**.
   - Create an API key that can index documents in Elasticsearch and manage Kibana connectors.
   - Paste that API key into SuperPlane.
4. **Alert rules**:
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
			Description: "Base URL of your Kibana instance, such as https://my-cluster.kb.us-east-1.aws.found.io:9243. In Elastic Cloud, get it from Deployments -> your deployment -> Manage.",
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
