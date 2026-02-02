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
	AuthTypeAppOAuth           = "appOAuth"
	OAuthAccessToken           = "accessToken"
	OAuthRefreshToken          = "refreshToken"
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
## GitLab OAuth Setup

**Step 1: Create a GitLab OAuth Application**

1. Go to GitLab → **User Settings** → **Applications**
   (For self-hosted: **Admin Area** → **Applications**)
2. Fill in the following:
   - **Name**: SuperPlane Integration (or which name you want)
   - **Redirect URI**: ` + "`%s`" + `
   - **Scopes**: Select: %s
3. Click **Save application**
4. Copy the **Client ID** and **Secret**

**Step 2: Enter Credentials**

Enter the **Client ID** and **Client Secret** in the fields above, then click **Save**.
`

	appConnectDescription = `Click **Connect to GitLab** to authorize SuperPlane to access your GitLab account.`

	patSetupDescription = `
## Personal Access Token Setup

**Step 1: Create a Personal Access Token**

1. Go to GitLab → **User Settings** → **Access Tokens**
2. Create a new token with:
   - **Name**: SuperPlane Integration
   - **Scopes**: Select: %s
3. Click **Create personal access token**
4. Copy the token value

**Step 2: Enter Token**

Paste the token into the **Personal Access Token** field above, then click **Save**.
`
)

func init() {
	registry.RegisterIntegration("gitlab", &GitLab{})
}

type GitLab struct {
}

type Configuration struct {
	AuthType     string  `json:"authType"`
	BaseURL      string  `json:"baseUrl"`
	ClientID     *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
	GroupID             *string `json:"groupId"`
	PersonalAccessToken *string `json:"personalAccessToken"`
}

type Metadata struct {
	State string `mapstructure:"state" json:"state"`
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
	scopes := strings.Join(scopeList, ", ")
	return fmt.Sprintf("For **App OAuth**, you can leave the **Client ID** and **Secret** fields empty initially. You will be guided through the OAuth application setup process with the required callback URL.\n\nFor **Personal Access Token**, generate a token in your GitLab User Settings > Access Tokens with the following scopes: %s", scopes)
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
			Name:     "authType",
			Label:    "Auth Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AuthTypeAppOAuth,
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
			Name:        "personalAccessToken",
			Label:       "Personal Access Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Personal Access Token from your GitLab user settings",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypePersonalAccessToken}},
			},
		},
		{
			Name:     "groupId",
			Label:    "Group ID",
			Type:     configuration.FieldTypeString,
			Description: "Group ID",
			Required: true,
		},
	}
}

func (g *GitLab) Components() []core.Component {
	return []core.Component{}
}

func (g *GitLab) Triggers() []core.Trigger {
	return []core.Trigger{}
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
		return g.personalAccessTokenSync(ctx, configuration)

	default:
		return fmt.Errorf("unknown authType: %s", configuration.AuthType)
	}
}

func (g *GitLab) oauthSync(ctx core.SyncContext, configuration Configuration) error {
	baseURL := configuration.BaseURL
	baseURL = strings.TrimSuffix(baseURL, "/")

	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.InstallationID)

	// Case 1: No credentials yet - show setup instructions
	if configuration.ClientID == nil || *configuration.ClientID == "" {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: fmt.Sprintf(appSetupDescription, callbackURL, strings.Join(scopeList, ", ")),
			URL:         fmt.Sprintf("%s/-/user_settings/applications", baseURL),
			Method:      "GET",
		})
		ctx.Integration.SetState("pending", "Enter Client ID and Secret")
		return nil
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		return err
	}
	
	// Case 2: Has credentials but no tokens - show auth button
	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshToken)
	if refreshToken == "" {
		state, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate state: %v", err)
		}
		ctx.Integration.SetMetadata(Metadata{State: state})

		authURL := fmt.Sprintf(
			"%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
			baseURL,
			url.QueryEscape(*configuration.ClientID),
			url.QueryEscape(callbackURL),
			url.QueryEscape(strings.Join(scopeList, " ")),
			url.QueryEscape(state),
		)

		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appConnectDescription,
			URL:         authURL,
			Method:      "GET",
		})
		ctx.Integration.SetState("pending", "Click Connect to GitLab to authorize")
		return nil
	}

	// STEP 3: Has tokens - refresh and set ready
	authService := NewAuthService(ctx.HTTP)
	tokenResp, err := authService.RefreshToken(baseURL, *configuration.ClientID, string(clientSecret), refreshToken)
	if err != nil {
		return fmt.Errorf("token expired, please re-authorize: %v", err)
	}

	if err := ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResp.AccessToken)); err != nil {
		return err
	}

	if err := ctx.Integration.SetSecret(OAuthRefreshToken, []byte(tokenResp.RefreshToken)); err != nil {
			return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify access token: %v", err)
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.SetState("ready", "")
	return ctx.Integration.ScheduleResync(tokenResp.GetExpiration())
}

func (g *GitLab) personalAccessTokenSync(ctx core.SyncContext, configuration Configuration) error {
	token := ""
	if configuration.PersonalAccessToken != nil {
		token = *configuration.PersonalAccessToken
	}
	
	if len(token) == 0 {
		baseURL := configuration.BaseURL

		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: fmt.Sprintf(patSetupDescription, strings.Join(scopeList, ", ")),
			URL:         baseURL,
			Method:      "GET",
		})
		ctx.Integration.SetState("pending", "Waiting for Personal Access Token")
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("invalid personal access token: %v", err)
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.SetState("ready", "")
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
		ClientID:     &strClientID,
		ClientSecret: &strClientSecret,
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

	authService := NewAuthService(ctx.HTTP)
	tokenResponse, err := authService.HandleCallback(ctx.Request, config, metadata.State, redirectURI)

	if err != nil {
		http.Redirect(ctx.Response, ctx.Request,
			fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
			http.StatusSeeOther)
		return
	}

	// Success - Save Tokens
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

	ctx.Integration.SetMetadata(Metadata{})

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := client.Verify(); err != nil {
		http.Redirect(ctx.Response, ctx.Request,
			fmt.Sprintf("%s/%s/settings/integrations/%s", redirectBaseURL, ctx.OrganizationID, ctx.Integration.ID().String()),
			http.StatusSeeOther)
		return
	}

	if err := ctx.Integration.ScheduleResync(tokenResponse.GetExpiration()); err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.SetState("ready", "")

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



func (g *GitLab) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (g *GitLab) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (g *GitLab) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (g *GitLab) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func normalizeBaseURL(url string) string {
	if url == "" {
		return "https://gitlab.com"
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "https://" + url
	}
	return url
}
