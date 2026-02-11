// Package buildkite implements the Buildkite integration for SuperPlane.
package buildkite

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
	Organization string `json:"organization"`
	APIToken     string `json:"apiToken"`
	WebhookToken string `json:"webhookToken"`
}

type Metadata struct {
	Organizations []string         `json:"organizations"`
	Webhook       *WebhookMetadata `json:"webhook,omitempty"`
	SetupComplete bool             `json:"setupComplete"`
	OrgSlug       string           `json:"orgSlug"`
}

func extractOrgSlug(orgInput string) (string, error) {
	if orgInput == "" {
		return "", fmt.Errorf("organization input is empty")
	}

	urlPattern := regexp.MustCompile(`(?:https?://)?(?:www\.)?buildkite\.com/(?:organizations/)?([^/]+)`)
	if matches := urlPattern.FindStringSubmatch(strings.TrimSpace(orgInput)); len(matches) > 1 {
		return matches[1], nil
	}

	// Just the slug (validate it looks like a valid org slug)
	slugPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*[a-zA-Z0-9]$`)
	if slugPattern.MatchString(strings.TrimSpace(orgInput)) {
		return strings.TrimSpace(orgInput), nil
	}

	return "", fmt.Errorf("invalid organization format: %s. Expected format: https://buildkite.com/my-org or just 'my-org'", orgInput)
}

func (b *Buildkite) createWebhookSetupAction(ctx core.SyncContext, orgSlug string) {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: fmt.Sprintf(
			"Click to setup webhook.\n\n**Webhook URL**:\n`%s/api/v1/integrations/%s/webhook`\n**Events**: `build.finished`\n**Pipelines**: All Pipelines",
			ctx.WebhooksBaseURL,
			ctx.Integration.ID(),
		),
		URL:    fmt.Sprintf("https://buildkite.com/organizations/%s/services/webhook/new", orgSlug),
		Method: "GET",
	})
}

func (b *Buildkite) createTokenSetupAction(ctx core.SyncContext, orgSlug string) {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: "Generate API token for triggering builds. Required permissions: `read_organizations`, `read_user`, `read_pipelines`, `read_builds`, `write_builds`.",
		URL:         "https://buildkite.com/user/api-access-tokens",
		Method:      "GET",
	})
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
	return "To create new Buildkite API key, open [Personal Settings > API Access Tokens](https://buildkite.com/user/api-access-tokens/new)."
}

func (b *Buildkite) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization URL",
			Type:        configuration.FieldTypeString,
			Description: "Buildkite organization URL (e.g. https://buildkite.com/my-org or just my-org)",
			Placeholder: "e.g. https://buildkite.com/my-org or my-org",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Buildkite API token with permissions: read_organizations, read_user, read_pipelines, read_builds, write_builds",
			Placeholder: "e.g. bkua_...",
			Required:    true,
		},
		{
			Name:        "webhookToken",
			Label:       "Webhook Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Buildkite webhook token (provided when you create webhook in Buildkite)",
			Placeholder: "e.g. c86b6bbbc...",
			Required:    true,
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
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	if config.Organization == "" {
		return fmt.Errorf("Organization is required")
	}

	orgSlug, err := extractOrgSlug(config.Organization)
	if err != nil {
		return fmt.Errorf("Invalid organization format: %v", err)
	}

	// Prompt user to create API token
	if config.APIToken == "" {
		b.createTokenSetupAction(ctx, orgSlug)
		return nil
	}

	// Prompt user to create webhook
	if config.WebhookToken == "" {
		b.createWebhookSetupAction(ctx, orgSlug)
		return nil
	}

	// Update metadata to track setup completion
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		metadata = Metadata{}
	}

	metadata.OrgSlug = orgSlug
	metadata.SetupComplete = true
	ctx.Integration.SetMetadata(metadata)

	ctx.Integration.RemoveBrowserAction()
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
	config := Configuration{}
	err := mapstructure.Decode(ctx.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.WebhookToken == "" {
		// Return empty webhook metadata if no token configured yet
		// The UI will prompt user to configure webhook token
		return WebhookMetadata{
			URL:             ctx.GetURL(),
			Token:           "",
			VerifySignature: false,
			Active:          false,
			Events:          []string{"build.finished"},
			Pipelines:       []string{"*"},
		}, nil
	}

	return WebhookMetadata{
		URL:             ctx.GetURL(),
		Token:           config.WebhookToken,
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
