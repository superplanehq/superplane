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
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	GitHubAppPEM           = "pem"
	GitHubAppClientSecret  = "clientSecret"
	GitHubAppWebhookSecret = "webhookSecret"

	appBootstrapDescription = `
To complete the GitHub app setup:

1. The "**Continue**" button/link will take you to GitHub with the app manifest pre-filled.
2. **Create GitHub App**: Give the new app a name, and click the "Create" button.
3. **Install GitHub App**: Install the new GitHub app in the user/organization.
`

	appInstallationDescription = `
To complete the GitHub app setup:
1. **Install GitHub App**: Install the new GitHub app in the user/organization.
`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("github", &GitHub{}, &GitHubWebhookHandler{})
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

func (g *GitHub) Instructions() string {
	return ""
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
	return []core.Component{
		&GetIssue{},
		&CreateIssue{},
		&CreateIssueComment{},
		&UpdateIssue{},
		&CreateReview{},
		&RunWorkflow{},
		&PublishCommitStatus{},
		&CreateRelease{},
		&GetRelease{},
		&UpdateRelease{},
		&DeleteRelease{},
		&GetWorkflowUsage{},
	}
}

func (g *GitHub) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPush{},
		&OnPullRequest{},
		&OnPRComment{},
		&OnIssue{},
		&OnIssueComment{},
		&OnRelease{},
		&OnTagCreated{},
		&OnBranchCreated{},
		&OnWorkflowRun{},
	}
}

func (g *GitHub) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (g *GitHub) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
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

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: appBootstrapDescription,
		URL:         g.browserActionURL(config.Organization),
		Method:      "POST",
		FormFields: map[string]string{
			"manifest": g.appManifest(ctx),
			"state":    state,
		},
	})

	ctx.Integration.SetMetadata(Metadata{
		Owner: config.Organization,
		State: state,
	})

	return nil
}

