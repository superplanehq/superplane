package dockerhub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Docker Hub to work with SuperPlane:

1. **Get Access Token**: In Docker Hub, go to Account Settings → Security → New Access Token
2. **Set Scopes**: Choose appropriate scopes (Read, Write, Delete as needed for your use case)
3. **Enter Credentials**: Provide your Docker Hub username and access token in the integration configuration
`

func init() {
	registry.RegisterIntegration("dockerhub", &DockerHub{})
}

type DockerHub struct{}

type Configuration struct {
	Username    string `json:"username"`
	AccessToken string `json:"accessToken"`
}

type Metadata struct {
	Username string `json:"username" mapstructure:"username"`
}

func (d *DockerHub) Name() string {
	return "dockerhub"
}

func (d *DockerHub) Label() string {
	return "Docker Hub"
}

func (d *DockerHub) Icon() string {
	return "docker"
}

func (d *DockerHub) Description() string {
	return "Interact with Docker Hub container registry"
}

func (d *DockerHub) Instructions() string {
	return installationInstructions
}

func (d *DockerHub) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your Docker Hub username",
		},
		{
			Name:        "accessToken",
			Label:       "Access Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Docker Hub access token for authentication",
		},
	}
}

func (d *DockerHub) Components() []core.Component {
	return []core.Component{
		&ListTags{},
	}
}

func (d *DockerHub) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnImagePushed{},
	}
}

func (d *DockerHub) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *DockerHub) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.Username == "" {
		return fmt.Errorf("username is required")
	}

	if config.AccessToken == "" {
		return fmt.Errorf("accessToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		Username: config.Username,
	})

	ctx.Integration.Ready()
	return nil
}

func (d *DockerHub) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/webhook") {
		d.handleWebhook(ctx)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (d *DockerHub) handleWebhook(ctx core.HTTPRequestContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("Error reading request body: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var payload WebhookPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		ctx.Logger.Errorf("Error parsing webhook payload: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx.Logger.Infof("Received Docker Hub webhook for repository: %s", payload.Repository.RepoName)
	ctx.Response.WriteHeader(http.StatusOK)
}

func (d *DockerHub) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	// Docker Hub webhooks are manually configured by users in Docker Hub UI
	return nil
}

func (d *DockerHub) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Repository == configB.Repository, nil
}

func (d *DockerHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (d *DockerHub) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	// Docker Hub webhooks are manually configured by users in Docker Hub UI
	// The webhook URL is provided to the user to configure in Docker Hub
	return nil, nil
}

func (d *DockerHub) Actions() []core.Action {
	return []core.Action{}
}

func (d *DockerHub) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

// WebhookConfiguration defines the configuration for Docker Hub webhooks
type WebhookConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

// WebhookPayload represents the payload sent by Docker Hub webhooks
type WebhookPayload struct {
	CallbackURL string         `json:"callback_url"`
	PushData    PushData       `json:"push_data"`
	Repository  RepositoryInfo `json:"repository"`
}

type PushData struct {
	Images   []string `json:"images"`
	PushedAt int64    `json:"pushed_at"`
	Pusher   string   `json:"pusher"`
	Tag      string   `json:"tag"`
}

type RepositoryInfo struct {
	CommentCount    int    `json:"comment_count"`
	DateCreated     int64  `json:"date_created"`
	Description     string `json:"description"`
	Dockerfile      string `json:"dockerfile"`
	FullDescription string `json:"full_description"`
	IsOfficial      bool   `json:"is_official"`
	IsPrivate       bool   `json:"is_private"`
	IsTrusted       bool   `json:"is_trusted"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Owner           string `json:"owner"`
	RepoName        string `json:"repo_name"`
	RepoURL         string `json:"repo_url"`
	StarCount       int    `json:"star_count"`
	Status          string `json:"status"`
}
