package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithOptions("jira", &Jira{}, registry.IntegrationRegistrationOptions{
		SetupProvider: &SetupProvider{},
	})
}

type Jira struct{}

type Metadata struct {
	User     *User     `json:"user,omitempty" mapstructure:"user,omitempty"`
	Projects []Project `json:"projects" mapstructure:"projects"`
	CloudID  string    `json:"cloudId,omitempty" mapstructure:"cloudId,omitempty"`
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
	return "Manage issues in Jira"
}

// Instructions is unused: connection is handled by the SetupProvider OAuth flow instead.
func (j *Jira) Instructions() string {
	return ""
}

// Configuration is empty: connection details are collected by the SetupProvider OAuth flow instead.
func (j *Jira) Configuration() []configuration.Field {
	return []configuration.Field{}
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
	return []core.Trigger{
		&OnIssue{},
	}
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

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("error listing projects: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{User: user, Projects: projects, CloudID: client.CloudID})
	ctx.Integration.Ready()
	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/redirect") {
		j.afterOAuthRedirect(ctx)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

// afterOAuthRedirect exchanges the authorization code for tokens and stores the connected site.
func (j *Jira) afterOAuthRedirect(ctx core.HTTPRequestContext) {
	code := ctx.Request.URL.Query().Get("code")
	state := ctx.Request.URL.Query().Get("state")
	if code == "" || state == "" {
		ctx.Logger.Errorf("missing code or state")
		http.Error(ctx.Response, "missing code or state", http.StatusBadRequest)
		return
	}

	expectedState, err := ctx.Integration.Properties().GetString(PropertyOAuthState)
	if err != nil || expectedState == "" || expectedState != state {
		ctx.Logger.Errorf("invalid OAuth state")
		http.Error(ctx.Response, "invalid OAuth state", http.StatusBadRequest)
		return
	}

	clientID, err := ctx.Integration.Properties().GetString(PropertyClientID)
	if err != nil {
		ctx.Logger.Errorf("failed to read client id: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	clientSecret, err := ctx.Integration.Secrets().Get(SecretOAuthClientSecret)
	if err != nil {
		ctx.Logger.Errorf("failed to read client secret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := exchangeCodeForToken(ctx.HTTP, clientID, clientSecret, code, redirectURI(ctx.BaseURL, ctx.Integration.ID().String()))
	if err != nil {
		ctx.Logger.Errorf("failed to exchange code for token: %v", err)
		http.Error(ctx.Response, "failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	resources, err := fetchAccessibleResources(ctx.HTTP, token.AccessToken)
	if err != nil {
		ctx.Logger.Errorf("failed to fetch accessible resources: %v", err)
		http.Error(ctx.Response, "failed to fetch accessible Jira sites", http.StatusInternalServerError)
		return
	}

	// PoC simplification: connect the first accessible site rather than prompting to choose one.
	site := resources[0]

	err = ctx.Integration.Properties().CreateMany([]core.IntegrationPropertyDefinition{
		{Name: PropertyCloudID, Label: "Jira Cloud ID", Type: core.IntegrationPropertyTypeString, Value: site.ID, Editable: false},
		{Name: PropertySiteURL, Label: "Jira Site URL", Type: core.IntegrationPropertyTypeString, Value: site.URL, Editable: false},
		{Name: PropertySiteName, Label: "Jira Site Name", Type: core.IntegrationPropertyTypeString, Value: site.Name, Editable: false},
	})
	if err != nil {
		ctx.Logger.Errorf("failed to save site properties: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.Secrets().CreateMany([]core.IntegrationSecretDefinition{
		{Name: SecretOAuthAccessToken, Label: "OAuth Access Token", Value: token.AccessToken, Editable: false},
		{Name: SecretOAuthRefreshToken, Label: "OAuth Refresh Token", Value: token.RefreshToken, Editable: false},
	})
	if err != nil {
		ctx.Logger.Errorf("failed to save OAuth tokens: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	// Best-effort immediate sync so the integration isn't left unverified until the next scheduled Sync().
	if client, err := NewClient(ctx.HTTP, ctx.Integration); err == nil {
		if user, err := client.GetCurrentUser(); err == nil {
			projects, _ := client.ListProjects()
			ctx.Integration.SetMetadata(Metadata{User: user, Projects: projects, CloudID: site.ID})
		}
	}
	ctx.Integration.Ready()

	err = ctx.IntegrationSetup.SetStep(core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         "done",
		Label:        "Jira connection completed successfully",
		Instructions: fmt.Sprintf("Connected to **%s** (%s).", site.Name, site.URL),
	})
	if err != nil {
		ctx.Logger.Errorf("failed to finish setup: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(ctx.Response, ctx.Request, ctx.BaseURL, http.StatusSeeOther)
}

func (j *Jira) Hooks() []core.Hook {
	return []core.Hook{}
}

func (j *Jira) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
