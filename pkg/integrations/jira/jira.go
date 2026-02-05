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

// WebhookConfiguration represents the configuration for a Jira webhook.
type WebhookConfiguration struct {
	EventType string `json:"eventType"`
	Project   string `json:"project"`
}

// WebhookMetadata stores the webhook ID for cleanup.
type WebhookMetadata struct {
	ID int64 `json:"id"`
}

func (j *Jira) Name() string {
	return "jira"
}

func (j *Jira) Label() string {
	return "Jira"
}

func (j *Jira) Icon() string {
	return "jira"
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
	return []core.Component{
		&CreateIssue{},
	}
}

func (j *Jira) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueCreated{},
	}
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
	var configA, configB WebhookConfiguration

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, fmt.Errorf("failed to decode config a: %w", err)
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, fmt.Errorf("failed to decode config b: %w", err)
	}

	return configA.EventType == configB.EventType && configA.Project == configB.Project, nil
}

func (j *Jira) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	var config WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	jqlFilter := fmt.Sprintf("project = %q", config.Project)
	events := []string{config.EventType}

	response, err := client.RegisterWebhook(ctx.Webhook.GetURL(), jqlFilter, events)
	if err != nil {
		return nil, fmt.Errorf("error registering webhook: %v", err)
	}

	if len(response.WebhookRegistrationResult) == 0 {
		return nil, fmt.Errorf("no webhook registration result returned")
	}

	result := response.WebhookRegistrationResult[0]
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("webhook registration failed: %v", result.Errors)
	}

	return WebhookMetadata{ID: result.CreatedWebhookID}, nil
}

func (j *Jira) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	var metadata WebhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.ID == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	return client.DeleteWebhook([]int64{metadata.ID})
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{}
}

func (j *Jira) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
