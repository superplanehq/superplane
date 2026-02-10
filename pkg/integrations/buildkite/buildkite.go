// Package buildkite implements the Buildkite integration for SuperPlane.
package buildkite

import (
	"encoding/json"
	"errors"
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
	registry.RegisterIntegration("buildkite", &Buildkite{})
}

type Buildkite struct{}

type Configuration struct {
	APIToken     string `json:"apiToken"`
	WebhookToken string `json:"webhookToken"`
}

type Metadata struct {
	Organizations []string         `json:"organizations"`
	Webhook       *WebhookMetadata `json:"webhook,omitempty"`
}

func (b *Buildkite) Name() string {
	return "buildkite"
}

func (b *Buildkite) Label() string {
	return "Buildkite"
}

func (b *Buildkite) Icon() string {
	return "workflow"
}

func (b *Buildkite) Description() string {
	return "Trigger and react to your Buildkite builds"
}

func (b *Buildkite) Instructions() string {
	return `
In order to connect Buildkite to Superplane, you need to create a webhook.

1. In Buildkite, go to **Settings → Notification Services**
2. Select **Add Webhook**
3. Configure the webhook with these settings:
   - **Description**: SuperPlane Integration
   - **Webhook URL**: Your SuperPlane webhook URL
   - **Events**: Select "build.finished"
   - **Pipelines**: Select "All Pipelines"
   - **Branch filtering**: Leave empty to receive all branches

Webhook URL is constructed in following way - https://**<Backend URL>**/api/v1/integrations/**<Integration ID>**/webhook.`
}

func (b *Buildkite) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Buildkite API token",
			Placeholder: "e.g. bkpi_...",
			Required:    true,
		},
		{
			Name:        "webhookToken",
			Label:       "Webhook Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Buildkite webhook token for verifying incoming webhook requests",
			Placeholder: "e.g. c86b6bbbc...",
			Required:    false,
		},
	}
}

func (b *Buildkite) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (b *Buildkite) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error validating API token: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (b *Buildkite) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/webhook") {
		b.handleWebhook(ctx)
		return
	}

	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (b *Buildkite) handleWebhook(ctx core.HTTPRequestContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secret, err := ctx.Integration.GetConfig("webhookToken")
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(secret) == 0 {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}
	if err := VerifyWebhook(ctx.Request.Header, body, secret); err != nil {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	eventType := ctx.Request.Header.Get("X-Buildkite-Event")
	if eventType == "" {
		if event, ok := payload["event"].(string); ok {
			eventType = event
		}
	}

	if eventType == "build.finished" {
		if err := b.routeToTriggerSubscriptions(ctx, payload); err != nil {
			ctx.Logger.WithError(err).Warn("some subscriptions failed to receive build.finished event")
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (b *Buildkite) routeToTriggerSubscriptions(ctx core.HTTPRequestContext, payload map[string]any) error {
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		return err
	}

	var sendErrors []error
	for _, subscription := range subscriptions {
		if !b.subscriptionApplies(ctx, subscription, "build.finished", payload) {
			continue
		}

		if err := subscription.SendMessage(payload); err != nil {
			sendErrors = append(sendErrors, err)
			ctx.Logger.WithError(err).Error("failed to forward build.finished webhook to subscription")
		}
	}

	if len(sendErrors) > 0 {
		return errors.Join(sendErrors...)
	}

	return nil
}

func (b *Buildkite) subscriptionApplies(ctx core.HTTPRequestContext, subscription core.IntegrationSubscriptionContext, eventType string, payload map[string]any) bool {
	var config BuildkiteSubscriptionConfiguration
	if err := mapstructure.Decode(subscription.Configuration(), &config); err != nil {
		ctx.Logger.Errorf("Failed to decode subscription configuration: %v", err)
		return false
	}

	if eventType != "build.finished" {
		return false
	}

	if config.Organization != "" && config.Organization != "*" {
		if org, ok := payload["organization"].(map[string]any); ok {
			if orgSlug, ok := org["slug"].(string); ok {
				if config.Organization != orgSlug {
					return false
				}
			}
		}
	}

	if config.Pipeline != "" && config.Pipeline != "*" {
		if pipeline, ok := payload["pipeline"].(map[string]any); ok {
			if pipelineSlug, ok := pipeline["slug"].(string); ok {
				if config.Pipeline != pipelineSlug {
					return false
				}
			}
		}
	}

	if config.Branch != "" {
		if build, ok := payload["build"].(map[string]any); ok {
			if branch, ok := build["branch"].(string); ok {
				if config.Branch != branch {
					return false
				}
			}
		}
	}

	return true
}

func (b *Buildkite) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	switch resourceType {
	case "organization":
		orgs, err := client.ListOrganizations()
		if err != nil {
			return nil, fmt.Errorf("error listing organizations: %v", err)
		}

		resources := make([]core.IntegrationResource, len(orgs))
		for i, org := range orgs {
			resources[i] = core.IntegrationResource{
				Type: "organization",
				ID:   org.Slug,
				Name: org.Name,
			}
		}
		return resources, nil

	case "pipeline":
		orgSlug := ctx.Parameters["organization"]
		if orgSlug == "" {
			return []core.IntegrationResource{}, nil
		}

		pipelines, err := client.ListPipelines(orgSlug)
		if err != nil {
			return nil, fmt.Errorf("error listing pipelines: %v", err)
		}

		resources := make([]core.IntegrationResource, len(pipelines))
		for i, pipeline := range pipelines {
			resources[i] = core.IntegrationResource{
				Type: "pipeline",
				ID:   pipeline.Slug,
				Name: pipeline.Name,
			}
		}
		return resources, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (b *Buildkite) Actions() []core.Action {
	return []core.Action{}
}

func (b *Buildkite) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

type WebhookMetadata struct {
	URL             string   `json:"url"`
	Token           string   `json:"token"`
	VerifySignature bool     `json:"verifySignature"`
	Active          bool     `json:"active"`
	Events          []string `json:"events"`
	Pipelines       []string `json:"pipelines"`
}

func (b *Buildkite) SetupWebhook(ctx core.WebhookContext) (any, error) {
	secret, err := ctx.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}
	return WebhookMetadata{
		URL:             ctx.GetURL(),
		Token:           string(secret),
		VerifySignature: true,
		Active:          true,
		Events:          []string{"build.finished"},
		Pipelines:       []string{"*"},
	}, nil
}

func (b *Buildkite) Components() []core.Component {
	return []core.Component{
		&TriggerBuild{},
	}
}

func (b *Buildkite) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnBuildFinished{},
	}
}
