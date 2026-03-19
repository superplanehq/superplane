package sentry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	DefaultBaseURL = "https://sentry.io"

	OAuthAccessTokenSecret  = "accessToken"
	OAuthRefreshTokenSecret = "refreshToken"

	ResourceTypeProject = "project"
	ResourceTypeTeam    = "team"
)

var scopeList = []string{
	"org:read",
	"project:read",
	"team:read",
	"event:read",
	"event:write",
}

const (
	appSetupDescription = `
- Click **Continue** to open Sentry Developer Settings.
- Create a **Public Integration** / **Sentry App**.
- Configure:
  - **Integration Name**: SuperPlane
  - **Slug**: any unique slug (for example ` + "`superplane`" + `)
  - **Redirect URL**: ` + "`%s`" + `
  - **Webhook URL**: ` + "`%s`" + `
  - **Webhook Subscriptions**: ` + "`issue`" + ` and ` + "`installation`" + `
  - **Scopes**: %s
- Copy the **Slug**, **Client ID**, and **Client Secret** into SuperPlane and save.
`

	appConnectDescription = `Click **Continue** to install the Sentry integration in your organization and authorize SuperPlane.`
)

func init() {
	registry.RegisterIntegration("sentry", &Sentry{})
}

type Sentry struct{}

type Configuration struct {
	BaseURL      string `json:"baseUrl" mapstructure:"baseUrl"`
	AppSlug      string `json:"appSlug" mapstructure:"appSlug"`
	ClientID     string `json:"clientId" mapstructure:"clientId"`
	ClientSecret string `json:"clientSecret" mapstructure:"clientSecret"`
}

type Metadata struct {
	InstallationID string               `json:"installationId" mapstructure:"installationId"`
	AppSlug        string               `json:"appSlug" mapstructure:"appSlug"`
	Organization   *OrganizationSummary `json:"organization,omitempty" mapstructure:"organization,omitempty"`
	Projects       []ProjectSummary     `json:"projects" mapstructure:"projects"`
	Teams          []TeamSummary        `json:"teams" mapstructure:"teams"`
}

type OrganizationSummary struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type ProjectSummary struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type TeamSummary struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type SubscriptionConfiguration struct {
	Resources []string `json:"resources" mapstructure:"resources"`
}

type WebhookInstallation struct {
	UUID string `json:"uuid" mapstructure:"uuid"`
}

type WebhookMessage struct {
	Resource     string              `json:"resource" mapstructure:"resource"`
	Action       string              `json:"action" mapstructure:"action"`
	Installation WebhookInstallation `json:"installation" mapstructure:"installation"`
	Data         map[string]any      `json:"data" mapstructure:"data"`
	Actor        map[string]any      `json:"actor,omitempty" mapstructure:"actor,omitempty"`
}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "bug"
}

func (s *Sentry) Description() string {
	return "React to issue events and update issues in Sentry"
}

func (s *Sentry) Instructions() string {
	return `
Leave **App Slug**, **Client ID**, and **Client Secret** empty to start the Sentry app setup wizard.

SuperPlane uses a Sentry **Public Integration** OAuth flow:
- create a Sentry app
- set the callback and webhook URLs shown in the setup prompt
- install the app into the target Sentry organization

Required scopes: ` + "`org:read`, `project:read`, `team:read`, `event:read`, `event:write`" + `.
`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseUrl",
			Label:       "Sentry URL",
			Type:        configuration.FieldTypeString,
			Description: "Sentry instance URL (or leave empty for https://sentry.io)",
			Default:     DefaultBaseURL,
		},
		{
			Name:        "appSlug",
			Label:       "App Slug",
			Type:        configuration.FieldTypeString,
			Description: "Slug of your Sentry public integration",
			Required:    false,
		},
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Description: "OAuth Client ID from your Sentry app",
			Required:    false,
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Description: "OAuth Client Secret from your Sentry app",
			Sensitive:   true,
			Required:    false,
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

