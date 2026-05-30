package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("plugin", &Plugin{})
}

type Plugin struct{}

type PluginConfiguration struct {
	ServerURL string `json:"serverUrl" mapstructure:"serverUrl"`
	AuthToken string `json:"authToken" mapstructure:"authToken"`
}

type PluginMetadata struct {
	ManifestName  string `json:"manifestName" mapstructure:"manifestName"`
	ManifestLabel string `json:"manifestLabel" mapstructure:"manifestLabel"`
}

func (p *Plugin) Name() string {
	return "plugin"
}

func (p *Plugin) Label() string {
	return "Plugin"
}

func (p *Plugin) Icon() string {
	return "puzzle"
}

func (p *Plugin) Description() string {
	return "Connect to custom plugin servers built with the SuperPlane Plugin SDK"
}

func (p *Plugin) Instructions() string {
	return `### Setup

1. Build a plugin server using the SuperPlane Plugin SDK
2. Deploy your plugin server so SuperPlane can reach it
3. Enter the server URL below
4. Optionally add an auth token if your server requires authentication

The plugin server exposes a manifest at ` + "`GET /manifest`" + ` describing available actions and their configuration fields.`
}

func (p *Plugin) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serverUrl",
			Label:       "Server URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "URL of the plugin server (e.g. https://my-plugin.example.com)",
		},
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Optional bearer token for authenticating with the plugin server",
		},
	}
}

func (p *Plugin) Actions() []core.Action {
	return []core.Action{
		&RunAction{},
	}
}

func (p *Plugin) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnEvent{},
	}
}

func (p *Plugin) Sync(ctx core.SyncContext) error {
	config := PluginConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}

	client := &Client{
		serverURL: config.ServerURL,
		authToken: config.AuthToken,
		httpDo:    ctx.HTTP.Do,
	}

	manifest, err := client.FetchManifest()
	if err != nil {
		return fmt.Errorf("failed to connect to plugin server: %w", err)
	}

	if manifest.Name == "" {
		return fmt.Errorf("plugin manifest is missing 'name' field")
	}

	setCachedManifest(manifest)

	ctx.Integration.SetMetadata(PluginMetadata{
		ManifestName:  manifest.Name,
		ManifestLabel: manifest.Label,
	})

	ctx.Integration.Ready()
	return nil
}

func (p *Plugin) Cleanup(ctx core.IntegrationCleanupContext) error {
	setCachedManifest(nil)
	return nil
}

func (p *Plugin) HandleRequest(ctx core.HTTPRequestContext) {
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
		if subType != "plugin_event" {
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

func (p *Plugin) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "action" {
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

	resources := make([]core.IntegrationResource, 0, len(manifest.Actions))
	for _, action := range manifest.Actions {
		resources = append(resources, core.IntegrationResource{
			Type: "action",
			ID:   action.Name,
			Name: action.Label,
		})
	}

	return resources, nil
}

func (p *Plugin) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *Plugin) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
