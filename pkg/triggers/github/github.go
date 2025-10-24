package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
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
	Integration string   `json:"integration"`
	Repository  string   `json:"repository"`
	Events      []string `json:"events"`
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

func (g *GitHub) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "GitHub integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "github",
				},
			},
		},
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     components.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []components.VisibilityCondition{
				{
					Field:  "integration",
					Values: []string{"*"},
				},
			},
			TypeOptions: &components.TypeOptions{
				Resource: &components.ResourceTypeOptions{
					Type: "repository",
				},
			},
		},
		{
			Name:     "events",
			Label:    "Events",
			Type:     components.FieldTypeMultiSelect,
			Required: true,
			VisibilityConditions: []components.VisibilityCondition{
				{
					Field:  "repository",
					Values: []string{"*"},
				},
			},
			TypeOptions: &components.TypeOptions{
				MultiSelect: &components.MultiSelectTypeOptions{
					Options: []components.FieldOption{
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

func (g *GitHub) Setup(ctx triggers.TriggerContext) error {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
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

	integration, err := ctx.IntegrationContext.GetIntegration(config.Integration)
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

	err = ctx.WebhookContext.Setup(&triggers.WebhookSetupOptions{
		IntegrationID: &integrationID,
		Resource:      resource,
		Configuration: config,
	})

	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	ctx.MetadataContext.Set(Metadata{
		Repository: &Repository{
			ID:   resource.Id(),
			Name: resource.Name(),
			URL:  resource.URL(),
		},
	})

	return nil
}

func (g *GitHub) Actions() []components.Action {
	return []components.Action{}
}

func (g *GitHub) HandleAction(ctx triggers.TriggerActionContext) error {
	return nil
}

func (g *GitHub) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusOK, nil
	}

	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// If event is not in the list of chosen events, ignore it.
	//
	if !slices.ContainsFunc(config.Events, func(event string) bool {
		return event == eventType
	}) {
		return http.StatusOK, nil
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.WebhookContext.GetSecret()
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

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
