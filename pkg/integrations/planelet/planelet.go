package planelet

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

func init() {
	registry.RegisterIntegration("planelet", &Planelet{})
}

type Planelet struct{}

type PlaneletConfiguration struct {
	ServerURL string `json:"serverUrl" mapstructure:"serverUrl"`
	AuthToken string `json:"authToken" mapstructure:"authToken"`
}

type PlaneletMetadata struct {
	ManifestID      string `json:"manifestId" mapstructure:"manifestId"`
	ManifestLabel   string `json:"manifestLabel" mapstructure:"manifestLabel"`
	ManifestIcon    string `json:"manifestIcon,omitempty" mapstructure:"manifestIcon"`
	ManifestIconURL string `json:"manifestIconUrl,omitempty" mapstructure:"manifestIconUrl"`
}

func (p *Planelet) Name() string {
	return "planelet"
}

func (p *Planelet) Label() string {
	return "Planelets"
}

func (p *Planelet) Icon() string {
	return "puzzle"
}

func (p *Planelet) Description() string {
	return "Connect to custom Planelet servers built with the SuperPlane Planelet SDK"
}

func (p *Planelet) Instructions() string {
	return `### Setup

1. Build a Planelet server using the SuperPlane Planelet SDK
2. Deploy your Planelet server so SuperPlane can reach it
3. Enter the server URL below
4. Optionally add an auth token if your server requires authentication

The Planelet server exposes a manifest at ` + "`GET /manifest`" + ` describing available actions, triggers, and their configuration parameters.`
}

func (p *Planelet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serverUrl",
			Label:       "Server URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "URL of the Planelet server (e.g. https://my-planelet.example.com)",
		},
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Optional bearer token for authenticating with the Planelet server",
		},
	}
}

func (p *Planelet) Actions() []core.Action {
	return []core.Action{
		&RunAction{},
	}
}

func (p *Planelet) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnEvent{},
		&WebhookTrigger{},
	}
}

func (p *Planelet) Sync(ctx core.SyncContext) error {
	config := PlaneletConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}

	client := &Client{
		serverURL: strings.TrimRight(config.ServerURL, "/"),
		authToken: config.AuthToken,
		httpDo:    ctx.HTTP.Do,
	}

	manifest, err := client.FetchManifest()
	if err != nil {
		return fmt.Errorf("failed to connect to Planelet server: %w", err)
	}

	if manifest.ID == "" {
		return fmt.Errorf("Planelet manifest is missing 'id' field")
	}

	setCachedManifest(manifest)

	ctx.Integration.SetMetadata(PlaneletMetadata{
		ManifestID:      manifest.ID,
		ManifestLabel:   manifest.Label,
		ManifestIcon:    manifest.Icon,
		ManifestIconURL: manifest.IconURL,
	})

	ctx.Integration.Ready()
	return nil
}

func (p *Planelet) Cleanup(ctx core.IntegrationCleanupContext) error {
	setCachedManifest(nil)
	return nil
}

func (p *Planelet) HandleRequest(ctx core.HTTPRequestContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("failed to read request body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("failed to parse event payload: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	eventType, _ := payload["eventType"].(string)

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("failed to list subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, sub := range subscriptions {
		config, ok := sub.Configuration().(map[string]any)
		if !ok {
			continue
		}

		subType, _ := config["type"].(string)
		if subType != "planelet_event" {
			continue
		}

		filterType, _ := config["eventType"].(string)
		if filterType != "" && filterType != eventType {
			continue
		}

		if err := sub.SendMessage(payload); err != nil {
			ctx.Logger.Errorf("failed to send message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (p *Planelet) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "action" && resourceType != "trigger" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	manifest, err := client.FetchManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	setCachedManifest(manifest)

	switch resourceType {
	case "action":
		resources := make([]core.IntegrationResource, 0, len(manifest.Actions))
		for _, action := range manifest.Actions {
			resources = append(resources, core.IntegrationResource{
				Type: "action",
				ID:   action.ID,
				Name: action.Label,
			})
		}
		return resources, nil
	case "trigger":
		resources := make([]core.IntegrationResource, 0, len(manifest.Triggers))
		for _, trigger := range manifest.Triggers {
			resources = append(resources, core.IntegrationResource{
				Type: "trigger",
				ID:   trigger.ID,
				Name: trigger.Label,
			})
		}
		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

func (p *Planelet) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *Planelet) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
