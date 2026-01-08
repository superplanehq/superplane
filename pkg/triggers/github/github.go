package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const MaxEventSize = 64 * 1024

func init() {
	registry.RegisterTrigger("github", &GitHub{})
}

type GitHub struct{}

type Metadata struct {
	Repository *Repository `json:"repository"`
}

type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Configuration struct {
	Integration string `json:"integration"`
	Repository  string `json:"repository"`
	EventType   string `json:"eventType"`
}

func (g *GitHub) Name() string {
	return "github"
}

func (g *GitHub) Label() string {
	return "GitHub"
}

func (g *GitHub) Description() string {
	return "Start a new execution chain when something happens in your GitHub repository"
}

func (g *GitHub) Icon() string {
	return "github"
}

func (g *GitHub) Color() string {
	return "gray"
}

func (g *GitHub) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "integration",
			Label:    "GitHub integration",
			Type:     configuration.FieldTypeIntegration,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Integration: &configuration.IntegrationTypeOptions{
					Type: "github",
				},
			},
		},
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "integration",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "repository",
				},
			},
		},
		{
			Name:     "eventType",
			Label:    "Event Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "repository",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Value: "push",
							Label: "Push",
						},
						{
							Value: "pull_request",
							Label: "Pull Request",
						},
					},
				},
			},
		},
	}
}

func (g *GitHub) Setup(ctx core.TriggerContext) error {
	var metadata Metadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	//
	// If metadata is set, it means the trigger was already setup
	//
	if metadata.Repository != nil {
		return nil
	}

	config := Configuration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Integration == "" {
		return fmt.Errorf("integration is required")
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	integration, err := ctx.Integration.GetIntegration(config.Integration)
	if err != nil {
		return fmt.Errorf("failed to get integration: %w", err)
	}

	resource, err := integration.Get("repository", config.Repository)
	if err != nil {
		return fmt.Errorf("failed to find repository %s: %w", config.Repository, err)
	}

	integrationID, err := uuid.Parse(config.Integration)
	if err != nil {
		return fmt.Errorf("integration ID is invalid: %w", err)
	}

	_, err = ctx.Webhook.Setup(&core.WebhookSetupOptions{
		IntegrationID: &integrationID,
		Resource:      resource,
		Configuration: config,
	})

	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	ctx.Metadata.Set(Metadata{
		Repository: &Repository{
			ID:   resource.Id(),
			Name: resource.Name(),
			URL:  resource.URL(),
		},
	})

	return nil
}

func (g *GitHub) Actions() []core.Action {
	return []core.Action{}
}

func (g *GitHub) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (g *GitHub) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// If event is not in the list of chosen events, ignore it.
	//
	if config.EventType != eventType {
		return http.StatusOK, nil
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// If the event is a push event for branch deletion, ignore it.
	//
	if isBranchDeletionEvent(eventType, data) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("github", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func isBranchDeletionEvent(eventType string, data map[string]any) bool {
	if eventType != "push" {
		return false
	}

	v, ok := data["deleted"]
	if !ok {
		return false
	}

	deleted, ok := v.(bool)
	if !ok {
		return false
	}

	return deleted
}
