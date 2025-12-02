package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

const MaxEventSize = 64 * 1024
const FilterTypeExactMatch = "exact-match"
const FilterTypeRegex = "regex"

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

type RefFilter struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Configuration struct {
	Integration string       `json:"integration"`
	Repository  string       `json:"repository"`
	EventType   string       `json:"eventType"`
	Refs        []*RefFilter `json:"refs,omitempty"`
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
		{
			Name:     "refs",
			Label:    "Refs",
			Type:     configuration.FieldTypeList,
			Required: false,
			Default: []map[string]any{
				{
					"type":  FilterTypeExactMatch,
					"value": "refs/heads/main",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "eventType",
					Values: []string{"push"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Match Type",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{
												Value: FilterTypeExactMatch,
												Label: "Exact Match",
											},
											{
												Value: FilterTypeRegex,
												Label: "Regex",
											},
										},
									},
								},
							},
							{
								Name:     "value",
								Label:    "Ref Pattern",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
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

	//
	// Validate ref filters
	//
	if err := validateRefFilters(config.Refs); err != nil {
		return err
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
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// If event is not the chosen one, ignore it.
	//
	if config.EventType != eventType {
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

	switch eventType {
	case "push":
		return g.handlePushEvent(ctx, data, config)

	case "pull_request":
		return g.handlePullRequestEvent(ctx, data)

	default:
		return http.StatusOK, nil
	}
}

func (g *GitHub) handlePushEvent(ctx triggers.WebhookRequestContext, data map[string]any, config Configuration) (int, error) {
	//
	// If the event is a push event for ref deletion, ignore it.
	//
	if isRefDeletionEvent(data) {
		return http.StatusOK, nil
	}

	//
	// CHeck if the branch matches the filter.
	//
	branch := getRef(data)
	if branch == "" {
		return http.StatusBadRequest, fmt.Errorf("failed to extract branch from push event")
	}

	allowed, err := isRefAllowed(branch, config.Refs)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error checking branch filter: %v", err)
	}

	if !allowed {
		return http.StatusOK, nil
	}

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (g *GitHub) handlePullRequestEvent(ctx triggers.WebhookRequestContext, data map[string]any) (int, error) {
	err := ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func isRefDeletionEvent(data map[string]any) bool {
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

func getRef(data map[string]any) string {
	refValue, ok := data["ref"]
	if !ok {
		return ""
	}

	ref, ok := refValue.(string)
	if !ok {
		return ""
	}

	return ref
}

func isRefAllowed(ref string, filters []*RefFilter) (bool, error) {
	//
	// If no filters configured, allow all refs
	//
	if len(filters) == 0 {
		return true, nil
	}

	for _, filter := range filters {
		switch filter.Type {
		case FilterTypeExactMatch:
			if ref == filter.Value {
				return true, nil
			}

		case FilterTypeRegex:
			matched, err := regexp.MatchString(filter.Value, ref)
			if err != nil {
				return false, fmt.Errorf("invalid regex pattern '%s': %w", filter.Value, err)
			}
			if matched {
				return true, nil
			}
		}
	}

	return false, nil
}

func validateRefFilters(filters []*RefFilter) error {
	for _, filter := range filters {
		if filter.Type == FilterTypeRegex {
			if _, err := regexp.Compile(filter.Value); err != nil {
				return fmt.Errorf("invalid regex pattern '%s': %w", filter.Value, err)
			}
		}
	}
	return nil
}
