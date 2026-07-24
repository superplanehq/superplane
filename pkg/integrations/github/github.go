package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/actions"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/admin"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/checks"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/contents"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/deployments"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/issues"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/metadata"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/statuses"
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

var defaultGitHubAppEvents = []string{
	"create",
	"issue_comment",
	"issues",
	"pull_request",
	"pull_request_review",
	"pull_request_review_comment",
	"push",
	"release",
	"status",
	"check_run",
	"workflow_run",
}

func init() {
	registry.RegisterIntegrationWithOptions("github", &GitHub{}, registry.IntegrationRegistrationOptions{
		WebhookHandler: &GitHubWebhookHandler{},
		SetupProvider:  &SetupProvider{},
	})
}

type GitHub struct {
}

type Configuration struct {
	Organization string `mapstructure:"organization" json:"organization"`
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

func (g *GitHub) Actions() []core.Action {
	return []core.Action{
		&admin.GetWorkflowUsage{},
		&checks.ListCheckRunsForRef{},
		&actions.RunWorkflow{},
		&contents.CreateRelease{},
		&contents.GetRelease{},
		&contents.UpdateRelease{},
		&contents.DeleteRelease{},
		&issues.AddIssueAssignee{},
		&issues.AddIssueLabel{},
		&issues.CreateIssue{},
		&issues.CreateIssueComment{},
		&issues.UpdateIssueComment{},
		&issues.GetIssue{},
		&issues.RemoveIssueLabel{},
		&issues.RemoveIssueAssignee{},
		&issues.UpdateIssue{},
		&metadata.GetRepositoryPermission{},
		&pulls.CreateReview{},
		&pulls.CreatePullRequest{},
		&pulls.MergePullRequest{},
		&pulls.MarkPullRequestReadyForReview{},
		&pulls.AddPullRequestReviewers{},
		&pulls.UpdatePullRequest{},
		&pulls.AddReaction{},
		&statuses.GetCombinedCommitStatus{},
		&statuses.PublishCommitStatus{},
		&deployments.CreateDeployment{},
		&deployments.CreateDeploymentStatus{},
	}
}

func (g *GitHub) Triggers() []core.Trigger {
	return []core.Trigger{
		&actions.OnWorkflowRun{},
		&contents.OnPush{},
		&contents.OnRelease{},
		&contents.OnTagCreated{},
		&contents.OnBranchCreated{},
		&issues.OnIssue{},
		&issues.OnIssueComment{},
		&pulls.OnPullRequest{},
		&pulls.OnPRComment{},
		&pulls.OnPRReviewComment{},
		&checks.OnCheckRun{},
		&statuses.OnCommitStatus{},
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

	metadata := common.Metadata{}
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

	ctx.Integration.SetMetadata(common.Metadata{
		Owner: config.Organization,
		State: state,
	})

	return nil
}

func (g *GitHub) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/redirect") {
		g.afterAppCreation(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/setup") {
		g.afterAppInstallation(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/webhook") {
		g.handleWebhook(ctx)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (g *GitHub) findWebhookSecret(ctx core.HTTPRequestContext) (string, error) {
	if ctx.Integration.LegacySetup() {
		return common.FindSecret(ctx.Integration, GitHubAppWebhookSecret)
	}

	return ctx.Integration.Secrets().Get(common.SecretAppWebhookSecret)
}

func (g *GitHub) handleWebhook(ctx core.HTTPRequestContext) {
	webhookSecret, err := g.findWebhookSecret(ctx)
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
		g.handleInstallationEvent(ctx, event)

	//
	// We don't actually use the repositories_added and repositories_removed fields in the events.
	// When we receive an installation_repositories event, we always reload the list of repositories using the API.
	//
	case *github.InstallationRepositoriesEvent:
		g.handleInstallationRepositoriesEvent(ctx)

	default:
		ctx.Logger.Warnf("ignoring eventType %s", eventType)
	}
}

func (g *GitHub) findInstallationID(ctx core.HTTPRequestContext) (string, error) {
	if ctx.Integration.LegacySetup() {
		metadata := common.Metadata{}
		err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
		if err != nil {
			return "", fmt.Errorf("failed to decode metadata: %v", err)
		}

		return metadata.InstallationID, nil
	}

	return ctx.Integration.Properties().GetString(common.PropertyAppInstallationID)
}

func (g *GitHub) handleInstallationEvent(ctx core.HTTPRequestContext, event *github.InstallationEvent) {
	installationID, err := g.findInstallationID(ctx)
	if err != nil {
		ctx.Logger.Errorf("failed to find installation ID: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	switch *event.Action {

	//
	// This is handled by the setup_url, so no need to do anything here.
	//
	case "created":
		ctx.Logger.Infof("installation %s created", installationID)
		return

	case "suspend":
		ctx.Logger.Infof("installation %s suspended", installationID)
		ctx.Integration.Error("app installation was suspended")
		return

	case "unsuspend":
		ctx.Logger.Infof("installation %s unsuspended", installationID)
		ctx.Integration.Ready()
		return

	//
	// When the app installation is deleted,
	// we need to prompt the user to re-install it.
	//
	case "deleted":
		g.handleInstallationDeletion(ctx, installationID)
		return

	default:
		ctx.Logger.Warnf("ignoring action: %s", *event.Action)
	}
}

func (g *GitHub) handleInstallationDeletion(ctx core.HTTPRequestContext, installationID string) {
	ctx.Logger.Infof("installation %s deleted", installationID)

	state, err := crypto.Base64String(32)
	if err != nil {
		ctx.Logger.Errorf("Failed to generate GitHub App state: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	//
	// Move the integration to error state
	//
	ctx.Integration.Error("App was uninstalled")

	//
	// If we are dealing with a legacy integration,
	// we need to update metadata and browser action.
	//
	if ctx.Integration.LegacySetup() {
		metadata := common.Metadata{}
		err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
		if err != nil {
			return
		}

		metadata.InstallationID = ""
		metadata.Repositories = []common.Repository{}
		metadata.State = state

		ctx.Integration.SetMetadata(metadata)
		ctx.Integration.NewBrowserAction(core.BrowserAction{
			Description: appInstallationDescription,
			URL:         fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", metadata.GitHubApp.Slug, state),
			Method:      "GET",
		})

		return
	}

	//
	// If we are dealing with an integration using the new setup,
	// we gotta cleanup the properties and update the integration setup state.
	//
	err = ctx.Integration.Properties().Delete(common.PropertyAppInstallationID, common.PropertyAppInstallationURL)
	if err != nil {
		ctx.Logger.Errorf("failed to delete properties: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	instructions, err := template.New("appInstallInstructions").Parse(string(appInstallInstructionsTemplate))
	if err != nil {
		ctx.Logger.Errorf("failed to parse template: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	appSlug, err := ctx.Integration.Properties().GetString(common.PropertyAppSlug)
	if err != nil {
		ctx.Logger.Errorf("failed to get app slug: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	owner, err := ctx.Integration.Properties().GetString(common.PropertyOwner)
	if err != nil {
		ctx.Logger.Errorf("failed to get owner: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Owner":   owner,
		"AppSlug": appSlug,
	}

	var buf bytes.Buffer
	if err := instructions.Execute(&buf, data); err != nil {
		ctx.Logger.Errorf("failed to execute template: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.Properties().Create(core.IntegrationPropertyDefinition{
		Name:  common.PropertyAppState,
		Label: "GitHub App State",
		Type:  core.IntegrationPropertyTypeString,
		Value: state,
	})

	if err != nil {
		ctx.Logger.Errorf("failed to create app state property: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.IntegrationSetup.SetStep(core.SetupStep{
		Type:         core.SetupStepTypeRedirectPrompt,
		Name:         SetupStepInstallApp,
		Label:        "Install the GitHub app",
		Instructions: buf.String(),
		RedirectPrompt: &core.RedirectPrompt{
			URL:    fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", appSlug, state),
			Method: "GET",
		},
	})

	if err != nil {
		ctx.Logger.Errorf("failed to set setup step: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (g *GitHub) handleInstallationRepositoriesEvent(ctx core.HTTPRequestContext) {
	//
	// Integrations from new setup flow do not store repositories in metadata,
	// so this is a no-op for them.
	//
	if !ctx.Integration.LegacySetup() {
		return
	}

	metadata := common.Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		ctx.Logger.Errorf("failed to decode metadata: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	client, err := newClientForAppInstallation(ctx.Integration, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		ctx.Logger.Errorf("failed to create client: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	repos, err := listInstallationRepositories(context.Background(), client)
	if err != nil {
		ctx.Logger.Errorf("failed to list repos: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	ctx.Logger.Infof("Updated repositories: %v", repos)

	metadata.Repositories = repos
	ctx.Integration.SetMetadata(metadata)
}

func (g *GitHub) afterAppCreation(ctx core.HTTPRequestContext) {
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

	if ctx.Integration.LegacySetup() {
		g.afterAppCreationLegacy(ctx, appData, state)
		return
	}

	//
	// Save app properties
	//
	appURL, err := common.AppURL(ctx.Integration.Properties(), appData.Slug)
	if err != nil {
		ctx.Logger.Errorf("failed to get app URL: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.Properties().CreateMany([]core.IntegrationPropertyDefinition{
		{
			Type:     core.IntegrationPropertyTypeString,
			Name:     common.PropertyAppID,
			Label:    "GitHub App ID",
			Value:    fmt.Sprintf("%d", appData.ID),
			Editable: false,
		},
		{
			Type:     core.IntegrationPropertyTypeString,
			Name:     common.PropertyAppSlug,
			Label:    "GitHub App Slug",
			Value:    appData.Slug,
			Editable: false,
		},
		{
			Type:     core.IntegrationPropertyTypeString,
			Name:     common.PropertyAppURL,
			Label:    "GitHub App URL",
			Value:    appURL,
			Editable: false,
		},
		{
			Type:     core.IntegrationPropertyTypeString,
			Name:     common.PropertyAppClientID,
			Label:    "GitHub App Client ID",
			Value:    appData.ClientID,
			Editable: false,
		},
	})

	if err != nil {
		ctx.Logger.Errorf("failed to save GitHub App properties: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	//
	// Save app secrets
	//
	err = ctx.Integration.Secrets().CreateMany([]core.IntegrationSecretDefinition{
		{
			Name:     common.SecretAppClientSecret,
			Label:    "GitHub App Client Secret",
			Value:    appData.ClientSecret,
			Editable: false,
		},
		{
			Name:     common.SecretAppWebhookSecret,
			Label:    "GitHub App Webhook Secret",
			Value:    appData.WebhookSecret,
			Editable: false,
		},
		{
			Name:     common.SecretAppPEM,
			Label:    "GitHub App Private Key (PEM)",
			Value:    appData.PEM,
			Editable: false,
		},
	})

	if err != nil {
		ctx.Logger.Errorf("failed to save GitHub App secrets: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	ctx.Logger.Infof("Successfully created GitHub App %s - %d", appData.Slug, appData.ID)

	//
	// Redirect to app installation page
	//
	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"https://github.com/apps/%s/installations/new?state=%s",
			appData.Slug,
			state,
		),
		http.StatusSeeOther,
	)
}

func (g *GitHub) afterAppCreationLegacy(ctx core.HTTPRequestContext, appData *GitHubAppData, state string) {
	metadata := common.Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return
	}

	//
	// Save installation metadata
	//
	metadata.GitHubApp = common.GitHubAppMetadata{
		ID:       appData.ID,
		Slug:     appData.Slug,
		ClientID: appData.ClientID,
	}

	ctx.Integration.SetMetadata(metadata)

	//
	// Save installation secrets
	//
	err = ctx.Integration.SetSecret(common.GitHubAppClientSecret, []byte(appData.ClientSecret))
	if err != nil {
		ctx.Logger.Errorf("failed to save client secret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.SetSecret(common.GitHubAppWebhookSecret, []byte(appData.WebhookSecret))
	if err != nil {
		ctx.Logger.Errorf("failed to save webhook secret: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.SetSecret(common.GitHubAppPEM, []byte(appData.PEM))
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

func (g *GitHub) afterAppInstallation(ctx core.HTTPRequestContext) {
	if ctx.Integration.LegacySetup() {
		ctx.Logger.Infof("handling app installation for legacy integration")
		g.afterAppInstallationLegacy(ctx)
		return
	}

	ctx.Logger.Infof("handling app installation for non-legacy integration")

	//
	// App installation has already been set up.
	// Just redirect to the SuperPlane app installation page.
	//
	installationID, err := ctx.Integration.Properties().GetString(common.PropertyAppInstallationID)
	if err == nil && installationID != "" {
		ctx.Logger.Infof("app installation %s already set up", installationID)
		http.Redirect(
			ctx.Response,
			ctx.Request,
			fmt.Sprintf(
				"%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String(),
			),
			http.StatusSeeOther,
		)
		return
	}

	state, err := ctx.Integration.Properties().GetString(common.PropertyAppState)
	if err != nil {
		ctx.Logger.Errorf("failed to get app state: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	installationID = ctx.Request.URL.Query().Get("installation_id")
	setupAction := ctx.Request.URL.Query().Get("setup_action")
	requestState := ctx.Request.URL.Query().Get("state")
	if installationID == "" || requestState != state {
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

	installationURL, err := common.AppInstallationURL(ctx.Integration.Properties(), installationID)
	if err != nil {
		ctx.Logger.Errorf("failed to get app installation URL: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.Properties().CreateMany([]core.IntegrationPropertyDefinition{
		{
			Name:     common.PropertyAppInstallationID,
			Label:    "GitHub App Installation ID",
			Type:     core.IntegrationPropertyTypeString,
			Value:    installationID,
			Editable: false,
		},
		{
			Name:     common.PropertyAppInstallationURL,
			Label:    "GitHub App Installation URL",
			Type:     core.IntegrationPropertyTypeString,
			Value:    installationURL,
			Editable: false,
		},
	})

	if err != nil {
		ctx.Logger.Errorf("failed to save installation ID: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	appID, err := ctx.Integration.Properties().GetString(common.PropertyAppID)
	if err != nil {
		ctx.Logger.Errorf("failed to get app ID: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	appIDNumber, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		ctx.Logger.Errorf("failed to parse app ID: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	client, err := newClientForAppInstallation(ctx.Integration, appIDNumber, installationID)
	if err != nil {
		ctx.Logger.Errorf("failed to create client: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	repos, err := listInstallationRepositories(context.Background(), client)
	if err != nil {
		ctx.Logger.Errorf("failed to list repos: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	err = ctx.Integration.Properties().Delete(common.PropertyAppState)
	if err != nil {
		ctx.Logger.Errorf("failed to delete app state: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)
	ctx.Integration.Ready()

	ctx.Logger.Infof("Successfully installed GitHub App - installation=%s", installationID)
	ctx.Logger.Infof("Repositories: %v", repos)

	http.Redirect(
		ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID().String(),
		),
		http.StatusSeeOther,
	)
}

func (g *GitHub) afterAppInstallationLegacy(ctx core.HTTPRequestContext) {
	metadata := common.Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return
	}

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
		return
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
	client, err := newClientForAppInstallation(ctx.Integration, metadata.GitHubApp.ID, installationID)
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

	repos, err := listInstallationRepositories(context.Background(), client)
	if err != nil {
		ctx.Logger.Errorf("failed to list repos: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
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
		"name":           `SuperPlane GH integration`,
		"public":         false,
		"url":            "https://superplane.com",
		"default_events": defaultGitHubAppEvents,
		"default_permissions": map[string]string{
			"issues":                      "write",
			"actions":                     "write",
			"checks":                      "read",
			"contents":                    "write",
			"pull_requests":               "write",
			"repository_hooks":            "write",
			"statuses":                    "write",
			"deployments":                 "write",
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

func listInstallationRepositories(ctx context.Context, client *github.Client) ([]common.Repository, error) {
	var allRepos []*github.Repository
	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		repos, resp, err := client.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos.Repositories...)
		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	out := make([]common.Repository, 0, len(allRepos))
	for _, r := range allRepos {
		if r == nil || r.ID == nil {
			continue
		}
		out = append(out, common.Repository{
			ID:   *r.ID,
			Name: r.GetName(),
			URL:  r.GetHTMLURL(),
		})
	}

	return out, nil
}

func (g *GitHub) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GitHub) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func newClientForAppInstallation(ctx core.IntegrationContext, appID int64, installationID string) (*github.Client, error) {
	installationNumber, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := findAppPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		http.DefaultTransport,
		appID,
		installationNumber,
		[]byte(pem),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create apps transport: %v", err)
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func findAppPrivateKey(ctx core.IntegrationContext) (string, error) {
	if ctx.LegacySetup() {
		return common.FindSecret(ctx, common.GitHubAppPEM)
	}

	return ctx.Secrets().Get(common.SecretAppPEM)
}

func (g *GitHub) ResolveSecrets(ctx core.IntegrationSecretContext) (map[string][]byte, error) {
	token, err := resolveAccessToken(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		integrationSecretGitHubToken: []byte(token),
	}, nil
}
