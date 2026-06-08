package jira

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("jira", &Jira{})
}

type Jira struct{}

type Metadata struct {
	User     *User     `json:"user,omitempty" mapstructure:"user,omitempty"`
	Projects []Project `json:"projects" mapstructure:"projects"`
	CloudID  string    `json:"cloudId,omitempty" mapstructure:"cloudId,omitempty"`
}

const installationInstructions = `
To connect Jira to SuperPlane:

1. Open [Atlassian API tokens](https://id.atlassian.com/manage-profile/security/api-tokens).
2. Click **Create API token**, give it a recognizable label, and copy the generated token.
3. Paste your **Jira Site URL** into SuperPlane. For Jira Cloud, this usually looks like ` + "`https://your-domain.atlassian.net`" + `.
4. Paste the Atlassian account **Email** that owns the API token.
5. Paste the generated **API Token**.
`

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
	return "Manage issues in Jira"
}

func (j *Jira) Instructions() string {
	return installationInstructions
}

func (j *Jira) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "siteUrl",
			Label:       "Site URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The base URL of your Jira Cloud site",
			Placeholder: "https://your-domain.atlassian.net",
		},
		{
			Name:        "email",
			Label:       "Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Atlassian account email associated with the API token",
			Placeholder: "you@example.com",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "API token from https://id.atlassian.com/manage-profile/security/api-tokens",
		},
	}
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{
		&CreateIssue{},
		&GetIssue{},
		&UpdateIssue{},
		&DeleteIssue{},
		&CreateIncident{},
		&GetIncident{},
		&DeleteIncident{},
		&GetWorkflow{},
		&TransitionIssue{},
		&ApproveWorkflow{},
		&CreateHeartbeat{},
		&PingHeartbeat{},
		&UpdateHeartbeat{},
		&DeleteHeartbeat{},
		&CreateAlert{},
		&GetAlert{},
		&DeleteAlert{},
		&UpdateAlert{},
	}
}

func (j *Jira) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (j *Jira) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jira) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying Jira credentials: %v", err)
	}

	cloudID, err := client.FetchCloudID()
	if err != nil {
		return fmt.Errorf("error resolving cloud id: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("error listing projects: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{User: user, Projects: projects, CloudID: cloudID})
	ctx.Integration.Ready()
	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (j *Jira) Hooks() []core.Hook {
	return []core.Hook{}
}

func (j *Jira) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
