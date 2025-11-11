package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

func init() {
	registry.RegisterIntegration("github", &GitHub{})
}

type GitHub struct {
}

type Configuration struct {
	Organization string `json:"organization"`
}

type Metadata struct {
	InstallationID string        `json:"installationId"`
	State          string        `json:"state"`
	Organization   string        `json:"organization"`
	Repositories   []string      `json:"repositories"`
	AppData        GitHubAppData `json:"appData"`
}

func (g *GitHub) Name() string {
	return "github"
}

func (g *GitHub) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name: "organization",
			Type: configuration.FieldTypeString,
		},
	}
}

func (g *GitHub) Components() []components.Component {
	return []components.Component{}
}

func (g *GitHub) Triggers() []triggers.Trigger {
	return []triggers.Trigger{}
}

func (g *GitHub) Sync(ctx applications.SyncContext) applications.SyncResponse {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return applications.SyncResponse{
			NewState:         "error",
			StateDescription: fmt.Sprintf("Failed to decode configuration: %v", err),
		}
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.AppContext.GetMetadata(), &metadata)
	if err != nil {
		return applications.SyncResponse{
			NewState:         "error",
			StateDescription: fmt.Sprintf("Failed to decode metadata: %v", err),
		}
	}

	if metadata.InstallationID != "" {
		return applications.SyncResponse{
			NewState: "ready",
			Signals: []applications.Signal{
				{Name: "repositories", Data: metadata.Repositories},
			},
		}
	}

	state, err := crypto.Base64String(32)
	if err != nil {
		return applications.SyncResponse{
			NewState:         "error",
			StateDescription: fmt.Sprintf("Failed to generate state: %v", err),
		}
	}

	ctx.AppContext.NewBrowserAction(applications.BrowserAction{
		URL:    browserActionURL(config.Organization),
		Method: "POST",
		FormFields: map[string]string{
			"manifest": getGitHubAppManifest(ctx),
			"state":    state,
		},
	})

	ctx.AppContext.SetMetadata(Metadata{
		Organization: config.Organization,
		State:        state,
	})

	return applications.SyncResponse{
		NewState: "in_progress",
	}
}

func (g *GitHub) HandleRequest(ctx applications.HttpRequestContext) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.AppContext.GetMetadata(), &metadata)
	if err != nil {
		return
	}

	logrus.Infof("metadata: %v", metadata)

	if strings.HasSuffix(ctx.Request.URL.Path, "/redirect") {
		afterAppCreation(ctx, metadata)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/setup") {
		afterAppInstallation(ctx, metadata)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/webhook") {
		logrus.Infof("webhook")
		// TODO: verify signature
		// TODO: decode payload
		// TODO: find components/triggers using this integration
		// TODO: call component/trigger
		return
	}

	logrus.Infof("unknown path: %s", ctx.Request.URL.Path)
}

func afterAppCreation(ctx applications.HttpRequestContext, metadata Metadata) {
	code := ctx.Request.URL.Query().Get("code")
	state := ctx.Request.URL.Query().Get("state")

	logrus.Infof("after app creation - code: %s, state: %s", code, state)

	if code == "" || state == "" {
		return
	}

	appData, err := createAppFromManifest(code)
	if err != nil {
		logrus.Errorf("failed to create app from manifest: %v", err)
		return
	}

	logrus.Infof("after app creation - appData: %v", appData)

	metadata.AppData = *appData
	ctx.AppContext.SetMetadata(metadata)

	http.Redirect(
		*ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"https://github.com/apps/%s/installations/new?state=%s",
			metadata.AppData.Slug,
			state,
		),
		http.StatusSeeOther,
	)
}

func afterAppInstallation(ctx applications.HttpRequestContext, metadata Metadata) {
	installationID := ctx.Request.URL.Query().Get("installation_id")
	setupAction := ctx.Request.URL.Query().Get("setup_action")
	state := ctx.Request.URL.Query().Get("state")

	logrus.Infof("after app installation - installationID: %s, setupAction: %s, state: %s", installationID, setupAction, state)

	if installationID == "" || state != metadata.State {
		logrus.Errorf("invalid installation ID or state")
		return
	}

	switch setupAction {
	case "install":
		afterAppInstallationInstall(ctx, installationID, metadata)

	case "update":
		// TODO
	}
}

