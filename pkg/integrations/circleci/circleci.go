package circleci

import (
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("circleci", &CircleCI{}, &CircleCIWebhookHandler{})
}

type CircleCI struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	Projects []string `json:"projects"`
}

func (c *CircleCI) Name() string {
	return "circleci"
}

func (c *CircleCI) Label() string {
	return "CircleCI"
}

func (c *CircleCI) Icon() string {
	return "workflow"
}

func (c *CircleCI) Description() string {
	return "Trigger and monitor CircleCI pipelines"
}

func (c *CircleCI) Instructions() string {
	return "Create a Personal API Token in CircleCI → User Settings → Personal API Tokens"
}

func (c *CircleCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "CircleCI Personal API Token",
			Placeholder: "Your CircleCI API token",
			Required:    true,
		},
	}
}

func (c *CircleCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *CircleCI) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Verify the API token by getting current user info
	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying API token: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (c *CircleCI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

type WebhookConfiguration struct {
	ProjectSlug string   `json:"projectSlug"`
	Events      []string `json:"events"`
}

var (
	defaultEvents = []string{"workflow-completed"}
	allowedEvents = map[string]struct{}{
		"workflow-completed": {},
		"job-completed":      {},
	}
)

func normalizeEvents(events []string) ([]string, error) {
	if len(events) == 0 {
		return defaultEvents, nil
	}

	unique := make([]string, 0, len(events))
	seen := map[string]struct{}{}

	for _, event := range events {
		if _, ok := allowedEvents[event]; !ok {
			return nil, fmt.Errorf("unsupported CircleCI event type: %s", event)
		}
		if _, exists := seen[event]; exists {
			continue
		}
		seen[event] = struct{}{}
		unique = append(unique, event)
	}

	return unique, nil
}

func (c *CircleCI) Actions() []core.Action {
	return []core.Action{}
}

func (c *CircleCI) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (c *CircleCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
	Name      string `json:"name"`
}

type CircleCIWebhookHandler struct{}

func (h *CircleCIWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	normalizedA, err := normalizeEvents(configA.Events)
	if err != nil {
		return false, err
	}

	normalizedB, err := normalizeEvents(configB.Events)
	if err != nil {
		return false, err
	}

	if configA.ProjectSlug != configB.ProjectSlug {
		return false, nil
	}

	for _, eventB := range normalizedB {
		if !slices.Contains(normalizedA, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *CircleCIWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	normalizedEvents, err := normalizeEvents(configuration.Events)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook events: %w", err)
	}

	hash := sha256.New()
	hash.Write([]byte(ctx.Webhook.GetID()))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%s", suffix[:16])

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	webhook, err := client.CreateWebhook(
		name,
		ctx.Webhook.GetURL(),
		string(webhookSecret),
		configuration.ProjectSlug,
		normalizedEvents,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating CircleCI webhook: %v", err)
	}

	return WebhookMetadata{
		WebhookID: webhook.ID,
		Name:      webhook.Name,
	}, nil
}

func (h *CircleCIWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.DeleteWebhook(metadata.WebhookID)
}

func (c *CircleCI) Components() []core.Component {
	return []core.Component{
		&RunPipeline{},
	}
}

func (c *CircleCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPipelineCompleted{},
	}
}
