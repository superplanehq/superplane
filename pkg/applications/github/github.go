package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	GitHubAppPEM           = "pem"
	GitHubAppClientSecret  = "clientSecret"
	GitHubAppWebhookSecret = "webhookSecret"

	appInstallationDescription = `
To complete the GitHub app setup:

1. The "**Continue**" button/link will take you to GitHub with the app manifest pre-filled.
2. **Create GitHub App**: Give the new app a name, and click the "Create" button.
3. **Install GitHub App**: Install the new GitHub app in the user/organization.
`
)

func init() {
	registry.RegisterApplication("github", &GitHub{})
}

type GitHub struct {
}

type Configuration struct {
	Organization string `json:"organization"`
}

type Metadata struct {
	InstallationID string            `mapstructure:"installationId" json:"installationId"`
	State          string            `mapstructure:"state" json:"state"`
	Owner          string            `mapstructure:"owner" json:"owner"`
	Repositories   []Repository      `mapstructure:"repositories" json:"repositories"`
	GitHubApp      GitHubAppMetadata `mapstructure:"githubApp" json:"githubApp"`
}

type GitHubAppMetadata struct {
	ID       int64  `mapstructure:"id" json:"id"`
	Slug     string `mapstructure:"slug" json:"slug"`
	ClientID string `mapstructure:"clientId" json:"clientId"`
}

func (g *GitHub) Name() string {
	return "github"
}

func (g *GitHub) Label() string {
	return "GitHub"
}

func (g *GitHub) Icon() string {
	return "github"
}

func (g *GitHub) Description() string {
	return "Manage and react to changes in your GitHub repositories"
}

func (g *GitHub) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Description: "Organization to install the app into. If not specified, the app will be installed into the user's account.",
		},
	}
}

func (g *GitHub) Components() []core.Component {
	return []core.Component{}
}

func (g *GitHub) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPush{},
		&OnPullRequest{},
	}
}

func (g *GitHub) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("Failed to decode metadata: %v", err)
	}

	//
	// App is already installed - do not do anything.
	//
	if metadata.InstallationID != "" {
		return nil
	}

	state, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("Failed to generate GitHub App state: %v", err)
	}

	ctx.AppInstallation.NewBrowserAction(core.BrowserAction{
		Description: appInstallationDescription,
		URL:         g.browserActionURL(config.Organization),
		Method:      "POST",
		FormFields: map[string]string{
			"manifest": g.appManifest(ctx),
			"state":    state,
		},
	})

	ctx.AppInstallation.SetMetadata(Metadata{
		Owner: config.Organization,
		State: state,
	})

	return nil
}

func (g *GitHub) HandleRequest(ctx core.HTTPRequestContext) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/redirect") {
		g.afterAppCreation(ctx, metadata)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/setup") {
		g.afterAppInstallation(ctx, metadata)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/webhook") {
		g.handleWebhook(ctx, metadata)
		return
	}

	logrus.Infof("unknown path: %s", ctx.Request.URL.Path)
}

func (g *GitHub) handleWebhook(ctx core.HTTPRequestContext, metadata Metadata) {
	signature := ctx.Request.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		logrus.Error("Missing X-Hub-Signature-256 header")
		return
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		logrus.Error("Missing signature")
		return
	}

	eventType := ctx.Request.Header.Get("X-GitHub-Event")
	if eventType == "" {
		logrus.Error("missing X-GitHub-Event header")
		return
	}

	if eventType != "installation" && eventType != "installation_repositories" {
		logrus.Infof("ignoring event type: %s", eventType)
		return
	}

	webhookSecret, err := findSecret(ctx.AppInstallation, GitHubAppWebhookSecret)
	if err != nil {
		logrus.Errorf("Error finding webhook secret: %v", err)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		logrus.Errorf("Error reading request body: %v", err)
		return
	}

	err = crypto.VerifySignature([]byte(webhookSecret), body, signature)
	if err != nil {
		logrus.Info("invalid signature")
		return
	}

	data := map[string]any{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		logrus.Errorf("error parsing request body: %v", err)
		return
	}

	action, ok := data["action"]
	if !ok {
		logrus.Error("missing action")
		return
	}

	actionStr, ok := action.(string)
	if !ok {
		logrus.Error("invalid action")
		return
	}

	switch eventType {
	case "installation":
		g.handleInstallationEvent(ctx, metadata, data, actionStr)
	case "installation_repositories":
		g.handleInstallationRepositoriesEvent(ctx, metadata, data, actionStr)
	}
}