func afterAppInstallationInstall(ctx applications.HttpRequestContext, installationID string, metadata Metadata) {
	logrus.Infof("after app installation install - installationID: %s", installationID)
	logrus.Infof("after app installation install - metadata: %v", metadata)

	ID, err := strconv.Atoi(installationID)
	if err != nil {
		logrus.Errorf("failed to parse installation ID: %v", err)
		return
	}

	f, err := os.CreateTemp("", "github-app.pem")
	if err != nil {
		logrus.Errorf("failed to create temp file: %v", err)
		return
	}

	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(metadata.AppData.PEM))
	if err != nil {
		logrus.Errorf("failed to write temp file: %v", err)
		return
	}

	itr, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		metadata.AppData.ID,
		int64(ID),
		f.Name(),
	)

	if err != nil {
		logrus.Errorf("failed to create apps transport: %v", err)
		return
	}

	client := github.NewClient(&http.Client{Transport: itr})
	response, _, err := client.Apps.ListRepos(context.Background(), &github.ListOptions{})
	if err != nil {
		logrus.Errorf("failed to list repos: %v", err)
		return
	}

	logrus.Infof("after app installation install - response: %v", response)

	repos := []string{}
	for _, r := range response.Repositories {
		repos = append(repos, r.GetFullName())
	}

	metadata.InstallationID = installationID
	metadata.Repositories = repos

	ctx.AppContext.SetMetadata(metadata)
	ctx.AppContext.RemoveBrowserAction()
	ctx.AppContext.SetState("ready")

	http.Redirect(
		*ctx.Response,
		ctx.Request,
		fmt.Sprintf(
			"%s/%s/settings/applications/%s", ctx.BaseURL, ctx.OrganizationID, ctx.InstallationID,
		),
		http.StatusSeeOther,
	)
}

func browserActionURL(organization string) string {
	if organization != "" {
		return fmt.Sprintf("https://github.com/organizations/%s/settings/apps/new", organization)
	}

	return "https://github.com/settings/apps/new"
}

func getGitHubAppManifest(ctx applications.SyncContext) string {
	manifest := map[string]any{
		"callback_urls": []string{
			fmt.Sprintf("%s/%s/settings/applications/%s", ctx.BaseURL, ctx.OrganizationID, ctx.InstallationID),
		},
		"default_events": []string{
			"issues",
			"workflow_run",
			"pull_request",
			"push",
			"issue_comment",
		},
		"default_permissions": map[string]string{
			"issues":        "write",
			"actions":       "write",
			"contents":      "write",
			"pull_requests": "write",
		},
		"hook_attributes": map[string]string{
			"url": fmt.Sprintf(`%s/api/v1/apps/%s/webhook`, ctx.BaseURL, ctx.InstallationID),
		},
		"setup_url":    fmt.Sprintf(`%s/api/v1/apps/%s/setup`, ctx.BaseURL, ctx.InstallationID),
		"name":         `Superplane GH integration`,
		"public":       false,
		"redirect_url": fmt.Sprintf(`%s/api/v1/apps/%s/redirect`, ctx.BaseURL, ctx.InstallationID),
		"url":          "https://superplane.com",
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return ""
	}

	return string(data)
}

type GitHubAppData struct {
	ClientID      string `mapstructure:"client_id" json:"client_id"`
	ClientSecret  string `mapstructure:"client_secret" json:"client_secret"`
	WebhookSecret string `mapstructure:"webhook_secret" json:"webhook_secret"`
	ID            int64  `mapstructure:"id" json:"id"`
	Slug          string `mapstructure:"slug" json:"slug"`
	PEM           string `mapstructure:"pem" json:"pem"`
}

func createAppFromManifest(code string) (*GitHubAppData, error) {
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

	logrus.Infof("createAppFromManifest - body: %s", string(body))

	var appData GitHubAppData
	err = json.Unmarshal(body, &appData)
	if err != nil {
		return nil, err
	}

	return &appData, nil
}
