package gitlab

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
	AuthTypePersonalAccessToken = "personalAccessToken"
	AuthTypeAppOAuth            = "appOAuth"
	OAuthAccessToken            = "accessToken"
	OAuthRefreshToken           = "refreshToken"
)

var scopeList = []string{
	"api",
	"read_user",
	"read_api",
	"write_repository",
	"read_repository",
}

const (
	appSetupDescription = `
- Click the **Continue** button to go to the Applications page in GitLab
- Add new application:
  - **Name**: SuperPlane
  - **Redirect URI**: ` + "`%s`" + `
  - **Scopes**: %s
- Copy the **Client ID** and **Client Secret**, and paste them in the fields below.
- Click **Save** to complete the setup.
`

	appConnectDescription = `Click **Continue** to authorize SuperPlane to access your GitLab account.`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("gitlab", &GitLab{}, &GitLabWebhookHandler{})
}

type GitLab struct {
}

type Configuration struct {
	AuthType     string `mapstructure:"authType" json:"authType"`
	BaseURL      string `mapstructure:"baseUrl" json:"baseUrl"`
	ClientID     string `mapstructure:"clientId" json:"clientId"`
	ClientSecret string `mapstructure:"clientSecret" json:"clientSecret"`
	GroupID      string `mapstructure:"groupId" json:"groupId"`
	AccessToken  string `mapstructure:"accessToken" json:"accessToken"`
}

type Metadata struct {
	State    *string           `mapstructure:"state,omitempty" json:"state,omitempty"`
	Projects []ProjectMetadata `mapstructure:"projects" json:"projects"`
	User     *UserMetadata     `mapstructure:"user,omitempty" json:"user,omitempty"`
}

type UserMetadata struct {
	ID       int    `mapstructure:"id" json:"id"`
	Name     string `mapstructure:"name" json:"name"`
	Username string `mapstructure:"username" json:"username"`
}

type ProjectMetadata struct {
	ID   int    `mapstructure:"id" json:"id"`
	Name string `mapstructure:"name" json:"name"`
	URL  string `mapstructure:"url" json:"url"`
}

func (g *GitLab) Name() string {
	return "gitlab"
}

func (g *GitLab) Label() string {
	return "GitLab"
}

func (g *GitLab) Icon() string {
	return "gitlab"
}

func (g *GitLab) Description() string {
	return "Manage and react to changes in your GitLab repositories"
}

func (g *GitLab) Instructions() string {
	return fmt.Sprintf(`
When connecting using App OAuth:
- Leave **Client ID** and **Secret** empty to start the setup wizard.

When connecting using Personal Access Token:
- Go to Preferences → Personal Access Token → Add New token
- Use **Scopes**: %s
- Copy the token and paste it into the **Access Token** configuration field, then click **Save**.
`, strings.Join(scopeList, ", "))
}

func (g *GitLab) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "GitLab URL",
			Type:        configuration.FieldTypeString,
			Description: "GitLab instance URL (or leave empty for https://gitlab.com)",
			Default:     "https://gitlab.com",
		},
		{
			Name:        "groupId",
			Label:       "Group ID",
			Type:        configuration.FieldTypeString,
			Description: "Group ID",
			Required:    true,
		},
		{
			Name:     "authType",
			Label:    "Auth Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "App OAuth", Value: AuthTypeAppOAuth},
						{Label: "Personal Access Token", Value: AuthTypePersonalAccessToken},
					},
				},
			},
		},
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Description: "OAuth Client ID from your GitLab app",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAppOAuth}},
			},
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "OAuth Client Secret from your GitLab app",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAppOAuth}},
			},
		},
		{
			Name:        "accessToken",
			Label:       "Access Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Access Token from your GitLab user settings",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypePersonalAccessToken}},
			},
		},
	}
}

func (g *GitLab) Components() []core.Component {
	return []core.Component{
		&CreateIssue{},
		&RunPipeline{},
		&GetPipeline{},
		&GetLatestPipeline{},
		&GetTestReportSummary{},
	}
}

func (g *GitLab) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
		&OnMergeRequest{},
		&OnMilestone{},
		&OnPipeline{},
		&OnRelease{},
		&OnTag{},
		&OnVulnerability{},
	}
}

func (g *GitLab) Sync(ctx core.SyncContext) error {

	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	configuration.BaseURL = normalizeBaseURL(configuration.BaseURL)

	if configuration.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	switch configuration.AuthType {
	case AuthTypeAppOAuth:
		return g.oauthSync(ctx, configuration)

	case AuthTypePersonalAccessToken:
		return g.personalAccessTokenSync(ctx)

	default:
		return fmt.Errorf("unknown authType: %s", configuration.AuthType)
	}
}