func (g *GitHub) handleInstallationEvent(ctx core.HTTPRequestContext, metadata Metadata, data map[string]any, action string) {
	switch action {
	case "created":
		logrus.Infof("installation created: %v", data)
	case "deleted":
		logrus.Infof("installation deleted: %v", data)
	case "suspend":
		logrus.Infof("installation suspended: %v", data)
	case "unsuspend":
		logrus.Infof("installation unsuspended: %v", data)
	default:
		logrus.Infof("ignoring action: %s", action)
	}
}

func (g *GitHub) handleInstallationRepositoriesEvent(ctx core.HTTPRequestContext, metadata Metadata, data map[string]any, action string) {
	client, err := NewClient(ctx.AppInstallation, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		logrus.Errorf("failed to create client: %v", err)
		return
	}

	response, _, err := client.Apps.ListRepos(context.Background(), &github.ListOptions{})
	if err != nil {
		logrus.Errorf("failed to list repos: %v", err)
		return
	}

	repos := []Repository{}
	for _, r := range response.Repositories {
		repos = append(repos, Repository{
			ID:   *r.ID,
			Name: r.GetName(),
			URL:  r.GetHTMLURL(),
		})
	}

	metadata.Repositories = repos
	ctx.AppInstallation.SetMetadata(metadata)
}

type WebhookConfiguration struct {
	EventType  string `json:"eventType"`
	Repository string `json:"repository"`
}

func (g *GitHub) RequestWebhook(ctx core.AppInstallationContext, configuration any) error {
	config := WebhookConfiguration{}
	err := mapstructure.Decode(configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	hooks, err := ctx.ListWebhooks()
	if err != nil {
		return fmt.Errorf("Failed to list webhooks: %v", err)
	}

	for _, hook := range hooks {
		c := WebhookConfiguration{}
		err := mapstructure.Decode(hook.Configuration, &c)
		if err != nil {
			return err
		}

		if c.Repository == config.Repository && c.EventType == config.EventType {
			ctx.AssociateWebhook(hook.ID)
			return nil
		}
	}

	return ctx.CreateWebhook(configuration)
}

type Webhook struct {
	ID          int64  `json:"id"`
	WebhookName string `json:"name"`
}

func (g *GitHub) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.GetMetadata(), &metadata)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		return nil, err
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(options.Configuration, &config)
	if err != nil {
		return nil, err
	}

	hook := &github.Hook{
		Active: github.Ptr(true),
		Events: []string{config.EventType},
		Config: &github.HookConfig{
			URL:         &options.URL,
			Secret:      github.Ptr(string(options.Secret)),
			ContentType: github.Ptr("json"),
		},
	}

	createdHook, _, err := client.Repositories.CreateHook(context.Background(), metadata.Owner, config.Repository, hook)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return &Webhook{ID: createdHook.GetID(), WebhookName: *createdHook.Name}, nil
}

func (g *GitHub) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.GetMetadata(), &metadata)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		return err
	}

	webhook := Webhook{}
	err = mapstructure.Decode(options.Metadata, &webhook)
	if err != nil {
		return err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(options.Configuration, &configuration)
	if err != nil {
		return err
	}

	_, err = client.Repositories.DeleteHook(context.Background(), metadata.Owner, configuration.Repository, webhook.ID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}

func (g *GitHub) afterAppCreation(ctx core.HTTPRequestContext, metadata Metadata) {
	code := ctx.Request.URL.Query().Get("code")
	state := ctx.Request.URL.Query().Get("state")

	if code == "" || state == "" {
		return
	}

	appData, err := g.createAppFromManifest(code)
	if err != nil {
		logrus.Errorf("failed to create app from manifest: %v", err)
		return
	}

	//
	// Save installation metadata
	//
	metadata.GitHubApp = GitHubAppMetadata{
		ID:       appData.ID,
		Slug:     appData.Slug,
		ClientID: appData.ClientID,
	}

	ctx.AppInstallation.SetMetadata(metadata)

	//
	// Save installation secrets
	//
	err = ctx.AppInstallation.SetSecret(GitHubAppClientSecret, []byte(appData.ClientSecret))
	if err != nil {
		logrus.Errorf("failed to save client secret: %v", err)
		return
	}

	err = ctx.AppInstallation.SetSecret(GitHubAppWebhookSecret, []byte(appData.WebhookSecret))
	if err != nil {
		logrus.Errorf("failed to save webhook secret: %v", err)
		return
	}

	err = ctx.AppInstallation.SetSecret(GitHubAppPEM, []byte(appData.PEM))
	if err != nil {
		logrus.Errorf("failed to save PEM: %v", err)
		return
	}

	//
	// Redirect to app installation page
	//
	http.Redirect(
		*ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"https://github.com/apps/%s/installations/new?state=%s",
			metadata.GitHubApp.Slug,
			state,
		),
		http.StatusSeeOther,
	)
}

