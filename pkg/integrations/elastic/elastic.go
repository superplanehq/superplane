package elastic

import (
	"fmt"
	"strings"

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
	AuthType  string `json:"authType"`
	APIKey    string `json:"apiKey"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

const installationInstructions = `
To connect Elastic to SuperPlane:

1. **Elasticsearch URL**: Full URL of your Elasticsearch cluster (e.g. ` + "`https://my-cluster.es.us-east-1.aws.found.io:9243`" + `).
2. **Kibana URL**: Full URL of your Kibana instance (e.g. ` + "`https://my-cluster.kb.us-east-1.aws.found.io:9243`" + `). Required for automatic Kibana webhook connector setup.
3. **Auth Method**:
   - **API Key** (recommended for Elastic Cloud): Go to Kibana → Stack Management → API Keys and create a new key. Paste the base64-encoded ` + "`id:api_key`" + ` value.
   - **Username / Password**: Provide the credentials for a user with the required privileges.
4. **Kibana alerts (trigger)**: SuperPlane automatically creates a signed Kibana Webhook connector. You still need to attach that connector to your alert rules in Kibana.
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
			Description: "Full URL of your Elasticsearch cluster (e.g. https://my-cluster.es.us-east-1.aws.found.io:9243).",
		},
		{
			Name:        "kibanaUrl",
			Label:       "Kibana URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full URL of your Kibana instance (e.g. https://my-cluster.kb.us-east-1.aws.found.io:9243).",
		},
		{
			Name:        "authType",
			Label:       "Auth Method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "apiKey",
			Description: "Choose whether SuperPlane should authenticate to Elasticsearch and Kibana with an API key or a username/password.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "API Key", Value: "apiKey"},
						{Label: "Username / Password", Value: "basic"},
					},
				},
			},
		},
		{
			Name:      "apiKey",
			Label:     "API Key",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Sensitive: true,
			Description: "Base64-encoded Elasticsearch API key (id:api_key format). " +
				"Create one in Kibana → Stack Management → API Keys.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"apiKey"}},
			},
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Username for basic authentication. Use an account with permission to index documents and manage Kibana connectors.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"basic"}},
			},
		},
		{
			Name:        "password",
			Label:       "Password",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Password for basic authentication.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"basic"}},
			},
		},
	}
}

func (e *Elastic) Components() []core.Component {
	return []core.Component{
		&IndexDocument{},
		&GetDocument{},
		&UpdateDocument{},
	}
}

func (e *Elastic) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertFires{},
		&OnDocumentIndexed{},
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

	if config.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	switch config.AuthType {
	case "apiKey":
		if config.APIKey == "" {
			return fmt.Errorf("apiKey is required when authType is apiKey")
		}
	case "basic":
		if config.Username == "" {
			return fmt.Errorf("username is required when authType is basic")
		}
		if config.Password == "" {
			return fmt.Errorf("password is required when authType is basic")
		}
	default:
		return fmt.Errorf("unknown authType %q: must be apiKey or basic", config.AuthType)
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
	ResourceTypeIndex       = "elastic.index"
	ResourceTypeDocument    = "elastic.document"
	ResourceTypeKibanaRule  = "elastic.kibana.rule"
	ResourceTypeKibanaSpace = "elastic.kibana.space"
)

func (e *Elastic) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	switch resourceType {
	case ResourceTypeIndex:
		indices, err := client.ListIndices()
		if err != nil {
			return nil, fmt.Errorf("error listing indices: %v", err)
		}
		resources := make([]core.IntegrationResource, 0, len(indices))
		for _, idx := range indices {
			resources = append(resources, core.IntegrationResource{ID: idx.Index, Name: idx.Index})
		}
		return resources, nil

	case ResourceTypeDocument:
		index := ctx.Parameters["index"]
		if index == "" || strings.Contains(index, "{{") {
			return []core.IntegrationResource{}, nil
		}

		documents, err := client.ListDocuments(index)
		if err != nil {
			return nil, fmt.Errorf("error listing documents: %v", err)
		}

		resources := make([]core.IntegrationResource, 0, len(documents))
		for _, doc := range documents {
			resources = append(resources, core.IntegrationResource{
				ID:   doc.ID,
				Name: doc.ID,
				Type: ResourceTypeDocument,
			})
		}
		return resources, nil

	case ResourceTypeKibanaRule:
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
		spaces, err := client.ListKibanaSpaces()
		if err != nil {
			return nil, fmt.Errorf("error listing Kibana spaces: %v", err)
		}
		resources := make([]core.IntegrationResource, 0, len(spaces))
		for _, s := range spaces {
			resources = append(resources, core.IntegrationResource{ID: s.ID, Name: s.Name})
		}
		return resources, nil
	}

	return nil, fmt.Errorf("unsupported resourceType %q", resourceType)
}

func (e *Elastic) Actions() []core.Action {
	return []core.Action{}
}

func (e *Elastic) HandleAction(_ core.IntegrationActionContext) error {
	return nil
}
