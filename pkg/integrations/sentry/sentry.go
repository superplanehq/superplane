package sentry

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("sentry", &Sentry{}, &SentryWebhookHandler{})
}

type Sentry struct{}

const (
	AuthTypeAPIToken = "apiToken"
)

type Configuration struct {
	AuthToken string `json:"authToken"`
	OrgSlug   string `json:"orgSlug"`
	BaseURL   string `json:"baseUrl,omitempty"`
}

type Metadata struct{}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "alert-circle"
}

func (s *Sentry) Description() string {
	return "Manage and react to issues in Sentry"
}

func (s *Sentry) Instructions() string {
	return ""
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Sentry authentication token with org:admin scope",
		},
		{
			Name:        "orgSlug",
			Label:       "Organization Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your Sentry organization slug (e.g., my-org)",
		},
		{
			Name:        "baseUrl",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Custom Sentry base URL for self-hosted Sentry (e.g., https://sentry.example.com). Leave blank for sentry.io",
			Placeholder: "https://sentry.example.com",
		},
	}
}

func (s *Sentry) Components() []core.Component {
	return []core.Component{
		&UpdateIssue{},
	}
}

func (s *Sentry) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	// Delete the auto-created Sentry App if it exists
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	// Check if we have Sentry App metadata in secrets
	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		return nil
	}

	var appSlug, clientSecret string
	for _, secret := range secrets {
		if secret.Name == "sentryAppSlug" {
			appSlug = string(secret.Value)
		} else if secret.Name == "sentryClientSecret" {
			clientSecret = string(secret.Value)
		}
	}

	// Only delete if we have both the app slug and client secret
	if appSlug != "" && clientSecret != "" {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("error creating client: %v", err)
		}

		err = client.DeleteSentryApp(appSlug)
		if err != nil {
			// Log error but don't fail cleanup - app may have been deleted manually
			fmt.Printf("Warning: failed to delete Sentry app %s: %v\n", appSlug, err)
		}
	}

	return nil
}

func (s *Sentry) Sync(ctx core.SyncContext) error {
	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if configuration.AuthToken == "" {
		return fmt.Errorf("authToken is required")
	}

	if configuration.OrgSlug == "" {
		return fmt.Errorf("orgSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Try to auto-create a Sentry App
	appName := "SuperPlane Integration"
	webhookURL := ctx.WebhooksBaseURL

	events := []SentryAppEvent{
		{Type: "issue.created"},
		{Type: "issue.resolved"},
		{Type: "issue.assigned"},
		{Type: "issue.ignored"},
		{Type: "issue.unresolved"},
	}

	sentryApp, err := client.CreateSentryApp(appName, webhookURL, events)
	if err != nil {
		// Auto-create failed - this could be due to insufficient permissions or self-hosted Sentry
		// Fall back to browser action for manual setup
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: fmt.Sprintf("To set up Sentry webhooks, please create a new Internal Integration manually:\n\n1. Go to Settings > Developer Settings > Internal Integrations in Sentry\n2. Create a new integration named 'SuperPlane'\n3. Set the webhook URL to: %s\n4. Enable issue events: created, resolved, assigned, ignored, unresolved\n5. After creating, provide the Client Secret and App Slug below", webhookURL),
			URL:         "https://sentry.io/settings/organizations/",
			Method:      "GET",
			FormFields: map[string]string{
				"sentryClientSecret": "Client Secret",
				"sentryAppSlug":      "App Slug",
			},
		})
		return nil
	}

	// Auto-create succeeded - store the app details in secrets
	err = ctx.Integration.SetSecret("sentryClientSecret", []byte(sentryApp.ClientSecret))
	if err != nil {
		return fmt.Errorf("error setting client secret: %v", err)
	}

	err = ctx.Integration.SetSecret("sentryAppSlug", []byte(sentryApp.Slug))
	if err != nil {
		return fmt.Errorf("error setting app slug: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	// No resource pickers in Sentry trigger/component; return empty list
	return []core.IntegrationResource{}, nil
}

func (s *Sentry) Actions() []core.Action {
	return []core.Action{}
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