func (g *GitHub) afterAppInstallation(ctx core.HTTPRequestContext, metadata Metadata) {
	installationID := ctx.Request.URL.Query().Get("installation_id")
	setupAction := ctx.Request.URL.Query().Get("setup_action")
	state := ctx.Request.URL.Query().Get("state")

	if installationID == "" || state != metadata.State {
		return
	}

	//
	// Installation updates are handled through the webhook events.
	//
	if setupAction != "install" {
		return
	}

	metadata.InstallationID = installationID
	client, err := NewClient(ctx.AppInstallation, metadata.GitHubApp.ID, installationID)
	if err != nil {
		logrus.Errorf("failed to create client: %v", err)
		return
	}

	if metadata.Owner == "" {
		ghApp, _, err := client.Apps.Get(context.Background(), metadata.GitHubApp.Slug)
		if err != nil {
			logrus.Errorf("failed to get app: %v", err)
			return
		}

		metadata.Owner = ghApp.Owner.GetLogin()
	}

	response, _, err := client.Apps.ListRepos(context.Background(), &github.ListOptions{})
	if err != nil {
		logrus.Errorf("failed to list repos: %v", err)
		return
	}

	repos := []Repository{}
	for _, r := range response.Repositories {
		repos = append(repos, Repository{
			ID:   *r.ID,
			Name: r.GetName(),
			URL:  r.GetHTMLURL(),
		})
	}

	metadata.Repositories = repos
	metadata.State = ""

	ctx.AppInstallation.SetMetadata(metadata)
	ctx.AppInstallation.RemoveBrowserAction()
	ctx.AppInstallation.SetState("ready")

	http.Redirect(
		*ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"%s/%s/settings/applications/%s", ctx.BaseURL, ctx.OrganizationID, ctx.AppInstallation.ID().String(),
		),
		http.StatusSeeOther,
	)
}

func (g *GitHub) browserActionURL(organization string) string {
	if organization != "" {
		return fmt.Sprintf("https://github.com/organizations/%s/settings/apps/new", organization)
	}

	return "https://github.com/settings/apps/new"
}

func (g *GitHub) appManifest(ctx core.SyncContext) string {
	manifest := map[string]any{
		"name":   `Superplane GH integration`,
		"public": false,
		"url":    "https://superplane.com",
		"default_permissions": map[string]string{
			"issues":           "write",
			"actions":          "write",
			"contents":         "write",
			"pull_requests":    "write",
			"repository_hooks": "write",
		},
		"setup_url":    fmt.Sprintf(`%s/api/v1/apps/%s/setup`, ctx.BaseURL, ctx.InstallationID),
		"redirect_url": fmt.Sprintf(`%s/api/v1/apps/%s/redirect`, ctx.BaseURL, ctx.InstallationID),
		"hook_attributes": map[string]any{
			"url": fmt.Sprintf(`%s/api/v1/apps/%s/webhook`, ctx.BaseURL, ctx.InstallationID),
		},
		"callback_urls": []string{
			fmt.Sprintf("%s/%s/settings/applications/%s", ctx.BaseURL, ctx.OrganizationID, ctx.InstallationID),
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return ""
	}

	return string(data)
}

/*
 * This is the response GitHub sends back after the GitHub app is created.
 * NOTE: this contains sensitive data, so we should not save this as part
 * of the installation metadata.
 */
type GitHubAppData struct {
	ID            int64  `mapstructure:"id" json:"id"`
	Slug          string `mapstructure:"slug" json:"slug"`
	ClientID      string `mapstructure:"client_id" json:"client_id"`
	ClientSecret  string `mapstructure:"client_secret" json:"client_secret"`
	WebhookSecret string `mapstructure:"webhook_secret" json:"webhook_secret"`
	PEM           string `mapstructure:"pem" json:"pem"`
}

func (g *GitHub) createAppFromManifest(code string) (*GitHubAppData, error) {
	URL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequest(http.MethodPost, URL, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var appData GitHubAppData
	err = json.Unmarshal(body, &appData)
	if err != nil {
		return nil, err
	}

	return &appData, nil
}