func (g *GitHub) HandleRequest(ctx core.HTTPRequestContext) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
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

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (g *GitHub) handleWebhook(ctx core.HTTPRequestContext, metadata Metadata) {
	webhookSecret, err := findSecret(ctx.Integration, GitHubAppWebhookSecret)
	if err != nil {
		ctx.Logger.Errorf("Error finding webhook secret: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	payload, err := github.ValidatePayload(ctx.Request, []byte(webhookSecret))
	if err != nil {
		ctx.Logger.Errorf("Error validating webhook payload: %v", err)
		http.Error(ctx.Response, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	eventType := github.WebHookType(ctx.Request)
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		ctx.Logger.Errorf("Error parsing webhook payload: %v", err)
		http.Error(ctx.Response, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	switch event := event.(type) {
	case *github.InstallationEvent:
		g.handleInstallationEvent(ctx, metadata, event)

	//
	// We don't actually use the repositories_added and repositories_removed fields in the events.
	// When we receive an installation_repositories event, we always reload the list of repositories using the API.
	//
	case *github.InstallationRepositoriesEvent:
		g.handleInstallationRepositoriesEvent(ctx, metadata)

	default:
		ctx.Logger.Warnf("ignoring eventType %s", eventType)
	}
}

func (g *GitHub) handleInstallationEvent(ctx core.HTTPRequestContext, metadata Metadata, event *github.InstallationEvent) {
	switch *event.Action {

	//
	// This is handled by the setup_url, so no need to do anything here.
	//
	case "created":
		ctx.Logger.Infof("installation %s created", metadata.InstallationID)
		return

	case "suspend":
		ctx.Logger.Infof("installation %s suspended", metadata.InstallationID)
		ctx.Integration.Error("app installation was suspended")
		return

	case "unsuspend":
		ctx.Logger.Infof("installation %s unsuspended", metadata.InstallationID)
		ctx.Integration.Ready()
		return

	//
	// When the app installation is deleted,
	// we need to prompt the user to re-install it.
	//
	case "deleted":
		ctx.Logger.Infof("installation %s deleted", metadata.InstallationID)

		state, err := crypto.Base64String(32)
		if err != nil {
			ctx.Logger.Errorf("Failed to generate GitHub App state: %v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
		}

		metadata.InstallationID = ""
		metadata.Repositories = []Repository{}
		metadata.State = state

		ctx.Integration.SetMetadata(metadata)
		ctx.Integration.Error("error")
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appInstallationDescription,
			URL:         fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", metadata.GitHubApp.Slug, state),
			Method:      "GET",
		})

	default:
		ctx.Logger.Warnf("ignoring action: %s", *event.Action)
	}
}

func (g *GitHub) handleInstallationRepositoriesEvent(ctx core.HTTPRequestContext, metadata Metadata) {
	client, err := NewClient(ctx.Integration, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		ctx.Logger.Errorf("failed to create client: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	response, _, err := client.Apps.ListRepos(context.Background(), &github.ListOptions{})
	if err != nil {
		ctx.Logger.Errorf("failed to list repos: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
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

	ctx.Logger.Infof("Updated repositories: %v", repos)

	metadata.Repositories = repos
	ctx.Integration.SetMetadata(metadata)
}

func (g *GitHub) afterAppCreation(ctx core.HTTPRequestContext, metadata Metadata) {
	code := ctx.Request.URL.Query().Get("code")
	state := ctx.Request.URL.Query().Get("state")

	if code == "" || state == "" {
		ctx.Logger.Errorf("missing code or state")
		http.Error(ctx.Response, "missing code or state", http.StatusBadRequest)
		return
	}

	appData, err := g.createAppFromManifest(ctx.HTTP, code)
	if err != nil {
		ctx.Logger.Errorf("failed to create app from manifest: %v", err)
		http.Error(ctx.Response, "failed to create app from manifest", http.StatusInternalServerError)
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

	ctx.Integration.SetMetadata(metadata)

	//
	// Save installation secrets
	//
	err = ctx.Integration.SetSecret(GitHubAppClientSecret, []byte(appData.ClientSecret))
	if err != nil {
		ctx.Logger.Errorf("failed to save client secret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.SetSecret(GitHubAppWebhookSecret, []byte(appData.WebhookSecret))
	if err != nil {
		ctx.Logger.Errorf("failed to save webhook secret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.SetSecret(GitHubAppPEM, []byte(appData.PEM))
	if err != nil {
		ctx.Logger.Errorf("failed to save PEM: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	ctx.Logger.Infof("Successfully created GitHub App %s - %d", metadata.GitHubApp.Slug, metadata.GitHubApp.ID)

	//
	// Redirect to app installation page
	//
	http.Redirect(
		ctx.Response,
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
	//
	// App installation has already been set up.
	// Just redirect to the SuperPlane app installation page.
	//
	if metadata.InstallationID != "" {
		ctx.Logger.Infof("app installation %s already set up", metadata.InstallationID)
		http.Redirect(
			ctx.Response,
			ctx.Request,
			fmt.Sprintf(
				"%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String(),
			),
			http.StatusSeeOther,
		)
	}

	installationID := ctx.Request.URL.Query().Get("installation_id")
	setupAction := ctx.Request.URL.Query().Get("setup_action")
	state := ctx.Request.URL.Query().Get("state")
	if installationID == "" || state != metadata.State {
		ctx.Logger.Errorf("invalid installation ID or state")
		http.Error(ctx.Response, "invalid installation ID or state", http.StatusBadRequest)
		return
	}

	//
	// Installation updates are handled through the webhook events.
	//
	if setupAction != "install" {
		ctx.Logger.Infof("Ignoring setup action %s for GitHub App installation %s", setupAction, installationID)
		return
	}

	metadata.InstallationID = installationID
	client, err := NewClient(ctx.Integration, metadata.GitHubApp.ID, installationID)
	if err != nil {
		ctx.Logger.Errorf("failed to create client: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if metadata.Owner == "" {
		ghApp, _, err := client.Apps.Get(context.Background(), metadata.GitHubApp.Slug)
		if err != nil {
			ctx.Logger.Errorf("failed to get app: %v", err)
			http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
			return
		}

		metadata.Owner = ghApp.Owner.GetLogin()
	}

	response, _, err := client.Apps.ListRepos(context.Background(), &github.ListOptions{})
	if err != nil {
		ctx.Logger.Errorf("failed to list repos: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
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

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	ctx.Logger.Infof("Successfully installed GitHub App %s - installation=%s", metadata.GitHubApp.Slug, metadata.InstallationID)
	ctx.Logger.Infof("Repositories: %v", metadata.Repositories)

	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String(),
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
		"name":   `SuperPlane GH integration`,
		"public": false,
		"url":    "https://superplane.com",
		"default_permissions": map[string]string{
			"issues":                      "write",
			"actions":                     "write",
			"contents":                    "write",
			"pull_requests":               "write",
			"repository_hooks":            "write",
			"statuses":                    "write",
			"organization_administration": "read",
		},
		"setup_url":    fmt.Sprintf(`%s/api/v1/integrations/%s/setup`, ctx.BaseURL, ctx.Integration.ID().String()),
		"redirect_url": fmt.Sprintf(`%s/api/v1/integrations/%s/redirect`, ctx.BaseURL, ctx.Integration.ID().String()),
		"hook_attributes": map[string]any{
			"url": fmt.Sprintf(`%s/api/v1/integrations/%s/webhook`, ctx.WebhooksBaseURL, ctx.Integration.ID().String()),
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

func (g *GitHub) createAppFromManifest(httpCtx core.HTTPContext, code string) (*GitHubAppData, error) {
	URL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequest(http.MethodPost, URL, nil)
	if err != nil {
		return nil, err
	}

	response, err := httpCtx.Do(req)
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

func (g *GitHub) Actions() []core.Action {
	return []core.Action{}
}

func (g *GitHub) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
