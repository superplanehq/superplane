package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	appSetupDescription = `
- Click **Continue** to open the Atlassian developer console.
- Create a new **OAuth 2.0 integration** if you do not have one yet.
- Add **Jira API** permission with the following scopes: ` + "`read:jira-user`, `read:jira-work`, `write:jira-work`, `manage:jira-webhook`, `offline_access`" + `.
- Under **Authorization**, add the following **Callback URL**: ` + "`%s`" + `.
- Copy the **Client ID** and **Secret** from the **Settings** page and paste them below.
- Click **Save** to continue.
`

	appConnectDescription = `Click **Continue** to authorize SuperPlane to access your Jira site.`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("jira", &Jira{}, &JiraWebhookHandler{})
}

type Jira struct{}

type Configuration struct {
	ClientID     string `json:"clientId" mapstructure:"clientId"`
	ClientSecret string `json:"clientSecret" mapstructure:"clientSecret"`
}

type Metadata struct {
	State     *string   `json:"state,omitempty" mapstructure:"state,omitempty"`
	CloudID   string    `json:"cloudId,omitempty" mapstructure:"cloudId"`
	SiteURL   string    `json:"siteUrl,omitempty" mapstructure:"siteUrl"`
	SiteName  string    `json:"siteName,omitempty" mapstructure:"siteName"`
	AccountID string    `json:"accountId,omitempty" mapstructure:"accountId"`
	User      *User     `json:"user,omitempty" mapstructure:"user,omitempty"`
	Projects  []Project `json:"projects" mapstructure:"projects"`
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
	return strings.Join([]string{
		"SuperPlane connects to Jira Cloud through an OAuth 2.0 (3LO) integration.",
		"",
		"**Setup steps:**",
		"",
		"1. Visit https://developer.atlassian.com/console/myapps/ and create an **OAuth 2.0 integration** (or pick an existing one).",
		"2. Under **Permissions**, enable **Jira API** and add scopes: `read:jira-user`, `read:jira-work`, `write:jira-work`, `manage:jira-webhook`, `offline_access`.",
		"3. Under **Authorization → OAuth 2.0 (3LO)**, set the **Callback URL** to the value SuperPlane displays when you start the setup.",
		"4. Paste the **Client ID** and **Secret** from the **Settings** page below, then save the integration.",
		"5. Click **Continue** to authorize the connection to your Jira site.",
	}, "\n")
}

func (j *Jira) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OAuth Client ID from the Atlassian developer console",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "OAuth Client Secret from the Atlassian developer console",
		},
	}
}

func (j *Jira) Actions() []core.Action {
	return []core.Action{
		&CreateIssue{},
		&GetIssue{},
		&UpdateIssue{},
		&DeleteIssue{},
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
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	if config.ClientID == "" || config.ClientSecret == "" {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: fmt.Sprintf(appSetupDescription, callbackURL),
			URL:         "https://developer.atlassian.com/console/myapps/",
			Method:      http.MethodGet,
		})
		return nil
	}

	accessToken, _ := findSecret(ctx.Integration, OAuthAccessToken)
	if accessToken == "" {
		return j.startAuthorization(ctx, config, callbackURL)
	}

	if err := j.refreshToken(ctx, config); err != nil {
		ctx.Logger.Errorf("Failed to refresh Jira token: %v", err)
		return j.startAuthorization(ctx, config, callbackURL)
	}

	if err := j.updateMetadata(ctx); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func (j *Jira) startAuthorization(ctx core.SyncContext, config Configuration, callbackURL string) error {
	metadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)

	if metadata.State == nil {
		state, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		metadata.State = &state
		ctx.Integration.SetMetadata(metadata)
	}

	authURL := BuildAuthorizationURL(config.ClientID, callbackURL, *metadata.State)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appConnectDescription,
		URL:         authURL,
		Method:      http.MethodGet,
	})

	return nil
}

func (j *Jira) refreshToken(ctx core.SyncContext, config Configuration) error {
	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshToken)
	if refreshToken == "" {
		ctx.Logger.Warn("Jira integration has no refresh token - not refreshing token")
		return nil
	}

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(config.ClientID, config.ClientSecret, refreshToken)
	if err != nil {
		_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(""))
		_ = ctx.Integration.SetSecret(OAuthRefreshToken, []byte(""))
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	if tokenResponse.AccessToken != "" {
		if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
			return fmt.Errorf("failed to save access token: %v", err)
		}
	}

	if tokenResponse.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			return fmt.Errorf("failed to save refresh token: %v", err)
		}
	}

	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

func (j *Jira) updateMetadata(ctx core.SyncContext) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	accessToken, _ := findSecret(ctx.Integration, OAuthAccessToken)
	if accessToken == "" {
		return fmt.Errorf("missing access token")
	}

	auth := NewAuth(ctx.HTTP)
	resources, err := auth.ListAccessibleResources(accessToken)
	if err != nil {
		return fmt.Errorf("failed to list accessible Jira sites: %v", err)
	}

	if len(resources) == 0 {
		return fmt.Errorf("no Jira sites accessible to this OAuth app — make sure the user authorized at least one site")
	}

	site := resources[0]
	metadata.CloudID = site.ID
	metadata.SiteURL = site.URL
	metadata.SiteName = site.Name

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to fetch current user: %v", err)
	}
	metadata.User = user
	metadata.AccountID = user.AccountID

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %v", err)
	}
	metadata.Projects = projects
	metadata.State = nil

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (j *Jira) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	clientIDBytes, err := ctx.Integration.GetConfig("clientId")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientSecretBytes, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientID := string(clientIDBytes)
	clientSecret := string(clientSecretBytes)

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	expectedState := ""
	if metadata.State != nil {
		expectedState = *metadata.State
	}

	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID().String())

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.HandleCallback(ctx.Request, clientID, clientSecret, expectedState, redirectURI)
	if err != nil {
		ctx.Logger.Errorf("Jira callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request,
			fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
			http.StatusSeeOther)
		return
	}

	if tokenResponse.AccessToken != "" {
		if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken)); err != nil {
			ctx.Response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if tokenResponse.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken)); err != nil {
			ctx.Response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err := ctx.Integration.ScheduleResync(tokenResponse.GetExpiration()); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	syncCtx := core.SyncContext{
		Logger:      ctx.Logger,
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}

	if err := j.updateMetadata(syncCtx); err != nil {
		ctx.Logger.Errorf("Jira callback error: failed to update metadata: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request,
		fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
		http.StatusSeeOther)
}

func (j *Jira) Hooks() []core.Hook {
	return []core.Hook{}
}

func (j *Jira) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
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