func (s *Sentry) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	config.BaseURL = normalizeBaseURL(config.BaseURL)

	if config.AppSlug == "" || config.ClientID == "" || config.ClientSecret == "" {
		return s.createSetupPrompt(ctx, config)
	}

	accessToken, _ := findSecret(ctx.Integration, OAuthAccessTokenSecret)
	if accessToken == "" {
		return s.createInstallPrompt(ctx, config)
	}

	refreshToken, _ := findSecret(ctx.Integration, OAuthRefreshTokenSecret)
	if refreshToken != "" {
		if err := s.refreshToken(ctx, config, refreshToken); err != nil {
			return err
		}
	}

	if err := s.updateMetadata(ctx, config); err != nil {
		ctx.Integration.Error(err.Error())
		return nil
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) createSetupPrompt(ctx core.SyncContext, config Configuration) error {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: fmt.Sprintf(
			appSetupDescription,
			callbackURL(ctx),
			eventsURL(ctx),
			strings.Join(scopeList, ", "),
		),
		URL:    fmt.Sprintf("%s/settings/account/developer-settings/", config.BaseURL),
		Method: http.MethodGet,
	})
	return nil
}

func (s *Sentry) createInstallPrompt(ctx core.SyncContext, config Configuration) error {
	installURL := fmt.Sprintf("%s/sentry-apps/%s/external-install/", config.BaseURL, url.PathEscape(config.AppSlug))
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appConnectDescription,
		URL:         installURL,
		Method:      http.MethodGet,
	})
	return nil
}

