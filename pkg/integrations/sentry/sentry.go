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

type Configuration struct {
	Organization string `json:"organization"`
	AuthToken    string `json:"authToken"`
}

type Metadata struct {
	Projects []Project `json:"projects"`
}

type Project struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "sentry"
}

func (s *Sentry) Description() string {
	return "Monitor and manage your Sentry issues"
}

func (s *Sentry) Instructions() string {
	return `## Setup Instructions

### Step 1: Create an Auth Token

1. Go to [sentry.io](https://sentry.io) and sign in
2. Navigate to **Settings > Auth Tokens** (under Developer Settings)
3. Click **Create New Token**
4. Select the following scopes:
   - ` + "`project:read`" + ` - To list and access projects
   - ` + "`project:write`" + ` - To create service hooks (webhooks) for triggers
   - ` + "`event:read`" + ` - To read issue events
   - ` + "`event:write`" + ` - To update issues (resolve, assign, etc.)
5. Click **Create Token** and copy the generated token

### Step 2: Find Your Organization Slug

Your organization slug is in your Sentry URL:
` + "```" + `
https://sentry.io/organizations/{org-slug}/
` + "```" + `

For example, if your URL is ` + "`https://sentry.io/organizations/acme-corp/`" + `, your organization slug is ` + "`acme-corp`" + `.

### Webhooks (Automatic)

SuperPlane automatically creates and manages webhooks for you using Sentry's Service Hooks API. When you configure an **On Issue Event** trigger:

- A service hook is automatically created in your Sentry project
- The webhook URL and secret are managed by SuperPlane
- When the trigger is removed, the service hook is automatically deleted

**Requirements:**
- Your Auth Token must have the ` + "`project:write`" + ` scope to create service hooks
- The 'servicehooks' feature must be enabled for your Sentry project

**Note:** Service hooks support ` + "`event.created`" + ` and ` + "`event.alert`" + ` event types.`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Description: "Your Sentry organization slug (e.g., 'acme-corp')",
			Placeholder: "e.g. acme-corp",
			Required:    true,
		},
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Sentry Auth Token with project:read, event:read, and event:write scopes",
			Required:    true,
		},
	}
}

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Sentry) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Verify the connection by getting organization details
	_, err = client.GetOrganization()
	if err != nil {
		return fmt.Errorf("error verifying connection: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (s *Sentry) Actions() []core.Action {
	return []core.Action{}
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Sentry) Components() []core.Component {
	return []core.Component{
		&UpdateIssue{},
	}
}

func (s *Sentry) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueEvent{},
	}
}