func (g *GitLab) oauthSync(ctx core.SyncContext, configuration Configuration) error {
	baseURL := configuration.BaseURL
	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	clientID, _ := ctx.Integration.GetConfig("clientId")
	clientSecret, _ := ctx.Integration.GetConfig("clientSecret")

	//
	// If no client ID or secret, show setup instructions
	//
	if string(clientID) == "" || string(clientSecret) == "" {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: fmt.Sprintf(appSetupDescription, callbackURL, strings.Join(scopeList, ", ")),
			URL:         fmt.Sprintf("%s/-/user_settings/applications", baseURL),
			Method:      "GET",
		})

		return nil
	}

	//
	// If access token is not available, ask user to authorize the app.
	//
	accessToken, _ := findSecret(ctx.Integration, OAuthAccessToken)
	if accessToken == "" {
		return g.handleOAuthNoAccessToken(ctx, baseURL, callbackURL, string(clientID))
	}

	//
	// If refresh token is available, refresh the token
	// and update the metadata.
	//
	err := g.refreshToken(ctx, baseURL, string(clientID), string(clientSecret))
	if err != nil {
		ctx.Logger.Errorf("Failed to refresh token: %v", err)
		return err
	}

	if err := g.updateMetadata(ctx); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func (g *GitLab) refreshToken(ctx core.SyncContext, baseURL, clientID, clientSecret string) error {
	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshToken)
	if refreshToken == "" {
		ctx.Logger.Warn("GitLab integration has no refresh token - not refreshing token")
		return nil
	}

	ctx.Logger.Info("Refreshing GitLab token")
	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(baseURL, clientID, clientSecret, refreshToken)

	if err != nil {
		_ = ctx.Integration.SetSecret(OAuthRefreshToken, []byte(""))
		_ = ctx.Integration.SetSecret(OAuthAccessToken, []byte(""))
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	if tokenResponse.AccessToken != "" {
		ctx.Logger.Info("Saving access token")
		err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken))
		if err != nil {
			return fmt.Errorf("failed to save access token: %v", err)
		}
	}

	if tokenResponse.RefreshToken != "" {
		ctx.Logger.Info("Saving refresh token")
		err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResponse.RefreshToken))
		if err != nil {
			return fmt.Errorf("failed to save refresh token: %v", err)
		}
	}

	ctx.Logger.Info("Token refreshed successfully")
	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

func (g *GitLab) handleOAuthNoAccessToken(ctx core.SyncContext, baseURL string, callbackURL string, clientID string) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("Failed to decode metadata while setting state: %v", err)
	}

	if metadata.State == nil {
		var err error
		s, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		metadata.State = &s
		ctx.Integration.SetMetadata(metadata)
	}

	authURL := fmt.Sprintf(
		"%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		baseURL,
		url.QueryEscape(clientID),
		url.QueryEscape(callbackURL),
		url.QueryEscape(strings.Join(scopeList, " ")),
		url.QueryEscape(*metadata.State),
	)

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appConnectDescription,
		URL:         authURL,
		Method:      "GET",
	})

	return nil
}

func (g *GitLab) personalAccessTokenSync(ctx core.SyncContext) error {
	token, err := ctx.Integration.GetConfig("accessToken")
	if err != nil {
		return fmt.Errorf("access token is required")
	}

	if string(token) == "" {
		return fmt.Errorf("access token is required")
	}

	if err := g.updateMetadata(ctx); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func (g *GitLab) updateMetadata(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	user, projects, err := client.FetchIntegrationData()
	if err != nil {
		return err
	}

	ps := []ProjectMetadata{}
	for _, p := range projects {
		ps = append(ps, ProjectMetadata{
			ID:   p.ID,
			Name: p.PathWithNamespace,
			URL:  p.WebURL,
		})
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	metadata.Projects = ps
	metadata.State = nil
	metadata.User = &UserMetadata{
		ID:       user.ID,
		Name:     user.Name,
		Username: user.Username,
	}

	ctx.Integration.SetMetadata(metadata)

	return nil
}

func (g *GitLab) HandleRequest(ctx core.HTTPRequestContext) {
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

	baseURL, _ := ctx.Integration.GetConfig("baseUrl")
	strBaseURL := normalizeBaseURL(string(baseURL))

	strClientID := string(clientID)
	strClientSecret := string(clientSecret)

	config := &Configuration{
		BaseURL:      strBaseURL,
		ClientID:     strClientID,
		ClientSecret: strClientSecret,
	}

	g.handleCallback(ctx, config)
}

func (g *GitLab) handleCallback(ctx core.HTTPRequestContext, config *Configuration) {
	redirectBaseURL := ctx.BaseURL
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	redirectURI := fmt.Sprintf("%s/api/v1/integrations/%s/callback", redirectBaseURL, ctx.Integration.ID().String())

	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.HandleCallback(ctx.Request, config, *metadata.State, redirectURI)

	if err != nil {
		ctx.Logger.Errorf("Callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request,
			fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
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

	if err := g.updateMetadata(core.SyncContext{
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}); err != nil {
		ctx.Logger.Errorf("Callback error: failed to update metadata: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request,
		fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
		http.StatusSeeOther)
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

func normalizeBaseURL(url string) string {
	if url == "" {
		return "https://gitlab.com"
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return strings.TrimSuffix(url, "/")
}

func (g *GitLab) Actions() []core.Action {
	return []core.Action{}
}

func (g *GitLab) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (g *GitLab) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}