func (s *Sentry) refreshToken(ctx core.SyncContext, config Configuration, refreshToken string) error {
	auth := NewAuth(ctx.HTTP)
	tokenResponse, err := auth.RefreshToken(config.BaseURL, config.ClientID, config.ClientSecret, currentInstallationID(ctx.Integration), refreshToken)
	if err != nil {
		_ = ctx.Integration.SetSecret(OAuthAccessTokenSecret, []byte(""))
		_ = ctx.Integration.SetSecret(OAuthRefreshTokenSecret, []byte(""))
		return fmt.Errorf("failed to refresh Sentry token: %w", err)
	}

	if err := storeTokenSecrets(ctx.Integration, tokenResponse); err != nil {
		return err
	}

	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Sentry) Actions() []core.Action {
	return []core.Action{}
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {
	switch {
	case strings.HasSuffix(ctx.Request.URL.Path, "/callback"):
		s.handleCallback(ctx)
	case strings.HasSuffix(ctx.Request.URL.Path, "/events"):
		s.handleWebhook(ctx)
	default:
		ctx.Response.WriteHeader(http.StatusNotFound)
	}
}

func (s *Sentry) handleCallback(ctx core.HTTPRequestContext) {
	config, err := s.loadConfiguration(ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("failed to load sentry config: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	auth := NewAuth(ctx.HTTP)
	tokenResponse, installationID, err := auth.HandleCallback(ctx.Request, config.BaseURL, config.ClientID, config.ClientSecret)
	if err != nil {
		ctx.Logger.Errorf("sentry callback failed: %v", err)
		http.Redirect(ctx.Response, ctx.Request, integrationSettingsURL(ctx), http.StatusSeeOther)
		return
	}

	if err := storeTokenSecrets(ctx.Integration, tokenResponse); err != nil {
		ctx.Logger.Errorf("failed to save sentry token secrets: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	metadata := Metadata{
		InstallationID: installationID,
		AppSlug:        config.AppSlug,
	}

	verifyResponse, err := auth.VerifyInstallation(config.BaseURL, installationID, tokenResponse.AccessToken)
	if err == nil {
		metadata.AppSlug = firstNonEmpty(verifyResponse.App.Slug, metadata.AppSlug)
		if verifyResponse.Organization.Slug != "" {
			metadata.Organization = &OrganizationSummary{
				Slug: verifyResponse.Organization.Slug,
			}
		}
	} else {
		ctx.Logger.Warnf("sentry installation verification skipped: %v", err)
	}

	ctx.Integration.SetMetadata(metadata)

	if err := s.resolveOrganization(ctx, config, tokenResponse.AccessToken); err != nil {
		ctx.Logger.Errorf("failed to resolve sentry organization: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := ctx.Integration.ScheduleResync(tokenResponse.GetExpiration()); err != nil {
		ctx.Logger.Errorf("failed to schedule sentry token refresh: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.updateMetadata(core.SyncContext{
		Logger:      ctx.Logger,
		HTTP:        ctx.HTTP,
		Integration: ctx.Integration,
	}); err != nil {
		ctx.Logger.Errorf("failed to sync sentry metadata after callback: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request, integrationSettingsURL(ctx), http.StatusSeeOther)
}

func (s *Sentry) resolveOrganization(ctx core.HTTPRequestContext, config Configuration, accessToken string) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Organization != nil && metadata.Organization.Slug != "" {
		return nil
	}

	client := NewAPIClient(ctx.HTTP, config.BaseURL, accessToken)
	organizations, err := client.ListOrganizations()
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(organizations) == 0 {
		return fmt.Errorf("no organizations available for the Sentry installation")
	}

	metadata.Organization = &OrganizationSummary{
		ID:   organizations[0].ID,
		Slug: organizations[0].Slug,
		Name: organizations[0].Name,
	}
	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (s *Sentry) handleWebhook(ctx core.HTTPRequestContext) {
	config, err := s.loadConfiguration(ctx.Integration)
	if err != nil {
		ctx.Logger.Errorf("failed to load sentry config: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("failed to read sentry webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := verifyWebhookSignature(ctx.Request.Header.Get("Sentry-Hook-Signature"), body, []byte(config.ClientSecret)); err != nil {
		ctx.Logger.Warnf("invalid sentry webhook signature: %v", err)
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	resource := strings.TrimSpace(ctx.Request.Header.Get("Sentry-Hook-Resource"))
	if resource == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload struct {
		Action       string              `json:"action"`
		Installation WebhookInstallation `json:"installation"`
		Data         map[string]any      `json:"data"`
		Actor        map[string]any      `json:"actor"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("failed to decode sentry webhook: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err == nil {
		if metadata.InstallationID != "" && payload.Installation.UUID != "" && metadata.InstallationID != payload.Installation.UUID {
			ctx.Response.WriteHeader(http.StatusOK)
			return
		}
	}

	message := WebhookMessage{
		Resource:     resource,
		Action:       payload.Action,
		Installation: payload.Installation,
		Data:         payload.Data,
		Actor:        payload.Actor,
	}

	if resource == "installation" && payload.Action == "deleted" {
		if err := s.handleInstallationDeleted(ctx, config, message); err != nil {
			ctx.Logger.Errorf("failed to handle sentry uninstall webhook: %v", err)
			ctx.Response.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	if err := s.dispatchWebhookMessage(ctx, message); err != nil {
		ctx.Logger.Errorf("failed to dispatch sentry webhook: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (s *Sentry) handleInstallationDeleted(ctx core.HTTPRequestContext, config Configuration, message WebhookMessage) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	metadata.InstallationID = ""
	metadata.Organization = nil
	metadata.Projects = nil
	metadata.Teams = nil
	ctx.Integration.SetMetadata(metadata)

	if err := ctx.Integration.SetSecret(OAuthAccessTokenSecret, []byte("")); err != nil {
		return fmt.Errorf("failed to clear sentry access token: %w", err)
	}

	if err := ctx.Integration.SetSecret(OAuthRefreshTokenSecret, []byte("")); err != nil {
		return fmt.Errorf("failed to clear sentry refresh token: %w", err)
	}

	if config.AppSlug != "" && config.ClientID != "" && config.ClientSecret != "" {
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appConnectDescription,
			URL:         fmt.Sprintf("%s/sentry-apps/%s/external-install/", config.BaseURL, url.PathEscape(config.AppSlug)),
			Method:      http.MethodGet,
		})
	}

	ctx.Integration.Error("Sentry app installation was removed. Reconnect the integration.")
	return nil
}

func (s *Sentry) dispatchWebhookMessage(ctx core.HTTPRequestContext, message WebhookMessage) error {
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to list sentry subscriptions: %w", err)
	}

	for _, subscription := range subscriptions {
		config := SubscriptionConfiguration{}
		if err := mapstructure.Decode(subscription.Configuration(), &config); err != nil {
			ctx.Logger.Warnf("failed to decode sentry subscription config: %v", err)
			continue
		}

		if len(config.Resources) > 0 && !slices.Contains(config.Resources, message.Resource) {
			continue
		}

		if err := subscription.SendMessage(message); err != nil {
			ctx.Logger.Errorf("failed to send sentry message to subscription: %v", err)
		}
	}

	return nil
}

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	switch resourceType {
	case ResourceTypeProject:
		resources := make([]core.IntegrationResource, 0, len(metadata.Projects))
		for _, project := range metadata.Projects {
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeProject,
				ID:   project.Slug,
				Name: displayName(project.Name, project.Slug),
			})
		}
		return resources, nil

	case ResourceTypeTeam:
		resources := make([]core.IntegrationResource, 0, len(metadata.Teams))
		for _, team := range metadata.Teams {
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTeam,
				ID:   team.Slug,
				Name: displayName(team.Name, team.Slug),
			})
		}
		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

func (s *Sentry) updateMetadata(ctx core.SyncContext, config ...Configuration) error {
	var cfg Configuration
	if len(config) > 0 {
		cfg = config[0]
	} else if decoded, err := s.loadConfiguration(ctx.Integration); err == nil {
		cfg = decoded
	} else {
		return err
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Organization == nil || metadata.Organization.Slug == "" {
		return fmt.Errorf("Sentry organization is not connected yet")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	organization, err := client.GetOrganization()
	if err != nil {
		return fmt.Errorf("failed to retrieve organization: %w", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	metadata.AppSlug = firstNonEmpty(metadata.AppSlug, cfg.AppSlug)
	metadata.Organization = &OrganizationSummary{
		ID:   organization.ID,
		Slug: organization.Slug,
		Name: organization.Name,
	}
	metadata.Projects = projects
	metadata.Teams = teams

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (s *Sentry) loadConfiguration(integration core.IntegrationContext) (Configuration, error) {
	baseURL, err := integration.GetConfig("baseUrl")
	if err != nil {
		return Configuration{}, err
	}

	appSlug, err := integration.GetConfig("appSlug")
	if err != nil {
		return Configuration{}, err
	}

	clientID, err := integration.GetConfig("clientId")
	if err != nil {
		return Configuration{}, err
	}

	clientSecret, err := integration.GetConfig("clientSecret")
	if err != nil {
		return Configuration{}, err
	}

	return Configuration{
		BaseURL:      normalizeBaseURL(string(baseURL)),
		AppSlug:      string(appSlug),
		ClientID:     string(clientID),
		ClientSecret: string(clientSecret),
	}, nil
}

func normalizeBaseURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return DefaultBaseURL
	}

	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}

	return strings.TrimSuffix(raw, "/")
}

func callbackURL(ctx core.SyncContext) string {
	return fmt.Sprintf("%s/api/v1/integrations/%s/callback", publicBaseURL(ctx.BaseURL, ctx.WebhooksBaseURL), ctx.Integration.ID().String())
}

func eventsURL(ctx core.SyncContext) string {
	return fmt.Sprintf("%s/api/v1/integrations/%s/events", publicBaseURL(ctx.BaseURL, ctx.WebhooksBaseURL), ctx.Integration.ID().String())
}

func publicBaseURL(baseURL, webhooksBaseURL string) string {
	if webhooksBaseURL != "" {
		return strings.TrimSuffix(webhooksBaseURL, "/")
	}

	return strings.TrimSuffix(baseURL, "/")
}

func integrationSettingsURL(ctx core.HTTPRequestContext) string {
	return fmt.Sprintf("%s/%s/settings/integrations/%s", strings.TrimSuffix(ctx.BaseURL, "/"), ctx.OrganizationID, ctx.Integration.ID().String())
}

func currentInstallationID(integration core.IntegrationContext) string {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return ""
	}
	return metadata.InstallationID
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

func storeTokenSecrets(integration core.IntegrationContext, tokenResponse *TokenResponse) error {
	if tokenResponse.AccessToken != "" {
		if err := integration.SetSecret(OAuthAccessTokenSecret, []byte(tokenResponse.AccessToken)); err != nil {
			return fmt.Errorf("failed to save sentry access token: %w", err)
		}
	}

	if tokenResponse.RefreshToken != "" {
		if err := integration.SetSecret(OAuthRefreshTokenSecret, []byte(tokenResponse.RefreshToken)); err != nil {
			return fmt.Errorf("failed to save sentry refresh token: %w", err)
		}
	}

	return nil
}

func verifyWebhookSignature(signature string, body, secret []byte) error {
	signature = strings.TrimSpace(signature)
	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected))) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func displayName(name, slug string) string {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)

	switch {
	case name == "":
		return slug
	case slug == "" || name == slug:
		return name
	default:
		return fmt.Sprintf("%s (%s)", name, slug)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
