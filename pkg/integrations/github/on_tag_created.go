package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnTagCreated struct{}

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

func (t *OnTagCreated) Documentation() string {
	return `The On Tag Created trigger starts a workflow execution when a new tag is created in a GitHub repository.

## Use Cases

- **Version tagging**: Trigger workflows when version tags are created
- **Release automation**: Automatically create releases from tags
- **Deployment triggers**: Deploy specific versions based on tags
- **Tag processing**: Process or validate tags as they're created

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Tags**: Configure which tags to listen for using predicates (e.g., equals "v*", starts with "release-")

## Event Data

Each tag event includes:
- **ref**: The tag reference (e.g., "refs/tags/v1.0.0")
- **ref_type**: Type of reference (tag)
- **repository**: Repository information
- **sender**: User who created the tag

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
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
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
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
	err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnTagCreatedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
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

func (t *OnTagCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = withWebhookLogger(ctx, t.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config := OnTagCreatedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		ctx.Logger.Errorf("Missing X-GitHub-Event header")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "create" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a create event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := verifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	//
	// Check ref_type - only process tags, not branches
	//
	refType, ok := data["ref_type"]
	if !ok {
		ctx.Logger.Errorf("Missing ref_type")
		return http.StatusBadRequest, nil, fmt.Errorf("missing ref_type")
	}

	rt, ok := refType.(string)
	if !ok {
		ctx.Logger.Errorf("Invalid ref_type")
		return http.StatusBadRequest, nil, fmt.Errorf("invalid ref_type")
	}

	if rt != "tag" {
		ctx.Logger.Infof("Ignoring event - ref_type %q is not a tag", rt)
		return http.StatusOK, nil, nil
	}

	ref, ok := data["ref"]
	if !ok {
		ctx.Logger.Errorf("Missing ref")
		return http.StatusBadRequest, nil, fmt.Errorf("missing ref")
	}

	r, ok := ref.(string)
	if !ok {
		ctx.Logger.Errorf("Invalid ref")
		return http.StatusBadRequest, nil, fmt.Errorf("invalid ref")
	}

	if !configuration.MatchesAnyPredicate(config.Tags, r) {
		ctx.Logger.Infof("Ignoring event - ref %q did not match configured filters", r)
		return http.StatusOK, nil, nil
	}

	err = ctx.Events.Emit("github.tagCreated", data)
	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnTagCreated) Cleanup(ctx core.TriggerContext) error {
	return nil
}
