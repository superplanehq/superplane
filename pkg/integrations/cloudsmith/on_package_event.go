package cloudsmith

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPackageEvent struct{}

type OnPackageEventSpec struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	Events     []string `json:"events" mapstructure:"events"`
}

type OnPackageEventMetadata struct {
	Repository  string `json:"repository"`
	WebhookSlug string `json:"webhookSlug"`
	WebhookURL  string `json:"webhookUrl"`
}

type PackageEventPayload struct {
	Meta struct {
		EventID string `json:"event_id"`
	} `json:"meta"`
	Data map[string]any `json:"data"`
}

func (p *OnPackageEvent) Name() string {
	return "cloudsmith.onPackageEvent"
}

func (p *OnPackageEvent) Label() string {
	return "On Package Event"
}

func (p *OnPackageEvent) Description() string {
	return "Triggers when a package event occurs in a Cloudsmith repository"
}

func (p *OnPackageEvent) Documentation() string {
	return `The On Package Event trigger starts a workflow execution when a package event occurs in a Cloudsmith repository.

## Use Cases

- **Build pipelines**: Trigger downstream workflows when a new package is synchronized
- **Release workflows**: Automate promotion or notification on package publish
- **Security automation**: React to quarantined or failed packages

## Configuration

- **Repository**: Cloudsmith repository in the format ` + "`namespace/repo`" + `
- **Events**: Package events to listen for (e.g. ` + "`package.synced`" + `, ` + "`package.deleted`" + `)

## Webhook Setup

This trigger creates a webhook in Cloudsmith automatically when the canvas is saved.`
}

func (p *OnPackageEvent) Icon() string {
	return "package"
}

func (p *OnPackageEvent) Color() string {
	return "gray"
}

func (p *OnPackageEvent) ExampleData() map[string]any {
	return onPackageEventExampleData()
}

func (p *OnPackageEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeRepository,
					Multi: false,
				},
			},
		},
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"package.synced"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Package Synced", Value: "package.synced"},
						{Label: "Package Deleted", Value: "package.deleted"},
						{Label: "Package Quarantined", Value: "package.quarantined"},
						{Label: "Package Failed", Value: "package.failed"},
					},
				},
			},
		},
	}
}

func (p *OnPackageEvent) Setup(ctx core.TriggerContext) error {
	metadata := OnPackageEventMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	spec := OnPackageEventSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repository := strings.TrimSpace(spec.Repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in the format of namespace/repo")
	}

	namespace := parts[0]
	repoSlug := parts[1]

	if metadata.Repository == repository && metadata.WebhookSlug != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	events := spec.Events
	if len(events) == 0 {
		events = []string{"package.synced"}
	}

	slugPerm, err := client.CreateWebhook(namespace, repoSlug, webhookURL, events)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return ctx.Metadata.Set(OnPackageEventMetadata{
		Repository:  repository,
		WebhookSlug: slugPerm,
		WebhookURL:  webhookURL,
	})
}

func (p *OnPackageEvent) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPackageEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPackageEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	spec := OnPackageEventSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var payload PackageEventPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	eventID := payload.Meta.EventID

	if len(spec.Events) > 0 {
		matched := false
		for _, e := range spec.Events {
			if e == eventID {
				matched = true
				break
			}
		}

		if !matched {
			ctx.Logger.Infof("Ignoring event type %s", eventID)
			return http.StatusOK, nil
		}
	}

	// Emit a normalized event: include event type alongside package data so the
	// frontend mapper can display it without knowing the raw Cloudsmith envelope.
	eventData := map[string]any{
		"event":   eventID,
		"package": payload.Data,
	}

	if err := ctx.Events.Emit("cloudsmith.package.event", eventData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil
}

func (p *OnPackageEvent) Cleanup(ctx core.TriggerContext) error {
	metadata := OnPackageEventMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.WebhookSlug == "" {
		return nil
	}

	parts := strings.Split(metadata.Repository, "/")
	if len(parts) != 2 {
		return nil
	}

	namespace := parts[0]
	repoSlug := parts[1]

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return client.DeleteWebhook(namespace, repoSlug, metadata.WebhookSlug)
}
