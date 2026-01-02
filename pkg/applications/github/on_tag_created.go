package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnTagCreated struct{}

type OnTagCreatedMetadata struct {
	Repository *Repository `json:"repository"`
}

type OnTagCreatedConfiguration struct {
	Repository string                    `json:"repository"`
	Tags       []configuration.Predicate `json:"tags"`
}

func (t *OnTagCreated) Name() string {
	return "github.onTagCreated"
}

func (t *OnTagCreated) Label() string {
	return "On Tag Created"
}

func (t *OnTagCreated) Description() string {
	return "Listen to GitHub tag creation events"
}

func (t *OnTagCreated) Icon() string {
	return "github"
}

func (t *OnTagCreated) Color() string {
	return "gray"
}

func (t *OnTagCreated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "tags",
			Label:    "Tags",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: true,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": ".*",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (t *OnTagCreated) Setup(ctx core.TriggerContext) error {
	var metadata OnTagCreatedMetadata
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

	config := OnTagCreatedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	appMetadata := Metadata{}
	err = mapstructure.Decode(ctx.AppInstallationContext.GetMetadata(), &appMetadata)
	if err != nil {
		return fmt.Errorf("error decoding app installation metadata: %v", err)
	}

	repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
		return r.Name == config.Repository
	})

	if repoIndex == -1 {
		return fmt.Errorf("repository %s is not accessible to app installation", config.Repository)
	}

	metadata.Repository = &appMetadata.Repositories[repoIndex]
	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType:  "create",
		Repository: config.Repository,
	})
}

func (t *OnTagCreated) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnTagCreated) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnTagCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnTagCreatedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "create")
	if err != nil {
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// Check ref_type - only process tags, not branches
	//
	refType, ok := data["ref_type"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing ref_type")
	}

	rt, ok := refType.(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid ref_type")
	}

	if rt != "tag" {
		return http.StatusOK, nil
	}

	//
	// Extract tag name from ref field
	//
	ref, ok := data["ref"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing ref")
	}

	r, ok := ref.(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid ref")
	}

	//
	// Match tag name against configured predicates
	//
	if !configuration.MatchesAnyPredicate(config.Tags, r) {
		return http.StatusOK, nil
	}

	err = ctx.EventContext.Emit("github.tagCreated", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
