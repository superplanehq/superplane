package jira

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	SecretOAuthAccessToken  = "accessToken"
	SecretOAuthRefreshToken = "refreshToken"
)

func init() {
	registry.RegisterIntegration("jira", &Jira{})
}

type Jira struct{}

type Metadata struct {
	State    *string   `json:"state,omitempty" mapstructure:"state,omitempty"`
	User     *User     `json:"user,omitempty" mapstructure:"user,omitempty"`
	Projects []Project `json:"projects" mapstructure:"projects"`
	CloudID  string    `json:"cloudId,omitempty" mapstructure:"cloudId,omitempty"`
	SiteURL  string    `json:"siteUrl,omitempty" mapstructure:"siteUrl,omitempty"`
	SiteName string    `json:"siteName,omitempty" mapstructure:"siteName,omitempty"`
}

const installationInstructions = `
SuperPlane connects to Jira with OAuth.

1. Click **Save** with Client ID and Client Secret empty to see the steps for creating an Atlassian OAuth app.
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
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Description: "OAuth Client ID from your Atlassian OAuth 2.0 (3LO) app. Leave empty and click Save to see setup instructions.",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "OAuth Client Secret from your Atlassian OAuth 2.0 (3LO) app",
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
	return []core.Trigger{
		&OnIssue{},
	}
}

func (j *Jira) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (j *Jira) Sync(ctx core.SyncContext) error {
	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	//
	// Sensitive configuration values are stored encrypted, and only
	// GetConfig decrypts them - never read the client secret from
	// ctx.Configuration directly.
	//
	clientID, _ := ctx.Integration.GetConfig("clientId")
	clientSecret, _ := ctx.Integration.GetConfig("clientSecret")

	//
	// Without app credentials, walk the user through creating the OAuth app.
	// Unlike Linear, Atlassian's app creation form can't be pre-filled via a
	// manifest URL, so the steps are shown as plain instructions instead of a
	// one-click "Continue".
	//
	if len(clientID) == 0 || len(clientSecret) == 0 {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appSetupInstructions(callbackURL),
		})

		return nil
	}

	//
	// With credentials but no access token, ask the user to authorize the app.
	//
	accessToken, _ := findSecret(ctx.Integration, SecretOAuthAccessToken)
	if accessToken == "" {
		return j.requestAuthorization(ctx, string(clientID), callbackURL)
	}

	if err := j.updateMetadata(ctx); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

// jsmScopesForInstructions lists the scopes shown in the setup instructions. offline_access is
// requested on the authorize URL too (see scopeList in oauth.go) but omitted here since it isn't
// selectable in the Developer Console's Permissions tab.
const jsmScopesForInstructions = "`read:jira-work`, `write:jira-work`, `manage:jira-webhook`, `read:jira-user`, " +
	"`read:servicedesk-request`, `write:servicedesk-request`, " +
	"`read:incident:jira-service-management`, `write:incident:jira-service-management`, " +
	"`read:ops-alert:jira-service-management`, `write:ops-alert:jira-service-management`, `delete:ops-alert:jira-service-management`, " +
	"`read:ops-config:jira-service-management`, `write:ops-config:jira-service-management`, `delete:ops-config:jira-service-management`"

func appSetupInstructions(callbackURL string) string {
	return fmt.Sprintf(`
**1. Create an OAuth 2.0 (3LO) app**

Open the [Atlassian Developer Console](https://developer.atlassian.com/console/myapps/) and create an **OAuth 2.0 (3LO)** app.

**2. Add the Jira API and the Jira Service Management API, with these scopes**

Go to the **Permissions** tab and add the **Jira API** and the **Jira Service Management API**, then add these scopes: %s.

**3. Configure the callback URL**

Go to the **Authorization** tab, click **Configure** next to OAuth 2.0 (3LO), and paste this callback URL:

%s

**4. Complete the installation setup**

Go to the **Settings** tab to find the app's **Client ID** and **Client Secret**. Paste them into the fields below and click **Save**.
`, jsmScopesForInstructions, callbackURL)
}

// requestAuthorization sends the user to Atlassian to approve the OAuth app, using a CSRF state
// that persists across Syncs until the callback consumes it.
func (j *Jira) requestAuthorization(ctx core.SyncContext, clientID, callbackURL string) error {
	metadata := readMetadata(ctx.Integration)

	if metadata.State == nil {
		state, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		metadata.State = &state
		ctx.Integration.SetMetadata(metadata)
	}

	authorizeURL := fmt.Sprintf(
		"%s?audience=api.atlassian.com&client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&prompt=consent",
		AuthorizeURL,
		url.QueryEscape(clientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape(scopeList),
		url.QueryEscape(*metadata.State),
	)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: "Click **Continue** to authorize SuperPlane to access your Jira site.",
		URL:         authorizeURL,
		Method:      "GET",
	})

	return nil
}

func (j *Jira) updateMetadata(ctx core.SyncContext) error {
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

	metadata := readMetadata(ctx.Integration)
	metadata.User = user
	metadata.Projects = projects
	ctx.Integration.SetMetadata(metadata)

	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	clientID, err := ctx.Integration.GetConfig("clientId")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	metadata := readMetadata(ctx.Integration)
	expectedState := ""
	if metadata.State != nil {
		expectedState = *metadata.State
	}

	settingsURL := fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID())
	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	auth := NewAuth(ctx.HTTP)
	token, err := auth.HandleCallback(ctx.Request, string(clientID), string(clientSecret), expectedState, redirectURI)
	if err != nil {
		ctx.Logger.Errorf("Callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	if err := ctx.Integration.SetSecret(SecretOAuthAccessToken, []byte(token.AccessToken)); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := ctx.Integration.SetSecret(SecretOAuthRefreshToken, []byte(token.RefreshToken)); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	resources, err := auth.AccessibleResources(token.AccessToken)
	if err != nil {
		ctx.Logger.Errorf("Callback error: failed to fetch accessible resources: %v", err)
		ctx.Integration.Error(fmt.Sprintf("connected, but failed to resolve Jira site: %v", err))
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	// PoC simplification: connect the first accessible site rather than prompting to choose one.
	site := resources[0]

	metadata.State = nil
	metadata.CloudID = site.ID
	metadata.SiteURL = site.URL
	metadata.SiteName = site.Name
	ctx.Integration.SetMetadata(metadata)

	//
	// Tokens and site are stored at this point, so a metadata failure must not
	// strand the user on a bare error page: surface it through the integration
	// state and send them back to settings, where a manual sync retries.
	//
	if err := j.updateMetadata(core.SyncContext{
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}); err != nil {
		ctx.Logger.Errorf("Callback error: failed to update metadata: %v", err)
		ctx.Integration.Error(fmt.Sprintf("connected, but failed to load workspace data: %v", err))
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
}

func readMetadata(integration core.IntegrationContext) Metadata {
	metadata := Metadata{}
	_ = mapstructure.Decode(integration.GetMetadata(), &metadata)
	return metadata
}

func findSecret(integration core.IntegrationContext, name string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value), nil
		}
	}

	return "", nil
}

func (j *Jira) Hooks() []core.Hook {
	return []core.Hook{}
}

func (j *Jira) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
