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

1. **Elasticsearch URL**:
   - **Elastic Cloud**: Start at https://cloud.elastic.co/home. In **Hosted deployments**, find your deployment, click **Manage** (or open it from **Open**), then copy the **Elasticsearch endpoint** from the **Application endpoints, cluster and component IDs** section.
   - **Self-managed Elastic**: Use your Elasticsearch server URL.
   - Example: ` + "`https://my-cluster.es.us-east-1.aws.found.io:9243`" + `.
2. **Kibana URL**:
   - **Elastic Cloud**: From the same deployment page opened from https://cloud.elastic.co/home, copy the **Kibana endpoint** from **Manage**.
   - **Self-managed Elastic**: Use the base URL of your Kibana instance.
   - Keep only the base URL: protocol, host, and port.
   - Example: ` + "`https://my-cluster.kb.us-east-1.aws.found.io:9243`" + `.
3. **Auth Method**:
   - **API Key** (recommended for Elastic Cloud): Go to Kibana → Stack Management → API Keys and create a new key. Paste the base64-encoded ` + "`id:api_key`" + ` value.
   - Create an API key that can index documents in Elasticsearch, access Kibana cases, and manage Kibana connectors.
   - **Username / Password**: Provide the credentials for a user with the required privileges.
4. **Kibana webhook connector**: SuperPlane creates or reuses one shared Kibana Webhook connector per integration and uses it across the Elastic triggers.
5. **Alert trigger**: For ` + "`When Alert Fires`" + `, SuperPlane attaches the shared connector to the selected Kibana alert rule automatically.
6. **Document and case triggers**: For ` + "`On Document Indexed`" + ` and ` + "`When Case Status Changes`" + `, SuperPlane automatically provisions the Kibana Elasticsearch query rules needed for those triggers.
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
	return "Index documents into Elasticsearch and receive Kibana webhooks"
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
			Name:        "authType",
			Label:       "Auth Method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "apiKey",
			Description: "Choose whether SuperPlane should authenticate with an API key or a username/password.",
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
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "API key used to authenticate Elastic API calls.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"apiKey"}},
			},
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Username for basic authentication.",
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
		&CreateCase{},
		&GetCase{},
		&UpdateCase{},
		&GetDocument{},
		&UpdateDocument{},
	}
}

func (e *Elastic) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertFires{},
		&OnCaseStatusChange{},
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
		config.AuthType = "apiKey"
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
	ResourceTypeIndex               = "elastic.index"
	ResourceTypeDocument            = "elastic.document"
	ResourceTypeKibanaRule          = "elastic.kibana.rule"
	ResourceTypeKibanaSpace         = "elastic.kibana.space"
	ResourceTypeKibanaAlertSeverity = "elastic.kibana.alert.severity"
	ResourceTypeKibanaAlertStatus   = "elastic.kibana.alert.status"
	ResourceTypeCase                = "elastic.case"
	ResourceTypeCaseStatus          = "elastic.case.status"
	ResourceTypeCaseSeverity        = "elastic.case.severity"
	ResourceTypeCaseVersion         = "elastic.case.version"
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

	case ResourceTypeDocument:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}

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

	case ResourceTypeCase:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}
		cases, err := client.ListCases()
		if err != nil {
			return nil, fmt.Errorf("error listing cases: %v", err)
		}
		resources := make([]core.IntegrationResource, 0, len(cases))
		for _, c := range cases {
			resources = append(resources, core.IntegrationResource{ID: c.ID, Name: c.Title})
		}
		return resources, nil

	case ResourceTypeCaseStatus:
		return []core.IntegrationResource{
			{ID: "open", Name: "Open"},
			{ID: "in-progress", Name: "In Progress"},
			{ID: "closed", Name: "Closed"},
		}, nil

	case ResourceTypeCaseSeverity:
		return []core.IntegrationResource{
			{ID: "critical", Name: "Critical"},
			{ID: "high", Name: "High"},
			{ID: "medium", Name: "Medium"},
			{ID: "low", Name: "Low"},
		}, nil

	case ResourceTypeCaseVersion:
		caseID := ctx.Parameters["caseId"]
		if caseID == "" {
			return nil, nil
		}
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %v", err)
		}
		c, err := client.GetCase(caseID)
		if err != nil {
			return nil, fmt.Errorf("error fetching case: %v", err)
		}
		return []core.IntegrationResource{{Type: ResourceTypeCaseVersion, ID: c.Version, Name: c.Version}}, nil

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
