package linear

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("linear", &Linear{}, &LinearWebhookHandler{})
}

type Linear struct{}

type Metadata struct {
	User         *User  `json:"user,omitempty" mapstructure:"user,omitempty"`
	Teams        []Team `json:"teams" mapstructure:"teams"`
	Organization string `json:"organization,omitempty" mapstructure:"organization,omitempty"`
	URLKey       string `json:"urlKey,omitempty" mapstructure:"urlKey,omitempty"`
}

const installationInstructions = `
To connect Linear to SuperPlane:

1. Open [Linear API settings](https://linear.app/settings/account/security) and go to **Personal API keys**.
2. Click **New API key**, give it a recognizable label such as ` + "`SuperPlane`" + `, and copy the generated key.
3. Paste the key into the **API Key** field below and click **Save**.

**Permissions:** components act as the user who owns the API key, so that user needs access to the
teams you want to automate.

- **Create Issue** requires the key owner to be a member of the team the issue is created in.
- **On Issue** creates a Linear webhook through the API. Only **workspace admins** can manage
  webhooks, so the key owner must be an admin of the workspace.

**Note:** actions performed by SuperPlane are attributed to the user who owns the API key, and the
integration stops working if that user leaves the workspace.
`

func (l *Linear) Name() string {
	return "linear"
}

func (l *Linear) Label() string {
	return "Linear"
}

func (l *Linear) Icon() string {
	return "linear"
}

func (l *Linear) Description() string {
	return "Manage and react to issues in Linear"
}

func (l *Linear) Instructions() string {
	return installationInstructions
}

func (l *Linear) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "Personal API key from https://linear.app/settings/account/security",
		},
	}
}

func (l *Linear) Actions() []core.Action {
	return []core.Action{
		&CreateIssue{},
	}
}

func (l *Linear) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (l *Linear) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	viewer, err := client.GetViewer()
	if err != nil {
		return fmt.Errorf("error verifying Linear credentials: %v", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("error listing teams: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		User:         viewer.User,
		Teams:        teams,
		Organization: viewer.Organization.Name,
		URLKey:       viewer.Organization.URLKey,
	})

	ctx.Integration.Ready()
	return nil
}

func (l *Linear) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *Linear) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (l *Linear) Hooks() []core.Hook {
	return []core.Hook{}
}

func (l *Linear) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
