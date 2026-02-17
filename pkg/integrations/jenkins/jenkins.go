package jenkins

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("jenkins", &Jenkins{}, &JenkinsWebhookHandler{})
}

type Jenkins struct{}

type Configuration struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	APIToken string `json:"apiToken"`
}

func (j *Jenkins) Name() string {
	return "jenkins"
}

func (j *Jenkins) Label() string {
	return "Jenkins"
}

func (j *Jenkins) Icon() string {
	return "jenkins"
}

func (j *Jenkins) Description() string {
	return "Trigger, monitor, and react to Jenkins builds"
}

func (j *Jenkins) Instructions() string {
	return `To set up the Jenkins integration:

1. Click your **user icon** (top right) -> **Security**
2. Under **API Token**, click **Add new Token**, give it a name, and click **Generate**
3. Copy the token and paste it in the **API Token** field below

### Webhook Setup (for triggers)

To receive build events, install the **Jenkins Notification Plugin**:

1. Go to **Manage Jenkins** -> **Manage Plugins** -> **Available** tab
2. Search for **Notification Plugin** and install it
3. In your Jenkins job configuration, add a **Notification Endpoint**:
   - **Format**: JSON
   - **Protocol**: HTTP
   - **URL**: Use the webhook URL shown on the integration details page after connecting`
}

func (j *Jenkins) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jenkins server URL",
			Placeholder: "e.g. https://jenkins.example.com",
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jenkins username",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Jenkins API token",
		},
	}
}

func (j *Jenkins) Components() []core.Component {
	return []core.Component{
		&TriggerBuild{},
	}
}

func (j *Jenkins) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnBuildFinished{},
	}
}

func (j *Jenkins) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.URL == "" {
		return fmt.Errorf("url is required")
	}

	if config.Username == "" {
		return fmt.Errorf("username is required")
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	_, err = client.GetServerInfo()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	// Create a shared integration-level webhook so the URL is available
	// immediately on the integration details page. Components and triggers
	// will reuse this webhook via RequestWebhook + CompareConfig.
	webhookID, err := ctx.Integration.EnsureIntegrationWebhook(WebhookConfiguration{})
	if err != nil {
		return fmt.Errorf("error ensuring webhook: %v", err)
	}

	metadata, _ := ctx.Integration.GetMetadata().(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}

	if webhookID != nil && ctx.WebhooksBaseURL != "" {
		metadata["webhookURL"] = fmt.Sprintf("%s/api/v1/webhooks/%s", ctx.WebhooksBaseURL, webhookID.String())
	}

	ctx.Integration.SetMetadata(metadata)

	ctx.Integration.Ready()
	return nil
}

func (j *Jenkins) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jenkins) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (j *Jenkins) Actions() []core.Action {
	return []core.Action{}
}

func (j *Jenkins) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (j *Jenkins) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "job" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	jobs, err := client.ListJobs()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(jobs))
	for _, job := range jobs {
		if job.FullName == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: job.FullName,
			ID:   job.FullName,
		})
	}

	return resources, nil
}
