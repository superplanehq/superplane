package jira

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("jira", &Jira{})
}

type Jira struct{}

type Configuration struct {
	BaseURL  string `json:"baseUrl"`
	Email    string `json:"email"`
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	Projects []Project `json:"projects"`
}

func (j *Jira) Name() string {
	return "jira"
}

func (j *Jira) Label() string {
	return "Jira"
}

func (j *Jira) Icon() string {
	return "clipboard-list"
}

func (j *Jira) Description() string {
	return "Manage and react to issues in Jira"
}

func (j *Jira) Instructions() string {
	return ""
}

func (j *Jira) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jira Cloud instance URL (e.g. https://your-domain.atlassian.net)",
		},
		{
			Name:        "email",
			Label:       "Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Email address for API authentication",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Jira API token",
		},
	}
}

func (j *Jira) Components() []core.Component {
	return []core.Component{}
}

func (j *Jira) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (j *Jira) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jira) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.BaseURL == "" {
		return fmt.Errorf("baseUrl is required")
	}

	if config.Email == "" {
		return fmt.Errorf("email is required")
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("error listing projects: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Projects: projects})
	ctx.Integration.Ready()
	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (j *Jira) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (j *Jira) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (j *Jira) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{}
}

func (j *Jira) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
